<script setup lang="ts">
import type { Task } from '@/data/mock'

defineProps<{
  tasks: Task[]
  activeTaskId: string | null
}>()

const emit = defineEmits<{
  select: [taskId: string]
  viewAll: []
}>()

function taskSub(task: Task) {
  return `${task.status} · ${task.duration.replace(' ', '')}`
}

function taskDur(task: Task) {
  return task.duration.replace(' 秒', 's').replace('秒', 's')
}
</script>

<template>
  <div class="s2v-wb-filmstrip">
    <div class="s2v-wb-filmstrip-head">
      <h3>最近任务视频</h3>
      <button type="button" class="s2v-wb-filmstrip-link" @click="emit('viewAll')">全部任务 →</button>
    </div>
    <div class="s2v-wb-film-row">
      <button
        v-for="task in tasks"
        :key="task.id"
        type="button"
        class="s2v-wb-film-card"
        :class="{ 's2v-wb-film-card-active': activeTaskId === task.id }"
        @click="emit('select', task.id)"
      >
        <div class="s2v-wb-film-thumb">
          <img :src="task.thumb" :alt="task.title">
          <span class="s2v-wb-film-dur s2v-mono">{{ taskDur(task) }}</span>
        </div>
        <div class="s2v-wb-film-meta">
          <div class="s2v-wb-film-title">{{ task.title }}</div>
          <div class="s2v-wb-film-sub">{{ taskSub(task) }}</div>
        </div>
      </button>
    </div>
  </div>
</template>
