<script setup lang="ts">
defineProps<{
  badge?: string
  kicker?: string
  title: string
}>()
</script>

<template>
  <header class="s2v-page-header">
    <div class="min-w-0">
      <div v-if="badge || kicker" class="flex flex-wrap items-center gap-2">
        <span v-if="badge" class="s2v-mono rounded border border-s2v-primary/25 bg-s2v-primary-soft px-2 py-0.5 text-xs font-bold text-s2v-primary">{{ badge }}</span>
        <span v-if="kicker" class="text-xs text-s2v-muted">{{ kicker }}</span>
      </div>
      <h1 class="s2v-page-title" :class="badge || kicker ? 'mt-1' : ''">{{ title }}</h1>
      <p v-if="$slots.description" class="s2v-subcopy"><slot name="description" /></p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
