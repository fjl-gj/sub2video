import { defineStore } from 'pinia'
import { computed, onScopeDispose, ref, shallowRef } from 'vue'
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
import { getAuthSession, onAuthSessionChange } from '@/shared/lib/auth-session'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const bootstrapped = ref(false)
  const bootstrapError = ref('')
  const publicSettings = ref<PublicSettings | null>(null)
  const session = shallowRef(getAuthSession())

  const unsubscribe = onAuthSessionChange((next, previous) => {
    session.value = next
    if (next.id !== previous.id || !next.accessToken) {
      user.value = null
      bootstrapError.value = ''
      bootstrapped.value = !next.accessToken
    } else if (!previous.accessToken && next.accessToken && !user.value) {
      // Another tab can finish signing in after this tab already returned to
      // the login page. The next navigation must reload that user's identity.
      bootstrapped.value = false
    }
  })
  onScopeDispose(unsubscribe)

  const isAuthenticated = computed(() => Boolean(user.value && session.value.accessToken))
  const displayName = computed(() => user.value?.workspaceName || user.value?.displayName || '')
  const balance = computed(() => user.value?.balance ?? 0)

  let bootstrapping: { sessionId: string; promise: Promise<void> } | null = null

  async function bootstrap() {
    const expected = getAuthSession()
    if (bootstrapping?.sessionId === expected.id) return bootstrapping.promise
    session.value = expected
    if (!expected.accessToken) {
      user.value = null
      bootstrapped.value = true
      return
    }
    bootstrapError.value = ''
    const promise = (async () => {
      try {
        const nextUser = await meApi()
        const current = getAuthSession()
        if (current.id !== expected.id) return
        session.value = current
        user.value = nextUser
      } catch {
        if (getAuthSession().id === expected.id) {
          // The HTTP layer alone invalidates proven-expired sessions. A
          // temporary outage must not erase tokens or an already loaded user.
          bootstrapError.value = '暂时无法确认登录状态，请重试'
        }
      } finally {
        if (getAuthSession().id === expected.id) bootstrapped.value = true
      }
    })()
    const pending = { sessionId: expected.id, promise }
    bootstrapping = pending
    try { await promise }
    finally { if (bootstrapping === pending) bootstrapping = null }
  }

  function completeAuthentication(data: { user: AuthUser; sessionId: string }) {
    const current = getAuthSession()
    if (current.id !== data.sessionId || !current.accessToken) throw new Error('登录会话已变化，请重试')
    session.value = current
    user.value = data.user
    bootstrapError.value = ''
    bootstrapped.value = true
    return data.user
  }

  async function login(email: string, password: string) {
    const data = await loginApi(email, password)
    return completeAuthentication(data)
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
    return completeAuthentication(data)
  }

  async function forgotPassword(email: string) {
    return forgotPasswordApi(email)
  }

  async function resetPassword(email: string, token: string, newPassword: string) {
    return resetPasswordApi(email, token, newPassword)
  }

  async function logout() {
    try {
      await logoutApi()
      return true
    } catch {
      // Local invalidation already happened. Callers must still leave private
      // pages when server revocation times out or is temporarily unavailable.
      return false
    }
  }

  function setBalance(value: number) {
    if (user.value) user.value = { ...user.value, balance: value }
  }

  return {
    user,
    bootstrapped,
    bootstrapError,
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
