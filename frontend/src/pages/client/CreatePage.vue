<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowUpRight } from 'lucide-vue-next'
import '@/features/create/workbench/studio.css'
import AssetPickDialog from '@/features/create/AssetPickDialog.vue'
import CreateFilmstrip from '@/features/create/workbench/CreateFilmstrip.vue'
import CreatePromptSubmit from '@/features/create/workbench/CreatePromptSubmit.vue'
import CreateSettingsPills from '@/features/create/workbench/CreateSettingsPills.vue'
import CreateSlotsTrack, { type SlotDescriptor } from '@/features/create/workbench/CreateSlotsTrack.vue'
import CreateStagePlayer from '@/features/create/workbench/CreateStagePlayer.vue'
import CreateWorkbenchToolbar from '@/features/create/workbench/CreateWorkbenchToolbar.vue'
import CreateAvailabilityNotice from '@/features/create/workbench/CreateAvailabilityNotice.vue'
import GuideCreationContext from '@/features/guides/GuideCreationContext.vue'
import { useGuidedCreation } from '@/features/guides/useGuidedCreation'
import DirectorCreationContext from '@/features/director/DirectorCreationContext.vue'
import { directorDraftScope, useDirectorCreation } from '@/features/director/useDirectorCreation'
import type { DirectorShotInput } from '@/api/director'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'
import { clearPendingCreation, loadPendingCreation, savePendingCreation, type PendingCreation } from '@/shared/lib/pending-creation'
import {
  clearCreateDraft,
  emptySlot,
  loadCreateDraft,
  MAX_REF_IMAGES,
  saveCreateDraft,
  type CreateFrameSlot,
  type CreateDraft,
  type ImageVariant,
  type WorkflowInputType,
} from '@/shared/lib/create-draft'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import {
  createTask,
  estimateTask,
  fetchCreationCatalog,
  fetchCreationOptions,
  fetchTask,
  listTasks,
  uploadImage,
  uploadReferenceAudio,
  type CreationCatalog,
  type CreationOptions,
} from '@/api/client'
import type { Task } from '@/entities/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

interface FrameSlot extends CreateFrameSlot {}

type AudioMode = 'none' | 'generate' | 'reference'

interface ReferenceAudio {
  file?: File
  uploadId?: string
  fileName: string
}

const router = useRouter()
const route = useRoute()
const app = useAppStore()
const auth = useAuthStore()
// Keep the mounted draft's owner even while logout clears the auth store.
const draftOwnerId = auth.user?.id || ''

const workflowInputType = ref<WorkflowInputType>('text')
const imageVariant = computed<ImageVariant>(() => workflowInputType.value === 'multi_image' ? 'multi' : 'keyframes')
const prompt = ref('')
const ratio = ref('9:16')
const durationSec = ref('15')
const resolution = ref('768P')
const selectedCarrierId = ref('')
const selectedModelId = ref('')
const audioMode = ref<AudioMode>('none')
const generateAudio = computed(() => audioMode.value !== 'none')
const referenceAudio = ref<ReferenceAudio | null>(null)
const catalog = ref<CreationCatalog>({ modes: [], carriers: [], models: [] })
const catalogLoaded = ref(false)
const catalogError = ref('')
const optionsError = ref('')
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

