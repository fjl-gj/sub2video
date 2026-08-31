<script setup lang="ts">
import { useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

const router = useRouter()
</script>

<template>
  <NoticeScreen title="页面不存在">
    <template #description>你打开的地址不在 Sub2Video 的产品路由里。</template>
    <p class="text-xs leading-5 text-s2v-muted">请返回首页概览，或从登录页重新进入创作工作流。</p>
    <div class="mt-5 grid gap-2">
      <S2VButton variant="primary" :block="true" @click="router.push('/app/overview')">回到首页</S2VButton>
      <S2VButton :block="true" @click="router.push('/auth/login')">去登录</S2VButton>
    </div>
  </NoticeScreen>
</template>
