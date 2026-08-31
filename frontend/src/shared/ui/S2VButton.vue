<script setup lang="ts">
import { LoaderCircle } from 'lucide-vue-next'
import { tv, type VariantProps } from 'tailwind-variants'
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const button = tv({
  base: 'inline-flex min-h-9 items-center justify-center gap-2 rounded-[7px] border px-3 text-xs font-semibold transition-colors focus-visible:outline-none disabled:pointer-events-none disabled:opacity-55',
  variants: {
    variant: {
      primary: 'border-s2v-primary bg-s2v-primary text-white hover:bg-s2v-primary-hover',
      secondary: 'border-s2v-line-strong bg-s2v-surface text-s2v-ink/75 hover:border-s2v-primary hover:bg-s2v-primary-soft hover:text-s2v-primary',
      ghost: 'border-transparent bg-transparent text-s2v-muted hover:bg-s2v-canvas hover:text-s2v-ink',
      danger: 'border-rose-200 bg-rose-50 text-rose-700 hover:bg-rose-100',
    },
    size: { sm: 'min-h-8 px-2.5', md: 'min-h-9 px-3', lg: 'min-h-11 px-4 text-sm' },
    block: { true: 'w-full' },
  },
  defaultVariants: { variant: 'secondary', size: 'md' },
})

type ButtonVariants = VariantProps<typeof button>
interface Props extends /* @vue-ignore */ ButtonVariants { loading?: boolean; disabled?: boolean; type?: 'button' | 'submit' | 'reset' }
const props = withDefaults(defineProps<Props>(), { variant: 'secondary', size: 'md', loading: false, disabled: false, type: 'button' })
const classes = computed(() => cn(button({ variant: props.variant, size: props.size, block: props.block }), props.class))
</script>

<template>
  <button :type="type" :class="classes" :disabled="disabled || loading">
    <LoaderCircle v-if="loading" class="h-3.5 w-3.5 animate-spin" />
    <slot v-else name="icon" />
    <slot />
  </button>
</template>
