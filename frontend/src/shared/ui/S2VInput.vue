<script setup lang="ts">
defineProps<{ modelValue?: string; placeholder?: string; type?: string; label?: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
<template>
  <label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70">
    <span v-if="label">{{ label }}</span>
    <input :value="modelValue" :type="type ?? 'text'" :placeholder="placeholder" class="h-10 w-full rounded-[7px] border border-s2v-line-strong bg-s2v-surface px-3 text-sm font-normal text-s2v-ink placeholder:text-s2v-muted/70" @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)" />
  </label>
</template>
