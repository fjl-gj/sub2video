<script setup lang="ts">
import { Paperclip, X } from 'lucide-vue-next'
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
  supportsReferenceAudio?: boolean
  audioMode?: 'none' | 'generate' | 'reference'
  referenceAudioName?: string
}>(), {
  durationOptions: () => ['5', '10', '15'], resolutionOptions: () => ['768P'],
  aspectRatioOptions: () => ['16:9', '9:16', '1:1'], carriers: () => [], models: () => [],
  carrierId: '', modelId: '', supportsGeneratedAudio: false, supportsReferenceAudio: true, audioMode: 'none', referenceAudioName: '',
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
function onReferenceAudioChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) emit('reference-audio-upload', file)
  input.value = ''
}
</script>

<template>
  <section class="s2v-studio-section" aria-label="生成设置">
    <div class="s2v-studio-field-heading"><h2>生成设置</h2></div>
    <div class="s2v-studio-fields">
      <label class="s2v-studio-field"><span>承运商</span>
        <select aria-label="承运商" :value="carrierId" @change="emit('update:carrierId', ($event.target as HTMLSelectElement).value)">
          <option value="">智能推荐</option><option v-for="carrier in carriers" :key="carrier.id" :value="carrier.id">{{ carrier.name }}</option>
        </select>
      </label>
      <label class="s2v-studio-field"><span>模型</span>
        <select aria-label="模型" :value="modelId" @change="emit('update:modelId', ($event.target as HTMLSelectElement).value)">
          <option value="">智能推荐</option><option v-for="model in models" :key="model.id" :value="model.id">{{ model.name }}</option>
        </select>
      </label>
      <label class="s2v-studio-field"><span>视频时长</span>
        <select aria-label="视频时长" :value="durationSec" @change="emit('update:durationSec', ($event.target as HTMLSelectElement).value)">
          <option v-for="sec in durationOptions" :key="sec" :value="sec">{{ sec }} 秒</option>
        </select>
      </label>
      <label class="s2v-studio-field"><span>分辨率</span>
        <select aria-label="输出分辨率" :value="resolution" @change="emit('update:resolution', ($event.target as HTMLSelectElement).value)">
          <option v-for="item in resolutionOptions" :key="item" :value="item">{{ item }}</option>
        </select>
      </label>
      <label class="s2v-studio-field"><span>画幅比例</span>
        <select aria-label="画幅比例" :value="ratio" @change="emit('update:ratio', ($event.target as HTMLSelectElement).value)">
          <option v-for="item in aspectRatioOptions" :key="item" :value="item">{{ item }}{{ item === '16:9' ? ' 横屏' : item === '9:16' ? ' 竖屏' : item === '1:1' ? ' 方形' : '' }}</option>
        </select>
      </label>
      <label class="s2v-studio-field"><span>音频</span>
        <select aria-label="音频设置" :value="audioMode" :disabled="!supportsGeneratedAudio" @change="emit('update:audioMode', ($event.target as HTMLSelectElement).value as 'none' | 'generate' | 'reference')">
          <option value="none">不使用音频</option>
          <option v-if="supportsGeneratedAudio" value="generate">自动生成音频</option>
          <option v-if="supportsGeneratedAudio && supportsReferenceAudio" value="reference">上传参考音频</option>
          <option v-else-if="audioMode === 'reference'" value="reference" disabled>参考音频暂不可用于导演台</option>
        </select>
      </label>
      <div v-if="supportsGeneratedAudio && supportsReferenceAudio && audioMode === 'reference'" class="s2v-studio-audio-upload">
        <input ref="referenceAudioInput" class="hidden" type="file" accept="audio/mpeg,audio/wav,audio/x-wav,audio/flac,audio/mp4,video/mp4,.mp3,.wav,.flac,.m4a,.mp4" aria-label="参考音频文件" @change="onReferenceAudioChange">
        <button type="button" title="上传参考音频" @click="referenceAudioInput?.click()"><Paperclip class="h-4 w-4" /><span>{{ referenceAudioName || '选择参考音频' }}</span></button>
        <button v-if="referenceAudioName" type="button" aria-label="移除参考音频" title="移除参考音频" @click="emit('clear-reference-audio')"><X class="h-4 w-4" /></button>
      </div>
    </div>
  </section>
</template>
