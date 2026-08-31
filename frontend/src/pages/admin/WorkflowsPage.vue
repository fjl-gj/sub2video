<script setup lang="ts">
import { Edit3, Plus, Workflow } from 'lucide-vue-next'
import { workflows } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
const app = useAppStore()
</script>
<template><section class="s2v-page"><header class="s2v-page-header"><div><p class="s2v-eyebrow">创作管理 / 工作流</p><h1 class="s2v-page-title">工作流</h1><p class="s2v-subcopy">定义客户端能看到的输入、参数、输出规格和上架状态。</p></div><S2VButton variant="primary" @click="app.notify('工作流编辑器将在后端接入后开放')"><template #icon><Plus class="h-4 w-4" /></template>新建工作流</S2VButton></header><div class="grid gap-4 lg:grid-cols-3"><S2VCard v-for="item in workflows" :key="item.id" class="p-4"><div class="flex items-start justify-between gap-3"><span class="grid h-9 w-9 place-items-center rounded-[7px] bg-emerald-50 text-emerald-600"><Workflow class="h-4 w-4" /></span><S2VBadge :tone="item.status === '已上架' ? 'success' : item.status === '草稿' ? 'warning' : 'neutral'">{{ item.status }}</S2VBadge></div><h2 class="mt-4 text-sm font-bold">{{ item.name }}</h2><p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ item.description }}</p><div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-[11px] text-s2v-muted"><div class="flex justify-between"><span>模型</span><strong class="text-s2v-ink">{{ item.provider }}</strong></div><div class="flex justify-between"><span>售价</span><strong class="font-mono text-s2v-ink">{{ item.price }}</strong></div><div class="flex justify-between"><span>更新</span><span>{{ item.updated }}</span></div></div><S2VButton class="mt-4 w-full" size="sm" @click="app.notify(`已打开 ${item.name} 配置`)"><Edit3 class="h-3.5 w-3.5" />编辑配置</S2VButton></S2VCard></div></section></template>
