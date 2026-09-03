<script setup lang="ts">
import { ArrowRight, Play } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import type { SceneTemplate } from '@/entities/types'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'

const props = defineProps<{ item: SceneTemplate }>()
defineEmits<{ apply: [] }>()

const videoRef = ref<HTMLVideoElement | null>(null)
const playing = ref(false)

const mediaStyle = computed(() => ({
  aspectRatio: props.item.aspectRatio.replace(':', ' / '),
  background: `linear-gradient(135deg, #${props.item.color}, #${props.item.accent})`,
}))

async function playPreview() {
  const video = videoRef.value
  if (!video) return
  try {
    await video.play()
    playing.value = true
  }
  catch {
    playing.value = false
  }
}

function stopPreview() {
  const video = videoRef.value
  if (!video) return
  video.pause()
  video.currentTime = 0
  playing.value = false
}
</script>

<template>
  <S2VCard variant="media">
    <div
      class="group relative w-full overflow-hidden"
      :style="mediaStyle"
      @mouseenter="playPreview"
      @mouseleave="stopPreview"
    >
      <video
        ref="videoRef"
        class="absolute inset-0 h-full w-full object-cover"
        :poster="item.poster"
        :src="item.preview"
        muted
        loop
        playsinline
        preload="metadata"
      />
      <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-s2v-ink/50 via-transparent to-s2v-ink/20" />
      <div class="pointer-events-none absolute left-2 top-2 flex flex-wrap gap-1">
        <span class="s2v-media-tag text-[10px] font-mono text-white">{{ item.category }}</span>
        <span class="s2v-media-tag text-[10px] font-mono text-white">{{ item.aspectRatio }}</span>
      </div>
      <div class="pointer-events-none absolute right-2 top-2">
        <span class="s2v-media-tag text-[10px] font-mono text-white">{{ item.resolution }} · {{ item.duration }}</span>
      </div>
      <div
        class="pointer-events-none absolute inset-0 grid place-items-center transition-opacity"
        :class="playing ? 'opacity-0' : 'opacity-100'"
      >
        <span class="grid h-10 w-10 place-items-center rounded-full bg-s2v-ink/45 text-white">
          <Play class="h-4 w-4" />
        </span>
      </div>
    </div>
    <div class="s2v-panel-body">
      <h2 class="text-sm font-bold">{{ item.title }}</h2>
      <p class="mt-1.5 line-clamp-2 text-xs leading-5 text-s2v-muted">{{ item.description }}</p>
      <div class="mt-3 flex items-center justify-end">
        <S2VButton size="sm" @click="$emit('apply')">套用 <ArrowRight class="h-3.5 w-3.5" /></S2VButton>
      </div>
    </div>
  </S2VCard>
</template>
