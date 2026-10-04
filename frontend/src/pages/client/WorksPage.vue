<script setup lang="ts">
import { Ban, BookmarkPlus, Download, ExternalLink, MoreHorizontal, Plus, RefreshCw } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { cancelTask as cancelTaskApi, listTasks } from '@/api/client'
import { downloadTaskFile } from '@/features/delivery/download'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import type { Task, TaskStatus } from '@/entities/types'
import { PAGE_SIZE } from '@/shared/lib/paginate'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

type WorksTask = Task & { savedToProject?: boolean; canCancel: boolean; reviewRequired: boolean }

const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()

const filter = ref('全部')
const page = ref(1)
const loading = ref(false)
const loadError = ref('')
const operationError = ref('')
const downloadingTask = ref('')
const tabs = ['全部', '排队中', '生成中', '已完成', '失败']
const refundOpen = ref(false)
const refundTask = ref<WorksTask | null>(null)
const refundBusy = ref(false)
const refundError = ref('')

const items = ref<WorksTask[]>([])
const total = ref(0)
const counts = ref<Record<string, number>>(Object.fromEntries(tabs.map((tab) => [tab, 0])))

const paged = computed(() => items.value)
const empty = computed(() => !loading.value && !loadError.value && items.value.length === 0)

let pollTimer: ReturnType<typeof setInterval> | null = null
let loadVersion = 0
let disposed = false

function mapTask(raw: {
  id: string
  title: string
  mode: string
  status: string
  progress: number
  duration: string
  resolution: string
  cost: number
  time: string
  thumbUrl: string
  savedToProject?: boolean
  canCancel?: boolean
  reviewRequired?: boolean
}): WorksTask {
  return {
    id: raw.id,
    title: raw.title,
    workflow: raw.mode,
    status: raw.status as TaskStatus,
    progress: raw.progress,
    duration: raw.duration,
    resolution: raw.resolution,
    cost: raw.cost,
    time: raw.time,
    thumb: raw.thumbUrl,
    savedToProject: raw.savedToProject,
    canCancel: raw.canCancel === true,
    reviewRequired: raw.reviewRequired === true,
  }
}

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function loadTasks() {
  const request = ++loadVersion
  loading.value = true
  try {
    const data = await listTasks({
      status: filter.value === '全部' ? undefined : filter.value,
      page: page.value,
      pageSize: PAGE_SIZE,
    })
    if (disposed || request !== loadVersion) return
    loadError.value = ''
    items.value = data.items.map(mapTask)
    total.value = data.total
    counts.value = {
      全部: data.counts['全部'] ?? data.total,
      排队中: data.counts['排队中'] ?? 0,
      生成中: data.counts['生成中'] ?? 0,
      已完成: data.counts['已完成'] ?? 0,
      失败: data.counts['失败'] ?? 0,
    }
  }
  catch {
    if (!disposed && request === loadVersion) loadError.value = '任务列表暂时无法读取，请重新加载。'
  }
  finally {
    if (!disposed && request === loadVersion) loading.value = false
  }
}

function reconfigureCreation(task: WorksTask) {
  void router.push({ path: `/app/works/${task.id}`, query: { action: 'reuse' } })
}

function openDetail(task: WorksTask) {
  void router.push(`/app/works/${task.id}`)
}

async function downloadResult(task: WorksTask) {
  if (downloadingTask.value) return
  downloadingTask.value = task.id
  operationError.value = ''
  try {
    await downloadTaskFile(task.id)
  } catch {
    operationError.value = '下载暂时不可用，请打开任务详情刷新状态后重试。'
  } finally { downloadingTask.value = '' }
}

function saveToProject(task: WorksTask) {
  void router.push({ path: `/app/works/${task.id}`, query: task.savedToProject ? {} : { action: 'save' } })
}

function askRefund(task: WorksTask) {
  if (!task.canCancel || task.reviewRequired) return
  refundTask.value = task
  refundError.value = ''
  refundOpen.value = true
}

