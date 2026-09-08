<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { RefreshCw, Route, Server, Workflow } from 'lucide-vue-next'
import {
  fetchAdminCreationModes,
  fetchAdminModelConfigurations,
  fetchAdminOfferings,
  fetchAdminSuppliers,
  fetchAdminTasks,
  type AdminCreationMode,
  type AdminModelConfiguration,
  type AdminOffering,
  type AdminSupplier,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const router = useRouter()
const app = useAppStore()
const loading = ref(false)
const suppliers = ref<AdminSupplier[]>([])
const offerings = ref<AdminOffering[]>([])
const modes = ref<AdminCreationMode[]>([])
const modelConfigurations = ref<AdminModelConfiguration[]>([])
const taskTotal = ref(0)
const queued = ref(0)
const running = ref(0)
const failed = ref(0)
const completed = ref(0)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const [s, o, m, configurations, all, q, run, fail, done] = await Promise.all([
      fetchAdminSuppliers(),
      fetchAdminOfferings(),
      fetchAdminCreationModes(),
      fetchAdminModelConfigurations(),
      fetchAdminTasks({ page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '排队中', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '生成中', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '失败', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '已完成', page: 1, pageSize: 1 }),
    ])
    suppliers.value = s.items
    offerings.value = o.items
    modes.value = m.items
    modelConfigurations.value = configurations.items
    taskTotal.value = all.total
    queued.value = q.total
    running.value = run.total
    failed.value = fail.total
    completed.value = done.total
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '加载运行大盘失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => { void load() })
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="运行大盘">
      <template #description>看任务量与供应商供给状态；调度细节在「供应商与模型」。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton size="sm" @click="router.push('/admin/supplier-models')">供应商链路</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <S2VCard variant="metric">
        <div class="text-xs font-medium text-s2v-muted">任务总量</div>
        <strong class="s2v-mono mt-2 block text-2xl font-bold tabular-nums text-s2v-ink">{{ taskTotal }}</strong>
        <span class="mt-1 block text-[10px] text-s2v-label">已完成 {{ completed }} · 失败 {{ failed }}</span>
      </S2VCard>
      <S2VCard variant="metric">
        <div class="text-xs font-medium text-s2v-muted">排队 / 生成中</div>
        <strong class="s2v-mono mt-2 block text-2xl font-bold tabular-nums text-s2v-primary">{{ queued }} / {{ running }}</strong>
        <span class="mt-1 block text-[10px] text-s2v-label">当前待调度与执行中</span>
      </S2VCard>
      <S2VCard variant="metric">
        <div class="text-xs font-medium text-s2v-muted">启用供应商</div>
        <strong class="s2v-mono mt-2 block text-2xl font-bold tabular-nums text-s2v-ink">{{ suppliers.filter((item) => item.enabled).length }}</strong>
        <span class="mt-1 block text-[10px] text-s2v-label">共 {{ suppliers.length }} 家接入</span>
      </S2VCard>
      <S2VCard variant="metric">
        <div class="text-xs font-medium text-s2v-muted">启用供给</div>
        <strong class="s2v-mono mt-2 block text-2xl font-bold tabular-nums text-s2v-ink">{{ offerings.filter((item) => item.enabled).length }}</strong>
        <span class="mt-1 block text-[10px] text-s2v-label">上架创作模式 {{ modes.filter((item) => item.status === 'published').length }}</span>
      </S2VCard>
    </div>

    <S2VCard variant="padded" class="mt-6">
      <div class="flex items-center justify-between gap-3">
        <h2 class="s2v-section-title">供应商供给链路</h2>
        <button type="button" class="text-xs font-semibold text-s2v-primary hover:underline" @click="router.push('/admin/supplier-models')">管理供应商 →</button>
      </div>
      <div class="mt-3 grid gap-3 sm:grid-cols-3">
        <div v-for="sp in suppliers" :key="sp.id" class="s2v-inset">
          <div class="flex items-center justify-between gap-2">
            <strong class="truncate text-xs">{{ sp.name }}</strong>
            <S2VBadge :tone="sp.enabled && sp.hasToken ? 'success' : 'warning'">
              {{ sp.enabled ? (sp.hasToken ? '就绪' : '缺 Token') : '停用' }}
            </S2VBadge>
          </div>
          <p class="mt-2 flex items-center gap-1.5 text-[11px] text-s2v-muted">
            <Server class="h-3.5 w-3.5 text-s2v-primary" />
            <span class="truncate font-mono">{{ sp.code }} · {{ sp.baseUrl || '未配置 URL' }}</span>
          </p>
          <p class="mt-1 text-[10px] text-s2v-label">
            关联供给 {{ offerings.filter((item) => item.supplierId === sp.id).length }} 条
          </p>
        </div>
        <p v-if="!suppliers.length" class="s2v-inset-dashed sm:col-span-3">暂无供应商，请先在「供应商与模型」接入。</p>
      </div>
    </S2VCard>

    <div class="mt-5 grid gap-4 lg:grid-cols-2">
      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">供给摘要</h2>
            <p class="mt-1 text-xs text-s2v-muted">供应商供给锁定模型、创作模式和上游工作流。</p>
          </div>
          <Route class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <div v-for="offering in offerings.slice(0, 6)" :key="offering.id" class="flex items-center gap-3 py-3">
            <div class="min-w-0 flex-1">
              <strong class="block truncate text-xs">{{ offering.name || offering.modelName }}</strong>
              <span class="mt-0.5 block truncate font-mono text-[10px] text-s2v-muted">{{ offering.supplierName }} · {{ offering.upstreamWorkflowId || '无 upstream id' }}</span>
            </div>
            <span class="shrink-0 font-mono text-[10px] text-s2v-muted">优先级 {{ offering.priority }}</span>
            <S2VBadge :tone="offering.enabled && offering.healthStatus !== 'unavailable' ? 'success' : 'warning'">{{ offering.enabled ? offering.healthStatus : '停用' }}</S2VBadge>
          </div>
          <p v-if="!offerings.length" class="py-4 text-xs text-s2v-muted">暂无供给。</p>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">工作流</h2>
            <p class="mt-1 text-xs text-s2v-muted">客户端固定工作流可配置展示和启用状态。</p>
          </div>
          <Workflow class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <div v-for="mode in modes.slice(0, 6)" :key="mode.id" class="flex items-center gap-3 py-3">
            <div class="min-w-0 flex-1">
              <strong class="block truncate text-xs">{{ mode.name }}</strong>
              <span class="mt-0.5 block truncate text-[10px] text-s2v-muted">可选模型 {{ modelConfigurations.filter((item) => item.workflowId === mode.id && item.enabled).length }} 个</span>
            </div>
            <S2VBadge :tone="mode.status === 'published' ? 'success' : 'warning'">{{ mode.status === 'published' ? '上架' : '停用' }}</S2VBadge>
          </div>
          <p v-if="!modes.length" class="py-4 text-xs text-s2v-muted">暂无创作模式。</p>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
