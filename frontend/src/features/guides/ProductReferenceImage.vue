<script setup lang="ts">
import { ImageOff } from 'lucide-vue-next'
import { shallowRef, watch } from 'vue'

const props = defineProps<{ src: string; alt: string }>()
const failed = shallowRef(false)
watch(() => props.src, () => { failed.value = false })
</script>

<template>
  <div class="s2v-home-reference-image">
    <img v-if="!failed" :src="src" :alt="alt" loading="lazy" @error="failed = true" />
    <div v-else class="s2v-home-reference-error" role="status"><ImageOff :size="24" aria-hidden="true" /><span>参考图暂不可用</span></div>
  </div>
</template>
