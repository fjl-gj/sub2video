<script setup lang="ts">
import { FileText, Image } from 'lucide-vue-next'
import type { CreateMode } from '@/shared/lib/create-draft'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import { cn } from '@/shared/lib/cn'

const mode = defineModel<CreateMode>('mode', { default: 'image' })

const cards = [
  {
    id: 'image' as const,
    title: '图生视频',
    description: '上传首尾帧或多张参考图，配合描述生成视频。',
    icon: Image,
    locked: false,
  },
  {
    id: 'text' as const,
    title: '文生视频',
    description: '仅填写画面描述，无需图片素材。',
    icon: FileText,
    locked: true,
  },
]
</script>

<template>
  <div class="grid gap-3 sm:grid-cols-2">
    <button
      v-for="card in cards"
      :key="card.id"
      type="button"
      class="s2v-create-mode-card"
      :class="cn(mode === card.id && 's2v-create-mode-card-active')"
      @click="mode = card.id"
    >
      <div class="flex items-center justify-between gap-2">
        <span class="s2v-icon-well s2v-icon-well-nav shrink-0" :class="mode === card.id && 's2v-icon-well-inverse'">
          <component :is="card.icon" class="h-4 w-4" />
        </span>
        <S2VBadge v-if="mode === card.id" tone="primary">当前</S2VBadge>
        <S2VBadge v-else-if="card.locked" tone="neutral">即将开放</S2VBadge>
      </div>
      <h3 class="mt-2.5 text-sm font-semibold text-s2v-ink">{{ card.title }}</h3>
      <p class="mt-1 text-xs leading-5 text-s2v-muted">{{ card.description }}</p>
    </button>
  </div>
</template>