const stageTitle = ref('预览与结果')
const stageSubtitle = ref('上传素材后可在这里查看画面')
const stagePreviewUrl = ref<string | undefined>(undefined)
const stageVideoUrl = ref<string | undefined>(undefined)
const stageLoading = ref(false)
const stageError = ref('')
let previewRequest = 0
let estimateRequest = 0
let optionsRequest = 0

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
const materialSignature = computed(() => JSON.stringify({
  first: firstFrame.value.uploadId || firstFrame.value.assetId || (firstFrame.value.file ? `local:${firstFrame.value.fileName || 'image'}` : ''),
  last: lastFrame.value.uploadId || lastFrame.value.assetId || (lastFrame.value.file ? `local:${lastFrame.value.fileName || 'image'}` : ''),
  extras: extraFrames.value.map((slot) => slot.uploadId || slot.assetId || (slot.file ? `local:${slot.fileName || 'image'}` : '')),
  refs: multiFrames.value.map((slot) => slot.uploadId || slot.assetId || (slot.file ? `local:${slot.fileName || 'image'}` : '')),
  audio: referenceAudio.value?.uploadId || (referenceAudio.value?.file ? `local:${referenceAudio.value.fileName}` : ''),
}))
const durationValid = computed(() => durationOptions.value.includes(durationSec.value))
const generationUnavailable = computed(() => catalogLoaded.value && !catalogError.value && !optionsError.value && !optionsLoading.value && (!activeModeId.value || !visibleModels.value.length || !creationOptions.value.durations.length))
const quotedCost = ref<number | null>(null)
const recentTasks = ref<Task[]>([])
const recentTasksLoading = ref(false)
const recentTasksError = ref('')
const estimateLoading = ref(false)
const estimateError = ref('')
const optionsLoading = ref(false)
const pendingCreation = ref<PendingCreation | null>(null)
const submissionError = ref('')
const recoveryError = ref('')
const tutorial = useGuidedCreation({
  mode: workflowInputType, prompt, duration: durationSec, resolution, ratio, catalog, catalogLoaded,
  hasMaterials: () => [firstFrame.value, lastFrame.value, ...extraFrames.value, ...multiFrames.value].some(frameReady) || referenceAudio.value !== null,
  disableAudio: () => { audioMode.value = 'none' },
})
const director = useDirectorCreation({
  ownerId: draftOwnerId, catalog, pending: pendingCreation,
  blocked: () => loading.value || !!recoveryError.value || !catalogLoaded.value || !!catalogError.value,
  persist: persistDraft,
  refreshPending: restorePendingCreation,
  apply: (draft) => { applyCreationDraft(draft); resetStageIdle(); void loadCreationOptions() },
  reset: applyDraftFromStorage,
  prepareShot: prepareDirectorShot,
})
const availableAssets = computed(() => director.context.value ? director.referenceAssets.value : app.assets)

async function applyTutorial() {
  tutorial.apply()
  await loadCreationOptions()
}

const canSubmit = computed(() => {
  return !pendingCreation.value && !recoveryError.value && quotedCost.value !== null && !estimateLoading.value && !estimateError.value && auth.balance >= quotedCost.value && catalogLoaded.value && !catalogError.value && !optionsError.value && !tutorial.invalid.value && !tutorial.unavailable.value && !generationUnavailable.value && !optionsLoading.value && durationValid.value && prompt.value.trim().length > 0
    && (workflowInputType.value === 'text' || hasRequiredImageInput.value)
    && (audioMode.value !== 'reference' || referenceAudio.value !== null)
    && (!director.requested.value || !!director.context.value)
    && (!director.context.value || audioMode.value !== 'reference')
    && !director.saving.value && !director.needsReapply.value
})

const hasRequiredImageInput = computed(() => {
  if (workflowInputType.value === 'text') return false
  if (imageVariant.value === 'keyframes') return frameReady(firstFrame.value) && frameReady(lastFrame.value)
  return multiFrames.value.filter((slot) => frameReady(slot)).length >= Math.max(creationOptions.value.minImages, 1)
})

function frameReady(slot: FrameSlot) {
  return Boolean(slot.file || slot.uploadId || slot.assetId)
}

const preparationHint = computed(() => {
  if (workflowInputType.value === 'first_last' && !hasRequiredImageInput.value) return '请先准备首帧与尾帧各一张图片，再查看费用。'
  if (workflowInputType.value === 'multi_image' && !hasRequiredImageInput.value) return `请先准备至少 ${Math.max(creationOptions.value.minImages, 1)} 张参考图，再查看费用。`
  if (audioMode.value === 'reference' && !referenceAudio.value) return '请先选择参考音频文件。'
  return ''
})
const submitHint = computed(() => {
  if (director.saving.value) return '正在保存镜头素材与参数，请稍候再提交生成。'
  if (director.needsReapply.value) return '请先重新读取并应用镜头，核对后再保存或生成。'
  if (director.requested.value && !director.context.value) return '请先应用镜头到创作，再确认素材与费用。'
  if (director.context.value && audioMode.value === 'reference') return '导演台暂不支持参考音频，请选择不使用音频或自动生成音频。'
  if (preparationHint.value) return preparationHint.value
  if (quotedCost.value !== null && auth.balance < quotedCost.value) return '余额不足，请前往钱包查看余额或使用兑换码。在线充值暂未开放。'
  if (!prompt.value.trim()) return '请填写这一镜的画面描述，再提交任务。'
  return ''
})

function slotLabel(descriptor: SlotDescriptor) {
  if (descriptor.key === 'first') return '首帧关键图'
  if (descriptor.key === 'last') return '尾帧关键图'
  return '参考素材'
}

