<script setup lang="ts">
import { computed, watch } from 'vue'
import S2VButton from './S2VButton.vue'

const page = defineModel<number>({ default: 1 })
const props = withDefaults(defineProps<{
  total: number
  pageSize?: number
}>(), { pageSize: 10 })

const pages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize) || 1))

watch(pages, (count) => {
  if (page.value > count) page.value = count
})
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-3">
    <span class="text-[11px] text-s2v-muted">共 {{ total }} 条，第 {{ page }} / {{ pages }} 页</span>
    <div class="flex gap-2">
      <S2VButton size="sm" :disabled="page <= 1" @click="page -= 1">上一页</S2VButton>
      <S2VButton size="sm" :disabled="page >= pages" @click="page += 1">下一页</S2VButton>
    </div>
  </div>
</template>
