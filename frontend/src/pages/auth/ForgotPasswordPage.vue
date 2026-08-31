<script setup lang="ts">
import { ArrowLeft, Mail } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'

const router = useRouter()
const email = ref('linmo@example.com')
const sent = ref(false)

function submit() {
  sent.value = true
  window.setTimeout(() => router.push('/auth/reset-password'), 600)
}
</script>

<template>
  <NoticeScreen title="找回密码">
    <template #description>输入注册邮箱，我们会发送重置链接。</template>
    <div class="grid gap-4 text-left">
      <S2VInput v-model="email" label="邮箱" type="email">
        <template #prefix><Mail class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
      <S2VButton variant="primary" size="lg" :block="true" :loading="sent" @click="submit">发送重置邮件</S2VButton>
    </div>
    <template #footer>
      <button class="flex items-center gap-1 font-semibold text-s2v-muted" type="button" @click="router.push('/auth/login')"><ArrowLeft class="h-3.5 w-3.5" />返回登录</button>
    </template>
  </NoticeScreen>
</template>
