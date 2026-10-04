<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  tone?: 'neutral' | 'primary' | 'success' | 'warning' | 'danger'
  class?: string
}>(), { tone: 'neutral' })

const classes = computed(() => cn('s2v-badge inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[11px] font-medium leading-none', {
  'border-s2v-line bg-s2v-canvas text-s2v-muted': props.tone === 'neutral',
  'border-s2v-primary/25 bg-s2v-primary-soft text-s2v-primary': props.tone === 'primary',
  'border-s2v-success/20 bg-s2v-success/10 text-s2v-success': props.tone === 'success',
  'border-s2v-warning/20 bg-s2v-warning/10 text-s2v-warning': props.tone === 'warning',
  'border-s2v-danger/20 bg-s2v-danger/10 text-s2v-danger': props.tone === 'danger',
}, props.class))
</script>

<template>
  <span :class="classes"><slot /></span>
</template>