function setStageFromSlot(slot: FrameSlot, descriptor: SlotDescriptor) {
  resetStageMedia()
  activeTaskId.value = null
  activeSlotKey.value = descriptor.key
  const asset = availableAssets.value.find((item) => item.id === slot.assetId)
  stagePreviewUrl.value = frameReady(slot) ? assetPreviewUrl(slot.assetId, slot.previewUrl || asset?.url) : undefined
  if (frameReady(slot)) {
    const asset = availableAssets.value.find((item) => item.id === slot.assetId)
    stageTitle.value = slot.fileName || asset?.name || '已选取素材'
    stageSubtitle.value = slotLabel(descriptor)
  }
}

async function setStageFromTask(taskId: string) {
  const task = recentTasks.value.find((item) => item.id === taskId)
  if (!task) return
  resetStageMedia()
  const request = previewRequest
  activeTaskId.value = taskId
  activeSlotKey.value = null
  stagePreviewUrl.value = task.thumb
  stageTitle.value = task.title
  stageSubtitle.value = `${task.status} · ${task.duration}`
  stageLoading.value = true
  try {
    const detail = await fetchTask(taskId)
    if (request !== previewRequest) return
    stagePreviewUrl.value = detail.thumbUrl || task.thumb
    stageVideoUrl.value = detail.deliveryStatus === 'ready' ? detail.downloadUrl || undefined : undefined
    stageSubtitle.value = `${detail.status} · ${detail.duration} · ${detail.resolution}`
  } catch {
    if (request === previewRequest) stageError.value = '暂时无法读取成片，请在任务详情查看最新状态'
  } finally {
    if (request === previewRequest) stageLoading.value = false
  }
}

function resetStageMedia() {
  previewRequest += 1
  stageVideoUrl.value = undefined
  stageLoading.value = false
  stageError.value = ''
}

function resetStageIdle() {
  resetStageMedia()
  activeTaskId.value = null
  activeSlotKey.value = null
  stagePreviewUrl.value = undefined
  stageTitle.value = '预览与结果'
  stageSubtitle.value = '上传素材后可在这里查看画面'
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
  const asset = availableAssets.value.find((item) => item.id === assetId)
  resetStageMedia()
  fillSlot(pickingSlotRef.value, { assetId, previewUrl: asset?.url })
  stageTitle.value = asset?.name ?? '已选取素材'
  stageSubtitle.value = '素材预览就绪'
  stagePreviewUrl.value = assetPreviewUrl(assetId, asset?.url)
  activeTaskId.value = null
  pickingSlotRef.value = null
  app.notify(`已装入素材: ${asset?.name ?? '图片'}`)
}

