<script setup lang="ts">
import { computed } from 'vue'
import { tv } from 'tailwind-variants'
import { cn } from '@/shared/lib/cn'

const logo = tv({
  base: 's2v-latin inline-flex shrink-0 select-none items-baseline font-semibold leading-none tracking-tight text-s2v-ink',
  variants: { size: { sm: 'text-base', md: 'text-xl', lg: 'text-2xl' } },
  defaultVariants: { size: 'md' },
})
const props = withDefaults(defineProps<{
  size?: 'sm' | 'md' | 'lg'
  variant?: 'wordmark' | 'compact'
  class?: string
}>(), { size: 'md', variant: 'wordmark' })
const classes = computed(() => cn(logo({ size: props.size }), props.class))
</script>

<template>
  <span :class="classes" aria-label="Sub2Video"><span>{{ variant === 'compact' ? 'S' : 'Sub' }}</span><span class="text-s2v-primary">2</span><span>{{ variant === 'compact' ? 'V' : 'Video' }}</span></span>
</template>
