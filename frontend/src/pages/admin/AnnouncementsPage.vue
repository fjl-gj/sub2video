<script setup lang="ts">
import { Megaphone, Plus, Send } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Notice" title="公告">
      <template #description>向创作者同步价格、服务状态和规则变更。</template>
      <template #actions>
        <S2VButton variant="primary" @click="app.createAnnouncement()"><Plus class="h-3.5 w-3.5" />新建公告</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 lg:grid-cols-2">
      <S2VCard v-for="notice in app.announcements" :key="notice.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <span class="s2v-icon-well s2v-icon-well-nav"><Megaphone class="h-4 w-4" /></span>
          <S2VBadge :tone="notice.status === '已发布' ? 'success' : 'warning'">{{ notice.status }}</S2VBadge>
        </div>
        <h2 class="mt-4 text-sm font-bold">{{ notice.title }}</h2>
        <p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ notice.desc }}</p>
        <div class="mt-4 flex items-center justify-between border-t border-s2v-line pt-3">
          <span class="text-[11px] text-s2v-muted">{{ notice.date }}</span>
          <S2VButton size="sm" @click="app.publishAnnouncement(notice.id)"><Send class="h-3.5 w-3.5" />{{ notice.status === '已发布' ? '撤回' : '发布' }}</S2VButton>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
