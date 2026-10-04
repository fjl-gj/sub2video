<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ArrowUpRight, Clapperboard, Download, Film, LoaderCircle, RefreshCw } from 'lucide-vue-next'
import { fetchTask, type ClientTaskDetail } from '@/api/client'
import S2VButton from '@/shared/ui/S2VButton.vue'
import type { RecentFilm } from './types'

const props = defineProps<{ film: RecentFilm | null; loading: boolean; unavailable: boolean }>()
const detail = ref<ClientTaskDetail | null>(null)
const detailLoading = ref(false)
const detailFailed = ref(false)
const mediaFailed = ref(false)
let requestVersion = 0

function safeMediaURL(value?: string) {
  if (!value) return ''
  try {
    const url = new URL(value, window.location.origin)
    return ['http:', 'https:'].includes(url.protocol) ? url.href : ''
  } catch { return '' }
}

const videoURL = computed(() => detail.value?.deliveryStatus === 'ready' ? safeMediaURL(detail.value.downloadUrl) : '')
const posterURL = computed(() => {
  const url = safeMediaURL(props.film?.thumbUrl)
  return url && !/\.(mp4|webm|mov)(?:[?#]|$)/i.test(url) ? url : ''
})
const emptyTitle = computed(() => {
  if (props.loading || detailLoading.value) return '正在读取成片'
  if (props.unavailable) return '暂时无法读取成片'
  if (!props.film) return '你的第一段成片，从这里开始'
  if (mediaFailed.value || detailFailed.value) return '预览暂不可用'
  return '成片正在准备交付'
})

async function loadDetail(id = props.film?.id) {
  const version = ++requestVersion
  detail.value = null
  detailFailed.value = false
  mediaFailed.value = false
  detailLoading.value = !!id
  if (!id) return
  try {
    const task = await fetchTask(id)
    if (version === requestVersion) {
      detail.value = task
      mediaFailed.value = false
    }
  } catch {
    if (version === requestVersion) detailFailed.value = true
  } finally {
    if (version === requestVersion) detailLoading.value = false
  }
}

watch([() => props.film?.id, () => props.loading], ([id, overviewLoading]) => {
  if (overviewLoading) {
    requestVersion++
    return
  }
  void loadDetail(id)
}, { immediate: true })

onBeforeUnmount(() => { requestVersion++ })
</script>

<template>
  <article class="s2v-studio-film">
    <div class="s2v-studio-screen">
      <video v-if="videoURL && !mediaFailed" :key="videoURL" :src="videoURL" controls playsinline preload="metadata" :aria-label="`${film?.title} 成片预览`" @error="mediaFailed = true" />
      <img v-else-if="posterURL && !mediaFailed" :src="posterURL" :alt="film?.title" @error="mediaFailed = true">
      <div v-else class="s2v-studio-screen-empty" role="status">
        <LoaderCircle v-if="loading || detailLoading" class="h-7 w-7 animate-spin" />
        <Film v-else-if="film" class="h-8 w-8" />
        <Clapperboard v-else class="h-8 w-8" />
        <strong>{{ emptyTitle }}</strong>
        <p>{{ film ? '可进入任务详情查看交付状态。' : unavailable ? '请重新加载工作空间。' : '完成一次生成后，视频会出现在这里。' }}</p>
      </div>
    </div>
    <div v-if="film" class="s2v-studio-film-footer">
      <div class="s2v-studio-film-caption"><strong>{{ film.title }}</strong><span>{{ film.duration }} · {{ film.resolution }}</span></div>
      <div class="s2v-studio-film-actions">
        <S2VButton v-if="detailFailed || mediaFailed" size="sm" variant="ghost" :loading="detailLoading" @click="loadDetail()"><RefreshCw class="h-3.5 w-3.5" />重新加载预览</S2VButton>
        <a v-if="videoURL" :href="videoURL" target="_blank" rel="noopener noreferrer" download class="s2v-studio-text-link"><Download class="h-3.5 w-3.5" />下载成片</a>
        <RouterLink :to="`/app/works/${film.id}`" class="s2v-studio-text-link">任务详情 <ArrowUpRight class="h-3.5 w-3.5" /></RouterLink>
      </div>
    </div>
  </article>
</template>
