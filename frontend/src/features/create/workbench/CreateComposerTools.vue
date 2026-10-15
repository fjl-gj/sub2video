<script setup lang="ts">
import { AtSign, Plus, SlidersHorizontal } from 'lucide-vue-next'

defineProps<{
  prefOpen?: boolean
}>()

defineEmits<{
  upload: []
  mention: []
  'toggle-pref': []
}>()
</script>

<template>
  <div class="s2v-studio-tools" role="toolbar" aria-label="创作工具">
    <button type="button" class="s2v-studio-tool-btn" aria-label="上传素材" title="上传素材" @click="$emit('upload')">
      <Plus class="h-4 w-4" aria-hidden="true" />
    </button>
    <button type="button" class="s2v-studio-tool-btn" aria-label="引用素材" title="@ 引用素材" @click="$emit('mention')">
      <AtSign class="h-4 w-4" aria-hidden="true" />
    </button>
    <button
      type="button"
      class="s2v-studio-tool-chip"
      :aria-pressed="prefOpen"
      aria-controls="studio-pref-panel"
      :aria-expanded="prefOpen"
      @click="$emit('toggle-pref')"
    >
      <SlidersHorizontal class="h-3.5 w-3.5" aria-hidden="true" />
      创作偏好
    </button>
  </div>
</template>
