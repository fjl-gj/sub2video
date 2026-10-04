import { computed, onMounted, ref } from 'vue'
import { fetchCreationCatalog, fetchOverview, type CreationCatalog } from '@/api/client'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { getAuthSession } from '@/shared/lib/auth-session'
import type { OverviewData, StudioMode } from './types'

const modeDefinitions = [
  { type: 'text', title: '文生视频', label: '文字 → 画面', description: '用动作、场景与镜头描述开启创作。' },
  { type: 'multi_image', title: '多图生成视频', label: '图片 → 动态', description: '组合参考图片，延续主体与画面风格。' },
  { type: 'first_last', title: '首尾帧生成视频', label: '首帧 → 尾帧', description: '确定起止画面，生成连贯的镜头变化。' },
] as const

export function useStudioOverview() {
  const app = useAppStore()
  const auth = useAuthStore()
  const data = ref<OverviewData | null>(null)
  const catalog = ref<CreationCatalog | null>(null)
  const loading = ref(true)
  const error = ref('')
  const catalogFailed = ref(false)
  let inFlight = false
  const workspaceName = computed(() => data.value?.workspace.displayName || auth.displayName || '创作者')
  const modes = computed<StudioMode[]>(() => modeDefinitions.map((item) => {
    const enabled = catalog.value?.modes.some((mode) => mode.inputType === item.type && mode.enabled)
    const supported = catalog.value?.models.some((model) => model.inputTypes.includes(item.type))
    const available = !!enabled && !!supported
    return {
      ...item,
      available,
      availability: loading.value ? '正在读取可用能力' : catalogFailed.value ? '暂时无法读取能力' : available ? '开始创作' : '暂不可用',
    }
  }))

  async function load() {
    if (inFlight) return
    inFlight = true
    loading.value = true
    error.value = ''
    catalogFailed.value = false
    const sessionId = getAuthSession().id
    const balanceRead = auth.beginBalanceRead()
    const [overviewResult, catalogResult] = await Promise.allSettled([fetchOverview(), fetchCreationCatalog()])
    if (getAuthSession().id !== sessionId) return
    if (overviewResult.status === 'fulfilled') {
      const overview = overviewResult.value as OverviewData
      data.value = overview
      auth.acceptBalanceRead(overview.wallet.balance, balanceRead)
      app.storageUsed = overview.storage.usedGb
      app.storageQuota = overview.storage.quotaGb
    } else {
      error.value = '暂时无法读取工作空间，请重试。'
    }
    if (catalogResult.status === 'fulfilled') {
      catalog.value = catalogResult.value
    } else {
      catalog.value = null
      catalogFailed.value = true
      if (!error.value) error.value = '暂时无法读取可用创作模式，请重新加载。'
    }
    loading.value = false
    inFlight = false
  }

  onMounted(() => { void load() })
  return { data, loading, error, modes, workspaceName, load }
}
