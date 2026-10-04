<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertCircle, ArrowLeft, ArrowUpRight, BookmarkPlus, Check, Download, Film, LoaderCircle, RefreshCw } from 'lucide-vue-next'
import { fetchProjects, saveTaskToProject, type ProjectSummary } from '@/api/client'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'
import { emptySlot, saveCreateDraft, type CreateFrameSlot, type WorkflowInputType } from '@/shared/lib/create-draft'
import { useTaskDelivery } from './useTaskDelivery'
import { downloadTaskFile } from './download'
import './delivery.css'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()
const taskId = computed(() => String(route.params.id || ''))
const { task, loading, refreshing, loadError, pollingPaused, pending, downloadUrl, refresh } = useTaskDelivery(taskId)
const mediaFailed = ref(false)
const mediaVersion = ref(0)
const downloading = ref(false)
const downloadError = ref('')
const saveOpen = ref(false)
const projectsLoading = ref(false)
const projectsError = ref('')
const projects = ref<ProjectSummary[]>([])
const selectedProject = ref('')
const saving = ref(false)
const saveError = ref('')
const savedDestination = ref<{ project: string; folder: string } | null>(null)
let projectRequest = 0
let disposed = false

const projectOptions = computed(() => [{ label: '请选择作品', value: '' }, ...projects.value.map((item) => ({ label: item.name, value: item.id }))])
const statusTone = computed(() => task.value?.reviewRequired ? 'warning' : task.value?.status === '失败' ? 'danger' : task.value?.status === '已完成' ? 'success' : 'primary')
const persistentDestination = computed(() => {
  const saved = task.value?.savedDestination
  return saved ? { project: saved.projectId, folder: saved.folderPath.join('/') } : savedDestination.value
})
const supportRoute = computed(() => ({ path: '/app/tickets', query: { taskId: taskId.value } }))
const deliveryLabel = computed(() => downloadUrl.value ? '可下载' : task.value?.deliveryStatus === 'pending_storage' ? '正在准备成片' : '尚未交付')
const previewTitle = computed(() => {
  if (task.value?.reviewRequired) return '提交结果正在核对'
  if (mediaFailed.value) return '视频预览暂不可用'
  if (task.value?.status === '失败') return '这一镜未能完成'
  if (task.value?.executionWaiting) return '任务正在等待恢复执行'
  if (task.value?.deliveryStatus === 'pending_storage') return '正在准备交付成片'
  if (task.value?.status === '生成中') return '你的画面正在生成'
  if (task.value?.status === '排队中') return '任务已提交，正在排队'
  return '成片暂不可预览'
})
const previewDescription = computed(() => {
  if (task.value?.reviewRequired) return '平台正在确认本次提交结果，请勿重复提交。费用不会重复扣除；如需了解进展，请关联这条任务联系支持。'
  if (mediaFailed.value) return '可以下载成片到本地播放，也可以重新加载预览。'
  if (task.value?.status === '失败') return '请在钱包查看消费与退款记录，或通过帮助与工单联系支持。'
  if (task.value?.deliveryStatus === 'pending_storage') return '视频已生成，完成保存后即可预览、下载和收藏到作品。'
  if (task.value?.executionWaiting) return '本次任务仍然保留，恢复后会继续执行。你可以稍后回来查看。'
  return '生成与交付状态会自动更新，你也可以稍后从任务列表继续查看。'
})

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.valueOf()) ? '—' : date.toLocaleString('zh-CN', { hour12: false })
}
async function reload() { await refresh(); mediaFailed.value = false; mediaVersion.value++ }
async function download() {
  if (!downloadUrl.value || downloading.value) return
  downloading.value = true
  downloadError.value = ''
  try { await downloadTaskFile(taskId.value) }
  catch { downloadError.value = '下载暂时不可用，请刷新任务状态后重试；仍有问题可关联任务联系支持。' }
  finally { downloading.value = false }
}

