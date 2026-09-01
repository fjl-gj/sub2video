<script setup lang="ts">
import { Ban, BookmarkPlus, Download, ExternalLink, MoreHorizontal, Plus, RefreshCw } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { PAGE_SIZE, slicePage } from '@/shared/lib/paginate'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

const router = useRouter()
const app = useAppStore()
const filter = ref('全部')
const page = ref(1)
const tabs = ['全部', '排队中', '生成中', '已完成', '失败']
const refundOpen = ref(false)
const refundTask = ref<(typeof app.tasks)[number] | null>(null)
const list = computed(() => filter.value === '全部' ? app.tasks : app.tasks.filter((task) => task.status === filter.value))
const paged = computed(() => slicePage(list.value, page.value, PAGE_SIZE))
const counts = computed(() => Object.fromEntries(tabs.map((tab) => [tab, tab === '全部' ? app.tasks.length : app.tasks.filter((task) => task.status === tab).length])))

watch(filter, () => { page.value = 1 })

function retry(id: string) {
  app.retryTask(id)
}

function askRefund(task: (typeof app.tasks)[number]) {
  refundTask.value = task
  refundOpen.value = true
}

function confirmRefund() {
  if (refundTask.value) app.cancelTask(refundTask.value.id)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="任务">
      <template #description>所有生成中的任务和已完成结果都在这里，失败任务可以重试或申请退款。完成的结果可保存到当前作品。</template>
      <template #actions>
        <S2VButton variant="primary" @click="router.push('/app/create')">
          <template #icon><Plus class="h-4 w-4" /></template>
          新建创作
        </S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-4">
      <S2VSegmented v-model="filter" :options="tabs" :counts="counts" />
    </div>

    <div class="s2v-table-wrap hidden md:block">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>任务</th>
            <th>工作流</th>
            <th>状态</th>
            <th>参数</th>
            <th>费用</th>
            <th>时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in paged" :key="task.id">
            <td>
              <div class="flex items-center gap-3">
                <img :src="task.thumb" :alt="task.title" class="s2v-thumb h-9 w-14" />
                <div>
                  <strong class="block text-xs text-s2v-ink">{{ task.title }}</strong>
                  <span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ task.id }}</span>
                </div>
              </div>
            </td>
            <td>{{ task.workflow }}</td>
            <td>
              <S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">
                {{ task.status }}<span v-if="task.status === '生成中'" class="ml-1">{{ task.progress }}%</span>
              </S2VBadge>
            </td>
            <td>{{ task.resolution }} · {{ task.duration }}</td>
            <td class="font-mono">¥ {{ task.cost.toFixed(2) }}</td>
            <td>{{ task.time }}</td>
            <td>
              <div class="flex items-center gap-1">
                <S2VIconButton v-if="task.status === '失败'" label="重新生成" @click="retry(task.id)"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-else-if="task.status === '生成中' || task.status === '排队中'" label="终止任务并退款" @click="askRefund(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
                <template v-else>
                  <S2VIconButton label="下载结果" @click="app.notify(`已模拟下载 ${task.title}.mp4`)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
                  <S2VIconButton :label="app.isWorkSaved(task.id) ? '已保存到作品' : '保存到当前作品'" @click="app.saveWorkToAsset(task.id)"><BookmarkPlus class="h-3.5 w-3.5" /></S2VIconButton>
                </template>
                <S2VIconButton label="查看详情" @click="app.notify(`${task.id} · ${task.workflow} · ${task.resolution}`)"><ExternalLink class="h-3.5 w-3.5" /></S2VIconButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid gap-3 md:hidden">
      <MobileRecordCard v-for="task in paged" :key="task.id" :title="task.title" :meta="task.id">
        <template #badge>
          <S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">
            {{ task.status }}<span v-if="task.status === '生成中'" class="ml-1">{{ task.progress }}%</span>
          </S2VBadge>
        </template>
        <template #body>{{ task.workflow }} · {{ task.resolution }} · {{ task.duration }} · ¥ {{ task.cost.toFixed(2) }}</template>
        <template #actions>
          <S2VIconButton v-if="task.status === '失败'" label="重新生成" @click="retry(task.id)"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
          <S2VIconButton v-else-if="task.status === '生成中' || task.status === '排队中'" label="终止任务并退款" @click="askRefund(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
          <template v-else>
            <S2VIconButton label="下载结果" @click="app.notify(`已模拟下载 ${task.title}.mp4`)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
            <S2VIconButton :label="app.isWorkSaved(task.id) ? '已保存到作品' : '保存到当前作品'" @click="app.saveWorkToAsset(task.id)"><BookmarkPlus class="h-3.5 w-3.5" /></S2VIconButton>
          </template>
          <S2VIconButton label="查看详情" @click="app.notify(`${task.id} · ${task.workflow} · ${task.resolution}`)"><ExternalLink class="h-3.5 w-3.5" /></S2VIconButton>
        </template>
      </MobileRecordCard>
    </div>

    <S2VPagination v-if="list.length" v-model="page" class="mt-4" :total="list.length" :page-size="PAGE_SIZE" />

    <S2VCard v-if="!list.length" variant="padded" class="mt-4 grid place-items-center text-center">
      <MoreHorizontal class="h-6 w-6 text-s2v-muted" />
      <p class="mt-2 text-xs text-s2v-muted">暂无符合条件的任务</p>
    </S2VCard>

    <S2VDialog
      v-model:open="refundOpen"
      tone="danger"
      title="终止渲染任务确认"
      :description="refundTask ? `确定要终止任务 ${refundTask.id} 吗？已预扣金额将即时全额退回钱包。` : ''"
      confirm-label="终止并退款"
      @confirm="confirmRefund"
    />
  </section>
</template>
