<script setup lang="ts">
import { AlertTriangle, ArrowLeft, BookmarkPlus, Download, LoaderCircle } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchTask, type ClientTaskDetail } from '@/api/client'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const task = ref<ClientTaskDetail | null>(null)
const loading = ref(false)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function tone(status: string) {
  if (status === '失败') return 'danger' as const
  if (status === '已完成') return 'success' as const
  return 'primary' as const
}

async function load() {
  loading.value = true
  try {
    task.value = await fetchTask(String(route.params.id))
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载任务详情失败'))
  } finally {
    loading.value = false
  }
}

function download() {
  if (!task.value?.downloadUrl) {
    app.notify('成片正在完成对象存储交付，暂不可下载')
    return
  }
  window.open(task.value.downloadUrl, '_blank', 'noopener,noreferrer')
}

async function saveToProject() {
  if (!task.value) return
  const ok = await app.saveWorkToAsset(task.value.id)
  if (ok) task.value.savedToProject = true
}

onMounted(() => { void load() })
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader :title="task?.title || '任务详情'">
      <template #description>查看生成状态、交付状态和可下载成片。</template>
      <template #actions>
        <S2VButton size="sm" @click="router.push('/app/works')"><ArrowLeft class="h-3.5 w-3.5" />返回任务</S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard v-if="loading" variant="padded" class="grid place-items-center text-center">
      <LoaderCircle class="h-5 w-5 animate-spin text-s2v-primary" />
      <p class="mt-2 text-xs text-s2v-muted">正在读取任务详情</p>
    </S2VCard>

    <template v-else-if="task">
      <div class="grid gap-4 lg:grid-cols-3">
        <S2VCard variant="padded" class="lg:col-span-2">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="font-mono text-[11px] text-s2v-muted">{{ task.id }}</p>
              <div class="mt-2 flex items-center gap-2">
                <S2VBadge :tone="tone(task.status)">{{ task.status }}<span v-if="task.status === '生成中'" class="ml-1">{{ task.progress }}%</span></S2VBadge>
                <span class="text-xs text-s2v-muted">{{ task.mode }}</span>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <S2VButton v-if="task.deliveryStatus === 'ready'" size="sm" variant="primary" @click="download"><Download class="h-3.5 w-3.5" />下载成片</S2VButton>
              <S2VButton v-if="task.deliveryStatus === 'ready' && !task.savedToProject" size="sm" @click="saveToProject"><BookmarkPlus class="h-3.5 w-3.5" />保存到作品</S2VButton>
            </div>
          </div>

          <img v-if="task.thumbUrl" :src="task.thumbUrl" :alt="task.title" class="mt-5 aspect-video w-full object-cover" />
          <div v-else class="s2v-inset-dashed mt-5 grid aspect-video place-items-center text-xs text-s2v-muted">成片预览将在持久化交付后可用</div>
        </S2VCard>

        <S2VCard variant="padded">
          <h2 class="s2v-section-title">生成设置</h2>
          <dl class="mt-4 grid gap-3 text-xs">
            <div class="flex items-center justify-between gap-3"><dt class="text-s2v-muted">视频时长</dt><dd class="font-mono">{{ task.duration }}</dd></div>
            <div class="flex items-center justify-between gap-3"><dt class="text-s2v-muted">输出分辨率</dt><dd class="font-mono">{{ task.resolution }}</dd></div>
            <div class="flex items-center justify-between gap-3"><dt class="text-s2v-muted">画幅比例</dt><dd class="font-mono">{{ task.aspectRatio }}</dd></div>
            <div class="flex items-center justify-between gap-3"><dt class="text-s2v-muted">自动生成音频</dt><dd>{{ task.generateAudio ? '需要' : '不需要' }}</dd></div>
            <div class="flex items-center justify-between gap-3"><dt class="text-s2v-muted">已预扣</dt><dd class="font-mono">¥ {{ task.cost.toFixed(2) }}</dd></div>
          </dl>
        </S2VCard>
      </div>

      <S2VCard v-if="task.executionWaiting || task.deliveryStatus === 'pending_storage' || task.failureReason" variant="padded" class="mt-4">
        <div class="flex items-start gap-3 text-xs">
          <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-s2v-warning" />
          <p v-if="task.executionWaiting">任务已创建并完成预扣，正在等待执行配置恢复。配置恢复后将继续按本次任务锁定的模型和参数执行。</p>
          <p v-else-if="task.deliveryStatus === 'pending_storage'">上游已完成，正在将成片持久化到对象存储。交付完成后可下载或保存到作品。</p>
          <p v-else>{{ task.failureReason }}</p>
        </div>
      </S2VCard>
    </template>

    <S2VCard v-else variant="padded" class="grid place-items-center text-center">
      <p class="text-xs text-s2v-muted">未找到该任务。</p>
    </S2VCard>
  </section>
</template>
