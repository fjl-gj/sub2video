import { CanceledError } from 'axios'
import { http } from '@/api/http'
import { beginAuthentication, getAuthSession, invalidateAuthSession, updateAuthTokens } from '@/shared/lib/auth-session'

export type AuthUser = {
  id: string
  email: string
  displayName: string
  role: string
  workspaceName: string
  balance: number
}

export type TokenPair = {
  accessToken: string
  refreshToken: string
  expiresIn: number
  tokenType: string
}

export type AuthResponse = {
  token: TokenPair
  user: AuthUser
}

export type PublicSettings = {
  emailVerifyEnabled: boolean
  passwordResetEnabled: boolean
  registrationEnabled: boolean
}

export async function loginApi(email: string, password: string) {
  const session = beginAuthentication()
  const { data } = await http.post<AuthResponse>('/auth/login', { email, password })
  if (!updateAuthTokens(session, data.token, 'authenticated')) throw new CanceledError('登录会话已变化，请重试')
  return { ...data, sessionId: session.id }
}

export async function fetchPublicSettings() {
  const { data } = await http.get<PublicSettings>('/settings/public')
  return data
}

export async function sendVerifyCode(email: string) {
  const { data } = await http.post<{ countdown: number }>('/auth/send-verify-code', { email })
  return data
}

export async function registerApi(email: string, password: string, displayName: string, verifyCode = '') {
  const session = beginAuthentication()
  const { data } = await http.post<AuthResponse>('/auth/register', { email, password, displayName, verifyCode })
  if (!updateAuthTokens(session, data.token, 'authenticated')) throw new CanceledError('登录会话已变化，请重试')
  return { ...data, sessionId: session.id }
}

export async function forgotPassword(email: string) {
  const { data } = await http.post<{ status: string }>('/auth/forgot-password', { email })
  return data
}

export async function resetPassword(email: string, token: string, newPassword: string) {
  const { data } = await http.post<{ status: string }>('/auth/reset-password', { email, token, newPassword })
  return data
}

export async function meApi() {
  const { data } = await http.get<AuthUser>('/auth/me')
  return data
}

export async function logoutApi() {
  const session = getAuthSession()
  // End the local session before waiting for the server. A late refresh or
  // logout response must never restore/clear a subsequent login.
  invalidateAuthSession(session, 'logout')
  if (session.refreshToken) await http.post('/auth/logout', { refreshToken: session.refreshToken })
}
