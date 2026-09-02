<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import AssetPickDialog from '@/features/create/AssetPickDialog.vue'
import CreateFilmstrip from '@/features/create/workbench/CreateFilmstrip.vue'
import CreatePromptSubmit from '@/features/create/workbench/CreatePromptSubmit.vue'
import CreateSettingsPills from '@/features/create/workbench/CreateSettingsPills.vue'
import CreateSlotsTrack, { type SlotDescriptor } from '@/features/create/workbench/CreateSlotsTrack.vue'
import CreateStagePlayer from '@/features/create/workbench/CreateStagePlayer.vue'
import CreateWorkbenchToolbar from '@/features/create/workbench/CreateWorkbenchToolbar.vue'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'
import { estimateCreateCost } from '@/shared/lib/create-cost'
import type { CreateResolution } from '@/shared/lib/create-cost'
import {
  clearCreateDraft,
  emptySlot,
  loadCreateDraft,
  MAX_KEYFRAME_EXTRAS,
  MAX_REF_IMAGES,
  saveCreateDraft,
  type CreateFrameSlot,
  type CreateMode,
  type ImageVariant,
} from '@/shared/lib/create-draft'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import { useAppStore } from '@/stores/app'

interface FrameSlot extends CreateFrameSlot {}

const router = useRouter()
const app = useAppStore()

const mode = ref<CreateMode>('image')
const imageVariant = ref<ImageVariant>('keyframes')
const prompt = ref('')
const ratio = ref('9:16')
const durationSec = ref('15')
const resolution = ref<CreateResolution>('768P')
const loading = ref(false)
const draftReady = ref(false)

const firstFrame = ref<FrameSlot>(emptySlot())
const lastFrame = ref<FrameSlot>(emptySlot())
const extraFrames = ref<FrameSlot[]>([])
const multiFrames = ref<FrameSlot[]>([emptySlot()])

const activeSlotKey = ref<string | null>(null)
const activeTaskId = ref<string | null>(null)
const pickingSlotRef = ref<FrameSlot | null>(null)

const stageTitle = ref('等待装填素材')
const stageSubtitle = ref('768P · 24fps')
const stagePreviewUrl = ref<string | undefined>(undefined)

let saveTimer: ReturnType<typeof setTimeout> | null = null

const assetDialogOpen = computed({
  get: () => pickingSlotRef.value !== null,
  set: (open: boolean) => {
    if (!open) pickingSlotRef.value = null
  },
})

const parsedDuration = computed(() => Number(durationSec.value))
const durationValid = computed(() => [5, 10, 15].includes(parsedDuration.value))
const cost = computed(() => estimateCreateCost(parsedDuration.value, resolution.value))
const recentTasks = computed(() => app.tasks.slice(0, 3))

const canSubmit = computed(() => mode.value === 'image' && durationValid.value && hasRequiredImageInput.value)

const hasRequiredImageInput = computed(() => {
  if (mode.value !== 'image') return false
  if (imageVariant.value === 'keyframes') return frameReady(firstFrame.value)
  return multiFrames.value.some((slot) => frameReady(slot))
})

function frameReady(slot: FrameSlot) {
  return slot.uploaded || Boolean(slot.assetId)
}

function slotLabel(descriptor: SlotDescriptor) {
  if (descriptor.key === 'first') return '首帧关键图'
  if (descriptor.key === 'last') return '尾帧关键图'
  return '参考素材'
}

function setStageFromSlot(slot: FrameSlot, descriptor: SlotDescriptor) {
  activeTaskId.value = null
  activeSlotKey.value = descriptor.key
  stagePreviewUrl.value = frameReady(slot) ? assetPreviewUrl(slot.assetId, slot.previewUrl) : undefined
  if (frameReady(slot)) {
    const asset = app.assets.find((item) => item.id === slot.assetId)
    stageTitle.value = slot.fileName || asset?.name || '已选取素材'
    stageSubtitle.value = slotLabel(descriptor)
  }
}

