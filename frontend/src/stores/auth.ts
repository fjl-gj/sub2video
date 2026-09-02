import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { loginApi, logoutApi, meApi, registerApi, type AuthUser } from '@/api/auth'
import { clearTokens, getAccessToken } from '@/api/http'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const bootstrapped = ref(false)

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

  async function register(email: string, password: string, displayName: string) {
    const data = await registerApi(email, password, displayName)
    user.value = data.user
    return data.user
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
    isAuthenticated,
    displayName,
    balance,
    bootstrap,
    login,
    register,
    logout,
    setBalance,
  }
})
