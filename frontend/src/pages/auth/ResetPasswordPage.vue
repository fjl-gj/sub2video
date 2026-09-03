<script setup lang="ts">
import { ArrowLeft, Check, LockKeyhole, Mail } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import NoticeScreen from '@/features/system/NoticeScreen.vue'
import { useAuthStore } from '@/stores/auth'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const email = ref(typeof route.query.email === 'string' ? route.query.email : '')
const token = typeof route.query.token === 'string' ? route.query.token : ''
const password = ref('')
const confirm = ref('')
const loading = ref(false)
const error = ref('')

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string } }; message?: string }
  return err.response?.data?.message || err.message || fallback
}

async function submit() {
  if (!email.value.trim() || !token || !password.value || password.value !== confirm.value || loading.value) return
  error.value = ''
  loading.value = true
  try {
    await auth.resetPassword(email.value.trim(), token, password.value)
    await router.push('/auth/login')
  } catch (e: unknown) {
    error.value = apiErrorMessage(e, '重置密码失败，请重新获取链接')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <NoticeScreen title="设置新密码">
    <template #description>重置链接有效。请设置至少 8 位的新密码。</template>
    <div class="grid gap-4 text-left">
      <S2VInput v-model="email" label="邮箱" type="email" placeholder="name@example.com">
        <template #prefix><Mail class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
      <S2VInput v-model="password" label="新密码" type="password" placeholder="至少 8 位">
        <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
      <S2VInput v-model="confirm" label="确认密码" type="password">
        <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
      <p v-if="error" class="text-xs text-s2v-danger">{{ error }}</p>
      <p v-else-if="!token" class="text-xs text-s2v-danger">重置链接无效或已过期，请重新获取。</p>
      <S2VButton variant="primary" size="lg" :block="true" :loading="loading" :disabled="!email.trim() || !token || !password || password !== confirm" @click="submit"><Check class="h-4 w-4" />保存并返回登录</S2VButton>
    </div>
    <template #footer>
      <button class="flex items-center gap-1 font-semibold text-s2v-muted" type="button" @click="router.push('/auth/login')"><ArrowLeft class="h-3.5 w-3.5" />返回登录</button>
    </template>
  </NoticeScreen>
</template>
