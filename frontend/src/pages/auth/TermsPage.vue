<script setup lang="ts">
import { useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'

const router = useRouter()
</script>

<template>
  <NoticeScreen title="服务条款">
    <template #description>使用 Sub2Video 即表示你同意以下创作与结算规则。</template>
    <div class="grid gap-3 text-left text-xs leading-6 text-s2v-muted">
      <p>生成任务在提交时预扣费用。确认失败或成功取消的任务，费用退回创作余额，以任务详情和钱包流水为准。任务进入执行或提交结果待核对后暂不可取消；结果尚未确认时，请勿重复提交。</p>
      <p>你主动保存的资产计入个人空间额度；任务临时文件按系统留存策略清理。</p>
      <p>请勿上传或生成侵犯第三方权利的内容。平台可冻结违规账号。</p>
    </div>
    <S2VButton class="mt-5" :block="true" @click="router.push('/auth/login')">返回登录</S2VButton>
  </NoticeScreen>
</template>
