<script setup lang="ts">
import { Search } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { computed, ref, watch } from 'vue'
import { type SceneAspectRatio, type SceneTemplate, sceneTemplates } from '@/data/mock'
import SceneCard from '@/features/garden/SceneCard.vue'
import { PLAZA_BATCH_SIZE } from '@/shared/lib/paginate'
import { useInfiniteScroll } from '@/shared/lib/use-infinite-scroll'
import { useAppStore } from '@/stores/app'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'

const router = useRouter()
const app = useAppStore()
const search = ref('')
const category = ref('全部')
const orientation = ref('全部')
const visibleCount = ref(PLAZA_BATCH_SIZE)
const sentinel = ref<HTMLElement | null>(null)

const categories = ['全部', '电影感', '自然', '商业', '生活', '叙事', '艺术']
const orientations = ['全部', '横屏', '竖屏', '方形']

const orientationMap: Record<string, SceneAspectRatio | null> = {
  全部: null,
  横屏: '16:9',
  竖屏: '9:16',
  方形: '1:1',
}

const filtered = computed(() => sceneTemplates.filter((item) => {
  const ratio = orientationMap[orientation.value]
  const matchesCategory = category.value === '全部' || item.category === category.value
  const matchesOrientation = !ratio || item.aspectRatio === ratio
  const matchesSearch = !search.value || `${item.title}${item.description}`.includes(search.value)
  return matchesCategory && matchesOrientation && matchesSearch
}))

const visible = computed(() => filtered.value.slice(0, visibleCount.value))
const hasMore = computed(() => visibleCount.value < filtered.value.length)
const useSquareGrid = computed(() => orientation.value === '方形')

function loadMore() {
  if (!hasMore.value) return
  visibleCount.value = Math.min(visibleCount.value + PLAZA_BATCH_SIZE, filtered.value.length)
}

useInfiniteScroll(sentinel, loadMore)

watch([category, orientation, search], () => {
  visibleCount.value = PLAZA_BATCH_SIZE
})

function useScene(item: SceneTemplate) {
  app.draftPrompt = `${item.title}，${item.description}`
  router.push('/app/create')
  app.notify(`已套用「${item.title}」`)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="广场" />

    <div class="mb-4 space-y-3">
      <S2VSegmented v-model="category" :options="categories" />
      <div class="flex flex-wrap items-center gap-2">
        <S2VSegmented v-model="orientation" :options="orientations" />
        <S2VInput v-model="search" class="ml-auto max-w-xs flex-1" size="sm" placeholder="搜索场景">
          <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
        </S2VInput>
      </div>
    </div>

    <div v-if="visible.length" :class="useSquareGrid ? 'grid gap-4 sm:grid-cols-2 lg:grid-cols-3' : 's2v-masonry'">
      <SceneCard v-for="item in visible" :key="item.id" :item="item" @apply="useScene(item)" />
    </div>
    <p v-else class="s2v-inset-dashed">没有匹配的场景，试试调整分类或比例筛选。</p>

    <div v-if="visible.length && hasMore" ref="sentinel" class="py-6 text-center text-xs text-s2v-muted">
      向下滚动加载更多
    </div>
    <p v-else-if="visible.length && filtered.length > PLAZA_BATCH_SIZE" class="py-6 text-center text-xs text-s2v-muted">
      已加载全部 {{ filtered.length }} 个场景
    </p>
  </section>
</template>
