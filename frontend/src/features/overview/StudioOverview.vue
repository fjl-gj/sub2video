<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, ArrowUpRight, Clapperboard, HardDrive, Images, Layers2, RefreshCw, TextCursorInput, Wallet } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import type { Task, TaskStatus } from '@/entities/types'
import { useAuthStore } from '@/stores/auth'
import ActiveTaskRow from './ActiveTaskRow.vue'
import RecentFilmPreview from './RecentFilmPreview.vue'
import type { OverviewData, StudioMode } from './types'
import S2VButton from '@/shared/ui/S2VButton.vue'

const props = defineProps<{ data: OverviewData | null; loading: boolean; error: string; modes: StudioMode[] }>()
defineEmits<{ reload: [] }>()
const router = useRouter()
const auth = useAuthStore()
const modeIcons = { text: TextCursorInput, multi_image: Images, first_last: Layers2 }
const latestFilm = computed(() => props.data?.tasks.recentCompleted[0] ?? null)
const otherFilms = computed(() => props.data?.tasks.recentCompleted.slice(1) ?? [])
const activeTask = computed<Task | null>(() => {
  const raw = props.data?.tasks.activeTask
  return raw ? { id: raw.id, title: raw.title, workflow: raw.mode, status: raw.status as TaskStatus, progress: raw.progress, duration: raw.duration, resolution: raw.resolution, cost: raw.cost, time: '', thumb: raw.thumbUrl || '' } : null
})
</script>

<template>
  <div class="s2v-studio-overview">
    <div v-if="error" class="s2v-studio-alert" role="alert">
      <span>{{ error }}</span>
      <S2VButton size="sm" :loading="loading" @click="$emit('reload')"><RefreshCw class="h-3.5 w-3.5" />重新加载</S2VButton>
    </div>
    <div class="s2v-studio-modes" aria-label="选择创作模式">
      <button v-for="mode in modes" :key="mode.type" class="s2v-studio-mode" type="button" :disabled="loading || !mode.available" @click="router.push({ path: '/app/create', query: { mode: mode.type } })">
        <span class="s2v-studio-mode-top"><span>{{ mode.label }}</span><component :is="modeIcons[mode.type]" class="h-4 w-4" /></span>
        <strong>{{ mode.title }}<ArrowUpRight class="h-4 w-4" /></strong>
        <span class="s2v-studio-mode-description">{{ mode.description }}</span>
        <span v-if="!mode.available || loading" class="s2v-studio-mode-availability">{{ mode.availability }}</span>
      </button>
    </div>
    <div class="s2v-studio-section-heading">
      <h2>最近成片</h2>
      <RouterLink to="/app/works" class="s2v-studio-text-link">查看全部任务 <ArrowRight class="h-3.5 w-3.5" /></RouterLink>
    </div>
    <div class="s2v-studio-results">
      <RecentFilmPreview :film="latestFilm" :loading="loading" :unavailable="!!error && !data" />
      <section class="s2v-studio-queue" aria-label="进行中的任务">
        <div class="s2v-studio-section-heading"><h2>进行中的任务</h2><span v-if="data" class="s2v-studio-count">{{ data.tasks.activeCount }}</span></div>
        <ActiveTaskRow v-if="activeTask" :task="activeTask" />
        <div v-else class="s2v-studio-queue-empty">
          <Clapperboard class="h-6 w-6" />
          <strong>{{ loading ? '正在读取任务' : error && !data ? '任务状态暂不可用' : '准备好下一个镜头' }}</strong>
          <p>{{ loading ? '稍候即可查看最新进度。' : error && !data ? '重新加载后查看最新任务状态。' : '当前没有进行中的任务，从上方选择一种方式开始创作。' }}</p>
        </div>
        <p v-if="activeTask" class="s2v-studio-queue-note">生成完成后，可在任务中预览、下载和保存成片。</p>
        <S2VButton v-if="activeTask" @click="router.push(`/app/works/${activeTask.id}`)">查看任务详情 <ArrowUpRight class="h-3.5 w-3.5" /></S2VButton>
        <S2VButton v-else @click="router.push('/app/create')">前往创作 <ArrowRight class="h-3.5 w-3.5" /></S2VButton>
      </section>
    </div>
    <div v-if="otherFilms.length" class="s2v-studio-recent-list">
      <RouterLink v-for="film in otherFilms" :key="film.id" :to="`/app/works/${film.id}`" class="s2v-studio-recent-item">
        <Clapperboard class="h-4 w-4" />
        <div><strong>{{ film.title }}</strong><span>{{ film.duration }} · {{ film.resolution }}</span></div>
        <ArrowUpRight class="h-4 w-4" />
      </RouterLink>
    </div>
    <footer class="s2v-studio-resources" aria-label="工作空间用量">
      <RouterLink to="/app/wallet"><Wallet class="h-3.5 w-3.5" /><span>创作余额</span><strong class="s2v-mono">{{ auth.user ? `¥ ${auth.balance.toFixed(2)}` : '—' }}</strong></RouterLink>
      <RouterLink to="/app/assets"><HardDrive class="h-3.5 w-3.5" /><span>存储空间</span><strong class="s2v-mono">{{ data ? `${data.storage.usedGb} / ${data.storage.quotaGb} GB` : '—' }}</strong></RouterLink>
      <span>{{ data ? `已保存 ${data.storage.fileCount} 个文件` : loading ? '正在同步工作空间' : '工作空间数据暂不可用' }}</span>
    </footer>
  </div>
</template>
