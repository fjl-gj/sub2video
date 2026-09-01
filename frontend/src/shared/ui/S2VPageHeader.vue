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
        <span v-if="badge" class="s2v-mono s2v-page-header-badge">{{ badge }}</span>
        <span v-if="kicker" class="text-xs text-s2v-label">{{ kicker }}</span>
      </div>
      <h1 class="s2v-page-title" :class="badge || kicker ? 'mt-1' : ''">{{ title }}</h1>
      <p v-if="$slots.description" class="s2v-subcopy"><slot name="description" /></p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
