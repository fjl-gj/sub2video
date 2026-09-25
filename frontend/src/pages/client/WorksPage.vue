<script setup lang="ts">
import { Ban, BookmarkPlus, Download, ExternalLink, MoreHorizontal, Plus, RefreshCw } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { cancelTask as cancelTaskApi, fetchTask, listTasks } from '@/api/client'
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

type WorksTask = Task & { savedToProject?: boolean }

const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()

const filter = ref('全部')
const page = ref(1)
const loading = ref(false)
const tabs = ['全部', '排队中', '生成中', '已完成', '失败']
const refundOpen = ref(false)
const refundTask = ref<WorksTask | null>(null)

const items = ref<WorksTask[]>([])
const total = ref(0)
const counts = ref<Record<string, number>>(Object.fromEntries(tabs.map((tab) => [tab, 0])))

const paged = computed(() => items.value)
const empty = computed(() => !loading.value && items.value.length === 0)

let pollTimer: ReturnType<typeof setInterval> | null = null

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
  }
}

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function syncBalance(balance: number) {
  app.balance = balance
  auth.setBalance(balance)
}

async function loadTasks() {
  loading.value = true
  try {
    const data = await listTasks({
      status: filter.value === '全部' ? undefined : filter.value,
      page: page.value,
      pageSize: PAGE_SIZE,
    })
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
  catch (error) {
    app.notify(apiErrorMessage(error, '加载任务失败'))
  }
  finally {
    loading.value = false
  }
}

function reconfigureCreation() {
  app.notify('请重新配置并重新预估费用后再提交')
  router.push('/app/create')
}

function openDetail(task: WorksTask) {
  void router.push(`/app/works/${task.id}`)
}

async function downloadResult(task: WorksTask) {
  try {
    const detail = await fetchTask(task.id)
    if (!detail.downloadUrl) {
      app.notify('成片正在完成对象存储交付，暂不可下载')
      return
    }
    window.open(detail.downloadUrl, '_blank', 'noopener,noreferrer')
  } catch (error) {
    app.notify(apiErrorMessage(error, '获取下载地址失败'))
  }
}

async function saveToProject(task: WorksTask) {
  const ok = await app.saveWorkToAsset(task.id)
  if (ok) {
    task.savedToProject = true
    await loadTasks()
  }
}

function askRefund(task: WorksTask) {
  refundTask.value = task
  refundOpen.value = true
}

async function confirmRefund() {
  if (!refundTask.value) return
  try {
    const result = await cancelTaskApi(refundTask.value.id)
    syncBalance(result.wallet.balance)
    app.notify(`已终止 ${result.task.id}，预扣费用已退回钱包`)
    await loadTasks()
  }
  catch (error) {
    app.notify(apiErrorMessage(error, '终止失败'))
  }
  finally {
    refundTask.value = null
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    const active = (counts.value['排队中'] ?? 0) + (counts.value['生成中'] ?? 0)
    if (active > 0) void loadTasks()
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
  stopPolling()
})
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="任务">
      <template #description>所有生成中的任务和已完成结果都在这里，失败任务可以重试或申请退款。完成的结果可保存到当前作品。</template>
      <template #actions>
        <S2VButton variant="primary" @click="router.push('/app/create')">
          <template #icon><Plus class="h-4 w-4" /></template>
          新建创作
        </S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-4">
      <S2VSegmented v-model="filter" :options="tabs" :counts="counts" />
    </div>

    <div class="s2v-table-wrap hidden md:block">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>任务</th>
            <th>工作流</th>
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
              <S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">
                {{ task.status }}<span v-if="task.status === '生成中'" class="ml-1">{{ task.progress }}%</span>
              </S2VBadge>
            </td>
            <td>{{ task.resolution }} · {{ task.duration }}</td>
            <td class="font-mono">¥ {{ task.cost.toFixed(2) }}</td>
            <td>{{ task.time }}</td>
            <td>
              <div class="flex items-center gap-1">
                <S2VIconButton v-if="task.status === '失败'" label="重新配置创作" @click="reconfigureCreation"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-else-if="task.status === '生成中' || task.status === '排队中'" label="终止任务并退款" @click="askRefund(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
                <template v-else>
                  <S2VIconButton label="下载结果" @click="downloadResult(task)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
                  <S2VIconButton :label="task.savedToProject || app.isWorkSaved(task.id) ? '已保存到作品' : '保存到当前作品'" @click="saveToProject(task)"><BookmarkPlus class="h-3.5 w-3.5" /></S2VIconButton>
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
          <S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">
            {{ task.status }}<span v-if="task.status === '生成中'" class="ml-1">{{ task.progress }}%</span>
          </S2VBadge>
        </template>
        <template #body>{{ task.workflow }} · {{ task.resolution }} · {{ task.duration }} · ¥ {{ task.cost.toFixed(2) }}</template>
        <template #actions>
          <S2VIconButton v-if="task.status === '失败'" label="重新配置创作" @click="reconfigureCreation"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
          <S2VIconButton v-else-if="task.status === '生成中' || task.status === '排队中'" label="终止任务并退款" @click="askRefund(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
          <template v-else>
            <S2VIconButton label="下载结果" @click="downloadResult(task)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
            <S2VIconButton :label="task.savedToProject || app.isWorkSaved(task.id) ? '已保存到作品' : '保存到当前作品'" @click="saveToProject(task)"><BookmarkPlus class="h-3.5 w-3.5" /></S2VIconButton>
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
      :description="refundTask ? `确定要终止任务 ${refundTask.id} 吗？已预扣金额将即时全额退回钱包。` : ''"
      confirm-label="终止并退款"
      @confirm="confirmRefund"
    />
  </section>
</template>
