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
import {
  clearCreateDraft,
  emptySlot,
  loadCreateDraft,
  MAX_REF_IMAGES,
  saveCreateDraft,
  type CreateFrameSlot,
  type ImageVariant,
  type WorkflowInputType,
} from '@/shared/lib/create-draft'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import {
  createTask,
  fetchCreationCatalog,
  fetchCreationOptions,
  listTasks,
  quoteTask,
  uploadImage,
  type CreationCatalog,
  type CreationOptions,
} from '@/api/client'
import type { Task } from '@/entities/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

interface FrameSlot extends CreateFrameSlot {}

const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()

const workflowInputType = ref<WorkflowInputType>('text')
const imageVariant = computed<ImageVariant>(() => workflowInputType.value === 'multi_image' ? 'multi' : 'keyframes')
const prompt = ref('')
const ratio = ref('9:16')
const durationSec = ref('15')
const resolution = ref('768P')
const selectedCarrierId = ref('')
const selectedModelId = ref('')
const generateAudio = ref(false)
const catalog = ref<CreationCatalog>({ modes: [], carriers: [], models: [] })
const catalogLoaded = ref(false)
const creationOptions = ref<CreationOptions>({
  durations: [], resolutions: [], aspectRatios: [], supportsGeneratedAudio: false,
  minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P',
})
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

const workflowOrder: Record<WorkflowInputType, number> = { text: 0, multi_image: 1, first_last: 2 }
const creationModes = computed(() => catalog.value.modes
  .filter((item) => item.enabled && item.inputType !== 'unsupported')
  .sort((left, right) => workflowOrder[left.inputType as WorkflowInputType] - workflowOrder[right.inputType as WorkflowInputType])
  .map((item) => ({ inputType: item.inputType as WorkflowInputType, name: item.name })))
const activeModeId = computed(() => catalog.value.modes.find((item) => item.inputType === workflowInputType.value && item.enabled)?.id || '')
const modeModels = computed(() => catalog.value.models.filter((item) => item.inputTypes.includes(workflowInputType.value)))
const visibleModels = computed(() => modeModels.value.filter((item) => !selectedCarrierId.value || item.carrierId === selectedCarrierId.value))
const visibleCarriers = computed(() => catalog.value.carriers.filter((carrier) => modeModels.value.some((model) => model.carrierId === carrier.id)))
const durationOptions = computed(() => creationOptions.value.durations.map(String))
const maxReferenceImages = computed(() => creationOptions.value.maxImages || MAX_REF_IMAGES)
const parsedDuration = computed(() => Number(durationSec.value))
const durationValid = computed(() => durationOptions.value.includes(durationSec.value))
const generationUnavailable = computed(() => catalogLoaded.value && (!activeModeId.value || !visibleModels.value.length || !creationOptions.value.durations.length))
const quotedCost = ref<number | null>(null)
const cost = computed(() => quotedCost.value ?? 0)
const recentTasks = ref<Task[]>([])

const canSubmit = computed(() => {
  return !generationUnavailable.value && durationValid.value && prompt.value.trim().length > 0 && (workflowInputType.value === 'text' || hasRequiredImageInput.value)
})

