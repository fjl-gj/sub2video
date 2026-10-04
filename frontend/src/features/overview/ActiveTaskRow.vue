<script setup lang="ts">
import type { Task } from '@/entities/types'
import S2VProgress from '@/shared/ui/S2VProgress.vue'

defineProps<{ task: Task }>()
</script>

<template>
  <div class="s2v-studio-active-task">
    <strong>{{ task.title }}</strong>
    <span>{{ task.workflow }} · {{ task.resolution }} · {{ task.duration }}</span>
    <div class="s2v-studio-task-progress"><span>{{ task.status }}</span><span class="s2v-mono">{{ task.progress }}%</span></div>
    <S2VProgress :value="task.progress" />
    <span>已扣费 ¥ {{ task.cost.toFixed(2) }}</span>
  </div>
</template>
