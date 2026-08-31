<script setup lang="ts">
import { CheckCircle2 } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'

const app = useAppStore()
</script>

<template>
  <Transition name="toast">
    <div v-if="app.toast" class="fixed bottom-20 right-4 z-50 flex max-w-xs items-center gap-2 rounded-control border border-emerald-200/80 bg-emerald-50 px-3 py-2.5 text-xs font-semibold text-emerald-800 shadow-subtle md:bottom-5 md:right-5">
      <CheckCircle2 class="h-4 w-4 shrink-0" />{{ app.toast }}
    </div>
  </Transition>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: opacity 0.18s, transform 0.18s; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateY(8px); }
</style>
