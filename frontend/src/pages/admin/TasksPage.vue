<script setup lang="ts">
import { Ban, Eye, RefreshCw, RotateCcw, Search, X } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import {
  cancelAdminTask,
  fetchAdminTask,
  fetchAdminTasks,
  retryAdminTask,
  type AdminTaskDetail,
  type AdminTaskItem,
} from '@/api/admin'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

const app = useAppStore()
const filter = ref('全部')
const search = ref('')
const tabs = ['全部', '排队中', '生成中', '已完成', '失败']
const items = ref<AdminTaskItem[]>([])
const total = ref(0)
const loading = ref(false)
const detail = ref<AdminTaskDetail | null>(null)
const detailOpen = ref(false)
const cancelTarget = ref<AdminTaskItem | null>(null)
const cancelOpen = ref(false)
const acting = ref(false)

const list = computed(() => items.value.filter((task) => !search.value || `${task.id}${task.title}${task.user}`.includes(search.value)))

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminTasks({
      status: filter.value === '全部' ? undefined : filter.value,
      page: 1,
      pageSize: 50,
    })
    items.value = data.items
    total.value = data.total
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '加载任务失败'))
  } finally {
    loading.value = false
  }
}

async function openDetail(id: string) {
  try {
    detail.value = await fetchAdminTask(id)
    detailOpen.value = true
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '加载任务详情失败'))
  }
}

function askCancel(task: AdminTaskItem) {
  cancelTarget.value = task
  cancelOpen.value = true
}

async function confirmCancel() {
  if (!cancelTarget.value) return
  acting.value = true
  try {
    const res = await cancelAdminTask(cancelTarget.value.id)
    app.notify(`已终止并退款 ¥${res.refund.toFixed(2)}`)
    cancelOpen.value = false
    await load()
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '终止失败'))
  } finally {
    acting.value = false
  }
}

async function doRetry(id: string) {
  acting.value = true
  try {
    await retryAdminTask(id)
    app.notify('已重新入队')
    await load()
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '重试失败'))
  } finally {
    acting.value = false
  }
}

