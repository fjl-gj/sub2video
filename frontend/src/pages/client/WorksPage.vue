<script setup lang="ts">
import { Download, ExternalLink, Filter, MoreHorizontal, Plus, RefreshCw } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'

const router = useRouter(); const app = useAppStore(); const filter = ref('全部')
const tabs = ['全部', '生成中', '已完成', '失败']
const list = computed(() => filter.value === '全部' ? app.tasks : app.tasks.filter((task) => task.status === filter.value))
function retry() { app.notify('任务已重新排队，费用不会重复扣除') }
</script>
<template>
  <section class="s2v-page"><header class="s2v-page-header"><div><p class="s2v-eyebrow">创作空间 / 作品</p><h1 class="s2v-page-title">我的作品</h1><p class="s2v-subcopy">所有生成中的任务和已完成结果都在这里，失败任务可以重试或申请退款。</p></div><S2VButton variant="primary" @click="router.push('/app/create')"><template #icon><Plus class="h-4 w-4" /></template>新建创作</S2VButton></header>
    <div class="mb-4 flex items-center justify-between gap-3"><div class="flex max-w-full items-center gap-1 overflow-x-auto rounded-[7px] border border-s2v-line bg-s2v-surface p-1"><button v-for="tab in tabs" :key="tab" :class="['whitespace-nowrap rounded px-3 py-1.5 text-xs font-semibold', filter === tab ? 'bg-s2v-primary-soft text-s2v-primary' : 'text-s2v-muted']" @click="filter = tab">{{ tab }} <span class="ml-1 font-mono">{{ tab === '全部' ? app.tasks.length : app.tasks.filter((t) => t.status === tab).length }}</span></button></div><button class="hidden items-center gap-1 text-xs text-s2v-muted sm:flex"><Filter class="h-3.5 w-3.5" />筛选</button></div>
    <div class="s2v-table-wrap"><table class="s2v-table"><thead><tr><th>作品</th><th>工作流</th><th>状态</th><th>参数</th><th>费用</th><th>时间</th><th>操作</th></tr></thead><tbody><tr v-for="task in list" :key="task.id"><td><div class="flex items-center gap-3"><img :src="task.thumb" :alt="task.title" class="h-9 w-14 rounded object-cover" /><div><strong class="block text-xs text-s2v-ink">{{ task.title }}</strong><span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ task.id }}</span></div></div></td><td>{{ task.workflow }}</td><td><S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">{{ task.status }}<span v-if="task.status === '生成中'" class="ml-1">{{ task.progress }}%</span></S2VBadge></td><td>{{ task.resolution }} · {{ task.duration }}</td><td class="font-mono">¥ {{ task.cost.toFixed(2) }}</td><td>{{ task.time }}</td><td><div class="flex items-center gap-1"><S2VIconButton v-if="task.status === '失败'" label="重新生成" @click="retry"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton><S2VIconButton v-else label="下载作品" @click="app.notify('原型阶段仅模拟下载')"><Download class="h-3.5 w-3.5" /></S2VIconButton><S2VIconButton label="查看详情" @click="app.notify(`已打开 ${task.id} 详情`)"><ExternalLink class="h-3.5 w-3.5" /></S2VIconButton></div></td></tr></tbody></table></div>
    <div v-if="!list.length" class="s2v-panel mt-4 grid place-items-center p-12 text-center"><MoreHorizontal class="h-6 w-6 text-s2v-muted" /><p class="mt-2 text-xs text-s2v-muted">暂无符合条件的作品</p></div>
  </section>
</template>
