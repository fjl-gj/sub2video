<script setup lang="ts">
import { Info } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'

const app = useAppStore()
</script>

<template>
  <Transition name="toast">
    <div v-if="app.toast" role="status" class="fixed bottom-20 right-4 z-50 flex max-w-xs items-center gap-2 rounded-control border border-s2v-line-strong bg-s2v-surface px-3 py-2.5 text-xs text-s2v-body shadow-subtle md:bottom-5 md:right-5">
      <Info class="h-4 w-4 shrink-0" />{{ app.toast }}
    </div>
  </Transition>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: opacity 0.18s, transform 0.18s; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateY(8px); }
</style>
