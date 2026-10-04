<script setup lang="ts">
import { ArrowRight, Plus } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import StudioOverview from '@/features/overview/StudioOverview.vue'
import { useStudioOverview } from '@/features/overview/useStudioOverview'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import '@/features/overview/studio.css'

const router = useRouter()
const { data, loading, error, modes, workspaceName, load } = useStudioOverview()
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader :title="`欢迎回来，${workspaceName}`">
      <template #description>继续上一次的创作，或从新的画面开始。</template>
      <template #actions>
        <S2VButton @click="router.push('/app/works')">查看任务 <ArrowRight class="h-3.5 w-3.5" /></S2VButton>
        <S2VButton variant="primary" @click="router.push('/app/create')">
          <template #icon><Plus class="h-4 w-4" /></template>
          继续创作
        </S2VButton>
      </template>
    </S2VPageHeader>
    <StudioOverview :data="data" :loading="loading" :error="error" :modes="modes" @reload="load" />
  </section>
</template>
