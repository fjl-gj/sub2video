<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const surface = computed(() => route.meta.surface as string | undefined)
</script>

<template>
  <div :data-surface="surface ?? 'client'">
    <RouterView />
  </div>
</template>
