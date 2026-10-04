<script setup lang="ts">
import { Pause, Play } from 'lucide-vue-next'
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

const video = ref<HTMLVideoElement | null>(null)
const posterUrl = '/brand/auth/iceland-coast.jpg'
const source = ref('')
const playing = ref(false)
const frameReady = ref(false)
const failed = ref(false)
const posterFailed = ref(false)
const playbackNote = ref('')
let motion: MediaQueryList | undefined

async function play() {
  if (failed.value) return
  source.value = '/brand/auth/iceland-coast.webm'
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
  <section class="s2v-auth-showcase" aria-labelledby="auth-showcase-title">
    <img v-if="!posterFailed" class="s2v-auth-poster" :src="posterUrl" alt="冰岛黑沙海岸，浪潮沿着山脚涌向岸边" fetchpriority="high" @error="posterFailed = true" />
    <video
      v-if="source && !failed"
      ref="video"
      class="s2v-auth-film"
      :class="{ 's2v-auth-film-playing': frameReady }"
      :src="source"
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
    <div class="s2v-auth-film-heading"><span>让想象，有画面。</span><span>SUB2VIDEO STUDIO</span></div>
    <div class="s2v-auth-showcase-copy">
      <p class="s2v-auth-kicker">从一个镜头，开始你的故事</p>
      <h2 id="auth-showcase-title">下一幕，<br />由你开场。</h2>
      <p class="s2v-auth-showcase-description">把文字写成画面，让静止的灵感流动起来。<br />你的短片，从这里开始。</p>
      <div class="s2v-auth-modes" aria-label="支持的创作模式"><span>文生视频</span><span>多图生成视频</span><span>首尾帧生成视频</span></div>
    </div>
    <div class="s2v-auth-film-footer">
      <div class="s2v-auth-film-credit">
        <span>实拍影像参考 · 冰岛海岸</span>
        <small><a href="https://commons.wikimedia.org/wiki/File:Ocean_waves_at_L%C3%A6kjavik_beach,_Iceland.webm" target="_blank" rel="noopener noreferrer">Alexander Grebenkov</a><span> / </span><a href="https://creativecommons.org/licenses/by/3.0/" target="_blank" rel="noopener noreferrer">CC BY 3.0</a></small>
      </div>
      <span v-if="playbackNote" class="s2v-auth-playback-note" aria-hidden="true">{{ playbackNote }}</span>
      <span class="sr-only" role="status">{{ playbackNote }}</span>
      <button v-if="!failed" class="s2v-auth-playback" type="button" :aria-label="playing ? '暂停背景影像' : '播放背景影像'" @click="togglePlayback"><Pause v-if="playing" :size="16" /><Play v-else :size="16" /></button>
    </div>
  </section>
</template>
