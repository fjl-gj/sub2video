<script setup lang="ts">
import { FileText, Images, PanelsTopLeft } from 'lucide-vue-next'
import type { WorkflowInputType } from '@/shared/lib/create-draft'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import { cn } from '@/shared/lib/cn'

const workflowInputType = defineModel<WorkflowInputType>('workflowInputType', { default: 'first_last' })

const cards = [
  { id: 'text' as const, title: '文生视频', description: '仅填写画面描述，生成完整视频。', icon: FileText },
  { id: 'multi_image' as const, title: '多图生成视频', description: '上传多张参考图，保持主体与视觉元素的一致性。', icon: Images },
  { id: 'first_last' as const, title: '首尾帧生成视频', description: '上传首帧与尾帧，控制视频的开始与结束画面。', icon: PanelsTopLeft },
]
</script>

<template>
  <div class="grid gap-3 sm:grid-cols-3">
    <button
      v-for="card in cards"
      :key="card.id"
      type="button"
      class="s2v-create-mode-card"
      :class="cn(workflowInputType === card.id && 's2v-create-mode-card-active')"
      @click="workflowInputType = card.id"
    >
      <div class="flex items-center justify-between gap-2">
        <span class="s2v-icon-well s2v-icon-well-nav shrink-0" :class="workflowInputType === card.id && 's2v-icon-well-inverse'">
          <component :is="card.icon" class="h-4 w-4" />
        </span>
        <S2VBadge v-if="workflowInputType === card.id" tone="primary">当前</S2VBadge>
      </div>
      <h3 class="mt-2.5 text-sm font-semibold text-s2v-ink">{{ card.title }}</h3>
      <p class="mt-1 text-xs leading-5 text-s2v-muted">{{ card.description }}</p>
    </button>
  </div>
</template>
