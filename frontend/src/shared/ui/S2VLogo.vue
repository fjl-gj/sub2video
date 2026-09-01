<script setup lang="ts">
import { tv } from 'tailwind-variants'
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const logo = tv({
  slots: {
    root: 's2v-latin inline-flex min-w-0 select-none items-end leading-none',
    sub: 'font-medium text-s2v-ink',
    bridge: 'relative mx-px inline-flex flex-col items-center',
    two: 'bg-gradient-to-br from-s2v-primary to-s2v-primary-accent bg-clip-text font-extrabold text-transparent',
    cues: 'mt-0.5 flex flex-col items-center gap-px',
    cueLong: 'h-px rounded-full bg-gradient-to-r from-s2v-primary to-s2v-primary-accent',
    cueShort: 'h-px rounded-full bg-gradient-to-r from-s2v-primary/70 to-s2v-primary-accent/70',
    video: 'font-bold text-s2v-ink',
  },
  variants: {
    size: {
      sm: {
        root: 'gap-0',
        sub: 'text-[13px]',
        two: 'text-[17px]',
        cueLong: 'w-2.5',
        cueShort: 'w-1.5',
        video: 'text-[13px]',
      },
      md: {
        sub: 'text-sm',
        two: 'text-xl',
        cueLong: 'w-3',
        cueShort: 'w-2',
        video: 'text-sm',
      },
      lg: {
        sub: 'text-base',
        two: 'text-[26px]',
        cueLong: 'w-3.5',
        cueShort: 'w-2.5',
        video: 'text-base',
      },
    },
    variant: {
      wordmark: {},
      compact: {
        bridge: 'hidden',
        video: 'hidden',
        sub: 'sr-only',
        two: 'text-lg',
      },
    },
  },
  defaultVariants: { size: 'md', variant: 'wordmark' },
})

const props = withDefaults(defineProps<{
  size?: 'sm' | 'md' | 'lg'
  variant?: 'wordmark' | 'compact'
  class?: string
}>(), {
  size: 'md',
  variant: 'wordmark',
})

const slots = computed(() => logo({ size: props.size, variant: props.variant }))
</script>

<template>
  <span :class="cn(slots.root(), props.class)" aria-label="Sub2Video">
    <span :class="slots.sub()">Sub</span>
    <span :class="slots.bridge()">
      <span :class="slots.two()">2</span>
      <span v-if="variant === 'wordmark'" :class="slots.cues()">
        <span :class="slots.cueLong()" />
        <span :class="slots.cueShort()" />
      </span>
    </span>
    <span :class="slots.video()">Video</span>
  </span>
</template>
