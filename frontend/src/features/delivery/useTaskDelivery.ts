import { computed, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { fetchTask, type ClientTaskDetail } from '@/api/client'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'

export function publicMediaUrl(value: string) {
  if (!value || value.startsWith('//')) return ''
  try {
    if (value.startsWith('/')) {
      const relative = new URL(value, window.location.origin)
      return relative.origin === window.location.origin ? assetPreviewUrl('', value) : ''
    }
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol) ? url.href : ''
  } catch { return '' }
}

export function useTaskDelivery(taskId: Ref<string>) {
  const task = ref<ClientTaskDetail | null>(null)
  const loading = ref(false)
  const refreshing = ref(false)
  const loadError = ref('')
  const notFound = ref(false)
  const pollingPaused = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let version = 0
  let disposed = false
  let failures = 0
  let terminalError = false

  const downloadUrl = computed(() => task.value?.deliveryStatus === 'ready' ? publicMediaUrl(task.value.downloadUrl) : '')
  const pending = computed(() => !!task.value && task.value.status !== '失败' && (
    ['排队中', '生成中'].includes(task.value.status)
    || task.value.deliveryStatus === 'pending_storage'
    || task.value.executionWaiting
  ))

  function clearPoll() {
    if (timer) clearTimeout(timer)
    timer = undefined
  }

  function schedulePoll() {
    clearPoll()
    if (disposed || terminalError || failures >= 3 || !pending.value || document.hidden) return
    timer = setTimeout(() => { void load() }, Math.min(5000 * 2 ** failures, 30000))
  }

  async function load(manual = false) {
    clearPoll()
    if (manual) {
      failures = 0
      terminalError = false
      pollingPaused.value = false
    }
    const request = ++version
    const id = taskId.value
    loading.value = !task.value
    refreshing.value = true
    if (manual) loadError.value = ''
    try {
      const result = await fetchTask(id)
      if (disposed || request !== version) return
      task.value = result
      notFound.value = false
      loadError.value = ''
      failures = 0
      pollingPaused.value = false
    } catch (error) {
      if (disposed || request !== version) return
      const status = (error as { response?: { status?: number } }).response?.status
      notFound.value = status === 404
      terminalError = status === 404 || status === 403 || status === 401
      if (terminalError) task.value = null
      failures++
      pollingPaused.value = failures >= 3
      loadError.value = status === 404
        ? '未找到这个任务，或你没有查看权限。'
        : status === 401
          ? '登录状态已失效，请重新登录后查看任务。'
          : status === 503
            ? '成片交付暂时不可用，请稍后刷新；任务状态仍会保留。'
          : '任务详情暂时无法读取，请重新加载。'
    } finally {
      if (!disposed && request === version) {
        loading.value = false
        refreshing.value = false
        schedulePoll()
      }
    }
  }

  function onVisibilityChange() {
    if (document.hidden) clearPoll()
    else if (pending.value && !refreshing.value && !terminalError && failures < 3) void load()
  }

  watch(taskId, () => {
    version++
    task.value = null
    notFound.value = false
    void load(true)
  }, { immediate: true })

  onMounted(() => document.addEventListener('visibilitychange', onVisibilityChange))
  onBeforeUnmount(() => {
    disposed = true
    version++
    clearPoll()
    document.removeEventListener('visibilitychange', onVisibilityChange)
  })

  return { task, loading, refreshing, loadError, notFound, pollingPaused, pending, downloadUrl, refresh: () => load(true) }
}
