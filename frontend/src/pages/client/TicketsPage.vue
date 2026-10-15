<script setup lang="ts">
import { ChevronRight, CircleHelp, MessageSquarePlus, Send, Ticket } from 'lucide-vue-next'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { createTicket, fetchTickets, type ClientTicketItem } from '@/api/client'
import { PAGE_SIZE } from '@/shared/lib/paginate'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'
import S2VTextarea from '@/shared/ui/S2VTextarea.vue'

const app = useAppStore()
const route = useRoute()
const showForm = ref(false)
const title = ref('')
const detail = ref('')
const taskId = ref('')
const category = ref('售后支持')
const page = ref(1)
const total = ref(0)
const items = ref<ClientTicketItem[]>([])
const loading = ref(false)
const submitting = ref(false)
const loadError = ref('')
const submitError = ref('')
const selectedTicket = ref<ClientTicketItem | null>(null)
const detailOpen = computed({ get: () => !!selectedTicket.value, set: (open: boolean) => { if (!open) selectedTicket.value = null } })
let loadVersion = 0
let disposed = false
const categories = ['售后支持', '生成异常', '售前咨询', '充值问题']
const openCount = computed(() => items.value.filter((item) => item.status !== '已解决').length)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  const request = ++loadVersion
  loading.value = true
  try {
    const data = await fetchTickets({ page: page.value, pageSize: PAGE_SIZE })
    if (disposed || request !== loadVersion) return
    loadError.value = ''
    items.value = data.items
    total.value = data.total
    app.tickets = data.items.map((t) => ({
      id: t.id,
      title: t.title,
      category: t.category,
      status: t.status as '待处理' | '处理中' | '已解决',
      updated: t.updated,
      task: t.task,
    }))
  } catch {
    if (!disposed && request === loadVersion) loadError.value = '工单列表暂时无法读取，请重新加载。'
  } finally {
    if (!disposed && request === loadVersion) loading.value = false
  }
}

async function submit() {
  if (!title.value.trim() || submitting.value) return
  submitting.value = true
  submitError.value = ''
  try {
    await createTicket({
      title: title.value.trim(),
      category: category.value,
      taskId: taskId.value.trim() || undefined,
      detail: detail.value.trim(),
    })
    title.value = ''
    detail.value = ''
    taskId.value = ''
    showForm.value = false
    page.value = 1
    app.notify('工单已提交')
    await load()
  } catch (e: unknown) {
    submitError.value = apiErrorMessage(e, '提交工单失败，请稍后重试。')
  } finally {
    submitting.value = false
  }
}

watch(() => route.query.taskId, (value) => {
  if (typeof value !== 'string' || !value.trim() || value.length > 200) return
  if (!taskId.value.trim()) taskId.value = value.trim()
  if (!title.value.trim() && !detail.value.trim()) category.value = '生成异常'
  showForm.value = true
}, { immediate: true })
watch(page, () => { void load() }, { immediate: true })
onBeforeUnmount(() => { disposed = true; loadVersion++ })
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="帮助与工单">
      <template #description>生成异常、退款、充值与售前请求。</template>
      <template #actions>
        <S2VBadge v-if="openCount" tone="warning">本页 {{ openCount }} 未解决</S2VBadge>
        <S2VButton variant="primary" @click="showForm = !showForm">
          <template #icon><MessageSquarePlus class="h-4 w-4" /></template>
          新建工单
        </S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard v-if="showForm" class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">提交工单</h2>
          <p class="mt-1 text-xs text-s2v-muted">请尽量提供任务编号，方便快速定位。</p>
        </div>
        <CircleHelp class="h-4 w-4 text-s2v-primary" />
      </div>
      <form class="s2v-panel-body grid gap-3" @submit.prevent="submit">
        <fieldset :disabled="submitting" class="grid gap-3">
        <S2VSegmented v-model="category" :options="categories" />
        <S2VInput v-model="title" label="工单标题" placeholder="简要说明遇到的问题" :disabled="submitting" />
        <S2VInput v-model="taskId" label="关联任务编号" placeholder="选填，可从任务详情带入" :disabled="submitting" />
        <S2VTextarea v-model="detail" label="问题描述" placeholder="描述遇到的问题，可包含订单号" :disabled="submitting" />
        <p v-if="submitError" role="alert">{{ submitError }}</p>
        <div class="flex justify-end">
          <S2VButton type="submit" variant="primary" :loading="submitting" :disabled="!title.trim() || submitting"><Send class="h-3.5 w-3.5" />提交</S2VButton>
        </div>
        </fieldset>
      </form>
    </S2VCard>

    <S2VCard>
      <div class="divide-y divide-s2v-line/70 px-4">
        <div v-if="loadError" class="py-4" role="alert"><p>{{ loadError }}{{ items.length ? '当前显示上次读取的记录。' : '' }}</p><S2VButton class="mt-3" :loading="loading" @click="load">重新加载工单</S2VButton></div>
        <button v-for="item in items" :key="item.id" class="flex w-full items-center gap-3 py-4 text-left" type="button" @click="selectedTicket = item">
          <span class="s2v-icon-well"><Ticket class="h-4 w-4" /></span>
          <span class="min-w-0 flex-1">
            <strong class="block truncate text-xs">{{ item.title }}</strong>
            <span class="mt-1 block text-[11px] text-s2v-muted">{{ item.id }} · {{ item.category }} · {{ item.updated }}</span>
          </span>
          <S2VBadge :tone="item.status === '已解决' ? 'success' : item.status === '待处理' ? 'warning' : 'primary'">{{ item.status }}</S2VBadge>
          <ChevronRight class="h-4 w-4 text-s2v-muted" />
        </button>
        <p v-if="!loading && !loadError && !items.length" class="py-4 text-xs text-s2v-muted">暂无工单记录。</p>
        <p v-if="loading" class="py-4 text-xs text-s2v-muted">加载中…</p>
      </div>
      <div v-if="total > PAGE_SIZE" class="border-t border-s2v-line px-4 py-3">
        <S2VPagination v-model="page" :total="total" :page-size="PAGE_SIZE" />
      </div>
    </S2VCard>
    <S2VDialog v-model:open="detailOpen" :title="selectedTicket?.title || '工单详情'" description="查看已提交的问题和当前处理状态。" hide-actions>
      <div v-if="selectedTicket" class="grid gap-4">
        <dl class="grid gap-3">
          <div><dt>工单编号</dt><dd>{{ selectedTicket.id }}</dd></div>
          <div><dt>处理状态</dt><dd>{{ selectedTicket.status }}</dd></div>
          <div><dt>问题分类</dt><dd>{{ selectedTicket.category }}</dd></div>
          <div><dt>更新时间</dt><dd>{{ selectedTicket.updated }}</dd></div>
          <div v-if="selectedTicket.task && selectedTicket.task !== '-'"><dt>关联任务</dt><dd><RouterLink :to="`/app/works/${encodeURIComponent(selectedTicket.task)}`">{{ selectedTicket.task }}</RouterLink></dd></div>
          <div><dt>问题描述</dt><dd class="whitespace-pre-wrap break-words">{{ selectedTicket.detail || '未填写问题描述。' }}</dd></div>
        </dl>
        <div class="flex justify-end"><S2VButton @click="detailOpen = false">关闭</S2VButton></div>
      </div>
    </S2VDialog>
  </section>
</template>
