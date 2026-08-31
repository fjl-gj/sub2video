<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{ tone?: 'neutral' | 'primary' | 'success' | 'warning' | 'danger'; class?: string }>(), { tone: 'neutral' })
const classes = computed(() => cn('inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-semibold', {
  'border-s2v-line bg-s2v-surface text-s2v-muted': props.tone === 'neutral',
  'border-indigo-200 bg-indigo-50 text-indigo-700': props.tone === 'primary',
  'border-emerald-200 bg-emerald-50 text-emerald-700': props.tone === 'success',
  'border-amber-200 bg-amber-50 text-amber-700': props.tone === 'warning',
  'border-rose-200 bg-rose-50 text-rose-700': props.tone === 'danger',
}, props.class))
</script>

<template><span :class="classes"><slot /></span></template>
