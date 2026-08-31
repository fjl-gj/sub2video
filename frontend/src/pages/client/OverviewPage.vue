<script setup lang="ts">
import { ArrowRight, Clock3, Plus } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const router = useRouter()
const app = useAppStore()
const activeTask = app.activeTasks[0]
const recentWorks = app.tasks.filter((task) => task.status === '已完成').slice(0, 2)

function statusTone(status: string) {
  return status === '失败' ? 'danger' : status === '已完成' ? 'success' : 'primary'
}

const metrics = [
  { label: '累计支出 (RMB)', value: '¥ 148.00', note: '本月消耗 ¥36.00' },
  { label: '可用余额 (RMB)', value: `¥ ${app.balance.toFixed(2)}`, note: '立即充值 →', tone: 'success' as const, to: '/app/wallet' },
  { label: '每秒视频均价', value: '¥ 0.20', unit: '/s', note: '768P/1080P 套餐按秒计' },
  { label: '空间资源占用', value: '1.9', unit: '/ 5GB', note: '已保存 6 个资产' },
  { label: '活跃渲染任务', value: String(app.activeTasks.length), note: '查看队列 →', tone: 'warning' as const, to: '/app/works' },
]
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="S2V Video Studio" kicker="多模态文生/图生视频创作系统" title="欢迎回来，李维工作室">
      <template #actions>
        <S2VButton class="flex-1 sm:flex-none" @click="router.push('/app/works')">查看任务队列 <ArrowRight class="h-3.5 w-3.5" /></S2VButton>
        <S2VButton variant="primary" class="flex-1 sm:flex-none" @click="router.push('/app/create')">
          <template #icon><Plus class="h-4 w-4" /></template>
          新建生成任务
        </S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid grid-cols-2 gap-3 sm:grid-cols-5">
      <S2VCard v-for="item in metrics" :key="item.label" variant="metric">
        <div class="text-[11px] font-medium text-s2v-muted">{{ item.label }}</div>
        <strong :class="['s2v-mono mt-2 block text-xl font-bold tabular-nums', item.tone === 'success' ? 'text-s2v-success' : item.tone === 'warning' ? 'text-s2v-warning' : 'text-s2v-ink']">
          {{ item.value }}<span v-if="item.unit" class="ml-1 text-xs font-normal text-s2v-muted">{{ item.unit }}</span>
        </strong>
        <button v-if="item.to" type="button" class="mt-1 block text-[10px] font-medium text-s2v-primary hover:underline" @click="router.push(item.to)">{{ item.note }}</button>
        <span v-else class="mt-1 block text-[10px] text-s2v-muted">{{ item.note }}</span>
      </S2VCard>
    </div>

    <S2VCard variant="padded" class="mt-6">
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-s2v-ink">
          <span class="h-2.5 w-2.5 rounded-full bg-s2v-warning" />正在执行的渲染流水 (预扣款机制)
        </div>
        <button type="button" class="text-[11px] font-semibold text-s2v-primary hover:underline" @click="router.push('/app/works')">进入完整任务中心 →</button>
      </div>
      <div v-if="activeTask" class="mt-3 flex flex-col gap-3 rounded-xl border border-s2v-line/60 bg-s2v-canvas p-3.5 sm:flex-row sm:items-center">
        <div class="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-slate-900 text-s2v-primary-soft"><Clock3 class="h-5 w-5" /></div>
        <div class="min-w-0 flex-1">
          <strong class="block truncate text-xs">{{ activeTask.title }} · {{ activeTask.resolution }} / {{ activeTask.duration }} (含音频)</strong>
          <span class="mt-1 block truncate text-[11px] text-s2v-muted">已预扣 ¥ {{ activeTask.cost.toFixed(2) }} · {{ activeTask.workflow }} · 进度 {{ activeTask.progress }}%</span>
        </div>
        <div class="flex items-center gap-3 sm:w-48">
          <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-s2v-line"><span class="block h-full rounded-full bg-s2v-primary" :style="{ width: `${activeTask.progress}%` }" /></div>
          <span class="s2v-mono text-[11px] text-s2v-primary">{{ activeTask.progress }}%</span>
        </div>
      </div>
      <div v-else class="mt-3 rounded-xl border border-dashed border-s2v-line p-5 text-center text-xs text-s2v-muted">当前没有进行中的任务</div>
    </S2VCard>

    <div class="mt-6 flex items-center justify-between">
      <h2 class="s2v-section-title">最近生成作品</h2>
      <button type="button" class="text-xs font-semibold text-s2v-primary hover:underline" @click="router.push('/app/works')">进入作品 →</button>
    </div>
    <div class="mt-3 grid gap-4 sm:grid-cols-2">
      <S2VCard v-for="work in recentWorks" :key="work.id" class="flex items-center gap-3 p-3.5">
        <img :src="work.thumb" :alt="work.title" class="h-16 w-24 shrink-0 rounded-lg object-cover" />
        <div class="min-w-0 flex-1">
          <strong class="block truncate text-xs">#{{ work.id.slice(-4) }}_{{ work.title }}.mp4</strong>
          <span class="mt-1 block text-[11px] text-s2v-muted">{{ work.resolution }} · {{ work.duration }} · ¥{{ work.cost.toFixed(2) }}</span>
          <S2VBadge class="mt-2" :tone="statusTone(work.status)">{{ work.status === '已完成' ? '生成成功' : work.status }}</S2VBadge>
        </div>
        <button type="button" class="text-[11px] font-semibold text-s2v-primary">下载</button>
      </S2VCard>
    </div>
  </section>
</template>