function setStageFromTask(taskId: string) {
  const task = app.tasks.find((item) => item.id === taskId)
  if (!task) return
  activeTaskId.value = taskId
  activeSlotKey.value = null
  stagePreviewUrl.value = task.thumb
  stageTitle.value = task.title
  stageSubtitle.value = `${task.status} · ${task.duration}`
}

function resetStageIdle() {
  activeTaskId.value = null
  activeSlotKey.value = null
  stagePreviewUrl.value = undefined
  stageTitle.value = '等待装填素材'
  stageSubtitle.value = `${resolution.value} · 24fps`
}

function fillSlot(slot: FrameSlot, payload: { assetId?: string; fileName?: string; previewUrl?: string }) {
  slot.uploaded = true
  slot.assetId = payload.assetId ?? ''
  slot.fileName = payload.fileName
  slot.previewUrl = payload.previewUrl
}

function handleSlotClick(descriptor: SlotDescriptor) {
  const slot = descriptor.slot
  if (!slot) return
  if (frameReady(slot)) {
    setStageFromSlot(slot, descriptor)
    return
  }
  pickingSlotRef.value = slot
  activeSlotKey.value = descriptor.key
}

function confirmAssetPick(assetId: string) {
  if (!pickingSlotRef.value) return
  const asset = app.assets.find((item) => item.id === assetId)
  fillSlot(pickingSlotRef.value, { assetId })
  stageTitle.value = asset?.name ?? '已选取素材'
  stageSubtitle.value = '素材预览就绪'
  stagePreviewUrl.value = assetPreviewUrl(assetId)
  activeTaskId.value = null
  pickingSlotRef.value = null
  app.notify(`已装入素材: ${asset?.name ?? '图片'}`)
}

function confirmLocalUpload(payload: { fileName: string; previewUrl: string }) {
  if (!pickingSlotRef.value) return
  fillSlot(pickingSlotRef.value, payload)
  stageTitle.value = payload.fileName
  stageSubtitle.value = '素材预览就绪'
  stagePreviewUrl.value = payload.previewUrl
  activeTaskId.value = null
  pickingSlotRef.value = null
  app.notify(`已装入素材: ${payload.fileName}`)
}

function addExtra() {
  if (extraFrames.value.length >= MAX_KEYFRAME_EXTRAS) return
  extraFrames.value.push(emptySlot())
}

function addMulti() {
  if (multiFrames.value.length >= MAX_REF_IMAGES) return
  multiFrames.value.push(emptySlot())
}

function workflowLabel() {
  if (mode.value === 'text') return 'H3 文生视频'
  return imageVariant.value === 'keyframes' ? 'H3 首尾帧图生视频' : 'H3 多图参考图生视频'
}

function taskStamp() {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${pad(now.getMonth() + 1)}${pad(now.getDate())} ${pad(now.getHours())}${pad(now.getMinutes())}`
}

function buildTaskTitle() {
  const seq = app.tasks.length + 1
  const variant = imageVariant.value === 'keyframes' ? '首尾帧' : '多图'
  return `图生·${variant} ${taskStamp()} #${seq}`
}

function applyDraftFromStorage() {
  const plazaPrompt = app.draftPrompt.trim()
  const saved = loadCreateDraft()
  if (saved) {
    mode.value = saved.mode
    imageVariant.value = saved.imageVariant
    ratio.value = saved.ratio
    durationSec.value = saved.durationSec
    resolution.value = saved.resolution
    firstFrame.value = { ...saved.firstFrame, picking: false }
    lastFrame.value = { ...saved.lastFrame, picking: false }
    extraFrames.value = saved.extraFrames.map((slot) => ({ ...slot, picking: false }))
    multiFrames.value = saved.multiFrames.length
      ? saved.multiFrames.map((slot) => ({ ...slot, picking: false }))
      : [emptySlot()]
    prompt.value = plazaPrompt || saved.prompt
  }
  else {
    prompt.value = plazaPrompt
  }
  if (plazaPrompt) app.draftPrompt = ''
  stageSubtitle.value = `${resolution.value} · 24fps`
}

