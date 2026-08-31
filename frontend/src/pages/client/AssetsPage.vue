<script setup lang="ts">
import { FileImage, FileMusic, FileVideo, Folder, Plus, Search, Upload } from 'lucide-vue-next'
import { ref } from 'vue'
import { assets } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
const app = useAppStore(); const search = ref(''); const selectedFolder = ref('全部')
const folders = ['全部', '场景', '人物', '产品', '视频', '品牌', '音频']
function filtered() { return assets.filter((item) => (selectedFolder.value === '全部' || item.folder === selectedFolder.value) && (!search.value || item.name.includes(search.value))) }
const iconFor = (type: string) => type === '视频' ? FileVideo : type === '音频' ? FileMusic : FileImage
</script>
<template><section class="s2v-page"><header class="s2v-page-header"><div><p class="s2v-eyebrow">创作空间 / 资产</p><h1 class="s2v-page-title">资产库</h1><p class="s2v-subcopy">只保存你主动留下的素材，任务临时文件不会无限占用空间。</p></div><S2VButton variant="primary" @click="app.notify('已模拟打开文件选择器')"><template #icon><Upload class="h-4 w-4" /></template>上传资产</S2VButton></header>
  <div class="mb-5 grid gap-3 rounded-panel border border-s2v-line bg-s2v-surface p-4 sm:grid-cols-[1fr_auto] sm:items-center"><div><div class="flex items-center gap-2"><Folder class="h-4 w-4 text-s2v-primary" /><strong class="text-xs">存储空间</strong><S2VBadge tone="primary">1.9 / 5 GB</S2VBadge></div><div class="mt-3 h-1.5 overflow-hidden rounded-full bg-slate-100"><span class="block h-full w-[38%] rounded-full bg-s2v-primary" /></div></div><span class="text-[11px] text-s2v-muted">任务输入 7 天自动清理 · 结果 30 天自动清理</span></div>
  <div class="mb-4 flex flex-wrap items-center gap-2"><div class="flex max-w-full items-center gap-1 overflow-x-auto"><button v-for="folder in folders" :key="folder" :class="['whitespace-nowrap rounded-full border px-3 py-1.5 text-xs font-semibold', selectedFolder === folder ? 'border-s2v-primary bg-s2v-primary-soft text-s2v-primary' : 'border-s2v-line bg-s2v-surface text-s2v-muted']" @click="selectedFolder = folder">{{ folder }}</button></div><label class="ml-auto flex min-w-[180px] items-center gap-2 rounded-[7px] border border-s2v-line bg-s2v-surface px-2.5 py-1.5 sm:max-w-[230px]"><Search class="h-3.5 w-3.5 text-s2v-muted" /><input v-model="search" class="min-w-0 flex-1 bg-transparent text-xs outline-none" placeholder="搜索资产" /></label></div>
  <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"><article v-for="item in filtered()" :key="item.id" class="overflow-hidden rounded-panel border border-s2v-line bg-s2v-surface"><div class="grid aspect-[4/3] place-items-center" :style="{ background: `linear-gradient(135deg, #${item.color}, #${item.accent})` }"><component :is="iconFor(item.type)" class="h-8 w-8 text-white/80" /></div><div class="p-3"><div class="flex items-center justify-between gap-2"><strong class="truncate text-xs">{{ item.name }}</strong><S2VBadge>{{ item.type }}</S2VBadge></div><div class="mt-1.5 flex items-center justify-between text-[11px] text-s2v-muted"><span>{{ item.size }} · {{ item.folder }}</span><button aria-label="资产更多操作" class="rounded p-1 text-s2v-muted hover:bg-s2v-canvas"><Plus class="h-3.5 w-3.5 rotate-45" /></button></div></div></article></div>
  <div v-if="!filtered().length" class="s2v-panel mt-4 grid place-items-center p-12 text-center"><Folder class="h-6 w-6 text-s2v-muted" /><p class="mt-2 text-xs text-s2v-muted">暂无匹配资产</p></div>
</section></template>
