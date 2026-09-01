<script setup lang="ts">
import { Lock } from 'lucide-vue-next'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'

withDefaults(defineProps<{
  title: string
  description: string
}>(), {})

defineEmits<{ switchImage: [] }>()
</script>

<template>
  <S2VCard variant="padded" class="grid gap-4 text-center">
    <span class="s2v-icon-well s2v-icon-well-nav mx-auto"><Lock class="h-4 w-4" /></span>
    <div>
      <h2 class="text-sm font-bold text-s2v-ink">{{ title }}</h2>
      <p class="mt-2 text-xs leading-6 text-s2v-muted">{{ description }}</p>
    </div>
    <p class="s2v-inset-dashed text-left text-xs leading-6">该工作流正在接入供应商能力，开放后会保留同样的参数栏与提交流程。</p>
    <S2VButton class="mx-auto" variant="primary" @click="$emit('switchImage')">先使用图生视频</S2VButton>
  </S2VCard>
</template>
