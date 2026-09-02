<script setup lang="ts">
import { Sparkles } from 'lucide-vue-next'

defineProps<{
  cost: number
  loading: boolean
  disabled: boolean
}>()

const prompt = defineModel<string>('prompt', { required: true })

const emit = defineEmits<{ submit: [] }>()
</script>

<template>
  <div class="s2v-wb-prompt-section">
    <div class="s2v-wb-section-head">
      <span>画面描述</span>
    </div>
    <textarea
      v-model="prompt"
      class="s2v-wb-prompt-textarea"
      placeholder="描述主体动作、镜头运动与环境变化…&#10;例如：主体缓缓低头，微风吹拂发丝，光影柔和自然"
    />
  </div>

  <div class="s2v-wb-submit-section">
    <div class="s2v-wb-cost-row">
      <span class="s2v-wb-cost-label">预计费用</span>
      <strong class="s2v-wb-cost-value s2v-mono">¥ {{ cost.toFixed(2) }}</strong>
    </div>
    <button
      type="button"
      class="s2v-wb-submit-btn"
      :disabled="disabled || loading"
      @click="emit('submit')"
    >
      <Sparkles class="h-4 w-4" />
      <span>{{ loading ? '提交中…' : '提交生成任务' }}</span>
    </button>
  </div>
</template>
