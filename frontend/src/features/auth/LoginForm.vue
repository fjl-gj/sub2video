<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authReturnPath } from '@/shared/lib/auth-redirect'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import { authErrorMessage } from './errors'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const app = useAppStore()
const email = ref('')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')

onMounted(() => { void auth.fetchPublicSettings().catch(() => undefined) })

function retrySession() { window.location.reload() }

async function login() {
  if (loading.value || !email.value.trim() || !password.value) return
  error.value = ''
  loading.value = true
  try {
    const user = await auth.login(email.value.trim(), password.value)
    app.balance = user.balance
    await router.replace(authReturnPath(route.query.redirect))
  } catch (e: unknown) {
    const status = (e as { response?: { status?: number } })?.response?.status
    error.value = status === 401 ? '邮箱或密码不正确，请重新输入。' : authErrorMessage(e, '登录失败，请检查邮箱和密码。')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="s2v-auth-form">
    <header class="s2v-auth-form-header">
      <p class="s2v-auth-form-eyebrow">继续你的下一镜</p>
      <h1>登录 Sub2Video</h1>
      <p>你的灵感、素材和作品，都在这里。</p>
    </header>
    <div v-if="route.query.reason === 'session-expired'" class="s2v-auth-status" role="status">登录已过期，请重新登录后继续。</div>
    <div v-else-if="route.query.reason === 'session-ended'" class="s2v-auth-status" role="status">账号登录状态已在其他页面变更，请重新登录。</div>
    <div v-else-if="route.query.reason === 'session-unavailable'" class="s2v-auth-status" role="status">暂时无法确认登录状态，请重试。<button type="button" class="s2v-auth-text-link" @click="retrySession">重新连接</button></div>
    <form class="s2v-auth-fields" :aria-busy="loading" @submit.prevent="login">
      <S2VInput v-model="email" label="邮箱" type="email" name="email" autocomplete="username" placeholder="name@example.com" required :disabled="loading" />
      <S2VInput v-model="password" label="密码" aria-label="密码" :type="showPassword ? 'text' : 'password'" name="password" autocomplete="current-password" placeholder="输入你的密码" required :disabled="loading">
        <template #suffix><button class="s2v-auth-password-action" type="button" :aria-label="showPassword ? '隐藏密码' : '显示密码'" :aria-pressed="showPassword" @click="showPassword = !showPassword"><EyeOff v-if="showPassword" :size="16" /><Eye v-else :size="16" /></button></template>
      </S2VInput>
      <div v-if="auth.publicSettings?.passwordResetEnabled !== false" class="s2v-auth-forgot"><RouterLink class="s2v-auth-text-link" to="/auth/forgot-password">忘记密码？</RouterLink></div>
      <p v-if="error" class="s2v-auth-error" role="alert">{{ error }}</p>
      <S2VButton type="submit" variant="primary" size="lg" :loading="loading" block>登录</S2VButton>
    </form>
    <p class="s2v-auth-legal">登录即表示你已阅读并同意 <RouterLink class="s2v-auth-text-link" to="/auth/terms">服务条款</RouterLink> 与 <RouterLink class="s2v-auth-text-link" to="/auth/privacy">隐私政策</RouterLink>。</p>
    <div class="s2v-auth-switch">
      <template v-if="auth.publicSettings?.registrationEnabled !== false"><span>第一次来这里？</span><button class="s2v-auth-text-link" type="button" @click="router.push({ path: '/auth/register', query: { redirect: authReturnPath(route.query.redirect) } })">创建新账号</button></template>
      <span v-else>新账号注册暂未开放</span>
    </div>
  </div>
</template>
