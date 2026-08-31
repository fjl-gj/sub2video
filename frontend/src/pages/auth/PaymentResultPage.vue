<script setup lang="ts">
import { CheckCircle2 } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

const route = useRoute()
const router = useRouter()
const amount = computed(() => Number(route.query.amount || 0))
</script>

<template>
  <NoticeScreen title="支付结果">
    <template #description>这是原型模拟的支付回跳页，不连接真实支付渠道。</template>
    <CheckCircle2 class="mx-auto h-9 w-9 text-s2v-success" />
    <S2VBadge class="mt-4" tone="success">支付成功</S2VBadge>
    <p class="mt-3 text-xs leading-5 text-s2v-muted">余额已增加 <strong class="s2v-mono text-s2v-ink">¥ {{ amount.toFixed(2) }}</strong>，可立即用于生成任务。</p>
    <S2VButton class="mt-5" variant="primary" :block="true" @click="router.push('/app/wallet')">返回钱包</S2VButton>
  </NoticeScreen>
</template>
