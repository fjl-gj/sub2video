const SESSION_KEY = 's2v_auth_session'
const ACCESS_KEY = 's2v_access_token'
const REFRESH_KEY = 's2v_refresh_token'

type SessionReason = 'initial' | 'authenticating' | 'authenticated' | 'renewed' | 'logout' | 'expired'

export type AuthSession = {
  id: string
  accessToken: string
  refreshToken: string
  reason: SessionReason
}

type SessionListener = (next: AuthSession, previous: AuthSession, source: 'local' | 'storage') => void
const listeners = new Set<SessionListener>()

function sessionId() {
  // getRandomValues also works on the HTTP UAT host, unlike randomUUID.
  return Array.from(crypto.getRandomValues(new Uint32Array(4)), (part) => part.toString(16)).join('-')
}

export function getAuthSession(): AuthSession {
  const stored = localStorage.getItem(SESSION_KEY)
  if (stored) {
    try {
      const session = JSON.parse(stored) as AuthSession
      if (typeof session.id === 'string' && typeof session.accessToken === 'string' && typeof session.refreshToken === 'string') return session
    } catch { /* Recover an invalid local record from the existing token keys. */ }
  }
  // Import existing sessions once. The combined record makes subsequent token
  // rotations atomic for other tabs; the old keys remain compatibility mirrors.
  const session: AuthSession = {
    id: sessionId(),
    accessToken: localStorage.getItem(ACCESS_KEY) || '',
    refreshToken: localStorage.getItem(REFRESH_KEY) || '',
    reason: 'initial',
  }
  localStorage.setItem(SESSION_KEY, JSON.stringify(session))
  return session
}

let observed = getAuthSession()

export function sameAuthSession(left: AuthSession, right: AuthSession) {
  return left.id === right.id && left.accessToken === right.accessToken && left.refreshToken === right.refreshToken
}

function publish(next: AuthSession, source: 'local' | 'storage') {
  if (sameAuthSession(next, observed)) return
  const previous = observed
  observed = next
  for (const listener of listeners) listener(next, previous, source)
}

function writeSession(session: AuthSession) {
  localStorage.setItem(SESSION_KEY, JSON.stringify(session))
  if (session.accessToken) localStorage.setItem(ACCESS_KEY, session.accessToken)
  else localStorage.removeItem(ACCESS_KEY)
  if (session.refreshToken) localStorage.setItem(REFRESH_KEY, session.refreshToken)
  else localStorage.removeItem(REFRESH_KEY)
  publish(session, 'local')
  return session
}

export function beginAuthentication() {
  return writeSession({ id: sessionId(), accessToken: '', refreshToken: '', reason: 'authenticating' })
}

export function updateAuthTokens(expected: AuthSession, tokens: { accessToken: string; refreshToken: string }, reason: 'authenticated' | 'renewed') {
  if (!sameAuthSession(expected, getAuthSession())) return false
  if (!tokens?.accessToken || !tokens?.refreshToken) throw new Error('登录服务返回了无效的会话，请重试')
  writeSession({ ...expected, accessToken: tokens.accessToken, refreshToken: tokens.refreshToken, reason })
  return true
}

export function invalidateAuthSession(expected: AuthSession, reason: 'logout' | 'expired') {
  if (!sameAuthSession(expected, getAuthSession())) return false
  writeSession({ id: sessionId(), accessToken: '', refreshToken: '', reason })
  return true
}

export function onAuthSessionChange(listener: SessionListener) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

window.addEventListener('storage', (event) => {
  if (event.storageArea === localStorage && (event.key === SESSION_KEY || event.key === null)) {
    publish(getAuthSession(), 'storage')
  }
})
