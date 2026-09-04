<script setup lang="ts">
import type { CreateMode, ImageVariant } from '@/shared/lib/create-draft'
import { useAppStore } from '@/stores/app'

defineProps<{
  mode: CreateMode
  imageVariant: ImageVariant
}>()

const emit = defineEmits<{
  'update:mode': [value: CreateMode]
  'update:imageVariant': [value: ImageVariant]
}>()

const app = useAppStore()
</script>

<template>
  <div class="s2v-wb-toolbar">
    <span class="s2v-wb-strip-label">模式</span>
    <div class="s2v-wb-segmented">
      <button
        type="button"
        class="s2v-wb-segment"
        :class="{ 's2v-wb-segment-active': mode === 'text' }"
        @click="emit('update:mode', 'text')"
      >
        文生视频
      </button>
      <button
        type="button"
        class="s2v-wb-segment"
        :class="{ 's2v-wb-segment-active': mode === 'image' && imageVariant === 'keyframes' }"
        @click="emit('update:mode', 'image'); emit('update:imageVariant', 'keyframes')"
      >
        首尾帧生成视频
      </button>
      <button
        type="button"
        class="s2v-wb-segment"
        :class="{ 's2v-wb-segment-active': mode === 'image' && imageVariant === 'multi' }"
        @click="emit('update:mode', 'image'); emit('update:imageVariant', 'multi')"
      >
        多图生成视频
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
