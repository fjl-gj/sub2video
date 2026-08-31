<script setup lang="ts">
import { Ban, Download, RefreshCw, Search } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

const app = useAppStore()
const filter = ref('全部')
const search = ref('')
const tabs = ['全部', '排队中', '生成中', '已完成', '失败']
const confirmOpen = ref(false)
const pending = ref<(typeof app.adminTasks)[number] | null>(null)
const list = computed(() => app.adminTasks.filter((task) => (filter.value === '全部' || task.status === filter.value) && (!search.value || `${task.id}${task.title}${task.user}`.includes(search.value))))

function askStop(task: (typeof app.adminTasks)[number]) {
  pending.value = task
  confirmOpen.value = true
}

function confirmStop() {
  if (pending.value) app.cancelTask(pending.value.id)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Task Hub" title="任务中心">
      <template #description>查看全站生成任务，处理卡住、失败和退款任务。</template>
      <template #actions>
        <S2VBadge tone="primary">{{ app.adminTasks.length }} 条任务</S2VBadge>
      </template>
    </S2VPageHeader>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <S2VSegmented v-model="filter" :options="tabs" />
      <S2VInput v-model="search" class="ml-auto max-w-xs flex-1" size="sm" placeholder="任务 ID / 用户">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>任务</th>
            <th>用户</th>
            <th>供应商</th>
            <th>状态</th>
            <th>费用</th>
            <th>提交时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in list" :key="task.id">
            <td>
              <strong class="block text-xs text-s2v-ink">{{ task.title }}</strong>
              <span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ task.id }}</span>
            </td>
            <td>{{ task.user }}</td>
            <td>{{ task.provider }}</td>
            <td><S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">{{ task.status }}</S2VBadge></td>
            <td class="font-mono">¥{{ task.cost.toFixed(2) }}</td>
            <td>{{ task.time }}</td>
            <td>
              <div class="flex items-center gap-1">
                <S2VIconButton label="重新调度" @click="task.status === '失败' ? app.retryTask(task.id) : app.notify(`已重新调度 ${task.id}`)"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-if="task.status === '生成中' || task.status === '排队中'" label="终止任务并退款" @click="askStop(task)"><Ban class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton label="导出任务记录" @click="app.notify('已生成任务记录（原型）')"><Download class="h-3.5 w-3.5" /></S2VIconButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <S2VDialog
      v-model:open="confirmOpen"
      tone="danger"
      title="终止任务并退款"
      :description="pending ? `确定终止 ${pending.id} 吗？预扣费用将原路退回用户钱包。` : ''"
      confirm-label="终止并退款"
      @confirm="confirmStop"
    />
  </section>
</template>