async function confirmRefund() {
  if (!refundTask.value || refundBusy.value) return
  refundBusy.value = true
  refundError.value = ''
  const balanceMutation = auth.beginBalanceMutation()
  try {
    const result = await cancelTaskApi(refundTask.value.id)
    auth.acceptBalanceMutation(balanceMutation, result.wallet.balance)
    app.notify(`已终止 ${result.task.id}，预扣费用已退回钱包`)
    refundOpen.value = false
    refundTask.value = null
    await loadTasks()
  }
  catch (error) {
    refundError.value = apiErrorMessage(error, '取消失败，请重新加载任务状态后再试。')
    await loadTasks()
  }
  finally {
    auth.finishBalanceMutation(balanceMutation)
    refundBusy.value = false
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    const active = (counts.value['排队中'] ?? 0) + (counts.value['生成中'] ?? 0)
    if (active > 0 && !loading.value && !loadError.value) void loadTasks()
  }, 4000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

watch(filter, () => {
  page.value = 1
})

watch([filter, page], () => {
  void loadTasks()
})

onMounted(() => {
  void loadTasks()
  startPolling()
})

onBeforeUnmount(() => {
  disposed = true
  loadVersion++
  stopPolling()
})
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="任务">
      <template #description>查看生成进展、成片与退款记录。任务进入执行后暂不可取消，提交结果待核对时请勿重复提交。</template>
      <template #actions>
        <S2VButton variant="primary" @click="router.push('/app/create')">
          <template #icon><Plus class="h-4 w-4" /></template>
          新建创作
        </S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard v-if="loadError" variant="padded" class="mb-4" role="alert">
      <p>{{ loadError }}{{ items.length ? '当前显示上次读取的任务。' : '' }}</p>
      <S2VButton class="mt-3" :loading="loading" @click="loadTasks">重新加载任务</S2VButton>
    </S2VCard>
    <S2VCard v-if="operationError" variant="padded" class="mb-4" role="alert"><p>{{ operationError }}</p></S2VCard>
    <p v-if="downloadingTask" class="mb-4" role="status">正在准备下载…</p>
    <p v-if="loading && !items.length" class="mb-4" role="status">正在读取任务…</p>

    <div class="mb-4">
      <S2VSegmented v-model="filter" :options="tabs" :counts="counts" />
    </div>

    <div class="s2v-table-wrap hidden md:block">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>任务</th>
            <th>创作模式</th>
            <th>状态</th>
            <th>参数</th>
            <th>费用</th>
            <th>时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in paged" :key="task.id">
            <td>
              <div class="flex items-center gap-3">
                <img :src="task.thumb" :alt="task.title" class="s2v-thumb h-9 w-14">
                <div>
                  <strong class="block text-xs text-s2v-ink">{{ task.title }}</strong>
                  <span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ task.id }}</span>
                </div>
              </div>
            </td>
            <td>{{ task.workflow }}</td>
            <td>
              <S2VBadge :tone="task.reviewRequired ? 'warning' : task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">
                {{ task.reviewRequired ? '待核对' : task.status }}<span v-if="task.status === '生成中' && !task.reviewRequired" class="ml-1">{{ task.progress }}%</span>
              </S2VBadge>
            </td>
            <td>{{ task.resolution }} · {{ task.duration }}</td>
            <td class="font-mono">¥ {{ task.cost.toFixed(2) }}</td>
            <td>{{ task.time }}</td>
            <td>
              <div class="flex items-center gap-1">
                <S2VIconButton v-if="task.status === '失败' && !task.reviewRequired" label="重新配置创作" @click="reconfigureCreation(task)"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-else-if="task.canCancel && !task.reviewRequired" label="终止任务并退款" @click="askRefund(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
                <template v-else-if="task.status === '已完成' && !task.reviewRequired">
                  <S2VIconButton label="下载结果" :disabled="!!downloadingTask" @click="downloadResult(task)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
                  <S2VIconButton :label="task.savedToProject ? '查看已保存作品' : '保存到作品'" @click="saveToProject(task)"><BookmarkPlus class="h-3.5 w-3.5" /></S2VIconButton>
                </template>
                <S2VIconButton label="查看详情" @click="openDetail(task)"><ExternalLink class="h-3.5 w-3.5" /></S2VIconButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid gap-3 md:hidden">
      <MobileRecordCard v-for="task in paged" :key="task.id" :title="task.title" :meta="task.id">
        <template #badge>
          <S2VBadge :tone="task.reviewRequired ? 'warning' : task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">
            {{ task.reviewRequired ? '待核对' : task.status }}<span v-if="task.status === '生成中' && !task.reviewRequired" class="ml-1">{{ task.progress }}%</span>
          </S2VBadge>
        </template>
        <template #body>{{ task.workflow }} · {{ task.resolution }} · {{ task.duration }} · ¥ {{ task.cost.toFixed(2) }}</template>
        <template #actions>
          <S2VIconButton v-if="task.status === '失败' && !task.reviewRequired" label="重新配置创作" @click="reconfigureCreation(task)"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
          <S2VIconButton v-else-if="task.canCancel && !task.reviewRequired" label="终止任务并退款" @click="askRefund(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
          <template v-else-if="task.status === '已完成' && !task.reviewRequired">
            <S2VIconButton label="下载结果" :disabled="!!downloadingTask" @click="downloadResult(task)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
            <S2VIconButton :label="task.savedToProject ? '查看已保存作品' : '保存到作品'" @click="saveToProject(task)"><BookmarkPlus class="h-3.5 w-3.5" /></S2VIconButton>
          </template>
          <S2VIconButton label="查看详情" @click="openDetail(task)"><ExternalLink class="h-3.5 w-3.5" /></S2VIconButton>
        </template>
      </MobileRecordCard>
    </div>

    <S2VPagination v-if="total" v-model="page" class="mt-4" :total="total" :page-size="PAGE_SIZE" />

    <S2VCard v-if="empty" variant="padded" class="mt-4 grid place-items-center text-center">
      <MoreHorizontal class="h-6 w-6 text-s2v-muted" />
      <p class="mt-2 text-xs text-s2v-muted">暂无符合条件的任务</p>
    </S2VCard>

    <S2VDialog
      v-model:open="refundOpen"
      tone="danger"
      title="终止渲染任务确认"
      :description="refundTask ? `申请取消任务 ${refundTask.id}。成功取消后费用退回创作余额；若任务已进入执行或待核对阶段，将无法取消。` : ''"
      hide-actions
    >
      <p v-if="refundError" role="alert">{{ refundError }}</p>
      <div class="mt-4 flex flex-wrap justify-end gap-2">
        <S2VButton :disabled="refundBusy" @click="refundOpen = false">返回</S2VButton>
        <S2VButton v-if="!refundError" variant="danger" :loading="refundBusy" @click="confirmRefund">终止并退款</S2VButton>
        <S2VButton v-else @click="refundOpen = false; refundTask && openDetail(refundTask)">查看最新任务状态</S2VButton>
      </div>
    </S2VDialog>
  </section>
</template>
