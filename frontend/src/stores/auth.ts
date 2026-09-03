import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  fetchPublicSettings as fetchPublicSettingsApi,
  forgotPassword as forgotPasswordApi,
  loginApi,
  logoutApi,
  meApi,
  registerApi,
  resetPassword as resetPasswordApi,
  sendVerifyCode as sendVerifyCodeApi,
  type AuthUser,
  type PublicSettings,
} from '@/api/auth'
import { clearTokens, getAccessToken } from '@/api/http'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const bootstrapped = ref(false)
  const publicSettings = ref<PublicSettings | null>(null)

  const isAuthenticated = computed(() => Boolean(user.value && getAccessToken()))
  const displayName = computed(() => user.value?.workspaceName || user.value?.displayName || '')
  const balance = computed(() => user.value?.balance ?? 0)

  async function bootstrap() {
    if (!getAccessToken()) {
      user.value = null
      bootstrapped.value = true
      return
    }
    try {
      user.value = await meApi()
    } catch {
      clearTokens()
      user.value = null
    } finally {
      bootstrapped.value = true
    }
  }

  async function login(email: string, password: string) {
    const data = await loginApi(email, password)
    user.value = data.user
    return data.user
  }

  async function fetchPublicSettings() {
    publicSettings.value = await fetchPublicSettingsApi()
    return publicSettings.value
  }

  async function sendVerifyCode(email: string) {
    return sendVerifyCodeApi(email)
  }

  async function register(email: string, password: string, displayName: string, verifyCode = '') {
    const data = await registerApi(email, password, displayName, verifyCode)
    user.value = data.user
    return data.user
  }

  async function forgotPassword(email: string) {
    return forgotPasswordApi(email)
  }

  async function resetPassword(email: string, token: string, newPassword: string) {
    return resetPasswordApi(email, token, newPassword)
  }

  async function logout() {
    await logoutApi()
    user.value = null
  }

  function setBalance(value: number) {
    if (user.value) user.value = { ...user.value, balance: value }
  }

  return {
    user,
    bootstrapped,
    publicSettings,
    isAuthenticated,
    displayName,
    balance,
    bootstrap,
    login,
    fetchPublicSettings,
    sendVerifyCode,
    register,
    forgotPassword,
    resetPassword,
    logout,
    setBalance,
  }
})
