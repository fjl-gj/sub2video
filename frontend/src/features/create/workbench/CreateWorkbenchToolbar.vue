<script setup lang="ts">
import type { WorkflowInputType } from '@/shared/lib/create-draft'
import { useAppStore } from '@/stores/app'

withDefaults(defineProps<{
  workflowInputType: WorkflowInputType
  modes?: Array<{ inputType: WorkflowInputType; name: string }>
}>(), {
  modes: () => [],
})

const emit = defineEmits<{
  'update:workflowInputType': [value: WorkflowInputType]
}>()

const app = useAppStore()
</script>

<template>
  <div class="s2v-studio-toolbar">
    <div class="s2v-studio-modes" aria-label="创作模式">
      <button
        v-for="mode in modes"
        :key="mode.inputType"
        type="button"
        class="s2v-studio-mode"
        :aria-pressed="workflowInputType === mode.inputType"
        @click="emit('update:workflowInputType', mode.inputType)"
      >
        {{ mode.name }}
      </button>
    </div>

    <label v-if="app.activeProjects.length" class="s2v-studio-project">
      <span>当前作品</span>
      <select
        aria-label="当前作品"
        :value="app.currentProjectId"
        @change="app.selectProject(($event.target as HTMLSelectElement).value)"
      >
        <option v-for="project in app.activeProjects" :key="project.id" :value="project.id">
          {{ project.name }}{{ project.id === app.currentProjectId ? '（当前作品）' : '' }}
        </option>
      </select>
    </label>
  </div>
</template>
