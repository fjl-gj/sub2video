<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

const host = ref<HTMLElement | null>(null)
const video = ref<HTMLVideoElement | null>(null)
const source = ref('')
const frameReady = ref(false)
const failed = ref(false)
const posterFailed = ref(false)
let motion: MediaQueryList | undefined
let desktop: MediaQueryList | undefined
let observer: IntersectionObserver | undefined
let idle: number | undefined
let timer: number | undefined
let visible = true
let disposed = false
let readyToStart = false
let playPending = false

function allowsAutomaticPlayback() {
  const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection
  return !disposed && !failed.value && readyToStart && !document.hidden && visible && !motion?.matches && !connection?.saveData && desktop?.matches
}

async function play() {
  if (!allowsAutomaticPlayback() || playPending) return
  playPending = true
  source.value = '/brand/auth/creator-studio-loop.mp4'
  await nextTick()
  let interrupted = false
  try {
    if (!allowsAutomaticPlayback() || !video.value) return
    video.value.muted = true
    await video.value.play()
    if (!allowsAutomaticPlayback()) video.value?.pause()
  }
  catch (error) {
    // A preference/visibility change can interrupt play() without a media
    // failure. Only a genuine autoplay failure retains the poster permanently.
    interrupted = error instanceof DOMException && error.name === 'AbortError'
    if (!interrupted && allowsAutomaticPlayback()) mediaError()
  }
  finally {
    playPending = false
    if (interrupted) reconcilePlayback()
  }
}

function reconcilePlayback() {
  if (!allowsAutomaticPlayback()) video.value?.pause()
  else if (!video.value || video.value.paused) void play()
}

function mediaError() {
  failed.value = true
  frameReady.value = false
}

function startAfterPageLoad() {
  const start = () => { readyToStart = true; reconcilePlayback() }
  if (typeof window.requestIdleCallback === 'function') idle = window.requestIdleCallback(start, { timeout: 1800 })
  else timer = window.setTimeout(start, 800)
}

onMounted(() => {
  motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  desktop = window.matchMedia('(min-width: 900px)')
  motion.addEventListener('change', reconcilePlayback)
  desktop.addEventListener('change', reconcilePlayback)
  document.addEventListener('visibilitychange', reconcilePlayback)
  // Keep the first paint on the local poster. The 2.1 MB brand film
  // is requested only after load/idle when automatic playback is permitted.
  if (document.readyState === 'complete') startAfterPageLoad()
  else window.addEventListener('load', startAfterPageLoad, { once: true })
  observer = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting
    reconcilePlayback()
  }, { threshold: 0.1 })
  if (host.value) observer.observe(host.value)
})

onBeforeUnmount(() => {
  disposed = true
  motion?.removeEventListener('change', reconcilePlayback)
  desktop?.removeEventListener('change', reconcilePlayback)
  document.removeEventListener('visibilitychange', reconcilePlayback)
  window.removeEventListener('load', startAfterPageLoad)
  observer?.disconnect()
  if (idle !== undefined) window.cancelIdleCallback(idle)
  if (timer !== undefined) window.clearTimeout(timer)
  video.value?.pause()
})
</script>

<template>
  <div ref="host" class="s2v-home-film">
    <img v-if="!posterFailed" class="s2v-home-film-poster" src="/brand/auth/creator-studio-poster.jpg" alt="动漫创作者在影像工作室调节镜头，镜头中映出山水风景" fetchpriority="high" @error="posterFailed = true" />
    <video v-if="source && !failed" ref="video" class="s2v-home-film-video" :class="{ 's2v-home-film-ready': frameReady }" :src="source" muted loop playsinline preload="none" aria-hidden="true" tabindex="-1" @loadeddata="frameReady = true" @error="mediaError" />
    <div class="s2v-home-film-shade" aria-hidden="true" />
  </div>
</template>
