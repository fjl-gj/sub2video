<script setup lang="ts">
import { ArrowLeft, Mail } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import { useAuthStore } from '@/stores/auth'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'

const router = useRouter()
const auth = useAuthStore()
const email = ref('')
const sent = ref(false)
const loading = ref(false)
const error = ref('')

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string } }; message?: string }
  return err.response?.data?.message || err.message || fallback
}

async function submit() {
  if (!email.value.trim() || loading.value) return
  error.value = ''
  loading.value = true
  try {
    const settings = auth.publicSettings ?? await auth.fetchPublicSettings()
    if (!settings.passwordResetEnabled) {
      error.value = '密码重置功能暂未开启'
      return
    }
    await auth.forgotPassword(email.value.trim())
    sent.value = true
  } catch (e: unknown) {
    error.value = apiErrorMessage(e, '发送重置邮件失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void auth.fetchPublicSettings().catch(() => undefined)
})
</script>

<template>
  <NoticeScreen title="找回密码">
    <template #description>{{ sent ? '如该邮箱已注册，重置链接已发送，请查收邮件。' : '输入注册邮箱，我们会发送重置链接。' }}</template>
    <div class="grid gap-4 text-left">
      <S2VInput v-model="email" label="邮箱" type="email">
        <template #prefix><Mail class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
      <p v-if="error" class="text-xs text-s2v-danger">{{ error }}</p>
      <S2VButton variant="primary" size="lg" :block="true" :loading="loading" :disabled="!email.trim()" @click="submit">{{ sent ? '重新发送重置邮件' : '发送重置邮件' }}</S2VButton>
    </div>
    <template #footer>
      <button class="flex items-center gap-1 font-semibold text-s2v-muted" type="button" @click="router.push('/auth/login')"><ArrowLeft class="h-3.5 w-3.5" />返回登录</button>
    </template>
  </NoticeScreen>
</template>
