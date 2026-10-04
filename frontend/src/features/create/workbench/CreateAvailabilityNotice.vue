<script setup lang="ts">
import { RefreshCw } from 'lucide-vue-next'
import S2VButton from '@/shared/ui/S2VButton.vue'
defineProps<{ error: string; loading: boolean }>()
defineEmits<{ retry: [] }>()
</script>
<template>
  <div class="s2v-studio-capability-error" role="alert">
    <p>{{ error }}</p>
    <S2VButton :loading="loading" @click="$emit('retry')"><RefreshCw class="h-4 w-4" />重新加载生成能力</S2VButton>
  </div>
</template>