async function loadProjects() {
  const request = ++projectRequest
  projectsLoading.value = true
  projectsError.value = ''
  try {
    const result = await fetchProjects(false)
    if (disposed || request !== projectRequest) return
    projects.value = result.items
    selectedProject.value = result.items.some((item) => item.id === result.currentProjectId) ? result.currentProjectId : ''
  } catch {
    if (!disposed && request === projectRequest) projectsError.value = '作品列表暂时无法读取，请重试。'
  } finally {
    if (!disposed && request === projectRequest) projectsLoading.value = false
  }
}
function openSave() { saveOpen.value = true; saveError.value = ''; void loadProjects() }

async function save() {
  if (saving.value || !selectedProject.value || !task.value || !downloadUrl.value) return
  const id = task.value.id
  const projectId = selectedProject.value
  saving.value = true
  saveError.value = ''
  try {
    const result = await saveTaskToProject(id, projectId)
    if (disposed || taskId.value !== id) return
    if (task.value) task.value.savedToProject = true
    savedDestination.value = { project: projectId, folder: result.path.join('/') }
    saveOpen.value = false
    app.notify('成片已保存到作品')
    void app.refreshStorage()
  } catch {
    if (!disposed && taskId.value === id) saveError.value = '保存失败，请稍后重试；如已在其他页面保存，可重新加载任务状态。'
  } finally { saving.value = false }
}
function viewProject() { void router.push(persistentDestination.value ? { path: '/app/assets', query: persistentDestination.value } : '/app/assets') }

function retryInputType(mode: string): WorkflowInputType {
  if (mode.includes('多图')) return 'multi_image'
  if (mode.includes('首尾')) return 'first_last'
  return 'text'
}

function draftSlot(source?: { type: 'upload' | 'asset'; id: string }): CreateFrameSlot {
  if (!source?.id) return emptySlot()
  return {
    ...emptySlot(),
    uploaded: true,
    assetId: source.type === 'asset' ? source.id : '',
    uploadId: source.type === 'upload' ? source.id : undefined,
  }
}

function retryTask() {
  const current = task.value
  if (!current || current.reviewRequired || !auth.user?.id) return
  if (!window.confirm('将使用这次任务的模式、素材和参数覆盖当前创作草稿，是否继续？')) return
  const source = current.sourceMeta || {}
  const inputType = retryInputType(current.mode)
  const refs = source.refs || []
  saveCreateDraft(auth.user.id, {
    workflowInputType: inputType,
    prompt: current.prompt || '',
    ratio: current.aspectRatio || '16:9',
    durationSec: String(current.durationSeconds || 5),
    resolution: current.resolution,
    carrierId: current.carrierId,
    modelId: current.modelId,
    audioMode: source.audio ? 'reference' : current.generateAudio ? 'generate' : 'none',
    referenceAudioId: source.audio?.type === 'upload' ? source.audio.id : undefined,
    referenceAudioName: source.audio ? '已保存参考音频' : undefined,
    firstFrame: draftSlot(source.first),
    lastFrame: draftSlot(source.last),
    extraFrames: (source.extras || []).map(draftSlot),
    multiFrames: refs.length ? refs.map(draftSlot) : [emptySlot()],
  })
  void router.push({ path: '/app/create', query: { mode: inputType } })
}

watch(taskId, () => { mediaFailed.value = false; downloadError.value = ''; saveOpen.value = false; savedDestination.value = null; projectRequest++ })
watch(downloadUrl, () => { mediaFailed.value = false })
watch(() => [task.value?.id, route.query.action], () => {
  if (!task.value || loadError.value) return
  const action = route.query.action
  if (action !== 'save' && action !== 'reuse') return
  void router.replace({ path: route.path, query: { ...route.query, action: undefined } })
  if (task.value.reviewRequired) return
  if (action === 'save' && downloadUrl.value && !task.value.savedToProject) openSave()
  else if (action === 'reuse' && ['失败', '已完成'].includes(task.value.status)) retryTask()
})
onBeforeUnmount(() => { disposed = true; projectRequest++ })
</script>

