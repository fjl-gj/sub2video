import { clearTokens, http, setTokens } from '@/api/http'

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
  const { data } = await http.post<AuthResponse>('/auth/login', { email, password })
  setTokens(data.token.accessToken, data.token.refreshToken)
  return data
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
  const { data } = await http.post<AuthResponse>('/auth/register', { email, password, displayName, verifyCode })
  setTokens(data.token.accessToken, data.token.refreshToken)
  return data
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
  const refreshToken = localStorage.getItem('s2v_refresh_token') || ''
  try {
    await http.post('/auth/logout', { refreshToken })
  } finally {
    clearTokens()
  }
}
