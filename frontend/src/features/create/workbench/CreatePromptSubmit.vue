<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, LoaderCircle, RefreshCw } from 'lucide-vue-next'
import S2VButton from '@/shared/ui/S2VButton.vue'
import type { PendingCreation } from '@/shared/lib/pending-creation'

const props = defineProps<{
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
  canEstimate?: boolean
}>()

const prompt = defineModel<string>('prompt', { required: true })
defineEmits<{ submit: []; retryEstimate: []; recover: [] }>()

const estimateDisabled = computed(() =>
  !!props.unavailable
  || !!props.pending
  || !!props.preparationHint
  || !!props.estimateLoading
  || !props.canEstimate
  || !prompt.value.trim(),
)
</script>

<template>
  <div class="s2v-studio-section s2v-studio-prompt-block">
    <label class="sr-only" for="studio-prompt">画面描述</label>
    <textarea
      id="studio-prompt"
      v-model="prompt"
      class="s2v-studio-prompt"
      rows="4"
      placeholder="描述你想要的镜头。点击下方 @ 可引用已有素材，或用 + 上传。"
    />
  </div>
  <div class="s2v-studio-composer-bar">
    <slot name="tools" />
    <div class="s2v-studio-price" aria-label="费用预估" aria-live="polite" :aria-busy="estimateLoading">
      <div>
        <strong v-if="!estimateLoading && !preparationHint && !estimateError && cost !== null" class="s2v-mono">¥ {{ cost.toFixed(2) }}</strong>
        <span v-else-if="estimateLoading" class="s2v-studio-estimate-state"><LoaderCircle class="h-4 w-4 animate-spin" />预估中</span>
        <span v-else-if="preparationHint" class="s2v-studio-estimate-state">{{ preparationHint }}</span>
        <span v-else-if="estimateError" class="s2v-studio-estimate-state">暂无法估价</span>
        <span v-else class="s2v-studio-estimate-state">先估价，再生成</span>
        <p>估价不扣费 · 提交时扣费</p>
      </div>
    </div>
    <div class="s2v-studio-submit-actions">
      <S2VButton
        v-if="!pending"
        variant="secondary"
        size="lg"
        aria-label="预估费用"
        :disabled="estimateDisabled"
        :loading="estimateLoading"
        @click="$emit('retryEstimate')"
      >估价</S2VButton>
      <S2VButton v-if="pending" variant="primary" size="lg" :disabled="loading" :loading="loading" @click="$emit('recover')">确认原提交结果</S2VButton>
      <S2VButton v-else variant="primary" size="lg" aria-label="提交生成任务" :disabled="disabled || loading" :loading="loading" @click="$emit('submit')">{{ loading ? '提交中…' : '生成' }}<ArrowRight v-if="!loading" class="h-4 w-4" /></S2VButton>
    </div>
  </div>
  <slot />
  <footer class="s2v-studio-submit">
    <p v-if="unavailable" class="s2v-studio-inline-error" role="status">当前没有可用生成能力，暂不可提交</p>
    <div v-else-if="estimateError && !estimateLoading" class="s2v-studio-estimate-error" role="status">
      <p>{{ estimateError }}</p>
      <button type="button" aria-label="重新预估费用" @click="$emit('retryEstimate')"><RefreshCw class="h-3.5 w-3.5" />重试</button>
    </div>
    <p v-if="!pending && submitHint && !unavailable && !preparationHint" class="s2v-studio-submit-note" role="status">{{ submitHint }}</p>
    <div v-if="pending" class="s2v-studio-estimate-error" role="status">
      <div>
        <p>有一笔提交等待确认，请先核对原提交。</p>
        <p>{{ pending.payload.durationSec }} 秒 · {{ pending.payload.resolution }} · {{ pending.payload.aspectRatio }}</p>
        <p>将按原参数确认或继续提交；已创建的任务不会再次扣费。当前编辑不会替换原提交。</p>
      </div>
    </div>
    <p v-if="submissionError" class="s2v-studio-inline-error" role="alert">{{ submissionError }}</p>
  </footer>
</template>
