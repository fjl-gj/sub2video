<script setup lang="ts">
import S2VCard from '@/shared/ui/S2VCard.vue'
import AuthBrand from '@/widgets/AuthBrand.vue'

defineProps<{ title: string }>()
</script>

<template>
  <div class="grid gap-5 text-center">
    <AuthBrand class="mx-auto" />
    <div>
      <h1 class="text-xl font-bold">{{ title }}</h1>
      <p v-if="$slots.description" class="mt-1.5 text-xs leading-5 text-s2v-muted"><slot name="description" /></p>
    </div>
    <S2VCard variant="padded">
      <slot />
    </S2VCard>
    <div v-if="$slots.footer" class="flex flex-wrap items-center justify-center gap-4 text-xs">
      <slot name="footer" />
    </div>
  </div>
</template>
