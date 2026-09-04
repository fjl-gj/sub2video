import { http } from '@/api/http'

export type AdminSupplier = {
  id: string
  code: string
  name: string
  baseUrl: string
  enabled: boolean
  hasToken: boolean
  status: string
}

export type AdminRoute = {
  id: string
  supplierId?: string
  supplierCode: string
  supplierName: string
  modelId?: string
  modelCode: string
  clientWorkflowId: string
  workflowId: string
  name: string
  enabled: boolean
  costPerSecPeak: number
  costPerSecOffpeak: number
  conditions: {
    max_duration?: number
    resolutions?: string[]
    require_source?: string[]
  }
  priority: number
}

export type AdminSettings = {
  smtpHost: string
  smtpPort: string
  smtpUsername: string
  smtpFrom: string
  smtpFromName: string
  smtpUseTls: boolean
  smtpPasswordConfigured: boolean
  emailVerifyEnabled: boolean
  passwordResetEnabled: boolean
  registrationEnabled: boolean
  frontendUrl: string
  inputRetentionDays: string
  resultRetentionDays: string
  notifyTask: boolean
  auditEnabled: boolean
  storageProvider: string
  cosSecretId: string
  cosBucket: string
  cosRegion: string
  cosUseIntranet: boolean
  cosSecretKeyConfigured: boolean
  minioEndpoint: string
  minioAccessKey: string
  minioBucket: string
  minioUseSsl: boolean
  minioPublicBase: string
  minioSecretKeyConfigured: boolean
  initialBalance: string
  storageQuotaGb: string
}

export async function fetchAdminSuppliers() {
  const { data } = await http.get('/admin/suppliers')
  return data as { items: AdminSupplier[] }
}

export async function createAdminSupplier(payload: {
  code: string
  name: string
  baseUrl?: string
  token?: string
  enabled?: boolean
}) {
  const { data } = await http.post('/admin/suppliers', payload)
  return data as AdminSupplier
}

export async function updateAdminSupplier(id: string, payload: {
  name?: string
  baseUrl?: string
  enabled?: boolean
  token?: string
}) {
  const { data } = await http.patch(`/admin/suppliers/${id}`, payload)
  return data as AdminSupplier
}

export async function fetchAdminRoutes() {
  const { data } = await http.get('/admin/routes')
  return data as { items: AdminRoute[] }
}

export async function createAdminRoute(payload: {
  supplierId: string
  modelId: string
  workflowId: string
  clientWorkflowId?: string
  name?: string
  enabled?: boolean
  costPerSecPeak?: number
  costPerSecOffpeak?: number
  conditions?: {
    max_duration?: number
    resolutions?: string[]
    require_source?: string[]
    aspect_ratios?: string[]
  }
}) {
  const { data } = await http.post('/admin/routes', payload)
  return data as AdminRoute
}

export async function updateAdminRoute(id: string, payload: {
  enabled?: boolean
  costPerSecPeak?: number
  costPerSecOffpeak?: number
  workflowId?: string
  name?: string
}) {
  const { data } = await http.patch(`/admin/routes/${id}`, payload)
  return data as AdminRoute
}

export type AdminModel = {
  id: string
  code: string
  name: string
  clientWorkflowId: string
  enabled: boolean
  carrierCode?: string
}

export type AdminCarrier = {
  id: string
  code: string
  name: string
  enabled: boolean
}

export async function fetchAdminCarriers() {
  const { data } = await http.get('/admin/carriers')
  return data as {
    carriers: AdminCarrier[]
    models: AdminModel[]
    cosConfigured: boolean
    autodlEnvToken: boolean
  }
}

export async function createAdminModel(payload: {
  carrierId: string
  code: string
  name: string
  description?: string
  clientWorkflowId?: string
}) {
  const { data } = await http.post('/admin/models', payload)
  return data as AdminModel
}