function scheduleDraftSave() {
  if (!draftReady.value) return
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    saveCreateDraft({
      mode: mode.value,
      imageVariant: imageVariant.value,
      prompt: prompt.value,
      ratio: ratio.value,
      durationSec: durationSec.value,
      resolution: resolution.value,
      firstFrame: firstFrame.value,
      lastFrame: lastFrame.value,
      extraFrames: extraFrames.value,
      multiFrames: multiFrames.value,
    })
  }, 320)
}

onMounted(() => {
  applyDraftFromStorage()
  draftReady.value = true
})

onBeforeUnmount(() => {
  if (saveTimer) clearTimeout(saveTimer)
})

watch([mode, imageVariant, prompt, ratio, durationSec, resolution, firstFrame, lastFrame, extraFrames, multiFrames], scheduleDraftSave, { deep: true })

watch([resolution, durationSec], () => {
  if (stageTitle.value === '等待装填素材') stageSubtitle.value = `${resolution.value} · 24fps`
})

watch(imageVariant, () => {
  activeSlotKey.value = null
})

function submit() {
  if (!canSubmit.value) {
    app.notify(imageVariant.value === 'keyframes' ? '请先上传或选择首帧图片' : '请至少选择一张参考图')
    return
  }
  loading.value = true
  window.setTimeout(() => {
    loading.value = false
    if (app.createTask(cost.value, {
      title: buildTaskTitle(),
      workflow: workflowLabel(),
      durationSec: parsedDuration.value,
      aspectRatio: ratio.value,
    })) {
      app.draftPrompt = ''
      prompt.value = ''
      clearCreateDraft()
      router.push('/app/works')
    }
  }, 420)
}
</script>

<template>
  <section class="s2v-page s2v-page-studio flex min-h-0 flex-1 flex-col">
    <S2VPageHeader title="创作">
      <template #actions>
        <S2VBadge tone="success">钱包余额: ¥ {{ app.balance.toFixed(2) }}</S2VBadge>
      </template>
    </S2VPageHeader>

    <div class="s2v-wb-card">
      <CreateWorkbenchToolbar
        :mode="mode"
        :image-variant="imageVariant"
        @update:mode="mode = $event"
        @update:image-variant="imageVariant = $event"
        @locked-text="app.notify('文生视频即将开放')"
      />

      <div class="s2v-wb-body">
        <div class="s2v-wb-stage-col">
          <CreateStagePlayer
            :ratio="ratio"
            :duration-sec="parsedDuration"
            :resolution="resolution"
            :title="stageTitle"
            :subtitle="stageSubtitle"
            :preview-url="stagePreviewUrl"
            @reset="resetStageIdle"
          />
          <CreateFilmstrip
            :tasks="recentTasks"
            :active-task-id="activeTaskId"
            @select="setStageFromTask"
            @view-all="router.push('/app/works')"
          />
        </div>

        <div class="s2v-wb-panel-col">
          <CreateSlotsTrack
            :variant="imageVariant"
            :first-frame="firstFrame"
            :last-frame="lastFrame"
            :extra-frames="extraFrames"
            :multi-frames="multiFrames"
            :active-slot-key="activeSlotKey"
            @slot-click="handleSlotClick"
            @add-extra="addExtra"
            @add-multi="addMulti"
          />
          <CreateSettingsPills
            :ratio="ratio"
            :duration-sec="durationSec"
            :resolution="resolution"
            @update:ratio="ratio = $event"
            @update:duration-sec="durationSec = $event"
            @update:resolution="resolution = $event"
          />
          <CreatePromptSubmit
            v-model:prompt="prompt"
            :cost="cost"
            :loading="loading"
            :disabled="!canSubmit"
            @submit="submit"
          />
        </div>
      </div>
    </div>

    <AssetPickDialog
      v-model:open="assetDialogOpen"
      @pick="confirmAssetPick"
      @upload="confirmLocalUpload"
    />
  </section>
</template>
