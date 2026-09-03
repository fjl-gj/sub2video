<script setup lang="ts">
import { ArrowLeft, Check, LockKeyhole, Mail, UserRound } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import AuthBrand from '@/widgets/AuthBrand.vue'

const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()
const name = ref('')
const email = ref('')
const password = ref('')
const verifyCode = ref('')
const agree = ref(true)
const loading = ref(false)
const sendingCode = ref(false)
const countdown = ref(0)
const error = ref('')
let countdownTimer: ReturnType<typeof window.setInterval> | undefined

const verificationRequired = computed(() => auth.publicSettings?.emailVerifyEnabled === true)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string } }; message?: string }
  return err.response?.data?.message || err.message || fallback
}

async function loadSettings() {
  try {
    await auth.fetchPublicSettings()
  } catch {
    error.value = '无法获取注册设置，请稍后重试'
  }
}

async function requestVerifyCode() {
  if (!email.value.trim() || sendingCode.value || countdown.value > 0) return
  error.value = ''
  sendingCode.value = true
  try {
    const result = await auth.sendVerifyCode(email.value.trim())
    countdown.value = result.countdown
    countdownTimer = window.setInterval(() => {
      countdown.value = Math.max(0, countdown.value - 1)
      if (countdown.value === 0 && countdownTimer) {
        window.clearInterval(countdownTimer)
        countdownTimer = undefined
      }
    }, 1000)
  } catch (e: unknown) {
    error.value = apiErrorMessage(e, '验证码发送失败，请稍后重试')
  } finally {
    sendingCode.value = false
  }
}

async function register() {
  if (!name.value || !email.value || !password.value || !agree.value || (verificationRequired.value && !verifyCode.value)) return
  error.value = ''
  loading.value = true
  try {
    const user = await auth.register(email.value.trim(), password.value, name.value.trim(), verifyCode.value.trim())
    app.balance = user.balance
    await router.push('/app/overview')
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
    error.value = msg || '注册失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadSettings()
})

onBeforeUnmount(() => {
  if (countdownTimer) window.clearInterval(countdownTimer)
})
</script>

<template>
  <div class="grid gap-5">
    <div class="flex items-center justify-between">
      <button class="flex items-center gap-1 text-xs font-semibold text-s2v-muted" type="button" @click="router.push('/auth/login')"><ArrowLeft class="h-3.5 w-3.5" />返回登录</button>
      <AuthBrand size="sm" />
    </div>
    <div>
      <h1 class="text-xl font-bold">创建账号</h1>
      <p class="mt-1.5 text-xs text-s2v-muted">注册后即可开始视频创作。</p>
    </div>
    <S2VCard variant="padded">
      <div class="grid gap-4">
        <S2VInput v-model="name" label="昵称" placeholder="你的创作者昵称">
          <template #prefix><UserRound class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <S2VInput v-model="email" label="邮箱" type="email" placeholder="name@example.com">
          <template #prefix><Mail class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <S2VInput v-model="password" label="密码" type="password" placeholder="至少 8 位">
          <template #prefix><LockKeyhole class="h-4 w-4 text-s2v-muted" /></template>
        </S2VInput>
        <div v-if="verificationRequired" class="grid gap-2">
          <S2VInput v-model="verifyCode" label="邮箱验证码" placeholder="请输入 6 位验证码">
            <template #prefix><Mail class="h-4 w-4 text-s2v-muted" /></template>
          </S2VInput>
          <S2VButton size="sm" :loading="sendingCode" :disabled="!email.trim() || countdown > 0" @click="requestVerifyCode">
            {{ countdown > 0 ? `${countdown} 秒后重发` : '发送验证码' }}
          </S2VButton>
        </div>
        <p v-if="error" class="text-xs text-s2v-danger">{{ error }}</p>
        <label class="flex items-start gap-2 text-[11px] leading-5 text-s2v-muted">
          <input v-model="agree" type="checkbox" class="mt-1 h-3.5 w-3.5 accent-s2v-primary" />
          <span>我已阅读并同意 <button class="font-semibold text-s2v-primary" type="button" @click="router.push('/auth/terms')">服务条款</button> 与 <button class="font-semibold text-s2v-primary" type="button" @click="router.push('/auth/privacy')">隐私政策</button></span>
        </label>
        <S2VButton variant="primary" size="lg" :loading="loading" :block="true" :disabled="verificationRequired && !verifyCode.trim()" @click="register"><Check class="h-4 w-4" />注册并进入</S2VButton>
      </div>
    </S2VCard>
  </div>
</template>
