<script setup lang="ts">
import { Filter, Search, Sparkles } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { computed, ref } from 'vue'
import { sceneTemplates } from '@/data/mock'
import SceneCard from '@/features/garden/SceneCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

const router = useRouter()
const app = useAppStore()
const search = ref('')
const category = ref('全部')
const categories = ['全部', '电影感', '自然', '商业', '生活', '叙事', '艺术']
const filtered = computed(() => sceneTemplates.filter((item) => (category.value === '全部' || item.category === category.value) && (!search.value || `${item.title}${item.description}`.includes(search.value))))

function useScene(item: (typeof sceneTemplates)[number]) {
  app.draftPrompt = `${item.title}，${item.description}`
  router.push('/app/create')
  app.notify(`已套用「${item.title}」`)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Garden" title="从灵感开始">
      <template #description>浏览可复用的镜头结构和提示词，把喜欢的场景带进创作。</template>
      <template #actions>
        <S2VBadge tone="primary"><Sparkles class="h-3 w-3" />官方精选</S2VBadge>
      </template>
    </S2VPageHeader>

    <div class="mb-5 flex flex-wrap items-center gap-2">
      <S2VSegmented v-model="category" :options="categories" />
      <S2VInput v-model="search" class="ml-auto max-w-xs flex-1" size="sm" placeholder="搜索场景">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
        <template #suffix><Filter class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <SceneCard v-for="item in filtered" :key="item.id" :item="item" @apply="useScene(item)" />
    </div>
  </section>
</template>
