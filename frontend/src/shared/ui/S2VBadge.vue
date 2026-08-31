<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  tone?: 'neutral' | 'primary' | 'success' | 'warning' | 'danger'
  class?: string
}>(), { tone: 'neutral' })

const classes = computed(() => cn('inline-flex items-center gap-1 rounded border px-1.5 py-0.5 font-mono text-[10px] font-semibold leading-none', {
  'border-s2v-line bg-s2v-canvas text-s2v-muted': props.tone === 'neutral',
  'border-s2v-primary/25 bg-s2v-primary-soft text-s2v-primary': props.tone === 'primary',
  'border-emerald-200/60 bg-emerald-50 text-emerald-700': props.tone === 'success',
  'border-amber-200/60 bg-amber-50 text-amber-700': props.tone === 'warning',
  'border-rose-200/50 bg-rose-50 text-rose-600': props.tone === 'danger',
}, props.class))
</script>

<template>
  <span :class="classes"><slot /></span>
</template>
