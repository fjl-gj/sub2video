<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Activity, Bot, RefreshCw, Server, Wifi } from 'lucide-vue-next'
import {
  fetchAdminOfferings,
  fetchAdminSuppliers,
  fetchAdminTasks,
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
const queued = ref(0)
const running = ref(0)
const failed = ref(0)
const completed = ref(0)

const metrics = computed(() => [
  { label: '启用供应商', value: String(suppliers.value.filter((item) => item.enabled).length), icon: Server },
  { label: '启用供给', value: String(offerings.value.filter((item) => item.enabled).length), icon: Bot },
  { label: '队列中', value: String(queued.value + running.value), icon: Activity },
  { label: '失败任务', value: String(failed.value), icon: Wifi },
])

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function supplierStatus(sp: AdminSupplier) {
  if (!sp.enabled) return { label: '停用', tone: 'warning' as const }
  if (!sp.hasToken) return { label: '缺 Token', tone: 'warning' as const }
  const linked = offerings.value.filter((item) => item.supplierId === sp.id && item.enabled)
  if (!linked.length) return { label: '无供给', tone: 'warning' as const }
  if (linked.some((item) => item.healthStatus === 'unavailable')) return { label: '供给异常', tone: 'danger' as const }
  if (linked.some((item) => item.healthStatus === 'degraded')) return { label: '降级', tone: 'warning' as const }
  return { label: '就绪', tone: 'success' as const }
}

async function load() {
  loading.value = true
  try {
    const [s, o, q, run, fail, done] = await Promise.all([
      fetchAdminSuppliers(),
      fetchAdminOfferings(),
      fetchAdminTasks({ status: '排队中', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '生成中', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '失败', page: 1, pageSize: 1 }),
      fetchAdminTasks({ status: '已完成', page: 1, pageSize: 1 }),
    ])
    suppliers.value = s.items
    offerings.value = o.items
    queued.value = q.total
    running.value = run.total
    failed.value = fail.total
    completed.value = done.total
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '加载监控失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => { void load() })
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="运维监控">
      <template #description>监控供应商凭据、供给健康与任务队列，不是智算集群拓扑。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton size="sm" @click="router.push('/admin/supplier-models')">供应商配置</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <S2VCard v-for="item in metrics" :key="item.label" variant="metric">
        <div class="flex items-center justify-between text-xs text-s2v-muted">
          <span>{{ item.label }}</span>
          <component :is="item.icon" class="h-4 w-4 text-s2v-primary" />
        </div>
        <strong class="s2v-stat-value">{{ item.value }}</strong>
        <span class="s2v-stat-note">实时数据</span>
      </S2VCard>
    </div>

    <div class="mt-5 grid gap-4 lg:grid-cols-2">
      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">供应商健康</h2>
            <p class="mt-1 text-xs text-s2v-muted">凭据、供给启用状态和 Worker 累计失败保护。</p>
          </div>
          <Wifi class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <div v-for="sp in suppliers" :key="sp.id" class="flex items-center gap-3 py-4">
            <span :class="supplierStatus(sp).tone === 'success' ? 's2v-icon-well s2v-icon-well-success' : 's2v-icon-well s2v-icon-well-muted'">
              <Server class="h-4 w-4" />
            </span>
            <div class="min-w-0 flex-1">
              <strong class="block text-xs">{{ sp.name }}</strong>
              <span class="mt-1 block truncate font-mono text-[11px] text-s2v-muted">{{ sp.code }} · {{ sp.baseUrl || '-' }}</span>
              <span class="mt-1 block text-[10px] text-s2v-label">启用供给 {{ offerings.filter((item) => item.supplierId === sp.id && item.enabled).length }} 条</span>
            </div>
            <S2VBadge :tone="supplierStatus(sp).tone">{{ supplierStatus(sp).label }}</S2VBadge>
          </div>
          <p v-if="!suppliers.length" class="py-4 text-xs text-s2v-muted">暂无供应商。</p>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">任务队列</h2>
            <p class="mt-1 text-xs text-s2v-muted">按平台任务状态汇总，不展示上游节点拓扑。</p>
          </div>
          <Activity class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="s2v-panel-body grid gap-3 text-xs">
          <div class="flex items-center justify-between rounded-s2v-nav bg-s2v-canvas px-3 py-3">
            <span class="text-s2v-muted">排队中</span>
            <strong class="font-mono">{{ queued }}</strong>
          </div>
          <div class="flex items-center justify-between rounded-s2v-nav bg-s2v-canvas px-3 py-3">
            <span class="text-s2v-muted">生成中</span>
            <strong class="font-mono">{{ running }}</strong>
          </div>
          <div class="flex items-center justify-between rounded-s2v-nav bg-s2v-canvas px-3 py-3">
            <span class="text-s2v-muted">已完成</span>
            <strong class="font-mono text-s2v-success">{{ completed }}</strong>
          </div>
          <div class="flex items-center justify-between rounded-s2v-nav bg-s2v-canvas px-3 py-3">
            <span class="text-s2v-muted">失败</span>
            <strong class="font-mono text-s2v-danger">{{ failed }}</strong>
          </div>
          <S2VButton size="sm" class="mt-1" @click="router.push('/admin/tasks')">打开任务中心</S2VButton>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
