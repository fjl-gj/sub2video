<script setup lang="ts">
defineProps<{
  modelValue: string
  options: string[]
  counts?: Record<string, number>
}>()

defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="flex max-w-full items-center gap-1 overflow-x-auto rounded-nav border border-s2v-line bg-s2v-surface p-1">
    <button
      v-for="option in options"
      :key="option"
      type="button"
      :class="['whitespace-nowrap rounded-nav px-2.5 py-1.5 text-xs font-semibold', modelValue === option ? 'bg-s2v-primary-soft text-s2v-primary' : 'text-s2v-muted hover:text-s2v-ink']"
      @click="$emit('update:modelValue', option)"
    >
      {{ option }}
      <span v-if="counts" class="s2v-mono ml-1">{{ counts[option] ?? 0 }}</span>
    </button>
  </div>
</template>
