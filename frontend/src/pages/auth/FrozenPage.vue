<script setup lang="ts">
import { ShieldBan } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

const router = useRouter()
</script>

<template>
  <NoticeScreen title="账号已冻结">
    <template #description>该账号暂时不能创建任务或使用余额。</template>
    <ShieldBan class="mx-auto h-9 w-9 text-s2v-danger" />
    <S2VBadge class="mt-4" tone="danger">已冻结</S2VBadge>
    <p class="mt-3 text-xs leading-5 text-s2v-muted">如需解冻，请提交工单并说明原因。管理端可在用户管理中查看账户状态。</p>
    <S2VButton class="mt-5" :block="true" @click="router.push('/auth/login')">返回登录</S2VButton>
  </NoticeScreen>
</template>
