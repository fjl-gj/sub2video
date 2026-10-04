<script setup lang="ts">
import GuideLibrary from '@/features/guides/GuideLibrary.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="广场">
      <template #description>从镜头教程开始，用自己的素材与想法练习创作。</template>
    </S2VPageHeader>
    <GuideLibrary searchable />
  </section>
</template>
