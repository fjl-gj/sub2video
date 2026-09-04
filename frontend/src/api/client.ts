import { http } from '@/api/http'

export async function fetchOverview() {
  const { data } = await http.get('/client/overview')
  return data
}
export async function fetchWallet() {
  const { data } = await http.get('/client/wallet')
  return data as { balance: number; products: Array<{ id: string; amount: number; bonus: number; price: number }> }
}

export async function fetchWalletTransactions(params?: { page?: number; pageSize?: number }) {
  const { data } = await http.get('/client/wallet/transactions', { params })
  return data as {
    items: Array<{ id: string; type: string; amount: number; desc: string; time: string }>
    total: number
    page: number
    pageSize: number
  }
}

export async function redeemCode(code: string) {
  const { data } = await http.post('/client/wallet/redeem', { code })
  return data as { credited: number; wallet: { balance: number } }
}

export async function quoteTask(payload: {
  workflowId: string
  durationSeconds: number
  resolution?: string
  aspectRatio?: string
  sourceType?: string
}) {
  const { data } = await http.post('/client/tasks/quote', payload)
  return data as {
    estimatedCost: number
    pricePerSecond: number
    priceSnapshot: { ruleId: string; version: string; pricePerSecond: number }
    route?: { id: string; name: string; workflowId: string; costPerSec: number; supplierCode: string }
  }
}

export async function createTask(payload: Record<string, unknown>) {
  const { data } = await http.post('/client/tasks', payload)
  return data as {
    task: { id: string; title: string; status: string; cost: number; workflow: string }
    wallet: { balance: number }
  }
}

export async function uploadImage(file: File) {
  const form = new FormData()
  form.append('file', file)
  const { data } = await http.post('/client/uploads', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  })
  return data as {
    id: string
    fileName: string
    mimeType: string
    sizeBytes: number
    url: string
  }
}

export async function listTasks(params?: { status?: string; page?: number; pageSize?: number }) {
  const { data } = await http.get('/client/tasks', { params })
  return data as {
    items: Array<{
      id: string
      title: string
      workflow: string
      status: string
      progress: number
      duration: string
      resolution: string
      cost: number
      time: string
      thumbUrl: string
      savedToProject?: boolean
    }>
    total: number
    page: number
    pageSize: number
    counts: Record<string, number>
  }
}

export async function cancelTask(id: string) {
  const { data } = await http.post(`/client/tasks/${id}/cancel`)
  return data as {
    task: { id: string; status: string }
    refundAmount: number
    wallet: { balance: number }
  }
}

export async function retryTask(id: string) {
  const { data } = await http.post(`/client/tasks/${id}/retry`)
  return data as { task: { id: string; status: string } }
}

export type ClientTicketItem = {
  id: string
  title: string
  category: string
  status: string
  task: string
  updated: string
}

export async function fetchTickets(params?: { page?: number; pageSize?: number }) {
  const { data } = await http.get('/client/tickets', { params })
  return data as { items: ClientTicketItem[]; total: number; page: number; pageSize: number }
}

export async function createTicket(payload: {
  title: string
  category?: string
  taskId?: string
  detail?: string
}) {
  const { data } = await http.post('/client/tickets', payload)
  return data as { ticket: { id: string; title: string; status: string } }
}

export type ProjectSummary = {
  id: string
  name: string
  desc: string
  createdAt: string
  trashed: boolean
  trashedAt?: string
  isDefault?: boolean
  fileCount: number
  sizeBytes?: number
}

export type ProjectTree = ProjectSummary & {
  folders: Record<string, import('@/data/projects').FsFolder>
}

export type ProjectFileItem = {
  id: string
  name: string
  type: 'image' | 'video' | 'audio'
  size: string
  time: string
  meta?: string
  sourceTask?: string
  url?: string
  remoteUrl?: string
}

export async function fetchStorageUsage() {
  const { data } = await http.get('/client/storage/usage')
  return data as {
    usedGb: number
    quotaGb: number
    fileCount: number
    projectCount: number
    trashCount: number
  }
}

export async function fetchProjects(trashed = false) {
  const { data } = await http.get('/client/projects', { params: { trashed } })
  return data as { items: ProjectSummary[]; currentProjectId: string }
}

export async function createProject(name: string) {
  const { data } = await http.post('/client/projects', { name })
  return data as { project: ProjectSummary }
}

export async function fetchProjectTree(id: string) {
  const { data } = await http.get(`/client/projects/${id}/tree`)
  return data as ProjectTree
}

export async function fetchProjectAssets(id: string, type?: string) {
  const { data } = await http.get(`/client/projects/${id}/assets`, { params: type ? { type } : undefined })
  return data as { items: ProjectFileItem[]; projectId: string }
}

export async function uploadProjectFile(projectId: string, path: string[], file: File) {
  const form = new FormData()
  form.append('file', file)
  const { data } = await http.post(`/client/projects/${projectId}/uploads`, form, {
    params: { path: path.join('/') },
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  })
  return data as { item: ProjectFileItem; path: string[] }
}

export async function deleteProjectFile(projectId: string, fileId: string) {
  const { data } = await http.delete(`/client/projects/${projectId}/files/${fileId}`)
  return data as { ok: boolean }
}

export async function trashProject(id: string) {
  const { data } = await http.post(`/client/projects/${id}/trash`)
  return data as { project: ProjectSummary }
}

export async function restoreProject(id: string) {
  const { data } = await http.post(`/client/projects/${id}/restore`)
  return data as { project: ProjectSummary }
}

export async function purgeProject(id: string) {
  const { data } = await http.delete(`/client/projects/${id}`, { params: { permanent: true } })
  return data as { ok: boolean }
}

export async function setCurrentProject(projectId: string) {
  const { data } = await http.put('/client/session/current-project', { projectId })
  return data as { currentProjectId: string }
}

export async function saveTaskToProject(taskId: string, projectId?: string) {
  const { data } = await http.post(`/client/tasks/${taskId}/save-to-project`, projectId ? { projectId } : {})
  return data as {
    asset: { id: string; name: string; type: string; sourceTask: string; url?: string }
    path: string[]
  }
}
