<script setup lang="ts">
import { Megaphone, Plus, Send } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const notices = [
  { title: 'H3 图生视频限时价格活动', desc: '活动期间 768P 价格低至 0.02 元/秒，具体以创作页报价为准。', status: '已发布', date: '今天 09:00' },
  { title: '服务条款更新说明', desc: '我们更新了视频内容使用规范和素材留存策略。', status: '草稿', date: '昨天 16:30' },
]
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Notice" title="公告">
      <template #description>向创作者同步价格、服务状态和规则变更。</template>
      <template #actions>
        <S2VButton variant="primary" @click="app.notify('公告编辑器将在后端接入后开放')"><Plus class="h-3.5 w-3.5" />新建公告</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 lg:grid-cols-2">
      <S2VCard v-for="notice in notices" :key="notice.title" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <span class="grid h-9 w-9 place-items-center rounded-nav bg-s2v-primary-soft text-s2v-primary"><Megaphone class="h-4 w-4" /></span>
          <S2VBadge :tone="notice.status === '已发布' ? 'success' : 'warning'">{{ notice.status }}</S2VBadge>
        </div>
        <h2 class="mt-4 text-sm font-bold">{{ notice.title }}</h2>
        <p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ notice.desc }}</p>
        <div class="mt-4 flex items-center justify-between border-t border-s2v-line pt-3">
          <span class="text-[11px] text-s2v-muted">{{ notice.date }}</span>
          <S2VButton size="sm" @click="app.notify(`已打开「${notice.title}」`)"><Send class="h-3.5 w-3.5" />{{ notice.status === '已发布' ? '查看' : '编辑' }}</S2VButton>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
