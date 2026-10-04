<script setup lang="ts">
import { ArrowRight, LoaderCircle, RefreshCw } from 'lucide-vue-next'
import S2VButton from '@/shared/ui/S2VButton.vue'
import type { PendingCreation } from '@/shared/lib/pending-creation'

defineProps<{
  cost: number | null
  loading: boolean
  disabled: boolean
  unavailable?: boolean
  estimateLoading?: boolean
  estimateError?: string
  preparationHint?: string
  submitHint?: string
  submissionError?: string
  pending?: PendingCreation | null
}>()

const prompt = defineModel<string>('prompt', { required: true })
defineEmits<{ submit: []; retryEstimate: []; recover: [] }>()
</script>

<template>
  <div class="s2v-studio-section">
    <div class="s2v-studio-field-heading"><label for="studio-prompt">画面描述</label><span>描述动作与镜头</span></div>
    <textarea id="studio-prompt" v-model="prompt" class="s2v-studio-prompt" rows="5" placeholder="描述主体动作、镜头运动与环境变化…&#10;例如：主体缓缓低头，微风吹拂发丝，光影柔和自然" />
  </div>
  <slot />
  <footer class="s2v-studio-submit">
    <div class="s2v-studio-price" aria-label="费用预估" aria-live="polite" :aria-busy="estimateLoading">
      <div><span>预估费用</span><p>按请求时长计费</p></div>
      <span v-if="estimateLoading" class="s2v-studio-estimate-state"><LoaderCircle class="h-4 w-4 animate-spin" />预估中</span>
      <span v-else-if="preparationHint" class="s2v-studio-estimate-state">等待素材</span>
      <span v-else-if="estimateError" class="s2v-studio-estimate-state">暂无法估价</span>
      <strong v-else-if="cost !== null" class="s2v-mono">¥ {{ cost.toFixed(2) }}</strong>
      <span v-else class="s2v-studio-estimate-state">等待生成设置</span>
    </div>
    <p v-if="unavailable" class="s2v-studio-inline-error" role="status">当前没有可用生成能力，暂不可提交</p>
    <div v-else-if="estimateError && !estimateLoading" class="s2v-studio-estimate-error" role="status">
      <p>{{ estimateError }}</p>
      <button type="button" aria-label="重新预估费用" @click="$emit('retryEstimate')"><RefreshCw class="h-3.5 w-3.5" />重试</button>
    </div>
    <p v-if="!pending && submitHint && !unavailable" class="s2v-studio-submit-note" role="status">{{ submitHint }}</p>
    <div v-if="pending" class="s2v-studio-estimate-error" role="status">
      <div>
        <p>有一笔提交等待确认，请先核对原提交。</p>
        <p>{{ pending.payload.durationSec }} 秒 · {{ pending.payload.resolution }} · {{ pending.payload.aspectRatio }}</p>
        <p>将按原参数确认或继续提交；已创建的任务不会再次扣费。当前编辑不会替换原提交。</p>
      </div>
    </div>
    <p v-if="submissionError" class="s2v-studio-inline-error" role="alert">{{ submissionError }}</p>
    <S2VButton v-if="pending" variant="primary" size="lg" :block="true" :disabled="loading" :loading="loading" @click="$emit('recover')">确认原提交结果</S2VButton>
    <S2VButton v-else variant="primary" size="lg" :block="true" :disabled="disabled || loading" :loading="loading" @click="$emit('submit')">{{ loading ? '提交中…' : '提交生成任务' }}<ArrowRight v-if="!loading" class="h-4 w-4" /></S2VButton>
    <p class="s2v-studio-submit-note">提交时按当前价格计费，失败自动退款</p>
  </footer>
</template>
