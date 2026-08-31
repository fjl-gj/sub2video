<script setup lang="ts">
import { ArrowRight } from 'lucide-vue-next'
import { sceneTemplates } from '@/data/mock'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'

defineProps<{ item: (typeof sceneTemplates)[number] }>()
defineEmits<{ apply: [] }>()
</script>

<template>
  <S2VCard variant="media">
    <div class="grid aspect-video place-items-center text-xs font-mono text-white" :style="{ background: `linear-gradient(135deg, #${item.color}, #${item.accent})` }">
      <span class="s2v-media-tag text-xs font-mono text-white">{{ item.category }} · H3</span>
    </div>
    <div class="s2v-panel-body">
      <h2 class="text-sm font-bold">{{ item.title }}</h2>
      <p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ item.description }}</p>
      <div class="mt-3 flex items-center justify-between">
        <span class="text-[11px] text-s2v-muted">768P · 15 秒</span>
        <S2VButton size="sm" @click="$emit('apply')">套用 <ArrowRight class="h-3.5 w-3.5" /></S2VButton>
      </div>
    </div>
  </S2VCard>
</template>