function confirmLocalUpload(payload: { file: File; fileName: string; previewUrl: string }) {
  if (!pickingSlotRef.value) return
  resetStageMedia()
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

async function ensureSlotUpload(slot: FrameSlot): Promise<{ type: 'upload' | 'asset'; id: string }> {
  if (slot.uploadId) return { type: 'upload', id: slot.uploadId }
  if (slot.file) {
    const uploaded = await uploadImage(slot.file)
    slot.uploadId = uploaded.id
    slot.previewUrl = uploaded.url
    slot.file = undefined
    return { type: 'upload', id: uploaded.id }
  }
  if (slot.assetId) return { type: 'asset', id: slot.assetId }
  throw new Error('请先上传或选择图片')
}

async function prepareDirectorShot(isCurrent: () => boolean): Promise<DirectorShotInput> {
  if (audioMode.value === 'reference') throw new Error('导演台暂不支持参考音频')
  // Freeze public settings before uploads. Incomplete references are valid
  // planning data; submission still validates the full creation capabilities.
  const input: DirectorShotInput = {
    prompt: prompt.value.trim(), modeId: activeModeId.value, carrierId: selectedCarrierId.value,
    modelId: selectedModelId.value, durationSec: Number.isFinite(parsedDuration.value) ? parsedDuration.value : 0,
    resolution: resolution.value, aspectRatio: ratio.value, generateAudio: generateAudio.value, sourceMeta: {},
  }
  const inputType = workflowInputType.value
  const first = firstFrame.value
  const last = lastFrame.value
  const extras = [...extraFrames.value]
  const refs = [...multiFrames.value]
  async function resolve(slot: FrameSlot) {
    if (!isCurrent()) throw new Error('镜头上下文已变化')
    const source = await ensureSlotUpload(slot)
    if (!isCurrent()) throw new Error('镜头上下文已变化')
    return source
  }
  if (inputType === 'first_last') {
    if (frameReady(first)) input.sourceMeta.first = await resolve(first)
    if (frameReady(last)) input.sourceMeta.last = await resolve(last)
    if (extras.some(frameReady)) {
      input.sourceMeta.extras = []
      for (const slot of extras.filter(frameReady)) input.sourceMeta.extras.push(await resolve(slot))
    }
  } else if (inputType === 'multi_image') {
    input.sourceMeta.refs = []
    for (const slot of refs.filter(frameReady)) input.sourceMeta.refs.push(await resolve(slot))
  }
  return input
}

function selectReferenceAudio(file: File) {
  referenceAudio.value = { file, fileName: file.name }
}

function clearReferenceAudio() {
  referenceAudio.value = null
}

async function ensureReferenceAudioUpload(): Promise<{ type: 'upload'; id: string }> {
  const selected = referenceAudio.value
  if (!selected) throw new Error('请先上传参考音频')
  if (selected.uploadId) return { type: 'upload', id: selected.uploadId }
  if (!selected.file) throw new Error('参考音频文件不可用，请重新上传')
  const uploaded = await uploadReferenceAudio(selected.file)
  selected.uploadId = uploaded.id
  selected.file = undefined
  return { type: 'upload', id: uploaded.id }
}

async function resolveSourcePayload(): Promise<Record<string, unknown> | undefined> {
  const sourceMeta: Record<string, unknown> = {}

  if (imageVariant.value === 'keyframes') {
    if (workflowInputType.value !== 'text') {
      if (!frameReady(firstFrame.value) || !frameReady(lastFrame.value)) throw new Error('请先上传或选择首帧和尾帧图片')
      sourceMeta.first = await ensureSlotUpload(firstFrame.value)
      sourceMeta.last = await ensureSlotUpload(lastFrame.value)
    }
  }
  else if (workflowInputType.value === 'multi_image') {
    const refs = []
    for (const slot of multiFrames.value) {
      if (frameReady(slot)) refs.push(await ensureSlotUpload(slot))
    }
    if (refs.length < Math.max(creationOptions.value.minImages, 1)) throw new Error(`请至少选择 ${Math.max(creationOptions.value.minImages, 1)} 张参考图`)
    sourceMeta.refs = refs
  }
  if (audioMode.value === 'reference') sourceMeta.audio = await ensureReferenceAudioUpload()
  return Object.keys(sourceMeta).length ? sourceMeta : undefined
}

// Estimate the selected visual reference count without uploading local files.
// Pending references are opaque placeholders used only for capability routing;
// submission uploads and validates the actual source ownership.
function estimateSourceMeta(): Record<string, unknown> | undefined {
  if (workflowInputType.value === 'text') return undefined
  const refFor = (slot: FrameSlot, index: number) => ({
    type: slot.uploadId ? 'upload' : slot.assetId ? 'asset' : 'pending',
    id: slot.uploadId || slot.assetId || `pending-image-${index + 1}`,
  })
  if (imageVariant.value === 'keyframes') {
    const refs: Record<string, unknown> = {}
    if (frameReady(firstFrame.value)) refs.first = refFor(firstFrame.value, 0)
    if (frameReady(lastFrame.value)) refs.last = refFor(lastFrame.value, 1)
    return Object.keys(refs).length ? refs : undefined
  }
  const refs = multiFrames.value.filter(frameReady).map(refFor)
  return refs.length ? { refs } : undefined
}

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function submit() {
  if (!canSubmit.value || loading.value) return
  const userId = auth.user?.id
  if (!userId) return
  loading.value = true
  submissionError.value = ''
  try {
    restorePendingCreation()
    if (pendingCreation.value || recoveryError.value) return
    const sourceMeta = await resolveSourcePayload()
    if (auth.user?.id !== userId) return
    // Another tab may have submitted while these materials were uploading.
    restorePendingCreation()
    if (pendingCreation.value || recoveryError.value) return
    const pending: PendingCreation = {
      requestId: `create_${Array.from(crypto.getRandomValues(new Uint32Array(4)), (n) => n.toString(16)).join('_')}`,
      payload: {
      modeId: activeModeId.value,
      carrierId: selectedCarrierId.value || undefined,
      modelId: selectedModelId.value || undefined,
      durationSec: parsedDuration.value,
      resolution: resolution.value,
      aspectRatio: ratio.value,
      generateAudio: generateAudio.value,
      title: buildTaskTitle(),
      prompt: prompt.value.trim(),
      sourceMeta,
      ...(director.context.value ? { directorContext: { ...director.context.value } } : {}),
      },
    }
    persistDraft()
    try { savePendingCreation(userId, pending) }
    catch { throw new Error('浏览器无法保存提交记录，暂未提交任务。请允许本地存储后重试。') }
    pendingCreation.value = pending
    await sendPendingCreation(userId, pending)
  }
  catch (error) {
    submissionError.value = apiErrorMessage(error, '素材准备失败，暂未提交任务，请重试。')
  }
  finally { loading.value = false }
}

function restorePendingCreation() {
  recoveryError.value = ''
  try { pendingCreation.value = auth.user?.id ? loadPendingCreation(auth.user.id) : null }
  catch { recoveryError.value = '无法读取上次提交记录，请先在任务中核对结果，或联系支持。' }
}

async function recoverSubmission() {
  if (loading.value || director.saving.value || !auth.user?.id || !pendingCreation.value) return
  const userId = auth.user.id
  loading.value = true
  submissionError.value = ''
  try { await sendPendingCreation(userId, pendingCreation.value) }
  finally { loading.value = false }
}

async function sendPendingCreation(userId: string, pending: PendingCreation) {
  let accepted = false
  const balanceMutation = auth.beginBalanceMutation()
  try {
    const result = await createTask(pending.payload, pending.requestId)
    if (!result.task?.id || !result.order?.id || !Number.isFinite(result.order.amount) || !Number.isFinite(result.wallet?.balance)) {
      throw new Error('提交响应不完整')
    }
    accepted = true
    if (auth.user?.id !== userId) return
    auth.acceptBalanceMutation(balanceMutation, result.wallet.balance)
    app.notify(`任务已提交，消费 ¥${result.order.amount.toFixed(2)}`)
    draftReady.value = false
    if (saveTimer) clearTimeout(saveTimer)
    app.draftPrompt = ''
    prompt.value = ''
    clearCreateDraft(userId, directorDraftScope(pending.payload.directorContext))
    await router.push({ path: `/app/works/${result.task.id}`, query: pending.payload.directorContext ? { directorProject: pending.payload.directorContext.projectId, directorShot: pending.payload.directorContext.shotId } : undefined })
    clearPendingCreation(userId, pending.requestId)
    pendingCreation.value = null
  }
  catch (error) {
    if (auth.user?.id !== userId) return
    const response = (error as { response?: { status?: number; data?: { code?: string } } }).response
    const code = response?.data?.code
    // Only explicit application rejections prove no task was committed. A
    // timeout, proxy error or unknown response keeps the original request.
    const directorNotFound = !!pending.payload.directorContext && response?.status === 404 && code === 'NOT_FOUND'
    const rejected = !accepted && ((code && ['BAD_REQUEST', 'PROMPT_REQUIRED', 'UNAVAILABLE_COMBINATION', 'INSUFFICIENT_BALANCE', 'SUPPLY_UNAVAILABLE', 'STORAGE_UNAVAILABLE', 'DIRECTOR_BOARD_CONFLICT', 'DIRECTOR_SHOT_MISSING', 'DIRECTOR_BOARD_INVALID', 'DIRECTOR_REFERENCE_INVALID', 'DIRECTOR_TASK_INVALID', 'DIRECTOR_SELECTION_NOT_READY'].includes(code)) || directorNotFound)
    if (rejected) {
      clearPendingCreation(userId, pending.requestId)
      pendingCreation.value = null
      submissionError.value = creationErrorMessage(error, '当前参数无法提交，请检查素材和生成设置后重试。')
      if (code === 'DIRECTOR_BOARD_CONFLICT' || code === 'DIRECTOR_SHOT_MISSING' || directorNotFound) {
        if (directorNotFound) submissionError.value = '作品或镜头已不可用，本次未提交。镜头草稿仍保留，请返回资产核对作品。'
        persistDraft()
        director.context.value = null
        draftReady.value = false
        void director.load()
      }
    } else {
      submissionError.value = '尚未确认提交结果，请核对原提交，不要另建相同任务。重试将使用原请求编号。'
    }
  }
  finally { auth.finishBalanceMutation(balanceMutation) }
}

function creationErrorMessage(error: unknown, fallback: string) {
  const code = (error as { response?: { data?: { code?: string } } }).response?.data?.code
  if (code === 'PROMPT_REQUIRED') return '请先填写画面描述，再提交任务。'
  if (code?.startsWith('DIRECTOR_')) return '镜头板或引用内容已变化，本次未提交。请重新读取并应用镜头后核对参数，或返回导演台。'
  if (code === 'INSUFFICIENT_BALANCE') return '余额不足，无法预扣。请前往钱包查看余额或使用兑换码。'
  if (code === 'STORAGE_UNAVAILABLE') return '素材与成片存储暂不可用，请稍后重试。本次未提交任务。'
  if (code === 'SUPPLY_UNAVAILABLE' || code === 'UNAVAILABLE_COMBINATION') return '当前模型或参数组合暂不可用，请调整参数或稍后重试。'
  return fallback
}

function creationOptionsErrorMessage(error: unknown) {
  const response = (error as { response?: { status?: number; data?: { code?: string } } }).response
  const code = response?.data?.code
  if (code === 'SUPPLY_UNAVAILABLE') return '当前生成能力暂不可用，请稍后重新加载，或联系帮助与工单。'
  if (code === 'UNAVAILABLE_COMBINATION') return '当前模型或参数组合暂不可用，请调整选择或稍后重新加载。'
  if (code === 'STORAGE_UNAVAILABLE') return '素材与成片存储暂不可用，请稍后重新加载。'
  if (!response) return '无法连接生成服务，请检查网络后重新加载。'
  if (response.status && response.status >= 500) return '生成服务暂时异常，请稍后重新加载。'
  return '生成选项暂不可用，请重新加载后再试。'
}

async function refreshEstimate() {
  const request = ++estimateRequest
  quotedCost.value = null
  estimateError.value = ''
  if (optionsLoading.value) {
    estimateLoading.value = true
    return
  }
  if (!activeModeId.value || !durationValid.value || !resolution.value || !ratio.value || preparationHint.value) {
    estimateLoading.value = false
    return
  }
  estimateLoading.value = true
  try {
    const result = await estimateTask({ modeId: activeModeId.value, carrierId: selectedCarrierId.value || undefined, modelId: selectedModelId.value || undefined, durationSec: parsedDuration.value, resolution: resolution.value, aspectRatio: ratio.value, generateAudio: generateAudio.value, sourceMeta: estimateSourceMeta() })
    if (request !== estimateRequest) return
    quotedCost.value = result.available ? result.billing.estimatedTotal : null
    if (!result.available) estimateError.value = '当前模型或参数组合暂不可用，请调整参数后重新估价。'
  } catch (error) {
    if (request === estimateRequest) estimateError.value = creationErrorMessage(error, '费用预估失败，请检查网络后重试。确认费用后才能提交。')
  } finally {
    if (request === estimateRequest) estimateLoading.value = false
  }
}

async function loadCreationOptions() {
  const request = ++optionsRequest
  optionsError.value = ''
  estimateRequest += 1
  quotedCost.value = null
  estimateError.value = ''
  if (!activeModeId.value) {
    optionsLoading.value = false
    estimateLoading.value = false
    creationOptions.value = { durations: [], resolutions: [], aspectRatios: [], supportsGeneratedAudio: false, minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P' }
    return
  }
  optionsLoading.value = true
  estimateLoading.value = true
  try {
    const next = await fetchCreationOptions(activeModeId.value, selectedModelId.value || undefined)
    if (request !== optionsRequest) return
    creationOptions.value = next
    tutorial.checkOptions(next)
    if (!next.durations.includes(parsedDuration.value)) durationSec.value = String(next.defaultDuration || next.durations[0] || 5)
    if (!next.resolutions.includes(resolution.value)) resolution.value = next.defaultResolution || next.resolutions[0] || '768P'
    if (!next.aspectRatios.includes(ratio.value)) ratio.value = next.aspectRatios[0] || '16:9'
    if (!next.supportsGeneratedAudio) {
      audioMode.value = 'none'
      clearReferenceAudio()
    }
  } catch (error) {
    if (request !== optionsRequest) return
    optionsError.value = creationOptionsErrorMessage(error)
    creationOptions.value = { durations: [], resolutions: [], aspectRatios: [], supportsGeneratedAudio: false, minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P' }
  } finally {
    if (request === optionsRequest) {
      optionsLoading.value = false
      void refreshEstimate()
    }
  }
}

async function loadCreationCatalog() {
  catalogLoaded.value = false
  catalogError.value = ''
  try {
    catalog.value = await fetchCreationCatalog()
    if (!tutorial.requested.value && !creationModes.value.some((item) => item.inputType === workflowInputType.value)) {
      const fallback = creationModes.value[0]
      if (fallback) workflowInputType.value = fallback.inputType
    }
    await loadCreationOptions()
  } catch (error) {
    catalogError.value = '创作目录加载失败，请检查网络后重新加载。'
    app.notify(apiErrorMessage(error, '加载创作目录失败'))
  } finally {
    catalogLoaded.value = true
  }
}

function syncTextStage() {
  resetStageMedia()
  activeTaskId.value = null
  activeSlotKey.value = null
  stagePreviewUrl.value = undefined
  stageTitle.value = '预览与结果'
  stageSubtitle.value = '生成完成后，可在任务中预览和下载成片'
}

function applyDraftFromStorage() {
  const plazaPrompt = app.draftPrompt.trim()
  const saved = loadCreateDraft(draftOwnerId, director.scope.value)
  if (saved) {
    applyCreationDraft(saved)
    prompt.value = plazaPrompt || saved.prompt
  }
  else {
    prompt.value = plazaPrompt
  }
  if (plazaPrompt) app.draftPrompt = ''
  applyModeFromRoute()
}

function applyCreationDraft(saved: CreateDraft) {
  workflowInputType.value = saved.workflowInputType
  ratio.value = saved.ratio
  durationSec.value = saved.durationSec
  resolution.value = saved.resolution
  selectedCarrierId.value = saved.carrierId || ''
  selectedModelId.value = saved.modelId || ''
  audioMode.value = saved.audioMode || 'none'
  referenceAudio.value = saved.referenceAudioId ? { uploadId: saved.referenceAudioId, fileName: saved.referenceAudioName || '已保存参考音频' } : null
  firstFrame.value = { ...saved.firstFrame, picking: false }
  lastFrame.value = { ...saved.lastFrame, picking: false }
  extraFrames.value = saved.extraFrames.map((slot) => ({ ...slot, picking: false }))
  multiFrames.value = saved.multiFrames.length ? saved.multiFrames.map((slot) => ({ ...slot, picking: false })) : [emptySlot()]
  prompt.value = saved.prompt
  draftReady.value = true
}

function applyModeFromRoute() {
  if (director.context.value) return
  const mode = route.query.mode
  if (mode === 'text' || mode === 'multi_image' || mode === 'first_last') workflowInputType.value = mode
}

function persistDraft() {
  if (!draftReady.value) return
  saveCreateDraft(draftOwnerId, {
    workflowInputType: workflowInputType.value,
    prompt: prompt.value,
    ratio: ratio.value,
    durationSec: durationSec.value,
    resolution: resolution.value,
    carrierId: selectedCarrierId.value || undefined,
    modelId: selectedModelId.value || undefined,
    audioMode: audioMode.value,
    referenceAudioId: referenceAudio.value?.uploadId,
    referenceAudioName: referenceAudio.value?.fileName,
    firstFrame: firstFrame.value,
    lastFrame: lastFrame.value,
    extraFrames: extraFrames.value,
    multiFrames: multiFrames.value,
  }, director.scope.value)
}

function scheduleDraftSave() {
  if (!draftReady.value) return
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(persistDraft, 320)
}

async function loadRecentTasks() {
  recentTasksLoading.value = true
  recentTasksError.value = ''
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
    recentTasksError.value = '暂时无法加载最近任务'
  } finally {
    recentTasksLoading.value = false
  }
}

onMounted(() => {
  window.addEventListener('pagehide', persistDraft)
  restorePendingCreation()
  director.restorePending()
  applyDraftFromStorage()
  if (!director.requested.value) tutorial.initialize()
  if (workflowInputType.value === 'text') syncTextStage()
  draftReady.value = true
  void loadRecentTasks()
  void loadCreationCatalog()
  void (async () => {
    try {
      await app.loadProjects()
      if (app.currentProjectId) await app.loadProjectTree(app.currentProjectId)
    } catch {
      if (auth.user?.id === draftOwnerId) app.notify('暂时无法加载作品素材，你仍可上传本地文件')
    }
  })()
})

onBeforeUnmount(() => {
  window.removeEventListener('pagehide', persistDraft)
  if (saveTimer) clearTimeout(saveTimer)
  persistDraft()
  estimateRequest += 1
  optionsRequest += 1
  previewRequest += 1
})

watch([workflowInputType, prompt, ratio, durationSec, resolution, selectedCarrierId, selectedModelId, audioMode, referenceAudio, firstFrame, lastFrame, extraFrames, multiFrames], scheduleDraftSave, { deep: true })

watch([activeModeId, selectedModelId], () => {
  quotedCost.value = null
  void loadCreationOptions()
})

watch([activeModeId, selectedCarrierId, selectedModelId, durationSec, resolution, ratio, generateAudio, materialSignature], () => {
  void refreshEstimate()
})

watch(audioMode, (next) => {
  if (next !== 'reference') clearReferenceAudio()
})

watch(selectedCarrierId, () => {
  // Draft restoration precedes the catalog request. An empty, not-yet-loaded
  // catalog must not clear an explicitly selected model.
  if (catalogLoaded.value && selectedModelId.value && !visibleModels.value.some((item) => item.id === selectedModelId.value)) selectedModelId.value = ''
})

watch(() => route.query.mode, applyModeFromRoute)
watch(() => auth.user?.id, restorePendingCreation)

watch(workflowInputType, (next) => {
  if (catalogLoaded.value && selectedModelId.value && !visibleModels.value.some((item) => item.id === selectedModelId.value)) selectedModelId.value = ''
  activeSlotKey.value = null
  if (next === 'text') syncTextStage()
  else resetStageIdle()
})

</script>

<template>
  <section class="s2v-page s2v-studio">
    <S2VPageHeader title="创作">
      <template #description>从一个画面出发，把想法变成下一段影片。</template>
      <template #actions>
        <div class="s2v-studio-page-actions">
          <span class="s2v-studio-balance">钱包余额: ¥ {{ auth.balance.toFixed(2) }}</span>
          <S2VButton size="sm" @click="router.push('/app/works')">查看任务<ArrowUpRight class="h-4 w-4" /></S2VButton>
        </div>
      </template>
    </S2VPageHeader>

    <DirectorCreationContext v-if="director.requested.value" :title="director.title.value" :applied="!!director.context.value" :loading="director.loading.value || !catalogLoaded" :blocked="loading || !!pendingCreation || !!recoveryError" :saving="director.saving.value" :can-save="director.canSave.value && audioMode !== 'reference'" :needs-reapply="director.needsReapply.value" :error="director.error.value" :notice="director.notice.value" :return-route="director.returnRoute.value" @apply="director.apply" @reload="director.load" @save="director.save" />

    <GuideCreationContext
      v-if="tutorial.requested.value && !director.requested.value"
      :guide="tutorial.guide.value" :shot="tutorial.shot.value"
      :invalid="tutorial.invalid.value" :unavailable="tutorial.unavailable.value"
      :applied="tutorial.applied.value" :adjusted="tutorial.adjusted.value"
      :loading="loading || optionsLoading"
      @apply="applyTutorial"
    />

    <fieldset :disabled="loading || director.saving.value" class="contents">
      <CreateWorkbenchToolbar
        :workflow-input-type="workflowInputType"
        :modes="creationModes"
        @update:workflow-input-type="workflowInputType = $event"
      />
    </fieldset>

    <CreateAvailabilityNotice
      v-if="catalogError || optionsError"
      :error="catalogError || optionsError"
      :loading="!catalogLoaded || optionsLoading"
      @retry="loadCreationCatalog"
    />

    <div class="s2v-studio-layout">
      <fieldset :disabled="loading || director.saving.value" class="s2v-studio-form">
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
        <CreatePromptSubmit
          v-model:prompt="prompt"
          :cost="quotedCost"
          :loading="loading"
          :estimate-loading="estimateLoading"
          :estimate-error="estimateError"
          :preparation-hint="preparationHint"
          :submit-hint="submitHint"
          :submission-error="recoveryError || submissionError"
          :pending="pendingCreation"
          :disabled="!canSubmit"
          :unavailable="generationUnavailable"
          @submit="submit"
          @retry-estimate="refreshEstimate"
          @recover="recoverSubmission"
        >
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
            :supports-reference-audio="!director.context.value"
            :audio-mode="audioMode"
            :reference-audio-name="referenceAudio?.fileName"
            @update:ratio="ratio = $event"
            @update:duration-sec="durationSec = $event"
            @update:resolution="resolution = $event"
            @update:carrier-id="selectedCarrierId = $event"
            @update:model-id="selectedModelId = $event"
            @update:audio-mode="audioMode = $event"
            @reference-audio-upload="selectReferenceAudio"
            @clear-reference-audio="clearReferenceAudio"
          />
        </CreatePromptSubmit>
      </fieldset>

      <div class="s2v-studio-results">
        <CreateStagePlayer
          :title="stageTitle"
          :subtitle="stageSubtitle"
          :preview-url="stagePreviewUrl"
          :video-url="stageVideoUrl"
          :loading="stageLoading"
          :error="stageError"
          :task-id="activeTaskId"
          @view-detail="router.push(`/app/works/${activeTaskId}`)"
        />
        <CreateFilmstrip
          :tasks="recentTasks"
          :active-task-id="activeTaskId"
          :loading="recentTasksLoading"
          :error="recentTasksError"
          @select="setStageFromTask"
          @view-all="router.push('/app/works')"
          @retry="loadRecentTasks"
        />
      </div>
    </div>

    <AssetPickDialog
      v-model:open="assetDialogOpen"
      :assets="director.context.value ? director.referenceAssets.value : undefined"
      @pick="confirmAssetPick"
      @upload="confirmLocalUpload"
    />
  </section>
</template>
