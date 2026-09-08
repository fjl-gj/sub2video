<script setup lang="ts">
import { Clock, Layers, Ratio, Volume2 } from 'lucide-vue-next'

withDefaults(defineProps<{
  ratio: string
  durationSec: string
  resolution: string
  durationOptions?: string[]
  resolutionOptions?: string[]
  aspectRatioOptions?: string[]
  carriers?: Array<{ id: string; name: string }>
  models?: Array<{ id: string; carrierId: string; name: string }>
  carrierId?: string
  modelId?: string
  supportsGeneratedAudio?: boolean
  generateAudio?: boolean
}>(), {
  durationOptions: () => ['5', '10', '15'],
  resolutionOptions: () => ['768P'],
  aspectRatioOptions: () => ['16:9', '9:16', '1:1'],
  carriers: () => [],
  models: () => [],
  carrierId: '',
  modelId: '',
  supportsGeneratedAudio: false,
  generateAudio: false,
})

const emit = defineEmits<{
  'update:ratio': [value: string]
  'update:durationSec': [value: string]
  'update:resolution': [value: string]
  'update:carrierId': [value: string]
  'update:modelId': [value: string]
  'update:generateAudio': [value: boolean]
}>()
</script>

<template>
  <div class="s2v-wb-panel-section">
    <div class="s2v-wb-section-head">
      <span>生成设置</span>
    </div>
    <div class="s2v-wb-settings-row">
      <label class="s2v-wb-setting-pill">
        <Layers class="h-3.5 w-3.5 text-s2v-muted" />
        <select aria-label="承运商" :value="carrierId" @change="emit('update:carrierId', ($event.target as HTMLSelectElement).value)">
          <option value="">智能推荐</option>
          <option v-for="carrier in carriers" :key="carrier.id" :value="carrier.id">{{ carrier.name }}</option>
        </select>
      </label>

      <label class="s2v-wb-setting-pill">
        <Layers class="h-3.5 w-3.5 text-s2v-muted" />
        <select aria-label="模型" :value="modelId" @change="emit('update:modelId', ($event.target as HTMLSelectElement).value)">
          <option value="">智能推荐</option>
          <option v-for="model in models" :key="model.id" :value="model.id">{{ model.name }}</option>
        </select>
      </label>

      <label class="s2v-wb-setting-pill">
        <Ratio class="h-3.5 w-3.5 text-s2v-muted" />
        <select aria-label="画幅比例" :value="ratio" @change="emit('update:ratio', ($event.target as HTMLSelectElement).value)">
          <option v-for="item in aspectRatioOptions" :key="item" :value="item">{{ item }}</option>
        </select>
      </label>

      <label class="s2v-wb-setting-pill">
        <Clock class="h-3.5 w-3.5 text-s2v-muted" />
        <select aria-label="视频时长" :value="durationSec" @change="emit('update:durationSec', ($event.target as HTMLSelectElement).value)">
          <option v-for="sec in durationOptions" :key="sec" :value="sec">{{ sec }} 秒</option>
        </select>
      </label>

      <label class="s2v-wb-setting-pill">
        <Layers class="h-3.5 w-3.5 text-s2v-muted" />
        <select aria-label="输出分辨率" :value="resolution" @change="emit('update:resolution', ($event.target as HTMLSelectElement).value)">
          <option v-for="item in resolutionOptions" :key="item" :value="item">{{ item }}</option>
        </select>
      </label>

      <label v-if="supportsGeneratedAudio" class="s2v-wb-setting-pill">
        <Volume2 class="h-3.5 w-3.5 text-s2v-muted" />
        <input :checked="generateAudio" type="checkbox" @change="emit('update:generateAudio', ($event.target as HTMLInputElement).checked)">
        <span>自动音频</span>
      </label>
    </div>
  </div>
</template>
