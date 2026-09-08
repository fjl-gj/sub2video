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
  <div class="s2v-wb-toolbar">
    <span class="s2v-wb-strip-label">模式</span>
    <div class="s2v-wb-segmented">
      <button
        v-for="mode in modes"
        :key="mode.inputType"
        type="button"
        class="s2v-wb-segment"
        :class="{ 's2v-wb-segment-active': workflowInputType === mode.inputType }"
        @click="emit('update:workflowInputType', mode.inputType)"
      >
        {{ mode.name }}
      </button>
    </div>

    <select
      class="s2v-wb-project-select"
      :value="app.currentProjectId"
      @change="app.selectProject(($event.target as HTMLSelectElement).value)"
    >
      <option v-for="project in app.activeProjects" :key="project.id" :value="project.id">
        {{ project.name }}{{ project.id === app.currentProjectId ? '（当前作品）' : '' }}
      </option>
    </select>
  </div>
</template>
