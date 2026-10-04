<script setup lang="ts">
import { ArrowUpRight, Film, ImageOff, LoaderCircle } from 'lucide-vue-next'
import { ref, watch } from 'vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

const props = defineProps<{
  title: string
  subtitle: string
  previewUrl?: string
  videoUrl?: string
  loading?: boolean
  error?: string
  taskId?: string | null
}>()

defineEmits<{ viewDetail: [] }>()
const mediaFailed = ref(false)
watch(() => [props.previewUrl, props.videoUrl], () => { mediaFailed.value = false })
</script>

<template>
  <section class="s2v-studio-preview" aria-label="预览与结果">
    <div class="s2v-studio-section-heading">
      <h2>预览与结果</h2>
      <span v-if="taskId" class="s2v-studio-caption">{{ videoUrl ? '成片已交付' : '任务结果' }}</span>
      <span v-else-if="previewUrl" class="s2v-studio-caption">参考素材</span>
    </div>
    <div class="s2v-studio-screen" :aria-busy="loading">
      <video v-if="videoUrl && !mediaFailed" :key="videoUrl" :src="videoUrl" :poster="previewUrl" controls playsinline preload="metadata" aria-label="生成视频预览" @error="mediaFailed = true" />
      <img v-else-if="previewUrl && !mediaFailed" :src="previewUrl" :alt="title" @error="mediaFailed = true">
      <div v-else class="s2v-studio-empty">
        <LoaderCircle v-if="loading" class="h-8 w-8 animate-spin" />
        <ImageOff v-else-if="mediaFailed" class="h-8 w-8" />
        <Film v-else class="h-8 w-8" />
        <h3>{{ loading ? '正在读取任务结果' : mediaFailed ? '暂时无法预览此素材' : taskId ? title : '你的下一段影片，从这里开始' }}</h3>
        <p>{{ mediaFailed ? '请重新选取素材，或在任务详情获取最新成片。' : taskId ? subtitle : '填写画面描述或添加参考图片。真实素材和生成结果将在这里呈现。' }}</p>
      </div>
      <span v-if="loading && previewUrl" class="s2v-studio-preview-loading"><LoaderCircle class="h-4 w-4 animate-spin" />正在读取成片</span>
    </div>
    <div class="s2v-studio-preview-meta">
      <div><h3>{{ title }}</h3><p>{{ error || subtitle }}</p></div>
      <S2VButton v-if="taskId" size="sm" @click="$emit('viewDetail')">查看任务详情<ArrowUpRight class="h-4 w-4" /></S2VButton>
    </div>
  </section>
</template>
