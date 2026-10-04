<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authReturnPath } from '@/shared/lib/auth-redirect'
import { useAuthStore } from '@/stores/auth'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import { authErrorMessage } from './errors'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const name = ref('')
const email = ref('')
const password = ref('')
const showPassword = ref(false)
const verifyCode = ref('')
const agree = ref(false)
const loading = ref(false)
const sendingCode = ref(false)
const countdown = ref(0)
const error = ref('')
const codeNotice = ref('')
const settingsState = ref<'loading' | 'ready' | 'error'>('loading')
let countdownTimer: ReturnType<typeof window.setInterval> | undefined
let codeRequest = 0

const verificationRequired = computed(() => auth.publicSettings?.emailVerifyEnabled === true)
const registrationOpen = computed(() => settingsState.value === 'ready' && auth.publicSettings?.registrationEnabled === true)
const validEmail = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim()))
const canRegister = computed(() => Boolean(registrationOpen.value && name.value.trim() && validEmail.value && password.value.length >= 8 && agree.value && (!verificationRequired.value || /^\d{6}$/.test(verifyCode.value.trim()))))

async function loadSettings() {
  settingsState.value = 'loading'
  try {
    await auth.fetchPublicSettings()
    settingsState.value = 'ready'
  } catch {
    settingsState.value = 'error'
  }
}

function stopCountdown() {
  if (countdownTimer) window.clearInterval(countdownTimer)
  countdownTimer = undefined
  countdown.value = 0
}

watch(email, () => {
  codeRequest += 1
  stopCountdown()
  sendingCode.value = false
  verifyCode.value = ''
  codeNotice.value = ''
})

async function requestVerifyCode() {
  if (!registrationOpen.value || !verificationRequired.value || !validEmail.value || sendingCode.value || countdown.value > 0) return
  const request = ++codeRequest
  error.value = ''
  codeNotice.value = ''
  sendingCode.value = true
  try {
    const result = await auth.sendVerifyCode(email.value.trim())
    if (request !== codeRequest) return
    codeNotice.value = '验证码已请求，请检查邮箱及垃圾邮件。'
    const until = Date.now() + Math.max(0, result.countdown) * 1000
    countdown.value = Math.max(0, result.countdown)
    countdownTimer = window.setInterval(() => {
      countdown.value = Math.max(0, Math.ceil((until - Date.now()) / 1000))
      if (countdown.value === 0) stopCountdown()
    }, 1000)
  } catch (e: unknown) {
    if (request === codeRequest) error.value = authErrorMessage(e, '验证码发送失败，请稍后重试。')
  } finally {
    if (request === codeRequest) sendingCode.value = false
  }
}

async function register() {
  if (!canRegister.value || loading.value) return
  error.value = ''
  loading.value = true
  try {
    await auth.register(email.value.trim(), password.value, name.value.trim(), verificationRequired.value ? verifyCode.value.trim() : '')
    await router.replace(authReturnPath(route.query.redirect))
  } catch (e: unknown) {
    error.value = authErrorMessage(e, '注册失败，请稍后重试。')
  } finally {
    loading.value = false
  }
}

onMounted(() => { void loadSettings() })
onBeforeUnmount(() => { codeRequest += 1; stopCountdown() })
</script>

<template>
  <div class="s2v-auth-form">
    <header class="s2v-auth-form-header"><p class="s2v-auth-form-eyebrow">每个故事，都值得被看见</p><h1>创建账号</h1><p>从第一镜开始，慢慢拍出你的故事。</p></header>
    <div v-if="settingsState === 'loading'" class="s2v-auth-status" role="status">正在读取注册设置…</div>
    <div v-else-if="settingsState === 'error'" class="s2v-auth-status" role="alert">无法获取注册设置，请重试。<button class="s2v-auth-text-link" type="button" @click="loadSettings">重新加载注册设置</button></div>
    <div v-else-if="!registrationOpen" class="s2v-auth-status" role="status">新账号注册暂未开放，已有账号可以继续登录。</div>
    <form v-if="registrationOpen" class="s2v-auth-fields" :aria-busy="loading" @submit.prevent="register">
      <S2VInput v-model="name" label="昵称" name="displayName" autocomplete="nickname" placeholder="你的创作者昵称" required maxlength="60" :disabled="loading" />
      <S2VInput v-model="email" label="邮箱" type="email" name="email" autocomplete="email" placeholder="name@example.com" required :disabled="loading" />
      <S2VInput v-model="password" label="密码" aria-label="密码" :type="showPassword ? 'text' : 'password'" name="password" autocomplete="new-password" placeholder="至少 8 位" required minlength="8" :disabled="loading">
        <template #suffix><button class="s2v-auth-password-action" type="button" :aria-label="showPassword ? '隐藏密码' : '显示密码'" :aria-pressed="showPassword" @click="showPassword = !showPassword"><EyeOff v-if="showPassword" :size="16" /><Eye v-else :size="16" /></button></template>
      </S2VInput>
      <div v-if="verificationRequired" class="s2v-auth-code">
        <S2VInput v-model="verifyCode" label="邮箱验证码" aria-label="邮箱验证码" name="verifyCode" autocomplete="one-time-code" inputmode="numeric" pattern="[0-9]{6}" maxlength="6" placeholder="请输入 6 位验证码" required :disabled="loading">
          <template #suffix><S2VButton size="sm" variant="ghost" :loading="sendingCode" :disabled="!validEmail || countdown > 0 || loading" @click="requestVerifyCode">{{ countdown > 0 ? `${countdown} 秒后重发` : '发送验证码' }}</S2VButton></template>
        </S2VInput>
        <p v-if="codeNotice" class="s2v-auth-code-help" role="status">{{ codeNotice }}</p>
      </div>
      <label class="s2v-auth-agreement"><input v-model="agree" type="checkbox" required :disabled="loading" aria-label="同意服务条款与隐私政策" /><span>我已阅读并同意 <RouterLink class="s2v-auth-text-link" to="/auth/terms" target="_blank">服务条款</RouterLink> 与 <RouterLink class="s2v-auth-text-link" to="/auth/privacy" target="_blank">隐私政策</RouterLink></span></label>
      <p v-if="error" class="s2v-auth-error" role="alert">{{ error }}</p>
      <S2VButton type="submit" variant="primary" size="lg" :loading="loading" block :disabled="!canRegister">注册并进入</S2VButton>
    </form>
    <div class="s2v-auth-switch"><span>已经有账号？</span><button class="s2v-auth-text-link" type="button" @click="router.push({ path: '/auth/login', query: { redirect: authReturnPath(route.query.redirect) } })">返回登录</button></div>
  </div>
</template>
