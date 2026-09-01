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
  'flex h-9 w-full items-center justify-between rounded-nav border px-2.5 text-[13px] font-medium transition',
  props.active
    ? 'border-s2v-primary/20 bg-s2v-primary-soft font-semibold text-s2v-primary-hover'
    : 'border-transparent text-s2v-nav hover:bg-s2v-nav-hover hover:text-s2v-ink',
))

const badgeClass = computed(() => cn('s2v-mono shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold leading-none', {
  'bg-s2v-primary-soft text-s2v-primary': props.badgeTone === 'primary',
  'bg-emerald-50 text-emerald-600': props.badgeTone === 'success',
  'border border-amber-200/60 bg-amber-50 text-amber-700': props.badgeTone === 'warning',
  'border border-rose-200/50 bg-rose-50 text-rose-600': props.badgeTone === 'danger',
}))
</script>

<template>
  <RouterLink :to="to" :class="classes">
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
