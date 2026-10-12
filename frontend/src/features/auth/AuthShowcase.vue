<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

const video = ref<HTMLVideoElement | null>(null)
const posterUrl = '/brand/auth/creator-studio-poster.jpg'
const source = ref('')
const playing = ref(false)
const frameReady = ref(false)
const failed = ref(false)
const posterFailed = ref(false)
const playbackNote = ref('')
let motion: MediaQueryList | undefined

async function play() {
  if (failed.value) return
  source.value = '/brand/auth/creator-studio-loop.mp4'
  await nextTick()
  try {
    await video.value?.play()
    playbackNote.value = ''
  } catch {
    playing.value = false
    if (!failed.value) playbackNote.value = '点击播放影像'
  }
}

function togglePlayback() {
  if (playing.value) video.value?.pause()
  else void play()
}

function mediaError() {
  failed.value = true
  playing.value = false
  playbackNote.value = '正在展示影像封面'
}

function pauseForPreference() {
  if (motion?.matches) video.value?.pause()
}

function pauseWhenHidden() {
  if (document.hidden) video.value?.pause()
}

onMounted(() => {
  motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  motion.addEventListener('change', pauseForPreference)
  document.addEventListener('visibilitychange', pauseWhenHidden)
  const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection
  // Small screens, data-saving and reduced-motion preferences start on the
  // local poster. A deliberate play action can always opt into the film.
  if (!motion.matches && !connection?.saveData && window.matchMedia('(min-width: 900px)').matches) void play()
})

onBeforeUnmount(() => {
  motion?.removeEventListener('change', pauseForPreference)
  document.removeEventListener('visibilitychange', pauseWhenHidden)
  video.value?.pause()
})
</script>

<template>
  <div class="s2v-auth-showcase">
    <div class="s2v-auth-backdrop">
    <img v-if="!posterFailed" class="s2v-auth-poster" :src="posterUrl" alt="动漫创作者在影像工作室调节镜头，镜头中映出山水风景" fetchpriority="high" @error="posterFailed = true" />
    <video
      v-if="source && !failed"
      ref="video"
      class="s2v-auth-film"
      :class="{ 's2v-auth-film-playing': frameReady }"
      :src="source"
      :poster="posterUrl"
      muted
      loop
      playsinline
      preload="none"
      aria-hidden="true"
      tabindex="-1"
      @playing="playing = true"
      @loadeddata="frameReady = true"
      @pause="playing = false"
      @error="mediaError"
    />
    <div class="s2v-auth-shade" aria-hidden="true" />
    </div>
    <footer class="s2v-auth-footer s2v-auth-film-footer">
      <slot />
      <span v-if="playbackNote" class="s2v-auth-playback-note" aria-hidden="true">{{ playbackNote }}</span>
      <span class="sr-only" role="status">{{ playbackNote }}</span>
      <button v-if="!failed" class="s2v-auth-playback" type="button" :aria-label="playing ? '暂停背景影像' : '播放背景影像'" @click="togglePlayback">{{ playing ? '暂停背景' : '播放背景' }}</button>
    </footer>
  </div>
</template>
