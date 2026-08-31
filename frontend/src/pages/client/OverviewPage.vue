<script setup lang="ts">
import { ArrowRight, Clock3, FolderOpen, Images, Plus, Sparkles, WalletCards } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { sceneTemplates } from '@/data/mock'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'

const router = useRouter()
const app = useAppStore()
const latest = app.tasks.slice(0, 3)
function applyTemplate(template: (typeof sceneTemplates)[number]) {
  app.draftPrompt = `${template.title}，${template.description}`
  router.push('/app/create')
  app.notify(`已将「${template.title}」带入创作台`)
}
</script>

<template>
  <section class="s2v-page">
    <header class="s2v-page-header">
      <div><p class="s2v-eyebrow">创作者工作台 / 2026 年 8 月 31 日</p><h1 class="s2v-page-title">下午好，林默</h1><p class="s2v-subcopy">把灵感变成下一条视频，所有生成结果都会在作品里持续跟踪。</p></div>
      <S2VButton variant="primary" size="lg" @click="router.push('/app/create')"><template #icon><Plus class="h-4 w-4" /></template>开始创作</S2VButton>
    </header>

    <div class="grid gap-4 rounded-panel border border-s2v-primary/20 bg-s2v-primary-soft/50 p-5 md:grid-cols-[1fr_auto] md:items-center">
      <div><div class="mb-2 flex items-center gap-2"><S2VBadge tone="primary">H3 VIDEO</S2VBadge><span class="text-xs font-semibold text-s2v-primary">当前推荐工作流</span></div><h2 class="text-lg font-bold">让一张图片开始运动</h2><p class="mt-1 max-w-xl text-xs leading-6 text-s2v-muted">选择图片、输入画面描述，H3 会为你生成自然的镜头运动。生成前会展示预计费用，余额不足时不会提交任务。</p></div>
      <S2VButton variant="secondary" @click="router.push('/app/create')">打开图生视频 <ArrowRight class="h-3.5 w-3.5" /></S2VButton>
    </div>

    <div class="my-5 grid grid-cols-2 gap-3 md:grid-cols-4">
      <S2VCard v-for="item in [{label:'钱包余额',value:`¥ ${app.balance.toFixed(2)}`,note:'可用于生成任务',icon:WalletCards},{label:'进行中',value:String(app.activeTasks.length),note:'个任务正在处理',icon:Clock3},{label:'已完成作品',value:String(app.completedTasks.length),note:'最近 30 天',icon:Images},{label:'资产空间',value:'1.9 GB',note:'剩余 3.1 GB',icon:FolderOpen}]" :key="item.label" class="p-4"><div class="flex items-center justify-between text-xs text-s2v-muted"><span>{{ item.label }}</span><component :is="item.icon" class="h-4 w-4 text-s2v-primary" /></div><strong class="mt-3 block text-xl font-bold tabular-nums">{{ item.value }}</strong><span class="mt-1 block text-[11px] text-s2v-muted">{{ item.note }}</span></S2VCard>
    </div>

    <div class="s2v-grid-2">
      <S2VCard><div class="s2v-panel-header"><div><h2 class="s2v-section-title">最近生成</h2><p class="mt-1 text-xs text-s2v-muted">从这里继续跟进任务状态</p></div><button class="text-xs font-semibold text-s2v-primary" @click="router.push('/app/works')">查看全部 <ArrowRight class="inline h-3.5 w-3.5" /></button></div><div class="divide-y divide-s2v-line/70 px-4"><div v-for="task in latest" :key="task.id" class="flex items-center gap-3 py-3"><img :src="task.thumb" :alt="task.title" class="h-10 w-16 rounded object-cover" /><div class="min-w-0 flex-1"><strong class="block truncate text-xs">{{ task.title }}</strong><span class="mt-1 block text-[11px] text-s2v-muted">{{ task.id }} · {{ task.time }}</span></div><S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">{{ task.status }}</S2VBadge></div></div></S2VCard>
      <S2VCard><div class="s2v-panel-header"><div><h2 class="s2v-section-title">今日灵感</h2><p class="mt-1 text-xs text-s2v-muted">从一个场景开始，不必从空白开始</p></div><Sparkles class="h-4 w-4 text-s2v-primary" /></div><div class="grid gap-3 p-4"> <button v-for="template in sceneTemplates.slice(0, 3)" :key="template.id" class="flex items-center gap-3 rounded-[7px] border border-s2v-line p-2.5 text-left transition hover:border-s2v-primary/50 hover:bg-s2v-primary-soft/40" @click="applyTemplate(template)"><span class="grid h-9 w-12 shrink-0 place-items-center rounded bg-slate-800 text-[9px] font-mono text-white">{{ template.category }}</span><span class="min-w-0 flex-1"><strong class="block truncate text-xs">{{ template.title }}</strong><span class="mt-1 block truncate text-[11px] text-s2v-muted">{{ template.description }}</span></span><ArrowRight class="h-3.5 w-3.5 text-s2v-muted" /></button></div></S2VCard>
    </div>
  </section>
</template>
