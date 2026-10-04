<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const surface = computed(() => (route.meta.surface as string | undefined) ?? 'client')
// Portals (dialogs, menus, toasts) must inherit the same theme as their page.
watchEffect(() => { document.documentElement.dataset.surface = surface.value })
</script>

<template>
  <div :data-surface="surface">
    <RouterView />
  </div>
</template>