export async function fetchAdminSettings() {
  const { data } = await http.get<AdminSettings>('/admin/settings')
  return data
}

export async function updateAdminSettings(
  payload: Partial<AdminSettings> & {
    smtpPassword?: string
    cosSecretKey?: string
    minioSecretKey?: string
  },
) {
  const { data } = await http.put<AdminSettings>('/admin/settings', payload)
  return data
}

export async function testAdminSMTP() {
  const { data } = await http.post<{ status: string }>('/admin/settings/test-smtp')
  return data
}

export type AdminWorkflow = {
  id: string
  code: string
  name: string
  description: string
  status: string
  statusRaw: string
  badge?: string
  modelId?: string
  modelCode: string
  carrier: string
  price: string
}

export type AdminPricingRule = {
  id: string
  name: string
  workflowCode: string
  resolution: string
  cost: string
  price: string
  pricePerSec: number
  duration: string
  status: string
  version: string
}

export type AdminTaskItem = {
  id: string
  title: string
  workflow: string
  workflowId: string
  status: string
  progress: number
  resolution: string
  durationSec: number
  cost: number
  user: string
  userEmail: string
  provider: string
  upstreamTaskId: string
  upstreamStatus: string
  failReason: string
  time: string
}

export async function fetchAdminWorkflows() {
  const { data } = await http.get('/admin/workflows')
  return data as { items: AdminWorkflow[] }
}

export async function createAdminWorkflow(payload: {
  code: string
  name: string
  description?: string
  modelId: string
  status?: string
  badge?: string
}) {
  const { data } = await http.post('/admin/workflows', payload)
  return data as { id: string; code: string; name: string; status: string; modelCode: string }
}

export async function updateAdminWorkflow(id: string, payload: {
  status?: string
  name?: string
  description?: string
  modelId?: string
  badge?: string
}) {
  const { data } = await http.patch(`/admin/workflows/${id}`, payload)
  return data as { id: string; code: string; status: string; name?: string; description?: string; modelCode?: string }
}

export async function fetchAdminPricing() {
  const { data } = await http.get('/admin/pricing')
  return data as { items: AdminPricingRule[] }
}

export async function updateAdminPricing(id: string, payload: { pricePerSec: number }) {
  const { data } = await http.patch(`/admin/pricing/${id}`, payload)
  return data as { id: string; pricePerSec: number; version: string }
}

export async function fetchAdminTasks(params?: { status?: string; page?: number; pageSize?: number }) {
  const { data } = await http.get('/admin/tasks', { params })
  return data as { items: AdminTaskItem[]; total: number; page: number; pageSize: number }
}

export type AdminTaskDetail = AdminTaskItem & {
  pricePerSec: number
  costPerSec: number
  supplierCode: string
  upstreamWorkflowId: string
  resultUrl: string
  resultOssUrl: string
  priceSnapshot: Record<string, unknown>
  routeSnapshot: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export async function fetchAdminTask(id: string) {
  const { data } = await http.get<AdminTaskDetail>(`/admin/tasks/${id}`)
  return data
}

export async function cancelAdminTask(id: string) {
  const { data } = await http.post(`/admin/tasks/${id}/cancel`)
  return data as { task: { id: string; status: string }; refund: number }
}

export async function retryAdminTask(id: string) {
  const { data } = await http.post(`/admin/tasks/${id}/retry`)
  return data as { task: { id: string; status: string } }
}

export type AdminTicketItem = {
  id: string
  title: string
  category: string
  status: string
  task: string
  user: string
  userEmail: string
  updated: string
}

export async function fetchAdminTickets(params?: { page?: number; pageSize?: number }) {
  const { data } = await http.get('/admin/tickets', { params })
  return data as { items: AdminTicketItem[]; total: number }
}

export async function updateAdminTicket(id: string, payload: { status: string }) {
  const { data } = await http.patch(`/admin/tickets/${id}`, payload)
  return data as { id: string; status: string }
}
