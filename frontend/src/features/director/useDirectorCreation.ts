import { computed, onBeforeUnmount, shallowRef, watch, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import { directorErrorMessage, directorShotDocument, fetchDirectorBoard, saveDirectorBoard, type DirectorBoard, type DirectorContext, type DirectorShotInput } from '@/api/director'
import { fetchProjectAssets, type CreationCatalog } from '@/api/client'
import { emptyCreateDraft, emptySlot, loadCreateDraft, type CreateDraft, type CreateFrameSlot, type WorkflowInputType } from '@/shared/lib/create-draft'
import type { PendingCreation } from '@/shared/lib/pending-creation'
import { getAuthSession } from '@/shared/lib/auth-session'
import type { Asset } from '@/data/projects'

export function directorDraftScope(context?: DirectorContext | null) { return context ? `${context.projectId}/${context.shotId}` : '' }

export function useDirectorCreation(options: {
  ownerId: string
  catalog: Ref<CreationCatalog>
  pending: Ref<PendingCreation | null>
  blocked: () => boolean
  persist: () => void
  refreshPending: () => void
  apply: (draft: CreateDraft) => void
  reset: () => void
  prepareShot: (isCurrent: () => boolean) => Promise<DirectorShotInput>
}) {
  const route = useRoute()
  const context = shallowRef<DirectorContext | null>(null)
  const board = shallowRef<DirectorBoard | null>(null)
  const appliedBoard = shallowRef<DirectorBoard | null>(null)
  const loading = shallowRef(false)
  const saving = shallowRef(false)
  const needsReapply = shallowRef(false)
  const error = shallowRef('')
  const notice = shallowRef('')
  const referenceAssets = shallowRef<Asset[]>([])
  const query = computed(() => ({ projectId: typeof route.query.directorProject === 'string' ? route.query.directorProject : '', shotId: typeof route.query.directorShot === 'string' ? route.query.directorShot : '' }))
  const requested = computed(() => !!(query.value.projectId || query.value.shotId || context.value))
  const shot = computed(() => board.value?.shots.find((item) => item.id === query.value.shotId))
  const title = computed(() => context.value && (context.value.projectId !== query.value.projectId || context.value.shotId !== query.value.shotId)
    ? options.pending.value?.payload.title || '待核对镜头'
    : shot.value?.title)
  const scope = computed(() => directorDraftScope(context.value))
  const canSave = computed(() => !!context.value && !!appliedBoard.value && !loading.value && !saving.value && !needsReapply.value && !options.pending.value && !options.blocked())
  const returnRoute = computed(() => {
    const target = context.value || query.value
    return target.projectId ? { path: `/app/projects/${encodeURIComponent(target.projectId)}/director`, query: { shot: target.shotId } } : { path: '/app/assets' }
  })
  let version = 0
  let disposed = false
  async function load() {
    if (saving.value) return
    const request = ++version
    board.value = null; error.value = ''; loading.value = false
    if (!query.value.projectId || !query.value.shotId) {
      if (requested.value && !context.value) error.value = '镜头入口不完整，请返回作品重新选择镜头。'
      return
    }
    const session = getAuthSession().id
    loading.value = true
    try {
      const result = await fetchDirectorBoard(query.value.projectId)
      if (disposed || request !== version || session !== getAuthSession().id) return
      board.value = result
      if (!shot.value) error.value = '这个镜头已不存在，请返回导演台重新选择。'
    } catch (cause) { if (!disposed && request === version && session === getAuthSession().id) error.value = directorErrorMessage(cause, '镜头信息暂时无法读取，请重试。') }
    finally { if (!disposed && request === version) loading.value = false }
  }
  function restorePending() { if (options.pending.value?.payload.directorContext) context.value = { ...options.pending.value.payload.directorContext } }
  async function apply() {
    options.refreshPending()
    if (loading.value || saving.value || options.blocked() || options.pending.value || !shot.value || !board.value) return
    const current = shot.value
    const selectedBoard = board.value
    const configured = current.modeId ? options.catalog.value.modes.find((mode) => mode.id === current.modeId && mode.enabled && mode.inputType !== 'unsupported') : options.catalog.value.modes.find((mode) => mode.inputType === 'text' && mode.enabled)
    if (!configured) { error.value = '这个镜头的创作模式暂不可用，请返回导演台或稍后重试。'; return }
    const request = ++version
    const session = getAuthSession().id
    loading.value = true; notice.value = ''
    try {
      const assets = new Map<string, { name: string }>()
      const projectAssets: Asset[] = []
      let assetsUnavailable = false
      try {
        const result = await fetchProjectAssets(selectedBoard.projectId, 'image')
        for (const asset of result.items || []) {
          if (asset.type !== 'image') continue
          assets.set(asset.id, asset)
          projectAssets.push({ id: asset.id, name: asset.name, type: '图片', size: asset.size, folder: '', updated: asset.time, url: `/api/v1/client/projects/${encodeURIComponent(selectedBoard.projectId)}/files/${encodeURIComponent(asset.id)}` })
        }
      } catch { assetsUnavailable = true }
      if (disposed || request !== version || session !== getAuthSession().id) return
      options.refreshPending()
      if (options.blocked() || options.pending.value) return
      options.persist()
      context.value = { projectId: selectedBoard.projectId, shotId: current.id, expectedRevision: selectedBoard.revision }
      appliedBoard.value = { ...selectedBoard, shots: selectedBoard.shots.map(directorShotDocument) }
      referenceAssets.value = projectAssets
      const savedDraft = loadCreateDraft(options.ownerId, scope.value)
      const cached = savedDraft && window.confirm('这个镜头有尚未提交的本地草稿，是否恢复？选择取消将使用导演台已保存的镜头内容。') ? savedDraft : null
      let unresolved = false
      const slot = (source?: { type: 'upload' | 'asset'; id: string }): CreateFrameSlot => {
        if (!source) return emptySlot()
        const asset = assets.get(source.id)
        if (source.type === 'asset' && !asset) unresolved = true
        return { ...emptySlot(), uploaded: true, assetId: source.type === 'asset' ? source.id : '', uploadId: source.type === 'upload' ? source.id : undefined,
          previewUrl: source.type === 'upload' ? `/api/v1/client/uploads/${encodeURIComponent(source.id)}` : asset ? `/api/v1/client/projects/${encodeURIComponent(selectedBoard.projectId)}/files/${encodeURIComponent(source.id)}` : undefined,
          fileName: source.type === 'upload' ? '已保存参考图片' : asset?.name || '参考素材待核对',
        }
      }
      const draft: CreateDraft = cached || { ...emptyCreateDraft(), workflowInputType: configured.inputType as WorkflowInputType, prompt: current.prompt, durationSec: String(current.durationSec || 5), resolution: current.resolution || '768P', ratio: current.aspectRatio || '16:9', carrierId: current.carrierId, modelId: current.modelId, audioMode: current.generateAudio ? 'generate' : 'none', firstFrame: slot(current.sourceMeta.first), lastFrame: slot(current.sourceMeta.last), extraFrames: (current.sourceMeta.extras || []).map(slot), multiFrames: current.sourceMeta.refs?.length ? current.sourceMeta.refs.map(slot) : [emptySlot()] }
      if (unresolved) notice.value = '部分参考素材不在当前作品或暂时无法读取预览。素材引用仍保留，请在素材槽重新选择并核对后提交。'
      else if (assetsUnavailable) notice.value = '这个作品的素材暂时无法读取，你仍可上传本地图片，或返回导演台后重试。'
      options.apply(draft)
      needsReapply.value = false
      error.value = ''
    } catch { if (!disposed && session === getAuthSession().id) error.value = '无法保留创作草稿，暂未应用镜头。请检查浏览器本地存储后重试。' }
    finally { if (!disposed && request === version) loading.value = false }
  }
  async function save() {
    options.refreshPending()
    if (!canSave.value || !context.value || !appliedBoard.value) return
    const selected = { ...context.value }
    const snapshot = appliedBoard.value
    const session = getAuthSession().id
    const request = ++version
    const isCurrent = () => !disposed && request === version && session === getAuthSession().id
      && context.value?.projectId === selected.projectId && context.value?.shotId === selected.shotId
      && query.value.projectId === selected.projectId && query.value.shotId === selected.shotId
    saving.value = true; error.value = ''; notice.value = ''
    let writeAttempted = false
    try {
      const input = await options.prepareShot(isCurrent)
      if (!isCurrent()) return
      options.refreshPending()
      if (options.pending.value) {
        error.value = '发现一笔提交等待确认，素材已保留在本地草稿，请先核对原提交。'
        options.persist()
        return
      }
      options.persist()
      const shots = snapshot.shots.map((item) => directorShotDocument(item.id === selected.shotId ? { ...item, ...input } : item))
      if (!shots.some((item) => item.id === selected.shotId)) {
        needsReapply.value = true
        error.value = '镜头已不可用，请重新读取并应用镜头。'
        return
      }
      writeAttempted = true
      const next = await saveDirectorBoard(selected.projectId, selected.expectedRevision, shots)
      if (!isCurrent()) return
      appliedBoard.value = next
      board.value = next
      context.value = { ...selected, expectedRevision: next.revision }
      notice.value = '镜头素材与参数已保存，未创建任务或扣费。'
    } catch (cause) {
      if (!isCurrent()) return
      // Retain uploaded IDs even when the board rejects a revision or a save
      // response is lost. Re-reading must never silently overwrite this draft.
      try { options.persist() } catch { /* Keep the mounted form intact. */ }
      const response = (cause as { response?: { status?: number; data?: { code?: string } } }).response
      const conflict = response?.status === 409 || response?.status === 404
      const uncertain = writeAttempted && (!response || response.status === undefined || response.status >= 500)
      needsReapply.value = conflict || uncertain
      error.value = conflict
        ? '镜头板已变化，本地编辑仍保留。请重新读取镜头并明确重新应用，再核对内容后保存。'
        : uncertain
          ? '尚未确认镜头保存结果，本地编辑仍保留。请重新读取镜头核对，不要反复保存。'
          : directorErrorMessage(cause, '镜头尚未保存。请检查素材与浏览器存储后重试，本次未创建任务或扣费。')
    } finally { if (!disposed && request === version) saving.value = false }
  }
  watch(query, (next, previous) => {
    if (previous && (next.projectId !== previous.projectId || next.shotId !== previous.shotId) && context.value && !options.pending.value) {
      options.persist()
      context.value = null
      appliedBoard.value = null
      needsReapply.value = false
      saving.value = false
      options.reset()
    }
    void load()
  }, { immediate: true })
  onBeforeUnmount(() => { disposed = true; version++ })
  return { context, requested, shot, title, scope, loading, saving, canSave, needsReapply, error, notice, referenceAssets, returnRoute, load, apply, save, restorePending }
}
