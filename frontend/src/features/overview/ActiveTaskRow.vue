<script setup lang="ts">
import type { Task } from '@/data/mock'
import { Clock3 } from 'lucide-vue-next'
import S2VProgress from '@/shared/ui/S2VProgress.vue'

defineProps<{ task: Task }>()
</script>

<template>
  <div class="s2v-inset flex flex-col gap-3 sm:flex-row sm:items-center">
    <div class="s2v-icon-well s2v-icon-well-inverse"><Clock3 class="h-5 w-5" /></div>
    <div class="min-w-0 flex-1">
      <strong class="block truncate text-xs">{{ task.title }} · {{ task.resolution }} / {{ task.duration }} (含音频)</strong>
      <span class="mt-1 block truncate text-[11px] text-s2v-muted">已预扣 ¥ {{ task.cost.toFixed(2) }} · {{ task.workflow }} · 进度 {{ task.progress }}%</span>
    </div>
    <div class="flex items-center gap-3 sm:w-48">
      <S2VProgress class="flex-1" :value="task.progress" />
      <span class="s2v-mono text-[11px] text-s2v-primary">{{ task.progress }}%</span>
    </div>
  </div>
</template>