watch(filter, () => { void load() })
onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="任务中心">
      <template #description>平台任务号为主；上游任务号仅作排障参考。可终止退款或失败重试。</template>
      <template #actions>
        <S2VBadge tone="primary">{{ total }} 条任务</S2VBadge>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <S2VSegmented v-model="filter" :options="tabs" />
      <S2VInput v-model="search" class="ml-auto max-w-xs flex-1" size="sm" placeholder="任务编号 / 用户">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap hidden md:block">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>任务编号</th>
            <th>用户</th>
            <th>供应商</th>
            <th>上游</th>
            <th>状态</th>
            <th>费用</th>
            <th>提交时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in list" :key="task.id">
            <td>
              <strong class="block text-xs text-s2v-ink">{{ task.title }}</strong>
              <span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ task.id }}</span>
              <span class="mt-0.5 block text-[10px] text-s2v-muted">{{ task.workflow }}</span>
            </td>
            <td>
              <span class="block text-xs">{{ task.user }}</span>
              <span class="mt-0.5 block text-[10px] text-s2v-muted">{{ task.userEmail }}</span>
            </td>
            <td class="font-mono text-[11px]">{{ task.provider }}</td>
            <td class="font-mono text-[10px] text-s2v-muted">{{ task.upstreamTaskId || '-' }}</td>
            <td><S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">{{ task.status }}</S2VBadge></td>
            <td class="font-mono">¥{{ task.cost.toFixed(2) }}</td>
            <td>{{ task.time }}</td>
            <td>
              <div class="flex items-center gap-1">
                <S2VIconButton label="详情" @click="openDetail(task.id)"><Eye class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-if="task.status === '排队中' || task.status === '生成中'" label="终止退款" @click="askCancel(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-if="task.status === '失败'" label="重试" :disabled="acting" @click="doRetry(task.id)"><RotateCcw class="h-3.5 w-3.5" /></S2VIconButton>
              </div>
            </td>
          </tr>
          <tr v-if="!list.length">
            <td colspan="8" class="text-center text-s2v-muted">暂无任务。</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid gap-3 md:hidden">
      <MobileRecordCard v-for="task in list" :key="task.id" :title="task.title" :meta="task.id">
        <template #badge>
          <S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">{{ task.status }}</S2VBadge>
        </template>
        <template #body>{{ task.user }} · {{ task.provider }} · ¥{{ task.cost.toFixed(2) }} · {{ task.time }}</template>
        <template #actions>
          <S2VIconButton label="详情" @click="openDetail(task.id)"><Eye class="h-3.5 w-3.5" /></S2VIconButton>
          <S2VIconButton v-if="task.status === '排队中' || task.status === '生成中'" label="终止退款" @click="askCancel(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
          <S2VIconButton v-if="task.status === '失败'" label="重试" :disabled="acting" @click="doRetry(task.id)"><RotateCcw class="h-3.5 w-3.5" /></S2VIconButton>
        </template>
      </MobileRecordCard>
      <p v-if="!list.length" class="text-xs text-s2v-muted">暂无任务。</p>
    </div>

    <S2VDialog
      v-model:open="cancelOpen"
      tone="danger"
      title="终止任务并退款"
      :description="cancelTarget ? `确认终止 ${cancelTarget.id}？将按扣费金额退回创作者钱包。` : ''"
      confirm-label="终止并退款"
      @confirm="confirmCancel"
    />

    <div v-if="detailOpen && detail" class="fixed inset-0 z-40 flex items-end justify-center bg-s2v-ink/40 p-4 sm:items-center" @click.self="detailOpen = false">
      <S2VCard class="max-h-[85vh] w-full max-w-lg overflow-y-auto">
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">任务详情</h2>
            <p class="mt-1 font-mono text-[11px] text-s2v-muted">{{ detail.id }}</p>
          </div>
          <S2VIconButton label="关闭" @click="detailOpen = false"><X class="h-3.5 w-3.5" /></S2VIconButton>
        </div>
        <div class="s2v-panel-body grid gap-3 text-xs">
          <div class="grid grid-cols-2 gap-2">
            <div><span class="text-s2v-muted">标题</span><p class="mt-0.5 font-medium">{{ detail.title }}</p></div>
            <div><span class="text-s2v-muted">状态</span><p class="mt-0.5"><S2VBadge :tone="detail.status === '失败' ? 'danger' : detail.status === '已完成' ? 'success' : 'primary'">{{ detail.status }}</S2VBadge></p></div>
            <div><span class="text-s2v-muted">用户</span><p class="mt-0.5">{{ detail.user }} · {{ detail.userEmail }}</p></div>
            <div><span class="text-s2v-muted">售价 / 成本</span><p class="mt-0.5 font-mono">¥{{ detail.cost.toFixed(2) }} / {{ detail.costPerSec.toFixed(4) }}/s</p></div>
            <div><span class="text-s2v-muted">供应商</span><p class="mt-0.5 font-mono">{{ detail.supplierCode || detail.provider || '-' }}</p></div>
            <div><span class="text-s2v-muted">上游任务</span><p class="mt-0.5 font-mono break-all">{{ detail.upstreamTaskId || '-' }}</p></div>
            <div><span class="text-s2v-muted">上游工作流</span><p class="mt-0.5 font-mono break-all">{{ detail.upstreamWorkflowId || '-' }}</p></div>
            <div><span class="text-s2v-muted">分辨率</span><p class="mt-0.5">{{ detail.resolution }} · {{ detail.durationSec }}s</p></div>
          </div>
          <div v-if="detail.failReason"><span class="text-s2v-muted">失败原因</span><p class="mt-0.5 text-s2v-danger">{{ detail.failReason }}</p></div>
          <div v-if="detail.resultOssUrl || detail.resultUrl">
            <span class="text-s2v-muted">结果地址</span>
            <p class="mt-0.5 break-all font-mono text-[10px]">{{ detail.resultOssUrl || detail.resultUrl }}</p>
          </div>
          <div>
            <span class="text-s2v-muted">价格快照</span>
            <pre class="s2v-note mt-1 overflow-x-auto font-mono text-[10px]">{{ JSON.stringify(detail.priceSnapshot, null, 2) }}</pre>
          </div>
          <div>
            <span class="text-s2v-muted">路由快照</span>
            <pre class="s2v-note mt-1 overflow-x-auto font-mono text-[10px]">{{ JSON.stringify(detail.routeSnapshot, null, 2) }}</pre>
          </div>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
