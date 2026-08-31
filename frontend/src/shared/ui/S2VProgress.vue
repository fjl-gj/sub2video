<script setup lang="ts">
defineProps<{ value: number }>()
</script>

<template>
  <div class="s2v-progress"><span :style="{ width: `${Math.min(100, Math.max(0, value))}%` }" /></div>
</template>
