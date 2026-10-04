<script setup lang="ts">
import { Film, LoaderCircle } from 'lucide-vue-next'
import type { Task } from '@/entities/types'
import { ref, watch } from 'vue'

const props = defineProps<{
  tasks: Task[]
  activeTaskId: string | null
  loading?: boolean
  error?: string
}>()
defineEmits<{ select: [taskId: string]; viewAll: []; retry: [] }>()
const failedImages = ref<Set<string>>(new Set())
watch(() => props.tasks, () => { failedImages.value = new Set() })
</script>

<template>
  <section class="s2v-studio-recent" aria-label="最近任务">
    <div class="s2v-studio-section-heading"><h2>最近任务</h2><button type="button" class="s2v-studio-text-link" @click="$emit('viewAll')">全部任务 →</button></div>
    <div v-if="loading" class="s2v-studio-recent-empty"><LoaderCircle class="h-4 w-4 animate-spin" />正在加载最近任务</div>
    <div v-else-if="error" class="s2v-studio-recent-empty"><p>{{ error }}</p><button type="button" class="s2v-studio-text-link" @click="$emit('retry')">重新加载</button></div>
    <p v-else-if="!tasks.length" class="s2v-studio-recent-empty">还没有生成任务。提交后，你的创作会出现在这里。</p>
    <div v-else class="s2v-studio-film-grid">
      <button v-for="task in tasks" :key="task.id" type="button" class="s2v-studio-film" :aria-pressed="activeTaskId === task.id" @click="$emit('select', task.id)">
        <div class="s2v-studio-film-thumb">
          <img v-if="task.thumb && !failedImages.has(task.id)" :src="task.thumb" :alt="task.title" @error="failedImages.add(task.id)">
          <Film v-else class="h-6 w-6" />
          <span>{{ task.duration.replace(' 秒', 's') }}</span>
        </div>
        <h3>{{ task.title }}</h3>
        <p>{{ task.status }} · {{ task.resolution }}</p>
      </button>
    </div>
  </section>
</template>
