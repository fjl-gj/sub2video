<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{ label: string; active?: boolean; class?: string }>(), { active: false })
const classes = computed(() => cn(
  'grid h-9 w-9 place-items-center rounded-nav border transition-colors',
  props.active ? 'border-s2v-primary bg-s2v-primary-soft text-s2v-primary' : 'border-s2v-line bg-s2v-surface text-s2v-muted hover:border-s2v-primary hover:text-s2v-primary',
  props.class,
))
</script>

<template>
  <button type="button" :aria-label="label" :title="label" :class="classes"><slot /></button>
</template>
