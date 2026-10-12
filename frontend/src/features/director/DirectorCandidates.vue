<script setup lang="ts">
import { computed, onBeforeUnmount, shallowRef, watch } from 'vue'
import { ArrowUpRight, Check, Download, Film, Plus, RefreshCw } from 'lucide-vue-next'
import { fetchTask, type ClientTaskDetail } from '@/api/client'
import type { DirectorShot } from '@/api/director'
import { downloadTaskFile } from '@/features/delivery/download'
import { publicMediaUrl } from '@/features/delivery/useTaskDelivery'
import S2VButton from '@/shared/ui/S2VButton.vue'
import { getAuthSession } from '@/shared/lib/auth-session'

const props = defineProps<{ projectId: string; shot: DirectorShot; disabled: boolean }>()
defineEmits<{ select: [taskId: string]; attach: [] }>()
const tasks = shallowRef<Record<string, ClientTaskDetail>>({})
const failed = shallowRef<string[]>([])
const loading = shallowRef(false)
const error = shallowRef('')
const downloading = shallowRef('')
const previewFailed = shallowRef(false)
let version = 0
let disposed = false
const chosen = computed(() => tasks.value[props.shot.selectedTaskId])
const previewUrl = computed(() => chosen.value?.deliveryStatus === 'ready' ? publicMediaUrl(chosen.value.downloadUrl) : '')
const taskRoute = (id: string) => ({ path: `/app/works/${encodeURIComponent(id)}`, query: { directorProject: props.projectId, directorShot: props.shot.id } })
async function refresh() {
  const request = ++version
  const session = getAuthSession().id
  loading.value = true; error.value = ''
  const ids = [...props.shot.candidateTaskIds]
  const results = await Promise.allSettled(ids.map((id) => fetchTask(id)))
  if (disposed || request !== version || session !== getAuthSession().id) return
  const next: Record<string, ClientTaskDetail> = {}
  const unavailable: string[] = []
  results.forEach((result, index) => {
    const id = ids[index]
    if (!id) return
    if (result.status === 'fulfilled') next[id] = result.value
    else unavailable.push(id)
  })
  tasks.value = next; failed.value = unavailable; loading.value = false
}
async function download(task: ClientTaskDetail) {
  if (downloading.value) return
  const session = getAuthSession().id
  downloading.value = task.id; error.value = ''
  try {
    await downloadTaskFile(task.id)
  } catch { if (!disposed && session === getAuthSession().id) error.value = '下载暂时不可用，请在任务详情中刷新交付状态后重试。' }
  finally { if (!disposed) downloading.value = '' }
}
watch(() => [props.shot.id, ...props.shot.candidateTaskIds], () => { tasks.value = {}; failed.value = []; void refresh() }, { immediate: true })
watch(() => props.shot.selectedTaskId, () => { previewFailed.value = false })
onBeforeUnmount(() => { disposed = true; version++ })
</script>

