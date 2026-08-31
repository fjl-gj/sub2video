<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{ variant?: 'default' | 'muted' | 'highlight'; class?: string }>(), { variant: 'default' })
const classes = computed(() => cn('s2v-panel', props.variant === 'muted' && 'bg-slate-50/70', props.variant === 'highlight' && 'border-s2v-primary/35 bg-s2v-primary-soft/35', props.class))
</script>

<template><section :class="classes"><slot /></section></template>
