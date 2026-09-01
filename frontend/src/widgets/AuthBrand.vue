<script setup lang="ts">
import S2VLogo from '@/shared/ui/S2VLogo.vue'

withDefaults(defineProps<{ size?: 'sm' | 'md' | 'lg'; variant?: 'wordmark' | 'compact' }>(), { size: 'md', variant: 'wordmark' })
</script>

<template>
  <S2VLogo :size="size" :variant="variant" />
</template>
