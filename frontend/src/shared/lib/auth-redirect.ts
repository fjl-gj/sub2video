/** Retain an in-app destination through login without accepting external URLs. */
export function authReturnPath(value: unknown): string {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//') || /[\\\u0000-\u0020]/.test(value)) return '/app/overview'
  try {
    const url = new URL(value, window.location.origin)
    if (url.origin !== window.location.origin || !/^\/(app|admin)(\/|$)/.test(url.pathname)) return '/app/overview'
    return `${url.pathname}${url.search}${url.hash}`
  } catch { return '/app/overview' }
}
