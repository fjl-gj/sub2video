<script setup lang="ts">
defineProps<{ modelValue?: string; options: Array<{ label: string; value: string }>; label?: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
<template>
  <label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70">
    <span v-if="label">{{ label }}</span>
    <select :value="modelValue" class="h-10 w-full rounded-[7px] border border-s2v-line-strong bg-s2v-surface px-3 text-sm font-normal text-s2v-ink" @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)">
      <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
    </select>
  </label>
</template>
