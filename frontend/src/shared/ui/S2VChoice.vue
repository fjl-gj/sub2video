<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  selected?: boolean
  disabled?: boolean
  variant?: 'option' | 'tile'
  class?: string
}>(), {
  selected: false,
  disabled: false,
  variant: 'option',
})

const classes = computed(() => cn(
  'border text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-55',
  props.variant === 'option' && 'h-9 rounded-nav',
  props.variant === 'tile' && 'rounded-nav p-3 text-left',
  props.selected
    ? 'border-s2v-primary bg-s2v-primary-soft text-s2v-primary'
    : 'border-s2v-line bg-s2v-surface text-s2v-muted hover:border-s2v-primary/40',
  props.class,
))
</script>

<template>
  <button type="button" :class="classes" :disabled="disabled">
    <slot />
  </button>
</template>
