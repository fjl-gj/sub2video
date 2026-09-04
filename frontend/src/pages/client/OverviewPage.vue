<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, Plus } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { fetchOverview } from '@/api/client'
import type { Task, TaskStatus } from '@/entities/types'
import ActiveTaskRow from '@/features/overview/ActiveTaskRow.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

type OverviewTask = {
  id: string
  title: string
  workflow: string
  status: string
  progress: number
  duration: string
  resolution: string
  cost: number
  thumbUrl?: string
}

type OverviewData = {
  workspace: { displayName: string }
  billing: { totalSpent: number; monthSpent: number }
  wallet: { balance: number }
  pricing: { referencePricePerSecond: number; referenceNote: string }
  storage: { usedGb: number; quotaGb: number; fileCount: number }
  tasks: {
    activeCount: number
    activeTask: OverviewTask | null
    recentCompleted: Array<{
      id: string
      title: string
      status: string
      resolution: string
      duration: string
      cost: number
      thumbUrl: string
    }>
  }
}

const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()

const data = ref<OverviewData | null>(null)
const loading = ref(false)

const workspaceName = computed(() => data.value?.workspace.displayName || auth.displayName || '创作者')
const activeTask = computed<Task | null>(() => {
  const raw = data.value?.tasks.activeTask
  if (!raw) return null
  return {
    id: raw.id,
    title: raw.title,
    workflow: raw.workflow,
    status: raw.status as TaskStatus,
    progress: raw.progress,
    duration: raw.duration,
    resolution: raw.resolution,
    cost: raw.cost,
    time: '',
    thumb: raw.thumbUrl || '',
  }
})
const recentWorks = computed(() => data.value?.tasks.recentCompleted ?? [])

function statusTone(status: string) {
  return status === '失败' ? 'danger' : status === '已完成' ? 'success' : 'primary'
}

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

const metrics = computed(() => {
  const d = data.value
  const balance = d?.wallet.balance ?? auth.balance ?? app.balance
  const activeCount = d?.tasks.activeCount ?? 0
  return [
    {
      label: '累计支出',
      value: `¥ ${(d?.billing.totalSpent ?? 0).toFixed(2)}`,
      note: `本月消耗 ¥${(d?.billing.monthSpent ?? 0).toFixed(2)}`,
    },
    {
      label: '可用余额',
      value: `¥ ${balance.toFixed(2)}`,
      note: '立即充值 →',
      tone: 'success' as const,
      to: '/app/wallet',
    },
    {
      label: '每秒视频均价',
      value: `¥ ${(d?.pricing.referencePricePerSecond ?? 0.5).toFixed(2)}`,
      unit: '/s',
      note: d?.pricing.referenceNote || '768P / 1080P 按秒计',
    },
    {
      label: '空间资源占用',
      value: String(d?.storage.usedGb ?? 0),
      unit: `/ ${d?.storage.quotaGb ?? 5}GB`,
      note: `已保存 ${d?.storage.fileCount ?? 0} 个文件`,
    },
    {
      label: '进行中的任务',
      value: String(activeCount),
      note: '查看队列 →',
      tone: 'warning' as const,
      to: '/app/works',
    },
  ]
})

async function load() {
  loading.value = true
  try {
    const overview = await fetchOverview() as OverviewData
    data.value = overview
    app.balance = overview.wallet.balance
    auth.setBalance(overview.wallet.balance)
    app.storageUsed = overview.storage.usedGb
    app.storageQuota = overview.storage.quotaGb
    void app.loadProjects()
  }
  catch (error) {
    app.notify(apiErrorMessage(error, '加载概览失败'))
  }
  finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader :title="`欢迎回来，${workspaceName}`">
      <template #actions>
        <S2VButton class="flex-1 sm:flex-none" @click="router.push('/app/works')">查看任务队列 <ArrowRight class="h-3.5 w-3.5" /></S2VButton>
        <S2VButton variant="primary" class="flex-1 sm:flex-none" @click="router.push('/app/create')">
          <template #icon><Plus class="h-4 w-4" /></template>
          新建生成任务
        </S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid grid-cols-2 gap-3 sm:grid-cols-5">
      <S2VCard v-for="(item, index) in metrics" :key="item.label" variant="metric" :class="index === metrics.length - 1 ? 'col-span-2 sm:col-span-1' : ''">
        <div class="text-[11px] font-medium text-s2v-label">{{ item.label }}</div>
        <strong :class="['s2v-mono mt-1 block text-xl font-bold tabular-nums', item.tone === 'success' ? 'text-s2v-success' : item.tone === 'warning' ? 'text-s2v-warning' : 'text-s2v-ink']">
          {{ item.value }}<span v-if="item.unit" class="ml-1 text-xs font-normal text-s2v-label">{{ item.unit }}</span>
        </strong>
        <button v-if="item.to" type="button" class="mt-0.5 block text-[10px] font-medium text-s2v-primary hover:underline" @click="router.push(item.to)">{{ item.note }}</button>
        <span v-else class="mt-0.5 block text-[10px] text-s2v-label">{{ item.note }}</span>
      </S2VCard>
    </div>

    <S2VCard variant="padded" class="mt-6">
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-s2v-body">
          <span class="h-2.5 w-2.5 animate-ping rounded-full bg-s2v-warning" />进行中的任务
        </div>
        <button type="button" class="text-[11px] font-semibold text-s2v-primary hover:underline" @click="router.push('/app/works')">查看全部 →</button>
      </div>
      <ActiveTaskRow v-if="activeTask" class="mt-3" :task="activeTask" />
      <div v-else class="s2v-inset-dashed mt-3">{{ loading ? '加载中…' : '当前没有进行中的任务' }}</div>
    </S2VCard>

    <div class="mt-6 flex items-center justify-between">
      <h2 class="s2v-section-title">最近完成任务</h2>
      <button type="button" class="text-xs font-semibold text-s2v-primary hover:underline" @click="router.push('/app/works')">进入任务 →</button>
    </div>
    <div class="mt-3 grid gap-4 sm:grid-cols-2">
      <S2VCard v-for="work in recentWorks" :key="work.id" variant="compact" class="flex items-center gap-3">
        <img :src="work.thumbUrl" :alt="work.title" class="s2v-thumb h-16 w-24 shrink-0">
        <div class="min-w-0 flex-1">
          <strong class="block truncate text-xs">#{{ work.id.slice(-4) }}_{{ work.title }}.mp4</strong>
          <span class="mt-1 block text-[11px] text-s2v-muted">{{ work.resolution }} · {{ work.duration }} · ¥{{ work.cost.toFixed(2) }}</span>
          <S2VBadge class="mt-2" :tone="statusTone(work.status)">{{ work.status === '已完成' ? '生成成功' : work.status }}</S2VBadge>
        </div>
        <button type="button" class="text-[11px] font-semibold text-s2v-primary">下载</button>
      </S2VCard>
      <S2VCard v-if="!loading && !recentWorks.length" variant="padded" class="grid place-items-center text-center sm:col-span-2">
        <p class="text-xs text-s2v-muted">暂无已完成的任务</p>
      </S2VCard>
    </div>
  </section>
</template>
