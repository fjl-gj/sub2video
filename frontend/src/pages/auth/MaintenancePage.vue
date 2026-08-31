<script setup lang="ts">
import { Wrench } from 'lucide-vue-next'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'

</script>

<template>
  <NoticeScreen title="系统维护中">
    <template #description>生成队列和充值暂时不可用，已提交任务不受影响。</template>
    <Wrench class="mx-auto h-9 w-9 text-s2v-warning" />
    <S2VBadge class="mt-4" tone="warning">维护中</S2VBadge>
    <p class="mt-3 text-xs leading-5 text-s2v-muted">请稍后再试。紧急问题可通过已登录会话的工单入口联系我们。</p>
  </NoticeScreen>
</template>