<template>
  <section class="s2v-page s2v-delivery">
    <S2VPageHeader :title="task?.title || '任务详情'">
      <template #description>查看这一镜的生成与交付状态，留下满意的结果。</template>
      <template #actions><S2VButton @click="router.push('/app/works')"><ArrowLeft class="h-4 w-4" />返回任务</S2VButton></template>
    </S2VPageHeader>
    <div v-if="loading" class="s2v-delivery-state" role="status"><LoaderCircle class="h-7 w-7 animate-spin" /><h2>正在读取任务详情</h2></div>
    <div v-else-if="!task" class="s2v-delivery-state" role="alert">
      <AlertCircle class="h-7 w-7" /><h2>{{ loadError || '暂时无法显示任务' }}</h2>
      <S2VButton :loading="refreshing" @click="reload"><RefreshCw class="h-4 w-4" />重新加载</S2VButton>
    </div>
    <template v-else>
      <div v-if="loadError" class="s2v-delivery-notice" role="alert">
        <AlertCircle class="h-4 w-4" /><p>{{ loadError }}{{ pollingPaused ? '自动刷新已暂停。' : '当前显示上次读取的状态。' }}</p>
        <S2VButton size="sm" :loading="refreshing" @click="reload">重新加载</S2VButton>
      </div>
      <div class="s2v-delivery-layout">
        <div class="s2v-delivery-main">
          <div class="s2v-delivery-screen">
            <video v-if="downloadUrl && !mediaFailed" :key="`${downloadUrl}-${mediaVersion}`" :src="downloadUrl" controls playsinline preload="metadata" :aria-label="`${task.title} 成片预览`" @error="mediaFailed = true" />
            <div v-else class="s2v-delivery-empty" role="status">
              <LoaderCircle v-if="pending && !mediaFailed" class="h-8 w-8 animate-spin" /><Film v-else class="h-8 w-8" />
              <h2>{{ previewTitle }}</h2><p>{{ previewDescription }}</p>
              <S2VButton v-if="mediaFailed" :loading="refreshing" @click="reload"><RefreshCw class="h-4 w-4" />重新加载预览</S2VButton>
              <RouterLink v-if="task.status === '失败'" class="s2v-delivery-link" to="/app/wallet">查看钱包流水 <ArrowUpRight class="h-4 w-4" /></RouterLink>
              <RouterLink v-if="task.status === '失败' || task.reviewRequired" class="s2v-delivery-link" :to="supportRoute">关联此任务联系支持 <ArrowUpRight class="h-4 w-4" /></RouterLink>
            </div>
          </div>
          <div class="s2v-delivery-after">
            <div><h2>{{ task.reviewRequired ? '任务已保留，请等待核对结果。' : downloadUrl ? '留下满意的一镜，继续下一次创作。' : pending ? '任务在后台继续，你可以安心离开。' : '调整思路，开始下一次创作。' }}</h2><p>{{ task.reviewRequired ? '无需再次提交同一任务；状态更新后可在这里继续查看。' : '每次创作都需要重新确认素材、参数与费用。' }}</p></div>
            <div class="s2v-delivery-after-actions">
              <S2VButton v-if="!task.reviewRequired && (task.status === '失败' || task.status === '已完成')" size="lg" @click="retryTask">按原参数再来一次 <RefreshCw class="h-4 w-4" /></S2VButton>
              <S2VButton v-if="!task.reviewRequired" size="lg" @click="router.push('/app/create')">继续创作 <ArrowUpRight class="h-4 w-4" /></S2VButton>
            </div>
          </div>
        </div>
        <aside class="s2v-delivery-details" aria-label="任务与交付">
          <div class="s2v-delivery-section-heading"><h2>任务与交付</h2><S2VButton size="sm" variant="ghost" :loading="refreshing" @click="reload"><RefreshCw class="h-3.5 w-3.5" />刷新状态</S2VButton></div>
          <div class="s2v-delivery-status-row"><span>生成状态</span><S2VBadge :tone="statusTone">{{ task.reviewRequired ? '待核对' : task.status }}<span v-if="task.status === '生成中' && !task.reviewRequired"> · {{ task.progress }}%</span></S2VBadge></div>
          <div class="s2v-delivery-status-row"><span>交付状态</span><span>{{ deliveryLabel }}</span></div>
          <div v-if="task.status === '失败'" class="s2v-delivery-refund" role="status">
            <p class="s2v-delivery-failure">{{ task.failureReason || '本次生成未完成' }}</p>
            <p v-if="task.refundAmount > 0">已退回 ¥{{ task.refundAmount.toFixed(2) }}<span v-if="task.refundedAt"> · {{ formatDate(task.refundedAt) }}</span></p>
            <p v-else-if="task.cost > 0">暂未查到退款记录，可查看钱包流水或关联任务联系支持。</p>
            <p v-else>本次任务未收取费用，无需退款。</p>
          </div>
          <dl class="s2v-delivery-parameters">
            <div><dt>任务编号</dt><dd class="font-mono">{{ task.id }}</dd></div>
            <div><dt>创作模式</dt><dd>{{ task.mode }}</dd></div>
            <div><dt>视频时长</dt><dd>{{ task.duration }}</dd></div>
            <div><dt>分辨率 / 画幅</dt><dd>{{ task.resolution }} / {{ task.aspectRatio || '—' }}</dd></div>
            <div><dt>自动生成音频</dt><dd>{{ task.generateAudio ? '需要' : '不需要' }}</dd></div>
            <div><dt>本次消费</dt><dd class="font-mono">¥ {{ task.cost.toFixed(2) }}</dd></div>
            <div><dt>创建时间</dt><dd>{{ formatDate(task.createdAt) }}</dd></div>
          </dl>
          <div class="s2v-delivery-actions">
            <S2VButton v-if="downloadUrl" size="lg" block :loading="downloading" :disabled="downloading" @click="download"><Download class="h-4 w-4" />{{ downloading ? '准备下载…' : '下载成片' }}</S2VButton>
            <p v-if="downloadError" role="alert">{{ downloadError }}</p>
            <template v-if="task.savedToProject">
              <p class="s2v-delivery-saved"><Check class="h-4 w-4" />已保存到作品</p>
              <S2VButton size="lg" variant="primary" block @click="viewProject">{{ persistentDestination ? '查看作品' : '前往资产查找' }} <ArrowUpRight class="h-4 w-4" /></S2VButton>
            </template>
            <S2VButton v-else-if="downloadUrl" size="lg" variant="primary" block @click="openSave"><BookmarkPlus class="h-4 w-4" />保存到作品</S2VButton>
            <p>{{ downloadUrl ? '无法播放或下载时，请刷新状态后再试。' : '成片交付完成后，可下载或保存到作品。' }}</p>
          </div>
        </aside>
      </div>
    </template>
    <S2VDialog v-model:open="saveOpen" title="保存到作品" description="选择成片归属的作品，保存后可在视频目录继续整理。" hide-actions>
      <div class="s2v-delivery-save">
        <p v-if="projectsLoading" role="status">正在读取作品列表…</p>
        <div v-else-if="projectsError" role="alert"><p>{{ projectsError }}</p><S2VButton size="sm" @click="loadProjects">重新加载作品</S2VButton></div>
        <template v-else-if="projects.length">
          <S2VSelect v-model="selectedProject" label="目标作品" :options="projectOptions" :disabled="saving" />
          <p>保存目录：视频 / 正片镜头</p>
        </template>
        <p v-else>还没有作品。先到资产页新建一个，再回来保存成片。</p>
        <p v-if="saveError" class="s2v-delivery-save-error" role="alert">{{ saveError }}</p>
        <div class="s2v-delivery-save-footer">
          <S2VButton :disabled="saving" @click="saveOpen = false">取消</S2VButton>
          <S2VButton v-if="!projectsLoading && !projectsError && !projects.length" variant="primary" @click="router.push('/app/assets')">前往资产</S2VButton>
          <S2VButton v-else variant="primary" :loading="saving" :disabled="projectsLoading || !!projectsError || !selectedProject" @click="save">保存成片</S2VButton>
        </div>
      </div>
    </S2VDialog>
  </section>
</template>
