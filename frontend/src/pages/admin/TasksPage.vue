<script setup lang="ts">
import { Download, RefreshCw, Search } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { fetchAdminTasks, type AdminTaskItem } from '@/api/admin'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

const app = useAppStore()
const filter = ref('全部')
const search = ref('')
const tabs = ['全部', '排队中', '生成中', '已完成', '失败']
const items = ref<AdminTaskItem[]>([])
const total = ref(0)
const loading = ref(false)

const list = computed(() => items.value.filter((task) => !search.value || `${task.id}${task.title}${task.user}`.includes(search.value)))

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminTasks({
      status: filter.value === '全部' ? undefined : filter.value,
      page: 1,
      pageSize: 50,
    })
    items.value = data.items
    total.value = data.total
  } catch (e: unknown) {
    app.notify(e instanceof Error ? e.message : '加载任务失败')
  } finally {
    loading.value = false
  }
}

async function copyId(id: string) {
  try {
    await navigator.clipboard.writeText(id)
    app.notify('已复制任务号')
  } catch {
    app.notify(id)
  }
}

watch(filter, () => { void load() })
onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="任务中心">
      <template #description>平台任务号为主；上游任务号仅作排障参考。</template>
      <template #actions>
        <S2VBadge tone="primary">{{ total }} 条任务</S2VBadge>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <S2VSegmented v-model="filter" :options="tabs" />
      <S2VInput v-model="search" class="ml-auto max-w-xs flex-1" size="sm" placeholder="任务编号 / 用户">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap hidden md:block">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>任务编号</th>
            <th>用户</th>
            <th>供应商</th>
            <th>上游</th>
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
              <span class="mt-0.5 block text-[10px] text-s2v-muted">{{ task.workflow }}</span>
            </td>
            <td>
              <span class="block text-xs">{{ task.user }}</span>
              <span class="mt-0.5 block text-[10px] text-s2v-muted">{{ task.userEmail }}</span>
            </td>
            <td class="font-mono text-[11px]">{{ task.provider }}</td>
            <td class="font-mono text-[10px] text-s2v-muted">{{ task.upstreamTaskId || '-' }}</td>
            <td><S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">{{ task.status }}</S2VBadge></td>
            <td class="font-mono">¥{{ task.cost.toFixed(2) }}</td>
            <td>{{ task.time }}</td>
            <td>
              <S2VIconButton label="复制任务号" @click="copyId(task.id)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
            </td>
          </tr>
          <tr v-if="!list.length">
            <td colspan="8" class="text-center text-s2v-muted">暂无任务。</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid gap-3 md:hidden">
      <MobileRecordCard v-for="task in list" :key="task.id" :title="task.title" :meta="task.id">
        <template #badge>
          <S2VBadge :tone="task.status === '失败' ? 'danger' : task.status === '已完成' ? 'success' : 'primary'">{{ task.status }}</S2VBadge>
        </template>
        <template #body>{{ task.user }} · {{ task.provider }} · ¥{{ task.cost.toFixed(2) }} · {{ task.time }}</template>
        <template #actions>
          <S2VIconButton label="复制任务号" @click="copyId(task.id)"><Download class="h-3.5 w-3.5" /></S2VIconButton>
        </template>
      </MobileRecordCard>
      <p v-if="!list.length" class="text-xs text-s2v-muted">暂无任务。</p>
    </div>
  </section>
</template>
