<script setup lang="ts">
import { Clock, Layers, Ratio } from 'lucide-vue-next'
import type { CreateResolution } from '@/shared/lib/create-cost'

withDefaults(defineProps<{
  ratio: string
  durationSec: string
  resolution: CreateResolution
  durationOptions?: string[]
  resolutionOptions?: CreateResolution[]
}>(), {
  durationOptions: () => ['5', '10', '15'],
  resolutionOptions: () => ['768P'],
})

const emit = defineEmits<{
  'update:ratio': [value: string]
  'update:durationSec': [value: string]
  'update:resolution': [value: CreateResolution]
}>()
</script>

<template>
  <div class="s2v-wb-panel-section">
    <div class="s2v-wb-section-head">
      <span>生成设置</span>
    </div>
    <div class="s2v-wb-settings-row">
      <label class="s2v-wb-setting-pill">
        <Ratio class="h-3.5 w-3.5 text-s2v-muted" />
        <select :value="ratio" @change="emit('update:ratio', ($event.target as HTMLSelectElement).value)">
          <option value="9:16">9:16 (竖屏)</option>
          <option value="16:9">16:9 (横屏)</option>
          <option value="1:1">1:1 (方形)</option>
        </select>
      </label>

      <label class="s2v-wb-setting-pill">
        <Clock class="h-3.5 w-3.5 text-s2v-muted" />
        <select :value="durationSec" @change="emit('update:durationSec', ($event.target as HTMLSelectElement).value)">
          <option v-for="sec in durationOptions" :key="sec" :value="sec">{{ sec }} 秒</option>
        </select>
      </label>

      <label class="s2v-wb-setting-pill">
        <Layers class="h-3.5 w-3.5 text-s2v-muted" />
        <select :value="resolution" @change="emit('update:resolution', ($event.target as HTMLSelectElement).value as CreateResolution)">
          <option v-for="item in resolutionOptions" :key="item" :value="item">{{ item }}</option>
        </select>
      </label>
    </div>
  </div>
</template>
