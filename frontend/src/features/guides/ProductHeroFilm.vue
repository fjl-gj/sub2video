<script setup lang="ts">
import { Pause, Play, RotateCcw } from 'lucide-vue-next'
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

const host = ref<HTMLElement | null>(null)
const video = ref<HTMLVideoElement | null>(null)
const source = ref('')
const playing = ref(false)
const frameReady = ref(false)
const failed = ref(false)
const posterFailed = ref(false)
const note = ref('')
let motion: MediaQueryList | undefined
let observer: IntersectionObserver | undefined
let idle: number | undefined
let timer: number | undefined
let visible = true
let disposed = false
let manuallyControlled = false

function allowsAutomaticPlayback() {
  const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection
  return !disposed && !manuallyControlled && !document.hidden && visible && !motion?.matches && !connection?.saveData && window.matchMedia('(min-width: 900px)').matches
}

async function play() {
  failed.value = false
  note.value = ''
  source.value = '/brand/auth/iceland-coast.webm'
  await nextTick()
  if (disposed || !video.value) return
  video.value.muted = true
  try { await video.value.play() }
  catch {
    playing.value = false
    if (!failed.value) note.value = '点击播放，查看实拍参考影像。'
  }
}

function togglePlayback() {
  manuallyControlled = true
  if (playing.value) video.value?.pause()
  else void play()
}

function mediaError() {
  failed.value = true
  playing.value = false
  frameReady.value = false
  note.value = '参考影像暂不可播放，创作和教程仍可使用。'
}

function pauseForPreference() { if (motion?.matches) video.value?.pause() }
function pauseWhenHidden() { if (document.hidden) video.value?.pause() }
function startAfterPageLoad() {
  if (!allowsAutomaticPlayback()) return
  const start = () => { if (allowsAutomaticPlayback()) void play() }
  if (typeof window.requestIdleCallback === 'function') idle = window.requestIdleCallback(start, { timeout: 1800 })
  else timer = window.setTimeout(start, 800)
}

onMounted(() => {
  motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  motion.addEventListener('change', pauseForPreference)
  document.addEventListener('visibilitychange', pauseWhenHidden)
  // Keep the first paint on the small local poster. The 2.2 MB reference film
  // is requested only after load/idle, or after an explicit play action.
  if (document.readyState === 'complete') startAfterPageLoad()
  else window.addEventListener('load', startAfterPageLoad, { once: true })
  observer = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting
    if (!visible) video.value?.pause()
  }, { threshold: 0.1 })
  if (host.value) observer.observe(host.value)
})

onBeforeUnmount(() => {
  disposed = true
  motion?.removeEventListener('change', pauseForPreference)
  document.removeEventListener('visibilitychange', pauseWhenHidden)
  window.removeEventListener('load', startAfterPageLoad)
  observer?.disconnect()
  if (idle !== undefined) window.cancelIdleCallback(idle)
  if (timer !== undefined) window.clearTimeout(timer)
  video.value?.pause()
})
</script>

<template>
  <div ref="host" class="s2v-home-film">
    <img v-if="!posterFailed" class="s2v-home-film-poster" src="/brand/auth/iceland-coast.jpg" alt="冰岛黑沙海岸实拍参考，浪潮沿着山脚涌向岸边" fetchpriority="high" @error="posterFailed = true" />
    <video v-if="source && !failed" ref="video" class="s2v-home-film-video" :class="{ 's2v-home-film-ready': frameReady }" :src="source" muted loop playsinline preload="none" aria-hidden="true" tabindex="-1" @playing="playing = true" @pause="playing = false" @loadeddata="frameReady = true" @error="mediaError" />
    <div class="s2v-home-film-shade" aria-hidden="true" />
    <div class="s2v-home-film-footer">
      <div class="s2v-home-film-credit"><span>实拍影像参考 · 冰岛海岸 · 非生成成片</span><small><a href="https://commons.wikimedia.org/wiki/File:Ocean_waves_at_L%C3%A6kjavik_beach,_Iceland.webm" target="_blank" rel="noopener noreferrer">Alexander Grebenkov</a> / <a href="https://creativecommons.org/licenses/by/3.0/" target="_blank" rel="noopener noreferrer">CC BY 3.0</a></small></div>
      <p v-if="note" class="s2v-home-film-note" role="status">{{ note }}</p>
      <button class="s2v-home-film-control" type="button" :aria-label="failed ? '重试首页参考影像' : playing ? '暂停首页参考影像' : '播放首页参考影像'" @click="togglePlayback"><RotateCcw v-if="failed" :size="17" /><Pause v-else-if="playing" :size="17" /><Play v-else :size="17" /></button>
    </div>
  </div>
</template>
