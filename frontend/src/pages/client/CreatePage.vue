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
import { createTask, quoteTask, uploadImage } from '@/api/client'
import type { Task } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

interface FrameSlot extends CreateFrameSlot {}

const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()

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

const canSubmit = computed(() => {
  if (!durationValid.value) return false
  if (mode.value === 'text') return prompt.value.trim().length > 0
  return mode.value === 'image' && hasRequiredImageInput.value
})

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

function fillSlot(slot: FrameSlot, payload: {
  assetId?: string
  fileName?: string
  previewUrl?: string
  file?: File
  uploadId?: string
}) {
  slot.uploaded = true
  slot.assetId = payload.assetId ?? ''
  slot.fileName = payload.fileName
  slot.previewUrl = payload.previewUrl
  slot.file = payload.file
  slot.uploadId = payload.uploadId
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

function confirmLocalUpload(payload: { file: File; fileName: string; previewUrl: string }) {
  if (!pickingSlotRef.value) return
  fillSlot(pickingSlotRef.value, {
    file: payload.file,
    fileName: payload.fileName,
    previewUrl: payload.previewUrl,
  })
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

function workflowId() {
  return mode.value === 'text' ? 'wf-h3-text' : 'wf-h3-image'
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
  if (mode.value === 'text') return `文生 ${taskStamp()} #${seq}`
  const variant = imageVariant.value === 'keyframes' ? '首尾帧' : '多图'
  return `图生·${variant} ${taskStamp()} #${seq}`
}

function primaryImageSlot(): FrameSlot | null {
  if (mode.value !== 'image') return null
  if (imageVariant.value === 'keyframes') {
    return frameReady(firstFrame.value) ? firstFrame.value : null
  }
  return multiFrames.value.find((slot) => frameReady(slot)) ?? null
}

async function resolveSource(): Promise<Record<string, string> | undefined> {
  if (mode.value === 'text') return { type: 'text' }
  const slot = primaryImageSlot()
  if (!slot) throw new Error('请先上传或选择图片')
  if (slot.uploadId) return { type: 'upload', uploadId: slot.uploadId }
  if (slot.file) {
    const uploaded = await uploadImage(slot.file)
    slot.uploadId = uploaded.id
    slot.file = undefined
    return { type: 'upload', uploadId: uploaded.id }
  }
  if (slot.assetId) return { type: 'asset', assetId: slot.assetId }
  throw new Error('请先上传或选择图片')
}

function syncBalance(balance: number) {
  app.balance = balance
  auth.setBalance(balance)
}

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function submit() {
  if (!canSubmit.value) {
    if (mode.value === 'text') app.notify('请先填写画面描述')
    else app.notify(imageVariant.value === 'keyframes' ? '请先上传或选择首帧图片' : '请至少选择一张参考图')
    return
  }
  if (loading.value) return
  loading.value = true
  try {
    const source = await resolveSource()
    const quote = await quoteTask({
      workflowId: workflowId(),
      durationSeconds: parsedDuration.value,
      resolution: resolution.value,
    })
    const result = await createTask({
      workflowId: workflowId(),
      title: buildTaskTitle(),
      prompt: prompt.value.trim(),
      aspectRatio: ratio.value,
      durationSeconds: parsedDuration.value,
      resolution: resolution.value,
      source,
      priceSnapshot: {
        ruleId: quote.priceSnapshot.ruleId,
        version: quote.priceSnapshot.version,
        pricePerSecond: quote.pricePerSecond,
        estimatedCost: quote.estimatedCost,
      },
    })
    syncBalance(result.wallet.balance)
    const localTask: Task = {
      id: result.task.id,
      title: result.task.title,
      workflow: result.task.workflow || workflowLabel(),
      status: (result.task.status as Task['status']) || '排队中',
      progress: 0,
      duration: `${parsedDuration.value} 秒`,
      resolution: resolution.value,
      cost: result.task.cost,
      time: '刚刚',
      thumb: stagePreviewUrl.value || 'https://images.unsplash.com/photo-1519608487953-e999c86e7455?auto=format&fit=crop&w=480&q=80',
    }
    app.tasks.unshift(localTask)
    app.notify(`任务已提交，预扣 ¥${result.task.cost.toFixed(2)}`)
    app.draftPrompt = ''
    prompt.value = ''
    clearCreateDraft()
    router.push('/app/works')
  }
  catch (error) {
    app.notify(apiErrorMessage(error, '提交失败，请稍后重试'))
  }
  finally {
    loading.value = false
  }
}

function syncTextStage() {
  activeTaskId.value = null
  activeSlotKey.value = null
  stagePreviewUrl.value = undefined
  stageTitle.value = prompt.value.trim() ? '文生描述已就绪' : '等待输入描述'
  stageSubtitle.value = `${resolution.value} · 文生视频`
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
  if (mode.value === 'text') syncTextStage()
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

watch(mode, (next) => {
  if (next === 'text') syncTextStage()
  else resetStageIdle()
})

watch(prompt, () => {
  if (mode.value !== 'text') return
  stageTitle.value = prompt.value.trim() ? '文生描述已就绪' : '等待输入描述'
})
</script>

<template>
  <section class="s2v-page s2v-page-fill">
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
            v-if="mode === 'image'"
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
