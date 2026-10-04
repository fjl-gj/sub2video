import { fetchTaskDownload } from '@/api/client'
import { publicMediaUrl } from './useTaskDelivery'

// The attachment response is signed separately from the inline preview. The
// server controls Content-Disposition so cross-origin files also download.
export async function downloadTaskFile(taskId: string) {
  const task = await fetchTaskDownload(taskId)
  const url = task.deliveryStatus === 'ready' ? publicMediaUrl(task.downloadUrl) : ''
  if (!url) throw new Error('成片仍在准备交付，请刷新任务状态后重试。')
  const link = document.createElement('a')
  link.href = url
  link.target = '_self'
  link.rel = 'noreferrer'
  link.download = `${task.title || task.id}.mp4`
  document.body.append(link)
  link.click()
  link.remove()
}
