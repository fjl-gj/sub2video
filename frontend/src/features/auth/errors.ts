export function authErrorMessage(error: unknown, fallback: string) {
  const response = (error as { response?: { status?: number; data?: { message?: string } } })?.response
  if (!response || (response.status ?? 0) >= 500) return '暂时无法连接服务，请稍后重试。'
  if (response.status === 429) return '操作过于频繁，请稍后重试。'
  return response.data?.message || fallback
}
