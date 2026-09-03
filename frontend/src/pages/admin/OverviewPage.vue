<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { RefreshCw, Route, Server, Workflow } from 'lucide-vue-next'
import {
  fetchAdminRoutes,
  fetchAdminSuppliers,
  fetchAdminTasks,
  fetchAdminWorkflows,
  type AdminRoute,
  type AdminSupplier,
  type AdminWorkflow,
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
const routes = ref<AdminRoute[]>([])
const workflows = ref<AdminWorkflow[]>([])
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
    const [s, r, w, all, q, run, fail, done] = await Promise.all([
      fetchAdminSuppliers(),
      fetchAdminRoutes(),
      fetchAdminWorkflows(),
      fetchAdminTasks({ page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '排队中', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '生成中', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '失败', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '已完成', page: 1, pageSize: 1 }),
    ])
    suppliers.value = s.items
    routes.value = r.items
    workflows.value = w.items
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
      <template #description>看任务量与供应商供给状态；调度细节在「模型与供应商」。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton size="sm" @click="router.push('/admin/providers')">供应商链路</S2VButton>
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
        <div class="text-xs font-medium text-s2v-muted">启用路由</div>
        <strong class="s2v-mono mt-2 block text-2xl font-bold tabular-nums text-s2v-ink">{{ routes.filter((item) => item.enabled).length }}</strong>
        <span class="mt-1 block text-[10px] text-s2v-label">上架工作流 {{ workflows.filter((item) => item.statusRaw === 'published' || item.status === '已上架').length }}</span>
      </S2VCard>
    </div>

    <S2VCard variant="padded" class="mt-6">
      <div class="flex items-center justify-between gap-3">
        <h2 class="s2v-section-title">供应商供给链路</h2>
        <button type="button" class="text-xs font-semibold text-s2v-primary hover:underline" @click="router.push('/admin/providers')">管理供应商 →</button>
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
            关联路由 {{ routes.filter((item) => item.supplierCode === sp.code).length }} 条
          </p>
        </div>
        <p v-if="!suppliers.length" class="s2v-inset-dashed sm:col-span-3">暂无供应商，请先在「模型与供应商」接入。</p>
      </div>
    </S2VCard>

    <div class="mt-5 grid gap-4 lg:grid-cols-2">
      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">路由摘要</h2>
            <p class="mt-1 text-xs text-s2v-muted">模型 → 供应商上游工作流，按成本选路。</p>
          </div>
          <Route class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <div v-for="rt in routes.slice(0, 6)" :key="rt.id" class="flex items-center gap-3 py-3">
            <div class="min-w-0 flex-1">
              <strong class="block truncate text-xs">{{ rt.name || rt.modelCode }}</strong>
              <span class="mt-0.5 block truncate font-mono text-[10px] text-s2v-muted">{{ rt.supplierCode }} · {{ rt.workflowId || '无 upstream id' }}</span>
            </div>
            <span class="shrink-0 font-mono text-[10px] text-s2v-muted">¥{{ rt.costPerSecPeak.toFixed(3) }}/s</span>
            <S2VBadge :tone="rt.enabled ? 'success' : 'warning'">{{ rt.enabled ? '启用' : '停用' }}</S2VBadge>
          </div>
          <p v-if="!routes.length" class="py-4 text-xs text-s2v-muted">暂无路由。</p>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">平台工作流</h2>
            <p class="mt-1 text-xs text-s2v-muted">客户端可见能力由上架状态控制。</p>
          </div>
          <Workflow class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <div v-for="wf in workflows.slice(0, 6)" :key="wf.id" class="flex items-center gap-3 py-3">
            <div class="min-w-0 flex-1">
              <strong class="block truncate text-xs">{{ wf.name }}</strong>
              <span class="mt-0.5 block truncate text-[10px] text-s2v-muted">{{ wf.carrier }} · {{ wf.modelCode }}</span>
            </div>
            <S2VBadge :tone="wf.status === '已上架' || wf.statusRaw === 'published' ? 'success' : 'warning'">{{ wf.status }}</S2VBadge>
          </div>
          <p v-if="!workflows.length" class="py-4 text-xs text-s2v-muted">暂无工作流。</p>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
