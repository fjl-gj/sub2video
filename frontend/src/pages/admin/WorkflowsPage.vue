<script setup lang="ts">
import { Edit3, RefreshCw, Workflow } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { fetchAdminWorkflows, updateAdminWorkflow, type AdminWorkflow } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const items = ref<AdminWorkflow[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminWorkflows()
    items.value = data.items
  } catch (e: unknown) {
    app.notify(e instanceof Error ? e.message : '加载工作流失败')
  } finally {
    loading.value = false
  }
}

async function toggle(item: AdminWorkflow) {
  const next = item.statusRaw === 'published' ? 'draft' : 'published'
  try {
    await updateAdminWorkflow(item.id, { status: next })
    app.notify(`${item.name} 已${next === 'published' ? '上架' : '下架'}`)
    await load()
  } catch (e: unknown) {
    app.notify(e instanceof Error ? e.message : '更新失败')
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="工作流">
      <template #description>平台工作流对客户端上架；经模型与供应商路由接到上游能力。</template>
      <template #actions>
        <S2VButton :disabled="loading" @click="load">
          <template #icon><RefreshCw class="h-4 w-4" /></template>
          刷新
        </S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 lg:grid-cols-3">
      <S2VCard v-for="item in items" :key="item.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <span class="s2v-icon-well s2v-icon-well-nav"><Workflow class="h-4 w-4" /></span>
          <S2VBadge :tone="item.status === '已上架' ? 'success' : item.status === '草稿' ? 'warning' : 'neutral'">{{ item.status }}</S2VBadge>
        </div>
        <h2 class="mt-4 text-sm font-bold">{{ item.name }}</h2>
        <p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ item.description }}</p>
        <div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-[11px] text-s2v-muted">
          <div class="flex justify-between"><span>承运 / 模型</span><strong class="text-s2v-ink">{{ item.carrier }} / {{ item.modelCode }}</strong></div>
          <div class="flex justify-between"><span>售价</span><strong class="font-mono text-s2v-ink">{{ item.price }}</strong></div>
          <div class="flex justify-between"><span>编码</span><span class="font-mono">{{ item.code }}</span></div>
        </div>
        <S2VButton class="mt-4" size="sm" :block="true" @click="toggle(item)"><Edit3 class="h-3.5 w-3.5" />{{ item.statusRaw === 'published' ? '下架' : '上架' }}</S2VButton>
      </S2VCard>
    </div>
    <p v-if="!items.length" class="s2v-inset-dashed">暂无平台工作流。</p>
  </section>
</template>
