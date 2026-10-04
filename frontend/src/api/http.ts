import axios, { CanceledError, type InternalAxiosRequestConfig } from 'axios'
import { getAuthSession, invalidateAuthSession, sameAuthSession, updateAuthTokens, type AuthSession } from '@/shared/lib/auth-session'

export function getAccessToken() { return getAuthSession().accessToken }
export function getRefreshToken() { return getAuthSession().refreshToken }

const REQUEST_TIMEOUT = 20000
type SessionRequest = InternalAxiosRequestConfig & { _authSession?: AuthSession; _authRetried?: boolean }

export const http = axios.create({ baseURL: '/api/v1', timeout: REQUEST_TIMEOUT })

function usesSession(config: InternalAxiosRequestConfig) {
  const path = (config.url || '').split('?')[0]
  return path !== '/settings/public' && (!path.startsWith('/auth/') || path === '/auth/me')
}

function sessionChanged() { return new CanceledError('登录会话已变化，请重试') }

async function performRefresh(expected: AuthSession): Promise<AuthSession> {
  const current = getAuthSession()
  if (current.id !== expected.id) throw sessionChanged()
  if (!sameAuthSession(expected, current)) return current
  try {
    const { data } = await axios.post('/api/v1/auth/refresh', { refreshToken: expected.refreshToken }, { timeout: REQUEST_TIMEOUT })
    if (!updateAuthTokens(expected, data.token, 'renewed')) {
      const latest = getAuthSession()
      if (latest.id !== expected.id) throw sessionChanged()
      return latest
    }
    return getAuthSession()
  } catch (error) {
    const latest = getAuthSession()
    if (latest.id !== expected.id) throw sessionChanged()
    if (!sameAuthSession(expected, latest)) return latest
    if (axios.isAxiosError(error) && (error.response?.status === 401 || error.response?.status === 403)) {
      invalidateAuthSession(expected, 'expired')
    }
    // Network failures and 5xx responses do not prove the session is invalid.
    throw error
  }
}

async function coordinatedRefresh(expected: AuthSession) {
  // Web Locks coordinates rotating tokens across tabs on HTTPS/localhost.
  // In browsers without it, same-tab singleflight and session fences still apply.
  if (!navigator.locks) return performRefresh(expected)
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT)
  try {
    return await navigator.locks.request('s2v-auth-refresh', { signal: controller.signal }, async () => {
      window.clearTimeout(timer)
      return performRefresh(expected)
    })
  } finally { window.clearTimeout(timer) }
}

let refreshing: { session: AuthSession; promise: Promise<AuthSession> } | null = null

function refreshSession(session: AuthSession) {
  if (refreshing && sameAuthSession(refreshing.session, session)) return refreshing.promise
  const pending = { session, promise: coordinatedRefresh(session) }
  refreshing = pending
  const finish = () => { if (refreshing === pending) refreshing = null }
  void pending.promise.then(finish, finish)
  return pending.promise
}

http.interceptors.request.use((config: SessionRequest) => {
  if (!usesSession(config)) return config
  const session = getAuthSession()
  if (config._authSession && config._authSession.id !== session.id) throw sessionChanged()
  config._authSession = session
  if (session.accessToken) config.headers.set('Authorization', `Bearer ${session.accessToken}`)
  else config.headers.delete('Authorization')
  return config
})

http.interceptors.response.use(
  (response) => {
    const request = response.config as SessionRequest
    if (request._authSession && request._authSession.id !== getAuthSession().id) throw sessionChanged()
    return response
  },
  async (error: unknown) => {
    if (!axios.isAxiosError(error)) return Promise.reject(error)
    const original = error.config as SessionRequest | undefined
    if (original?._authSession && original._authSession.id !== getAuthSession().id) return Promise.reject(sessionChanged())
    if (error.response?.status !== 401 || !original?._authSession || !usesSession(original)) return Promise.reject(error)
    const requested = original._authSession
    const current = getAuthSession()
    if (requested.id !== current.id) return Promise.reject(sessionChanged())
    if (original._authRetried) {
      invalidateAuthSession(requested, 'expired')
      return Promise.reject(error)
    }
    original._authRetried = true
    // A late response may have used the previous access token even though
    // another request (or tab) has already completed renewal.
    if (current.accessToken && current.accessToken !== requested.accessToken) return http(original)
    if (!current.refreshToken) {
      invalidateAuthSession(current, 'expired')
      return Promise.reject(error)
    }
    await refreshSession(current)
    if (getAuthSession().id !== requested.id) throw sessionChanged()
    // Replay the original config, preserving data and Idempotency-Key.
    return http(original)
  },
)
