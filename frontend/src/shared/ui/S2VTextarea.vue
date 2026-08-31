<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  label?: string
  rows?: number
}>(), { rows: 5 })

defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <label class="grid w-full gap-1.5 text-xs font-semibold text-s2v-ink/70">
    <span v-if="label">{{ label }}</span>
    <textarea
      :value="modelValue"
      :rows="rows"
      :placeholder="placeholder"
      class="min-h-28 w-full resize-y rounded-control border border-s2v-line bg-s2v-canvas p-3 text-sm font-normal leading-6 text-s2v-ink placeholder:text-s2v-muted/65"
      @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
    />
  </label>
</template>
