<script setup lang="ts">
import { onBeforeUnmount, shallowRef, watch } from 'vue'
import { listTasks } from '@/api/client'
import { getAuthSession } from '@/shared/lib/auth-session'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
const props = defineProps<{ open: boolean; attachedIds: string[]; busy: boolean }>()
defineEmits<{ 'update:open': [value: boolean]; attach: [id: string] }>()
const tasks = shallowRef<Awaited<ReturnType<typeof listTasks>>['items']>([])
const total = shallowRef(0)
const page = shallowRef(1)
const loading = shallowRef(false)
const error = shallowRef('')
let version = 0
let disposed = false
async function load() {
  const request = ++version
  const session = getAuthSession().id
  loading.value = true; error.value = ''
  try {
    const result = await listTasks({ page: page.value, pageSize: 6 })
    if (disposed || request !== version || session !== getAuthSession().id) return
    tasks.value = result.items; total.value = result.total
  } catch { if (!disposed && request === version) error.value = '任务列表暂时无法读取，请重试。' }
  finally { if (!disposed && request === version) loading.value = false }
}
watch(() => props.open, (open) => { if (open) { page.value = 1; void load() } else version++ })
watch(page, () => { if (props.open) void load() })
onBeforeUnmount(() => { disposed = true; version++ })
</script>

<template>
  <S2VDialog :open="open" title="关联已有任务" description="选择当前账号的任务加入镜头候选，不创建任务、不再次扣费。" hide-actions @update:open="$emit('update:open', $event)">
    <div class="director-task-picker">
      <p v-if="loading" role="status">正在读取任务…</p>
      <div v-else-if="error" role="alert"><p>{{ error }}</p><S2VButton @click="load">重新加载任务</S2VButton></div>
      <p v-else-if="!tasks.length">还没有任务。请先在创作页完成一次创作。</p>
      <ul v-else class="director-picker-list"><li v-for="task in tasks" :key="task.id"><div><strong>{{ task.title }}</strong><p>{{ task.status }} · {{ task.duration }} · {{ task.resolution }}</p></div><S2VButton size="lg" :disabled="busy || attachedIds.includes(task.id)" @click="$emit('attach', task.id)">{{ attachedIds.includes(task.id) ? '已关联' : '加入候选' }}</S2VButton></li></ul>
      <S2VPagination v-if="total > 6" v-model="page" :total="total" :page-size="6" />
      <S2VButton size="lg" :disabled="busy" @click="$emit('update:open', false)">关闭</S2VButton>
    </div>
  </S2VDialog>
</template>

<style scoped>
.director-task-picker { display: grid; gap: 20px; color: rgb(var(--s2v-muted)); font-size: 13px; line-height: 1.8; }
.director-picker-list { display: grid; gap: 16px; }
.director-picker-list li { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; justify-content: space-between; padding-bottom: 16px; border-bottom: 1px solid rgb(var(--s2v-line)); }
.director-picker-list li > div { flex: 1; min-width: 0; }
.director-picker-list strong { display: block; color: rgb(var(--s2v-ink)); font-weight: 500; overflow-wrap: anywhere; }
.director-picker-list p { margin-top: 4px; font-size: 12px; }
</style>
