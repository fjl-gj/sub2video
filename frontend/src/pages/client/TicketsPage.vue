<script setup lang="ts">
import { ChevronRight, CircleHelp, MessageSquarePlus, Send, Ticket } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { PAGE_SIZE, slicePage } from '@/shared/lib/paginate'
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
const categories = ['售后支持', '生成异常', '售前咨询', '充值问题']
const paged = computed(() => slicePage(app.tickets, page.value, PAGE_SIZE))

function submit() {
  if (!title.value.trim()) return
  app.createTicket(title.value.trim(), category.value, taskId.value.trim() || '-')
  title.value = ''
  detail.value = ''
  taskId.value = ''
  showForm.value = false
  page.value = 1
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="帮助与工单">
      <template #description>遇到生成异常、退款或售前问题，可以关联任务和订单提交给我们。</template>
      <template #actions>
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
        <S2VInput v-model="taskId" placeholder="关联任务 ID，例如 tsk_88840" />
        <S2VTextarea v-model="detail" placeholder="描述遇到的问题，可包含订单号" />
        <div class="flex justify-end">
          <S2VButton variant="primary" :disabled="!title.trim()" @click="submit"><Send class="h-3.5 w-3.5" />提交</S2VButton>
        </div>
      </div>
    </S2VCard>

    <S2VCard>
      <div class="divide-y divide-s2v-line/70 px-4">
        <button v-for="item in paged" :key="item.id" class="flex w-full items-center gap-3 py-4 text-left" type="button" @click="app.notify(`${item.id} · ${item.task}`)">
          <span class="s2v-icon-well"><Ticket class="h-4 w-4" /></span>
          <span class="min-w-0 flex-1">
            <strong class="block truncate text-xs">{{ item.title }}</strong>
            <span class="mt-1 block text-[11px] text-s2v-muted">{{ item.id }} · {{ item.category }} · {{ item.updated }}</span>
          </span>
          <S2VBadge :tone="item.status === '已解决' ? 'success' : item.status === '待处理' ? 'warning' : 'primary'">{{ item.status }}</S2VBadge>
          <ChevronRight class="h-4 w-4 text-s2v-muted" />
        </button>
      </div>
      <div v-if="app.tickets.length" class="border-t border-s2v-line px-4 py-3">
        <S2VPagination v-model="page" :total="app.tickets.length" :page-size="PAGE_SIZE" />
      </div>
    </S2VCard>
  </section>
</template>
