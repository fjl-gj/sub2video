<script setup lang="ts">
import { ArrowRight, Filter, Search, Sparkles } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { ref } from 'vue'
import { sceneTemplates } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'

const router = useRouter()
const app = useAppStore()
const search = ref('')
const category = ref('全部')
const categories = ['全部', '电影感', '自然', '商业', '生活', '叙事', '艺术']
const filtered = () => sceneTemplates.filter((item) => (category.value === '全部' || item.category === category.value) && (!search.value || `${item.title}${item.description}`.includes(search.value)))
function useScene(item: (typeof sceneTemplates)[number]) { app.draftPrompt = `${item.title}，${item.description}`; router.push('/app/create'); app.notify(`已套用「${item.title}」`) }
</script>
<template>
  <section class="s2v-page">
    <header class="s2v-page-header"><div><p class="s2v-eyebrow">创作空间 / 创意园</p><h1 class="s2v-page-title">从灵感开始</h1><p class="s2v-subcopy">浏览可复用的镜头结构和提示词，把喜欢的场景带进创作台。</p></div><S2VBadge tone="primary"><Sparkles class="h-3 w-3" />官方精选</S2VBadge></header>
    <div class="mb-5 flex flex-wrap items-center gap-2"><div class="flex max-w-full items-center gap-2 overflow-x-auto rounded-[7px] border border-s2v-line bg-s2v-surface p-1"><button v-for="item in categories" :key="item" :class="['whitespace-nowrap rounded px-2.5 py-1.5 text-xs font-semibold', category === item ? 'bg-s2v-primary-soft text-s2v-primary' : 'text-s2v-muted hover:text-s2v-ink']" @click="category = item">{{ item }}</button></div><div class="ml-auto flex min-w-[210px] flex-1 items-center gap-2 rounded-[7px] border border-s2v-line bg-s2v-surface px-2.5 py-1.5 sm:max-w-[250px]"><Search class="h-3.5 w-3.5 text-s2v-muted" /><input v-model="search" class="min-w-0 flex-1 bg-transparent text-xs outline-none" placeholder="搜索场景" /><Filter class="h-3.5 w-3.5 text-s2v-muted" /></div></div>
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"> <article v-for="item in filtered()" :key="item.id" class="overflow-hidden rounded-panel border border-s2v-line bg-s2v-surface shadow-subtle"><div class="grid aspect-video place-items-center text-xs font-mono text-white" :style="{ background: `linear-gradient(135deg, #${item.color}, #${item.accent})` }"><span class="rounded-full border border-white/30 bg-black/20 px-2 py-1">{{ item.category }} · H3</span></div><div class="p-4"><h2 class="text-sm font-bold">{{ item.title }}</h2><p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ item.description }}</p><div class="mt-3 flex items-center justify-between"><span class="text-[11px] text-s2v-muted">768P · 15 秒</span><S2VButton size="sm" variant="secondary" @click="useScene(item)">套用 <ArrowRight class="h-3.5 w-3.5" /></S2VButton></div></div></article></div>
  </section>
</template>
