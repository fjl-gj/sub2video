<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/cn'

const props = withDefaults(defineProps<{
  to: string
  active?: boolean
  badge?: string
  badgeTone?: 'primary' | 'success' | 'warning' | 'danger'
}>(), { active: false, badgeTone: 'primary' })

const classes = computed(() => cn(
  's2v-nav-item flex w-full items-center justify-between rounded-nav border px-3 text-sm font-medium transition',
  props.active
    ? 'border-transparent bg-s2v-primary-soft font-semibold text-s2v-primary-hover'
    : 'border-transparent text-s2v-nav hover:bg-s2v-nav-hover hover:text-s2v-ink',
))

const badgeClass = computed(() => cn('s2v-mono shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold leading-none', {
  'bg-s2v-primary-soft text-s2v-primary': props.badgeTone === 'primary',
  'bg-s2v-success/10 text-s2v-success': props.badgeTone === 'success',
  'bg-s2v-warning/10 text-s2v-warning': props.badgeTone === 'warning',
  'bg-s2v-danger/10 text-s2v-danger': props.badgeTone === 'danger',
}))
</script>

<template>
  <RouterLink :to="to" :class="classes" :aria-current="active ? 'page' : undefined">
    <span class="flex min-w-0 items-center gap-2.5">
      <span class="s2v-nav-slot" :class="active ? 'text-s2v-primary' : 'text-s2v-label'">
        <slot name="icon" />
      </span>
      <span class="truncate"><slot /></span>
    </span>
    <span v-if="badge" :class="badgeClass">{{ badge }}</span>
    <slot v-else name="trailing" />
  </RouterLink>
</template>
