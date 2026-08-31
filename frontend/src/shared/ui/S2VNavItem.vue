<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  to: string
  active?: boolean
  badge?: string
}>(), { active: false })

const classes = computed(() => cn(
  'flex h-9 w-full items-center justify-between rounded-nav border px-2.5 text-[13px] font-medium transition',
  props.active
    ? 'border-s2v-primary/20 bg-s2v-primary-soft font-semibold text-s2v-primary-hover'
    : 'border-transparent text-s2v-muted hover:bg-s2v-canvas hover:text-s2v-ink',
))
</script>

<template>
  <RouterLink :to="to" :class="classes">
    <span class="flex min-w-0 items-center gap-2.5">
      <span class="s2v-nav-slot" :class="active ? 'text-s2v-primary' : 'text-s2v-muted'">
        <slot name="icon" />
      </span>
      <span class="truncate"><slot /></span>
    </span>
    <span v-if="badge" class="s2v-mono shrink-0 rounded border border-s2v-primary/20 bg-s2v-primary-soft px-1.5 py-0.5 text-[10px] font-bold leading-none text-s2v-primary">{{ badge }}</span>
    <slot v-else name="trailing" />
  </RouterLink>
</template>