<template>
  <section class="director-candidates" aria-label="镜头候选">
    <div class="director-section-title"><div><h2>候选成片</h2><p>比较不同版本，选用适合故事的一镜。</p></div><S2VButton size="lg" variant="ghost" :loading="loading" @click="refresh"><RefreshCw class="h-4 w-4" aria-hidden="true" />刷新候选</S2VButton></div>
    <div v-if="chosen" class="director-preview">
      <video v-if="previewUrl && !previewFailed" :key="previewUrl" :src="previewUrl" controls playsinline preload="metadata" :aria-label="`${chosen.title} 已选成片预览`" @error="previewFailed = true" />
      <div v-else class="director-preview-unavailable"><Film class="h-7 w-7" aria-hidden="true" /><p>已选成片暂时无法预览，可打开任务详情或下载查看。</p></div>
      <p class="director-preview-caption"><Check class="h-4 w-4" aria-hidden="true" />当前选用：{{ chosen.title }}</p>
    </div>
    <p v-if="loading" class="director-hint" role="status">正在读取真实任务状态…</p>
    <p v-if="error" class="director-error" role="alert">{{ error }}</p>
    <div v-if="!shot.candidateTaskIds.length" class="director-candidates-empty"><Film class="h-7 w-7" aria-hidden="true" /><h3>还没有候选成片</h3><p>前往创作生成这一镜，或关联一个已有任务。任务关联不会再次扣费。</p></div>
    <div v-else class="director-candidate-grid">
      <article v-for="id in shot.candidateTaskIds" :key="id" class="director-candidate" :aria-label="`候选 ${tasks[id]?.title || id}`">
        <template v-if="tasks[id]">
          <div class="director-candidate-heading"><h3>{{ tasks[id].title }}</h3><span>{{ shot.selectedTaskId === id ? '已选用' : tasks[id].reviewRequired ? '待核对' : tasks[id].deliveryStatus === 'pending_storage' ? '交付准备中' : tasks[id].status }}</span></div>
          <p>{{ tasks[id].duration }} · {{ tasks[id].resolution }} · {{ tasks[id].aspectRatio }}</p>
          <p v-if="tasks[id].deliveryStatus === 'pending_storage'">正在准备交付，完成后可选用。</p>
          <p v-if="tasks[id].status === '失败'">{{ tasks[id].failureReason || '生成未完成' }}{{ tasks[id].refundAmount > 0 ? ` · 已退款 ¥${tasks[id].refundAmount.toFixed(2)}` : '' }}</p>
          <div class="director-candidate-actions"><RouterLink :to="taskRoute(id)" class="director-link">查看任务<ArrowUpRight class="h-4 w-4" aria-hidden="true" /></RouterLink><S2VButton v-if="tasks[id].deliveryStatus === 'ready'" size="lg" variant="ghost" :loading="downloading === id" :disabled="!!downloading" @click="download(tasks[id])"><Download class="h-4 w-4" aria-hidden="true" />下载成片</S2VButton></div>
          <S2VButton v-if="tasks[id].status === '已完成' && tasks[id].deliveryStatus === 'ready' && !tasks[id].reviewRequired" size="lg" :variant="shot.selectedTaskId === id ? 'secondary' : 'primary'" block :disabled="disabled || shot.selectedTaskId === id" @click="$emit('select', id)">{{ shot.selectedTaskId === id ? '已选用这一版' : '采用这一版' }}</S2VButton>
        </template>
        <template v-else><h3>候选任务</h3><p>{{ failed.includes(id) ? '暂时无法读取，可能已不可用或没有权限。可刷新后重试。' : '正在读取…' }}</p><RouterLink :to="taskRoute(id)" class="director-link">查看任务</RouterLink></template>
      </article>
    </div>
    <S2VButton class="mt-5" size="lg" :disabled="disabled || shot.candidateTaskIds.length >= 20" @click="$emit('attach')"><Plus class="h-4 w-4" aria-hidden="true" />关联已有任务</S2VButton>
    <p class="director-hint">每个镜头最多保留 20 个候选。选用结果与任务是否已交付分别记录。</p>
  </section>
</template>

<style scoped>
.director-candidates { min-width: 0; }
.director-preview { margin: 20px 0; }
.director-preview video { display: block; width: 100%; max-height: 420px; aspect-ratio: 16 / 9; object-fit: contain; background: rgb(var(--s2v-media-overlay)); border-radius: var(--s2v-radius-control); }
.director-preview-unavailable, .director-candidates-empty { display: grid; place-items: center; gap: 14px; min-height: 180px; padding: 24px; border: 1px dashed rgb(var(--s2v-line-strong)); border-radius: var(--s2v-radius-control); text-align: center; color: rgb(var(--s2v-muted)); font-size: 13px; line-height: 1.8; }
.director-candidates-empty { margin-top: 22px; }
.director-candidates-empty h3 { color: rgb(var(--s2v-ink)); font-size: 15px; font-weight: 500; }
.director-preview-caption { display: flex; gap: 8px; align-items: center; margin-top: 10px; color: rgb(var(--s2v-muted)); font-size: 12px; }
.director-candidate-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr)); gap: 14px; margin-top: 20px; }
.director-candidate { padding: 18px; border: 1px solid rgb(var(--s2v-line)); border-radius: var(--s2v-radius-control); min-width: 0; }
.director-candidate-heading { display: flex; gap: 12px; justify-content: space-between; align-items: start; }
.director-candidate h3 { font-size: 14px; color: rgb(var(--s2v-ink)); font-weight: 500; overflow-wrap: anywhere; }
.director-candidate-heading span { font-size: 12px; color: rgb(var(--s2v-muted)); flex-shrink: 0; }
.director-candidate p { margin-top: 10px; font-size: 12px; color: rgb(var(--s2v-muted)); line-height: 1.8; overflow-wrap: anywhere; }
.director-candidate-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; justify-content: space-between; margin: 10px 0; }
</style>
