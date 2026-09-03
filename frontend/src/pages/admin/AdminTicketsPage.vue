<script setup lang="ts">
import { Check, CircleAlert, RefreshCw, Search } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { fetchAdminTickets, updateAdminTicket, type AdminTicketItem } from '@/api/admin'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const search = ref('')
const items = ref<AdminTicketItem[]>([])
const total = ref(0)
const loading = ref(false)
const openCount = computed(() => items.value.filter((item) => item.status !== '已解决').length)
const list = computed(() => items.value.filter((row) => !search.value || `${row.id}${row.user}${row.title}`.includes(search.value)))

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminTickets({ page: 1, pageSize: 50 })
    items.value = data.items
    total.value = data.total
    app.adminTickets = data.items.map((t) => ({
      id: t.id,
      title: t.title,
      user: t.user,
      type: t.category,
      status: t.status as '待处理' | '处理中' | '已解决',
      updated: t.updated,
    }))
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '加载工单失败'))
  } finally {
    loading.value = false
  }
}

async function resolve(id: string) {
  try {
    await updateAdminTicket(id, { status: '已解决' })
    app.notify('工单已标记解决')
    await load()
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '更新失败'))
  }
}

async function markProcessing(id: string) {
  try {
    await updateAdminTicket(id, { status: '处理中' })
    app.notify('已标记处理中')
    await load()
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '更新失败'))
  }
}

onMounted(() => { void load() })
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="工单中心">
      <template #description>处理生成异常与售前咨询；退款请在任务中心终止对应任务。</template>
      <template #actions>
        <S2VBadge tone="warning"><CircleAlert class="h-3 w-3" />{{ openCount }} 条待处理</S2VBadge>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-4 max-w-xs">
      <S2VInput v-model="search" size="sm" placeholder="工单号 / 用户 / 标题">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap hidden md:block">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>工单</th>
            <th>用户</th>
            <th>分类</th>
            <th>关联任务</th>
            <th>状态</th>
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
            <td>
              <span class="block text-xs">{{ item.user }}</span>
              <span class="mt-0.5 block text-[10px] text-s2v-muted">{{ item.userEmail }}</span>
            </td>
            <td>{{ item.category }}</td>
            <td class="font-mono text-[11px]">{{ item.task }}</td>
            <td><S2VBadge :tone="item.status === '已解决' ? 'success' : item.status === '待处理' ? 'warning' : 'primary'">{{ item.status }}</S2VBadge></td>
            <td>{{ item.updated }}</td>
            <td>
              <div class="flex items-center gap-1">
                <S2VIconButton v-if="item.status === '待处理'" label="开始处理" @click="markProcessing(item.id)"><RefreshCw class="h-3.5 w-3.5" /></S2VIconButton>
                <S2VIconButton v-if="item.status !== '已解决'" label="标记已解决" @click="resolve(item.id)"><Check class="h-3.5 w-3.5" /></S2VIconButton>
              </div>
            </td>
          </tr>
          <tr v-if="!list.length">
            <td colspan="7" class="text-center text-s2v-muted">{{ loading ? '加载中…' : '暂无工单。' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid gap-3 md:hidden">
      <MobileRecordCard v-for="item in list" :key="item.id" :title="item.title" :meta="item.id">
        <template #badge>
          <S2VBadge :tone="item.status === '已解决' ? 'success' : item.status === '待处理' ? 'warning' : 'primary'">{{ item.status }}</S2VBadge>
        </template>
        <template #body>{{ item.user }} · {{ item.category }} · {{ item.task }}</template>
        <template #actions>
          <S2VIconButton v-if="item.status !== '已解决'" label="标记已解决" @click="resolve(item.id)"><Check class="h-3.5 w-3.5" /></S2VIconButton>
        </template>
      </MobileRecordCard>
      <p v-if="!list.length" class="text-xs text-s2v-muted">{{ loading ? '加载中…' : `共 ${total} 条工单。` }}</p>
    </div>
  </section>
</template>
