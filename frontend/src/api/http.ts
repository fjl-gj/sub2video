import axios from 'axios'

const ACCESS_KEY = 's2v_access_token'
const REFRESH_KEY = 's2v_refresh_token'

export function getAccessToken() {
  return localStorage.getItem(ACCESS_KEY) || ''
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_KEY) || ''
}

export function setTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(ACCESS_KEY, accessToken)
  localStorage.setItem(REFRESH_KEY, refreshToken)
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_KEY)
  localStorage.removeItem(REFRESH_KEY)
}

export const http = axios.create({
  baseURL: '/api/v1',
  timeout: 20000,
})

http.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => res,
  async (error) => {
    const status = error.response?.status as number | undefined
    const original = error.config as { _retry?: boolean; headers: Record<string, string> } | undefined
    if (status === 401 && original && !original._retry && getRefreshToken()) {
      original._retry = true
      try {
        const { data } = await axios.post('/api/v1/auth/refresh', {
          refreshToken: getRefreshToken(),
        })
        const token = data.token as { accessToken: string; refreshToken: string }
        setTokens(token.accessToken, token.refreshToken)
        original.headers.Authorization = `Bearer ${token.accessToken}`
        return http(original)
      } catch {
        clearTokens()
      }
    }
    return Promise.reject(error)
  },
)
