import { computed, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router'
import { attachDirectorCandidate, directorErrorMessage, directorShotDocument, fetchDirectorBoard, saveDirectorBoard, type DirectorBoard, type DirectorShot } from '@/api/director'
import { fetchProjectTree } from '@/api/client'
import { getAuthSession, onAuthSessionChange } from '@/shared/lib/auth-session'

export function useDirectorWorkspace() {
  const route = useRoute()
  const router = useRouter()
  const ownerSession = getAuthSession().id
  const projectId = computed(() => String(route.params.projectId || ''))
  const projectName = shallowRef('作品')
  const board = ref<DirectorBoard | null>(null)
  const savedContent = shallowRef('')
  const activeShotId = shallowRef('')
  const loading = shallowRef(false)
  const saving = shallowRef(false)
  const error = shallowRef('')
  const message = shallowRef('')
  let version = 0
  let disposed = false
  const dirty = computed(() => !!board.value && JSON.stringify(board.value.shots) !== savedContent.value)
  const activeShot = computed(() => board.value?.shots.find((shot) => shot.id === activeShotId.value) || null)
  const selectedCount = computed(() => board.value?.shots.filter((shot) => shot.selectedTaskId).length || 0)

  function accept(next: DirectorBoard) {
    board.value = next
    savedContent.value = JSON.stringify(next.shots)
    if (!next.shots.some((shot) => shot.id === activeShotId.value)) {
      const requested = typeof route.query.shot === 'string' ? route.query.shot : ''
      activeShotId.value = next.shots.find((shot) => shot.id === requested)?.id || next.shots[0]?.id || ''
    }
  }
  async function load(confirmDiscard = false) {
    if (saving.value || (confirmDiscard && dirty.value && !window.confirm('重新加载会放弃尚未保存的镜头编辑。可以先导出清单，是否继续？'))) return
    const request = ++version
    const id = projectId.value
    const session = getAuthSession().id
    loading.value = true
    error.value = ''; message.value = ''
    try {
      const [next, project] = await Promise.all([fetchDirectorBoard(id), fetchProjectTree(id)])
      if (disposed || request !== version || session !== getAuthSession().id) return
      projectName.value = project.name
      accept(next)
    } catch (cause) {
      if (!disposed && request === version && session === getAuthSession().id) error.value = directorErrorMessage(cause, '导演台暂时无法读取，请重新加载。')
    } finally { if (!disposed && request === version) loading.value = false }
  }
  async function save() {
    if (!board.value || saving.value) return false
    if (!dirty.value) return true
    const current = board.value
    const session = getAuthSession().id
    saving.value = true; error.value = ''; message.value = ''
    try {
      const next = await saveDirectorBoard(current.projectId, current.revision, current.shots)
      if (disposed || session !== getAuthSession().id || current.projectId !== projectId.value) return false
      accept(next)
      message.value = '镜头板已保存'
      return true
    } catch (cause) {
      if (!disposed && session === getAuthSession().id) error.value = directorErrorMessage(cause, '保存结果尚未确认，你的编辑仍保留。请重新加载核对，也可先导出清单。')
      return false
    } finally { if (!disposed) saving.value = false }
  }
  function addShot() {
    if (!board.value || saving.value || loading.value || board.value.shots.length >= 100) return
    const id = `shot_${Array.from(crypto.getRandomValues(new Uint32Array(3)), (n) => n.toString(16)).join('_')}`
    board.value.shots.push({ id, title: `镜头 ${board.value.shots.length + 1}`, prompt: '', modeId: '', carrierId: '', modelId: '', durationSec: 0, resolution: '', aspectRatio: '', generateAudio: false, sourceMeta: {}, candidateTaskIds: [], selectedTaskId: '' })
    activeShotId.value = id
    message.value = ''
  }
  function updateShot(value: Pick<DirectorShot, 'title' | 'prompt'>) {
    if (!activeShot.value || saving.value || loading.value) return
    Object.assign(activeShot.value, value)
    message.value = ''
  }
  function moveShot(id: string, direction: -1 | 1) {
    if (!board.value || saving.value || loading.value) return
    const index = board.value.shots.findIndex((shot) => shot.id === id)
    const target = index + direction
    if (index < 0 || target < 0 || target >= board.value.shots.length) return
    const shot = board.value.shots.splice(index, 1)[0]
    if (shot) board.value.shots.splice(target, 0, shot)
    message.value = ''
  }
  function removeShot() {
    if (!board.value || !activeShot.value || saving.value || !window.confirm(`移除「${activeShot.value.title}」？关联任务和已交付视频会保留，保存镜头板后生效。`)) return
    board.value.shots = board.value.shots.filter((shot) => shot.id !== activeShotId.value)
    activeShotId.value = board.value.shots[0]?.id || ''
    message.value = ''
  }
  async function createShot() {
    if (!activeShot.value || !await save() || !activeShot.value) return
    await router.push({ path: '/app/create', query: { directorProject: projectId.value, directorShot: activeShot.value.id } })
  }
  async function selectCandidate(taskId: string) {
    if (!activeShot.value || saving.value) return
    activeShot.value.selectedTaskId = taskId
    await save()
  }
  async function attachCandidate(taskId: string) {
    const shotId = activeShotId.value
    if (!shotId || !await save() || !board.value) return false
    const id = projectId.value
    const session = getAuthSession().id
    saving.value = true; error.value = ''; message.value = ''
    try {
      const next = await attachDirectorCandidate(id, shotId, taskId)
      if (disposed || session !== getAuthSession().id || id !== projectId.value) return false
      accept(next); message.value = '已有任务已加入候选，未创建新任务'
      return true
    } catch (cause) {
      if (!disposed && session === getAuthSession().id) error.value = directorErrorMessage(cause, '关联结果尚未确认，请重新加载核对。重复关联不会创建新任务。')
      return false
    } finally { if (!disposed) saving.value = false }
  }
  function exportBoard() {
    if (!board.value) return
    const document = { format: 'sub2video-director-board', version: 1, description: '镜头编排与选片清单，不是合成视频。', saved: !dirty.value, projectName: projectName.value, projectId: board.value.projectId, revision: board.value.revision, updatedAt: board.value.updatedAt, shots: board.value.shots.map(directorShotDocument) }
    const url = URL.createObjectURL(new Blob([JSON.stringify(document, null, 2)], { type: 'application/json;charset=utf-8' }))
    const link = window.document.createElement('a')
    link.href = url; link.download = `director-${projectId.value}.json`
    window.document.body.append(link); link.click(); link.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  }
  const stopSessionListener = onAuthSessionChange((next) => {
    if (next.id === ownerSession) return
    version++
    board.value = null; savedContent.value = ''; activeShotId.value = ''
    projectName.value = '作品'; loading.value = false; saving.value = false
    error.value = ''; message.value = ''
  })
  function beforeUnload(event: BeforeUnloadEvent) { if (getAuthSession().id === ownerSession && dirty.value) { event.preventDefault(); event.returnValue = '' } }
  window.addEventListener('beforeunload', beforeUnload)
  onBeforeRouteLeave(() => getAuthSession().id !== ownerSession || !getAuthSession().accessToken || !dirty.value || window.confirm('镜头编辑尚未保存。离开会丢失修改，是否继续？'))
  onBeforeRouteUpdate((to, from) => to.params.projectId === from.params.projectId || getAuthSession().id !== ownerSession || !dirty.value || window.confirm('镜头编辑尚未保存。切换作品会丢失修改，是否继续？'))
  watch(projectId, () => { board.value = null; activeShotId.value = ''; void load() }, { immediate: true })
  onBeforeUnmount(() => { disposed = true; version++; stopSessionListener(); window.removeEventListener('beforeunload', beforeUnload) })
  return { projectId, projectName, board, activeShotId, activeShot, dirty, selectedCount, loading, saving, error, message, load, save, addShot, updateShot, moveShot, removeShot, createShot, selectCandidate, attachCandidate, exportBoard }
}
