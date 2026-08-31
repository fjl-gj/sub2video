<script setup lang="ts">
import { Film } from 'lucide-vue-next'

withDefaults(defineProps<{ size?: 'sm' | 'md' }>(), { size: 'md' })
</script>

<template>
  <span
    :class="['grid place-items-center rounded-control bg-s2v-primary text-white shadow-subtle', size === 'sm' ? 'h-9 w-9' : 'h-11 w-11']"
  >
    <Film :class="size === 'sm' ? 'h-4 w-4' : 'h-5 w-5'" />
  </span>
</template>
