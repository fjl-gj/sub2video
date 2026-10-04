<script setup lang="ts">
import { ImageOff } from 'lucide-vue-next'
import { ref, watch } from 'vue'
const props = defineProps<{ src: string; alt: string }>()
const failed = ref(false)
watch(() => props.src, () => { failed.value = false })
</script>
<template>
  <div class="s2v-guide-image">
    <img v-if="!failed" :src="src" :alt="alt" loading="lazy" @error="failed = true">
    <div v-else class="s2v-guide-image-error"><ImageOff class="h-6 w-6" /><span>参考图暂不可用</span></div>
    <span class="s2v-guide-image-label">构图参考 · 非生成成片</span>
  </div>
</template>
