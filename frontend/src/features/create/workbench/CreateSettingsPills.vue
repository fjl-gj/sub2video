<script setup lang="ts">
import { Clock, Layers, Paperclip, Ratio, Volume2, X } from 'lucide-vue-next'
import { ref } from 'vue'

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
  audioMode?: 'none' | 'generate' | 'reference'
  referenceAudioName?: string
}>(), {
  durationOptions: () => ['5', '10', '15'],
  resolutionOptions: () => ['768P'],
  aspectRatioOptions: () => ['16:9', '9:16', '1:1'],
  carriers: () => [],
  models: () => [],
  carrierId: '',
  modelId: '',
  supportsGeneratedAudio: false,
  audioMode: 'none',
  referenceAudioName: '',
})

const emit = defineEmits<{
  'update:ratio': [value: string]
  'update:durationSec': [value: string]
  'update:resolution': [value: string]
  'update:carrierId': [value: string]
  'update:modelId': [value: string]
  'update:audioMode': [value: 'none' | 'generate' | 'reference']
  'reference-audio-upload': [file: File]
  'clear-reference-audio': []
}>()

const referenceAudioInput = ref<HTMLInputElement | null>(null)

function chooseReferenceAudio() {
  referenceAudioInput.value?.click()
}

function onReferenceAudioChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) emit('reference-audio-upload', file)
  input.value = ''
}
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
        <select aria-label="音频设置" :value="audioMode" @change="emit('update:audioMode', ($event.target as HTMLSelectElement).value as 'none' | 'generate' | 'reference')">
          <option value="none">不使用音频</option>
          <option value="generate">自动生成音频</option>
          <option value="reference">上传参考音频</option>
        </select>
      </label>

      <template v-if="supportsGeneratedAudio && audioMode === 'reference'">
        <input ref="referenceAudioInput" class="hidden" type="file" accept="audio/mpeg,audio/wav,audio/x-wav,audio/flac,audio/mp4,video/mp4,.mp3,.wav,.flac,.m4a,.mp4" aria-label="参考音频文件" @change="onReferenceAudioChange">
        <button type="button" class="s2v-wb-setting-pill" title="上传参考音频" @click="chooseReferenceAudio">
          <Paperclip class="h-3.5 w-3.5 text-s2v-muted" />
          <span>{{ referenceAudioName || '选择音频文件' }}</span>
        </button>
        <button v-if="referenceAudioName" type="button" class="s2v-wb-setting-pill" title="移除参考音频" aria-label="移除参考音频" @click="emit('clear-reference-audio')">
          <X class="h-3.5 w-3.5 text-s2v-muted" />
        </button>
      </template>
    </div>
  </div>
</template>
