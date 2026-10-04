<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AuthShell from '@/features/auth/AuthShell.vue'

const route = useRoute()
const isEntry = computed(() => ['/auth/login', '/auth/register'].includes(route.path))
</script>

<template>
  <AuthShell v-if="isEntry"><RouterView /></AuthShell>
  <main v-else class="grid min-h-screen place-items-center bg-s2v-canvas p-5">
    <div class="w-full max-w-md"><RouterView /></div>
  </main>
</template>
