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

export type CreationCatalog = {
  modes: Array<{ id: string; inputType: 'text' | 'first_last' | 'multi_image' | 'unsupported'; name: string; description: string; enabled: boolean }>
  carriers: Array<{ id: string; name: string }>
  models: Array<{ id: string; carrierId: string; name: string; description: string; inputTypes: Array<'text' | 'first_last' | 'multi_image'> }>
}

export type CreationOptions = {
  durations: number[]
  resolutions: string[]
  aspectRatios: string[]
  supportsGeneratedAudio: boolean
  minImages: number
  maxImages: number
  defaultDuration: number
  defaultResolution: string
}

export async function fetchCreationCatalog() {
  const { data } = await http.get('/client/create/catalog')
  return data as CreationCatalog
}

export async function fetchCreationOptions(modeId: string, modelId?: string) {
  const { data } = await http.get('/client/create/options', { params: { modeId, modelId } })
  return data as CreationOptions
}

export async function estimateTask(payload: {
	modeId: string
	carrierId?: string
	modelId?: string
	durationSec: number
	resolution: string
	aspectRatio: string
	generateAudio: boolean
	sourceMeta?: Record<string, unknown>
}) {
	const { data } = await http.post('/client/tasks/estimate', payload)
	return data as {
		available: boolean
		selection: { carrier: { id: string; name: string }; model: { id: string; name: string } }
		billing: { basis: string; durationSec: number; resolution: string; generateAudio: boolean; pricePerSecond: number; estimatedTotal: number; currency: string }
	}
}

export async function createTask(payload: {
	modeId: string
	carrierId?: string
	modelId?: string
	durationSec: number
	resolution: string
	aspectRatio: string
	generateAudio: boolean
	title: string
	prompt: string
	sourceMeta?: Record<string, unknown>
	directorContext?: import('@/api/director').DirectorContext
}, idempotencyKey?: string) {
	const { data } = await http.post('/client/tasks', payload, { headers: idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : undefined })
	return data as {
		task: { id: string; title: string; status: string; cost: number; mode: string }
		order: { id: string; amount: number; currency: string; status: string }
		wallet: { balance: number }
		idempotentReplay?: boolean
	}
}

export async function uploadImage(file: File) {
	return uploadCreationSource('/client/uploads', file)
}

export async function uploadReferenceAudio(file: File) {
	return uploadCreationSource('/client/uploads/reference-audio', file)
}

async function uploadCreationSource(path: string, file: File) {
  const form = new FormData()
  form.append('file', file)
  const { data } = await http.post(path, form, {
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
      mode: string
      status: string
      progress: number
      duration: string
      resolution: string
      cost: number
      time: string
      thumbUrl: string
      savedToProject?: boolean
      canCancel?: boolean
      reviewRequired?: boolean
    }>
    total: number
    page: number
    pageSize: number
    counts: Record<string, number>
  }
}

export type ClientTaskDetail = {
  id: string
  title: string
  mode: string
  modeId?: string
  carrierId?: string
  modelId?: string
  prompt?: string
  sourceMeta?: {
    first?: { type: 'upload' | 'asset'; id: string }
    last?: { type: 'upload' | 'asset'; id: string }
    extras?: Array<{ type: 'upload' | 'asset'; id: string }>
    refs?: Array<{ type: 'upload' | 'asset'; id: string }>
    audio?: { type: 'upload' | 'asset'; id: string }
  }
  status: string
  progress: number
  duration: string
  durationSeconds: number
  resolution: string
  aspectRatio: string
  generateAudio: boolean
  cost: number
  refundAmount: number
  refundedAt?: string | null
  thumbUrl: string
  savedToProject: boolean
  canCancel?: boolean
  reviewRequired?: boolean
  savedDestination?: { projectId: string; folderPath: string[] }
  failureReason: string
  executionWaiting: boolean
  deliveryStatus: 'ready' | 'pending_storage' | 'not_ready'
  downloadUrl: string
  createdAt: string
  updatedAt: string
}

export async function fetchTask(id: string) {
  const { data } = await http.get<ClientTaskDetail>(`/client/tasks/${id}`)
  return data
}

export async function fetchTaskDownload(id: string) {
  const { data } = await http.get<ClientTaskDetail>(`/client/tasks/${id}`, { params: { download: true } })
  return data
}

export async function cancelTask(id: string) {
  const { data } = await http.post(`/client/tasks/${id}/cancel`)
  return data as {
    task: { id: string; status: string }
    refundAmount: number
    wallet: { balance: number }
  }
}

export type ClientTicketItem = {
  id: string
  title: string
  category: string
  status: string
  task: string
  updated: string
  detail?: string
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