const hasRequiredImageInput = computed(() => {
  if (workflowInputType.value === 'text') return false
  if (imageVariant.value === 'keyframes') return frameReady(firstFrame.value) && frameReady(lastFrame.value)
  return multiFrames.value.filter((slot) => frameReady(slot)).length >= Math.max(creationOptions.value.minImages, 1)
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
  const task = recentTasks.value.find((item) => item.id === taskId)
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
  // Filled slots can be replaced by opening the picker again.
  pickingSlotRef.value = slot
  activeSlotKey.value = descriptor.key
  if (frameReady(slot)) setStageFromSlot(slot, descriptor)
}

function clearFrameSlot(slot: FrameSlot) {
  if (slot.previewUrl?.startsWith('blob:')) URL.revokeObjectURL(slot.previewUrl)
  slot.uploaded = false
  slot.assetId = ''
  slot.picking = false
  slot.previewUrl = undefined
  slot.fileName = undefined
  slot.file = undefined
  slot.uploadId = undefined
}

function handleClearSlot(descriptor: SlotDescriptor) {
  const slot = descriptor.slot
  if (!slot) return
  clearFrameSlot(slot)
  if (activeSlotKey.value === descriptor.key) resetStageIdle()
  app.notify('已清除，可重新上传或选取')
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

function addMulti() {
  if (multiFrames.value.length >= maxReferenceImages.value) return
  multiFrames.value.push(emptySlot())
}

function taskStamp() {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${pad(now.getMonth() + 1)}${pad(now.getDate())} ${pad(now.getHours())}${pad(now.getMinutes())}`
}

function buildTaskTitle() {
  const seq = recentTasks.value.length + 1
  const titles: Record<WorkflowInputType, string> = { text: '文生', first_last: '首尾帧', multi_image: '多图' }
  return `${titles[workflowInputType.value]} ${taskStamp()} #${seq}`
}

async function ensureSlotUpload(slot: FrameSlot): Promise<{ type: string; id: string }> {
  if (slot.uploadId) return { type: 'upload', id: slot.uploadId }
  if (slot.file) {
    const uploaded = await uploadImage(slot.file)
    slot.uploadId = uploaded.id
    slot.file = undefined
    return { type: 'upload', id: uploaded.id }
  }
  if (slot.assetId) return { type: 'asset', id: slot.assetId }
  throw new Error('请先上传或选择图片')
}

async function resolveSourcePayload(): Promise<Record<string, unknown> | undefined> {
  if (workflowInputType.value === 'text') return undefined

  if (imageVariant.value === 'keyframes') {
    if (!frameReady(firstFrame.value) || !frameReady(lastFrame.value)) throw new Error('请先上传或选择首帧和尾帧图片')
    const first = await ensureSlotUpload(firstFrame.value)
    const last = await ensureSlotUpload(lastFrame.value)
    return { first, last }
  }

  const refs = []
  for (const slot of multiFrames.value) {
    if (frameReady(slot)) refs.push(await ensureSlotUpload(slot))
  }
  if (refs.length < Math.max(creationOptions.value.minImages, 1)) throw new Error(`请至少选择 ${Math.max(creationOptions.value.minImages, 1)} 张参考图`)
  return { refs }
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
    if (!prompt.value.trim()) app.notify('请先填写画面描述')
    else if (workflowInputType.value === 'first_last') app.notify('请先上传或选择首帧和尾帧图片')
    else app.notify('请至少选择一张参考图')
    return
  }
  if (loading.value) return
  loading.value = true
  try {
    const sourceMeta = await resolveSourcePayload()
    const quote = await quoteTask({
      modeId: activeModeId.value,
      modelId: selectedModelId.value || undefined,
      durationSeconds: parsedDuration.value,
      resolution: resolution.value,
      aspectRatio: ratio.value,
      generateAudio: generateAudio.value,
    })
    quotedCost.value = quote.price.total
    const result = await createTask({
      quoteId: quote.quoteId,
      title: buildTaskTitle(),
      prompt: prompt.value.trim(),
      sourceMeta,
    })
    syncBalance(result.wallet.balance)
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

async function loadCreationOptions() {
  if (!activeModeId.value) {
    creationOptions.value = { durations: [], resolutions: [], aspectRatios: [], supportsGeneratedAudio: false, minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P' }
    return
  }
  try {
    const next = await fetchCreationOptions(activeModeId.value, selectedModelId.value || undefined)
    creationOptions.value = next
    if (!next.durations.includes(parsedDuration.value)) durationSec.value = String(next.defaultDuration || next.durations[0] || 5)
    if (!next.resolutions.includes(resolution.value)) resolution.value = next.defaultResolution || next.resolutions[0] || '768P'
    if (!next.aspectRatios.includes(ratio.value)) ratio.value = next.aspectRatios[0] || '16:9'
    if (!next.supportsGeneratedAudio) generateAudio.value = false
  } catch {
    creationOptions.value = { durations: [], resolutions: [], aspectRatios: [], supportsGeneratedAudio: false, minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P' }
  }
}

async function loadCreationCatalog() {
  catalogLoaded.value = false
  try {
    catalog.value = await fetchCreationCatalog()
    if (!creationModes.value.some((item) => item.inputType === workflowInputType.value)) {
      const fallback = creationModes.value[0]
      if (fallback) workflowInputType.value = fallback.inputType
    }
    await loadCreationOptions()
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载创作目录失败'))
  } finally {
    catalogLoaded.value = true
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
    workflowInputType.value = saved.workflowInputType
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
      workflowInputType: workflowInputType.value,
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

async function loadRecentTasks() {
  try {
    const result = await listTasks({ page: 1, pageSize: 3 })
    recentTasks.value = result.items.map((task) => ({
      ...task,
      workflow: task.mode,
      status: task.status as Task['status'],
      thumb: task.thumbUrl,
    }))
  } catch {
    recentTasks.value = []
  }
}

onMounted(() => {
  applyDraftFromStorage()
  if (workflowInputType.value === 'text') syncTextStage()
  draftReady.value = true
  void loadRecentTasks()
	void loadCreationCatalog()
  void (async () => {
    await app.loadProjects()
    if (app.currentProjectId) await app.loadProjectTree(app.currentProjectId)
  })()
})

onBeforeUnmount(() => {
  if (saveTimer) clearTimeout(saveTimer)
})

watch([workflowInputType, prompt, ratio, durationSec, resolution, firstFrame, lastFrame, extraFrames, multiFrames], scheduleDraftSave, { deep: true })

watch([activeModeId, selectedModelId], () => {
  quotedCost.value = null
  void loadCreationOptions()
})

watch(selectedCarrierId, () => {
  if (selectedModelId.value && !visibleModels.value.some((item) => item.id === selectedModelId.value)) selectedModelId.value = ''
})

watch([resolution, durationSec], () => {
  if (stageTitle.value === '等待装填素材') stageSubtitle.value = `${resolution.value} · 24fps`
})

watch(workflowInputType, (next) => {
  activeSlotKey.value = null
  if (next === 'text') syncTextStage()
  else resetStageIdle()
})

watch(prompt, () => {
  if (workflowInputType.value !== 'text') return
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
        :workflow-input-type="workflowInputType"
        :modes="creationModes"
        @update:workflow-input-type="workflowInputType = $event"
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
            v-if="workflowInputType !== 'text'"
            :variant="imageVariant"
            :first-frame="firstFrame"
            :last-frame="lastFrame"
            :extra-frames="extraFrames"
            :multi-frames="multiFrames"
            :max-images="maxReferenceImages"
            :active-slot-key="activeSlotKey"
            @slot-click="handleSlotClick"
            @clear-slot="handleClearSlot"
            @add-multi="addMulti"
          />
          <CreateSettingsPills
            :ratio="ratio"
            :duration-sec="durationSec"
            :resolution="resolution"
            :duration-options="durationOptions"
            :resolution-options="creationOptions.resolutions"
            :aspect-ratio-options="creationOptions.aspectRatios"
            :carriers="visibleCarriers"
            :models="visibleModels"
            :carrier-id="selectedCarrierId"
            :model-id="selectedModelId"
            :supports-generated-audio="creationOptions.supportsGeneratedAudio"
            :generate-audio="generateAudio"
            @update:ratio="ratio = $event"
            @update:duration-sec="durationSec = $event"
            @update:resolution="resolution = $event"
            @update:carrier-id="selectedCarrierId = $event"
            @update:model-id="selectedModelId = $event"
            @update:generate-audio="generateAudio = $event"
          />
          <CreatePromptSubmit
            v-model:prompt="prompt"
            :cost="cost"
            :loading="loading"
            :disabled="!canSubmit"
            :unavailable="generationUnavailable"
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
