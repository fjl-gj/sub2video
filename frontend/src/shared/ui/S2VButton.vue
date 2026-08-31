<script setup lang="ts">
import { LoaderCircle } from 'lucide-vue-next'
import { tv } from 'tailwind-variants'
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const button = tv({
  base: 'inline-flex items-center justify-center gap-2 rounded-control text-xs transition-colors focus-visible:outline-none disabled:pointer-events-none disabled:opacity-55',
  variants: {
    variant: {
      primary: 'border-0 bg-s2v-primary font-bold text-white shadow-subtle hover:bg-s2v-primary-hover',
      secondary: 'border border-s2v-line bg-s2v-surface font-semibold text-s2v-body shadow-subtle hover:bg-s2v-canvas',
      ghost: 'border-0 bg-transparent font-semibold text-s2v-muted hover:bg-s2v-canvas hover:text-s2v-ink',
      danger: 'border border-rose-200 bg-rose-50 font-semibold text-rose-700 hover:bg-rose-100',
    },
    size: {
      sm: 'min-h-8 px-2.5',
      md: 'min-h-9 px-3.5 py-2.5',
      lg: 'min-h-11 px-5',
    },
    block: { true: 'w-full' },
  },
  defaultVariants: { variant: 'secondary', size: 'md' },
})

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  block?: boolean
  loading?: boolean
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
  class?: string
}>(), {
  variant: 'secondary',
  size: 'md',
  loading: false,
  disabled: false,
  type: 'button',
})

const classes = computed(() => cn(button({ variant: props.variant, size: props.size, block: props.block }), props.class))
</script>

<template>
  <button :type="type" :class="classes" :disabled="disabled || loading">
    <LoaderCircle v-if="loading" class="h-3.5 w-3.5 animate-spin" />
    <slot v-else name="icon" />
    <slot />
  </button>
</template>
