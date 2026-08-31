<script setup lang="ts">
import { Check, CircleAlert, MessageSquare, RotateCcw, Search } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const search = ref('')
const refundOpen = ref(false)
const pending = ref<(typeof app.adminTickets)[number] | null>(null)
const openCount = computed(() => app.adminTickets.filter((item) => item.status !== '已解决').length)
const list = computed(() => app.adminTickets.filter((row) => !search.value || `${row.id}${row.user}${row.title}`.includes(search.value)))

function askRefund(item: (typeof app.adminTickets)[number]) {
  pending.value = item
  refundOpen.value = true
}

function confirmRefund() {
  if (pending.value) app.approveTicketRefund(pending.value.id)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Tickets" title="工单中心">
      <template #description>处理退款、生成异常和售前咨询，支持关联任务后再执行操作。</template>
      <template #actions>
        <S2VBadge tone="warning"><CircleAlert class="h-3 w-3" />{{ openCount }} 条待处理</S2VBadge>
      </template>
    </S2VPageHeader>

    <div class="mb-4 max-w-xs">
      <S2VInput v-model="search" size="sm" placeholder="工单号 / 用户 / 标题">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>工单</th>
            <th>用户</th>
            <th>分类</th>
            <th>状态</th>
            <th>关联金额</th>
            <th>更新时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in list" :key="item.id">
            <td>
              <strong class="block text-xs text-s2v-ink">{{ item.title }}</strong>
              <span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ item.id }}</span>
            </td>
            <td>{{ item.user }}</td>
            <td>{{ item.type }}</td>
            <td><S2VBadge :tone="item.status === '已解决' ? 'success' : item.status === '待处理' ? 'warning' : 'primary'">{{ item.status }}</S2VBadge></td>
            <td class="font-mono">{{ item.amount ? `¥${item.amount.toFixed(2)}` : '-' }}</td>
            <td>{{ item.updated }}</td>
            <td>
              <div class="flex items-center gap-1">
                <S2VIconButton label="回复工单" @click="app.notify(`已打开 ${item.id} 对话`)"><MessageSquare class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-if="item.amount && item.status !== '已解决'" label="核准退款" @click="askRefund(item)"><RotateCcw class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-else-if="item.status !== '已解决'" label="标记已解决" @click="app.resolveTicket(item.id)"><Check class="h-3.5 w-3.5" /></S2VIconButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <S2VDialog
      v-model:open="refundOpen"
      tone="danger"
      title="核准退款"
      :description="pending ? `确认将工单 ${pending.id} 关联金额原路退回创作者钱包？` : ''"
      confirm-label="核准退款"
      @confirm="confirmRefund"
    />
  </section>
</template>
