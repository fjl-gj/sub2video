<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Pause, Play, RotateCcw } from 'lucide-vue-next'

const props = defineProps<{
  ratio: string
  durationSec: number
  resolution: string
  title: string
  subtitle: string
  previewUrl?: string
}>()

const emit = defineEmits<{
  reset: []
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const isPlaying = ref(false)
const playhead = ref(0)
let animId = 0
let rainDrops: Array<{ x: number; y: number; len: number; speed: number; opacity: number }> = []

const monitorClass = computed(() => {
  if (props.ratio === '16:9') return 's2v-wb-monitor-r169'
  if (props.ratio === '1:1') return 's2v-wb-monitor-r11'
  return 's2v-wb-monitor-r916'
})

const timecode = computed(() => {
  const curSec = Math.floor(playhead.value * props.durationSec)
  const curPad = String(curSec).padStart(2, '0')
  const durPad = String(props.durationSec).padStart(2, '0')
  return `00:${curPad} / 00:${durPad}`
})

function canvasSize() {
  if (props.ratio === '16:9') return { width: 1280, height: 720 }
  if (props.ratio === '1:1') return { width: 1080, height: 1080 }
  return { width: 720, height: 1280 }
}

function initRain() {
  const { width, height } = canvasSize()
  rainDrops = Array.from({ length: 70 }, () => ({
    x: Math.random() * width,
    y: Math.random() * height,
    len: 12 + Math.random() * 20,
    speed: 4 + Math.random() * 7,
    opacity: 0.15 + Math.random() * 0.35,
  }))
}

function renderCanvas() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const { width, height } = canvasSize()

  const grad = ctx.createLinearGradient(0, 0, 0, height)
  grad.addColorStop(0, '#0f172a')
  grad.addColorStop(0.5, '#1e1b4b')
  grad.addColorStop(1, '#09090b')
  ctx.fillStyle = grad
  ctx.fillRect(0, 0, width, height)

  const cx = width / 2
  const cy = height * 0.45
  const radial = ctx.createRadialGradient(cx, cy, 30, cx, cy, 360)
  radial.addColorStop(0, 'rgba(99, 102, 241, 0.35)')
  radial.addColorStop(0.6, 'rgba(79, 70, 229, 0.08)')
  radial.addColorStop(1, 'rgba(0,0,0,0)')
  ctx.fillStyle = radial
  ctx.beginPath()
  ctx.arc(cx, cy, 380, 0, Math.PI * 2)
  ctx.fill()

  ctx.strokeStyle = '#93c5fd'
  rainDrops.forEach((drop) => {
    ctx.lineWidth = 1
    ctx.globalAlpha = drop.opacity
    ctx.beginPath()
    ctx.moveTo(drop.x, drop.y)
    ctx.lineTo(drop.x, drop.y + drop.len)
    ctx.stroke()
    if (isPlaying.value) {
      drop.y += drop.speed
      if (drop.y > height) {
        drop.y = -drop.len
        drop.x = Math.random() * width
      }
    }
  })
  ctx.globalAlpha = 1

  const vig = ctx.createRadialGradient(width / 2, height / 2, width * 0.35, width / 2, height / 2, width * 0.75)
  vig.addColorStop(0, 'rgba(0,0,0,0)')
  vig.addColorStop(1, 'rgba(0,0,0,0.65)')
  ctx.fillStyle = vig
  ctx.fillRect(0, 0, width, height)

  if (isPlaying.value) {
    playhead.value += 0.003
    if (playhead.value > 1) playhead.value = 0
    animId = requestAnimationFrame(renderCanvas)
  }
}

function resizeCanvas() {
  const canvas = canvasRef.value
  if (!canvas) return
  const { width, height } = canvasSize()
  canvas.width = width
  canvas.height = height
  initRain()
  renderCanvas()
}

function togglePlayback() {
  isPlaying.value = !isPlaying.value
  if (isPlaying.value) renderCanvas()
  else cancelAnimationFrame(animId)
}

function resetPlayback() {
  isPlaying.value = false
  cancelAnimationFrame(animId)
  playhead.value = 0
  renderCanvas()
  emit('reset')
}

watch(() => props.ratio, () => {
  resizeCanvas()
})

onMounted(() => {
  resizeCanvas()
})

onBeforeUnmount(() => {
  cancelAnimationFrame(animId)
})
</script>

<template>
  <div class="s2v-wb-stage-box">
    <div class="s2v-wb-stage-topbar">
      <strong>播放台</strong>
      <span class="s2v-mono">{{ timecode }}</span>
    </div>

    <div class="s2v-wb-viewport">
      <div class="s2v-wb-monitor" :class="monitorClass">
        <img v-if="previewUrl" :src="previewUrl" alt="">
        <canvas v-else ref="canvasRef" />

        <button
          v-if="!previewUrl"
          type="button"
          class="s2v-wb-play-overlay"
          :aria-label="isPlaying ? '暂停' : '播放'"
          @click="togglePlayback"
        >
          <span class="s2v-wb-play-puck">
            <Pause v-if="isPlaying" class="h-4 w-4" />
            <Play v-else class="h-4 w-4 fill-current" />
          </span>
        </button>
      </div>
    </div>

    <div class="s2v-wb-stage-foot">
      <button v-if="!previewUrl" type="button" class="s2v-wb-stage-btn" @click="togglePlayback">
        <Pause v-if="isPlaying" class="h-3 w-3" />
        <Play v-else class="h-3 w-3" />
        <span>{{ isPlaying ? '暂停' : '播放' }}</span>
      </button>
      <button v-if="!previewUrl" type="button" class="s2v-wb-stage-btn" @click="resetPlayback">
        <RotateCcw class="h-3 w-3" />
        <span>重置</span>
      </button>
      <div class="s2v-wb-stage-info">
        <span class="s2v-wb-stage-title">{{ title }}</span>
        <span class="s2v-wb-stage-sub s2v-mono">{{ subtitle }}</span>
      </div>
    </div>
  </div>
</template>
