<script setup lang="ts">
import { ArrowUpRight, Search } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { shotGuides } from './catalog'
import GuideImage from './GuideImage.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import './guides.css'
withDefaults(defineProps<{ searchable?: boolean }>(), { searchable: false })
const query = ref('')
const visible = computed(() => shotGuides.filter((guide) => `${guide.title}${guide.modeName}${guide.category}${guide.description}`.includes(query.value.trim())))
</script>
<template>
  <section class="s2v-guides" aria-label="镜头教程">
    <S2VInput v-if="searchable" v-model="query" label="查找教程" placeholder="按题材或创作模式查找" class="mb-6 max-w-md"><template #prefix><Search class="h-4 w-4" /></template></S2VInput>
    <div class="s2v-guide-grid">
      <RouterLink v-for="guide in visible" :key="guide.id" :to="`/guides/${guide.id}`" class="s2v-guide-card">
        <GuideImage :src="guide.poster" :alt="`${guide.title}构图参考`" />
        <div class="s2v-guide-card-copy"><span class="s2v-guide-eyebrow">{{ guide.category }} / {{ guide.modeName }}</span><h3>{{ guide.title }}<ArrowUpRight class="h-4 w-4" /></h3><p>{{ guide.description }}</p><div class="s2v-guide-card-foot"><span>{{ guide.shots.length }} 个镜头 · 可独立跟做</span><span>查看教程 →</span></div></div>
      </RouterLink>
    </div>
    <p v-if="!visible.length" class="s2v-guide-empty" role="status">没有找到匹配的教程，试试其他关键词。</p>
  </section>
</template>
