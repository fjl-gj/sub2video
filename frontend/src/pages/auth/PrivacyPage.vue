<script setup lang="ts">
import { useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

const router = useRouter()
</script>

<template>
  <NoticeScreen title="隐私政策">
    <template #description>我们只收集完成创作、计费和售后所必需的信息。</template>
    <div class="grid gap-3 text-left text-xs leading-6 text-s2v-muted">
      <p>账号使用邮箱和昵称标识。供应商凭据只保存在管理端，不会下发到客户端。</p>
      <p>任务提示词、上传素材和生成结果用于完成你发起的工作流，并按留存策略删除。</p>
      <p>当前版本暂未开放在线充值支付。创作消费与退款记录用于余额核算和售后处理。</p>
    </div>
    <S2VButton class="mt-5" :block="true" @click="router.push('/auth/login')">返回登录</S2VButton>
  </NoticeScreen>
</template>
