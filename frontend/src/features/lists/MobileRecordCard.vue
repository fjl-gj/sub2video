<script setup lang="ts">
import S2VCard from '@/shared/ui/S2VCard.vue'

defineProps<{ title: string; meta?: string }>()
</script>

<template>
  <S2VCard variant="compact">
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <strong class="block truncate text-xs">{{ title }}</strong>
        <span v-if="meta" class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ meta }}</span>
      </div>
      <slot name="badge" />
    </div>
    <p v-if="$slots.body" class="mt-2 text-[11px] leading-5 text-s2v-muted"><slot name="body" /></p>
    <div v-if="$slots.actions" class="mt-3 flex items-center gap-1"><slot name="actions" /></div>
  </S2VCard>
</template>
