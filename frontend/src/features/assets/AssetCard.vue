<script setup lang="ts">
import { computed } from 'vue'
import { FileImage, FileMusic, FileVideo, Plus } from 'lucide-vue-next'
import { assets } from '@/data/mock'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'

const props = defineProps<{ item: (typeof assets)[number] }>()
defineEmits<{ dismiss: [] }>()

const icon = computed(() => props.item.type === '视频' ? FileVideo : props.item.type === '音频' ? FileMusic : FileImage)
</script>

<template>
  <S2VCard variant="media">
    <div class="grid aspect-[4/3] place-items-center" :style="{ background: `linear-gradient(135deg, #${item.color}, #${item.accent})` }">
      <component :is="icon" class="h-8 w-8 text-white/80" />
    </div>
    <div class="s2v-panel-body-sm">
      <div class="flex items-center justify-between gap-2">
        <strong class="truncate text-xs">{{ item.name }}</strong>
        <S2VBadge>{{ item.type }}</S2VBadge>
      </div>
      <div class="mt-1.5 flex items-center justify-between text-[11px] text-s2v-muted">
        <span>{{ item.size }} · {{ item.folder }}</span>
        <button aria-label="资产更多操作" class="rounded-nav p-1 text-s2v-muted hover:bg-s2v-canvas" type="button" @click="$emit('dismiss')"><Plus class="h-3.5 w-3.5 rotate-45" /></button>
      </div>
    </div>
  </S2VCard>
</template>
