<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  variant?: 'default' | 'metric' | 'padded' | 'muted' | 'highlight' | 'media' | 'compact'
  class?: string
}>(), { variant: 'default' })

const classes = computed(() => cn(
  's2v-panel',
  props.variant === 'metric' && 'min-h-24 p-3.5',
  props.variant === 'padded' && 'p-5',
  props.variant === 'muted' && 'bg-s2v-canvas/70',
  props.variant === 'highlight' && 'border-s2v-primary/30 bg-s2v-primary-soft/25',
  props.variant === 'media' && 'overflow-hidden',
  props.variant === 'compact' && 'p-3.5',
  props.class,
))
</script>

<template>
  <section :class="classes"><slot /></section>
</template>
