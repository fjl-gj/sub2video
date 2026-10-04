import type { createTask } from '@/api/client'

export type PendingCreation = {
  requestId: string
  payload: Parameters<typeof createTask>[0]
}

const key = (userId: string) => `sub2video.pending-creation.${userId}`

// Persist the exact request before sending it. A response lost in transit does
// not establish failure; replaying this request lets the server return its order.
export function loadPendingCreation(userId: string): PendingCreation | null {
  const raw = localStorage.getItem(key(userId))
  if (!raw) return null
  const value = JSON.parse(raw) as PendingCreation
  if (!value.requestId || !value.payload?.modeId || typeof value.payload.prompt !== 'string') {
    throw new Error('无法读取上次提交记录，请先在任务中核对结果，或联系支持。')
  }
  return value
}

export function savePendingCreation(userId: string, value: PendingCreation) {
  localStorage.setItem(key(userId), JSON.stringify(value))
}

export function clearPendingCreation(userId: string, requestId: string) {
  if (loadPendingCreation(userId)?.requestId === requestId) localStorage.removeItem(key(userId))
}
