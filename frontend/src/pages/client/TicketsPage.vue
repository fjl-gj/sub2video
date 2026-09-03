<script setup lang="ts">
import { ChevronRight, CircleHelp, MessageSquarePlus, Send, Ticket } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { createTicket, fetchTickets, type ClientTicketItem } from '@/api/client'
import { PAGE_SIZE } from '@/shared/lib/paginate'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'
import S2VTextarea from '@/shared/ui/S2VTextarea.vue'

const app = useAppStore()
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
const categories = ['售后支持', '生成异常', '售前咨询', '充值问题']
const openCount = computed(() => items.value.filter((item) => item.status !== '已解决').length)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const data = await fetchTickets({ page: page.value, pageSize: PAGE_SIZE })
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
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '加载工单失败'))
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (!title.value.trim()) return
  submitting.value = true
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
    app.notify(apiErrorMessage(e, '提交工单失败'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => { void load() })
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="帮助与工单">
      <template #description>遇到生成异常、退款或售前问题，可以关联任务和订单提交给我们。</template>
      <template #actions>
        <S2VBadge v-if="openCount" tone="warning">{{ openCount }} 待处理</S2VBadge>
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
      <div class="s2v-panel-body grid gap-3">
        <S2VSegmented v-model="category" :options="categories" />
        <S2VInput v-model="title" placeholder="工单标题" />
        <S2VInput v-model="taskId" placeholder="关联任务 ID，例如 tsk_20260326_xxxx" />
        <S2VTextarea v-model="detail" placeholder="描述遇到的问题，可包含订单号" />
        <div class="flex justify-end">
          <S2VButton variant="primary" :loading="submitting" :disabled="!title.trim()" @click="submit"><Send class="h-3.5 w-3.5" />提交</S2VButton>
        </div>
      </div>
    </S2VCard>

    <S2VCard>
      <div class="divide-y divide-s2v-line/70 px-4">
        <button v-for="item in items" :key="item.id" class="flex w-full items-center gap-3 py-4 text-left" type="button" @click="app.notify(`${item.id} · ${item.task}`)">
          <span class="s2v-icon-well"><Ticket class="h-4 w-4" /></span>
          <span class="min-w-0 flex-1">
            <strong class="block truncate text-xs">{{ item.title }}</strong>
            <span class="mt-1 block text-[11px] text-s2v-muted">{{ item.id }} · {{ item.category }} · {{ item.updated }}</span>
          </span>
          <S2VBadge :tone="item.status === '已解决' ? 'success' : item.status === '待处理' ? 'warning' : 'primary'">{{ item.status }}</S2VBadge>
          <ChevronRight class="h-4 w-4 text-s2v-muted" />
        </button>
        <p v-if="!loading && !items.length" class="py-4 text-xs text-s2v-muted">暂无工单记录。</p>
        <p v-if="loading" class="py-4 text-xs text-s2v-muted">加载中…</p>
      </div>
      <div v-if="total > PAGE_SIZE" class="border-t border-s2v-line px-4 py-3">
        <S2VPagination v-model="page" :total="total" :page-size="PAGE_SIZE" @update:model-value="load" />
      </div>
    </S2VCard>
  </section>
</template>
