<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  type?: string
  label?: string
  size?: 'md' | 'sm'
  min?: string | number
  disabled?: boolean
  class?: string
}>(), {
  type: 'text',
  size: 'md',
  disabled: false,
})

defineEmits<{ 'update:modelValue': [value: string] }>()

const rootClass = computed(() => cn('grid w-full gap-1.5 text-xs font-semibold text-s2v-ink/70', props.class))
const shell = computed(() => cn(
  'flex w-full items-center gap-2 border border-s2v-line bg-s2v-canvas focus-within:border-s2v-primary focus-within:ring-1 focus-within:ring-s2v-primary',
  props.size === 'sm' ? 'h-9 rounded-nav px-2.5' : 'h-10 rounded-control px-3',
  props.disabled && 'opacity-55',
))
</script>

<template>
  <label :class="rootClass">
    <span v-if="label">{{ label }}</span>
    <span :class="shell">
      <slot name="prefix" />
      <input
        v-bind="$attrs"
        :value="modelValue"
        :type="type"
        :min="min"
        :placeholder="placeholder"
        :disabled="disabled"
        class="min-w-0 flex-1 bg-transparent text-sm font-normal text-s2v-ink outline-none placeholder:text-s2v-muted/70 disabled:cursor-not-allowed"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
      <slot name="suffix" />
    </span>
  </label>
</template>
