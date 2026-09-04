<script setup lang="ts">
import { computed } from 'vue'
import { FileImage, FileMusic, FileVideo, Plus } from 'lucide-vue-next'
import type { FsItem } from '@/data/projects'
import { assetPreviewUrl } from '@/shared/lib/asset-preview'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'

const props = defineProps<{ item: FsItem }>()
defineEmits<{ dismiss: [] }>()

const icon = computed(() => props.item.type === 'video' ? FileVideo : props.item.type === 'audio' ? FileMusic : FileImage)
const label = computed(() => props.item.type === 'video' ? '视频' : props.item.type === 'audio' ? '音频' : '图片')
const preview = computed(() => props.item.type === 'image' ? assetPreviewUrl(props.item.id, props.item.url || props.item.remoteUrl) : '')
</script>

<template>
  <S2VCard variant="media">
    <div class="grid aspect-[4/3] place-items-center overflow-hidden bg-s2v-canvas">
      <img v-if="preview" :src="preview" :alt="item.name" class="h-full w-full object-cover" />
      <component :is="icon" v-else class="h-8 w-8 text-s2v-primary" />
    </div>
    <div class="s2v-panel-body-sm">
      <div class="flex items-center justify-between gap-2">
        <strong class="truncate text-xs">{{ item.name }}</strong>
        <S2VBadge>{{ label }}</S2VBadge>
      </div>
      <div class="mt-1.5 flex items-center justify-between text-[11px] text-s2v-muted">
        <span>{{ item.size }}{{ item.meta ? ` · ${item.meta}` : '' }}</span>
        <button aria-label="移除文件" class="rounded-nav p-1 text-s2v-muted hover:bg-s2v-canvas" type="button" @click="$emit('dismiss')"><Plus class="h-3.5 w-3.5 rotate-45" /></button>
      </div>
    </div>
  </S2VCard>
</template>
