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
  supplierCode: string
  supplierName: string
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
}

export async function fetchAdminSuppliers() {
  const { data } = await http.get('/admin/suppliers')
  return data as { items: AdminSupplier[] }
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

export async function fetchAdminCarriers() {
  const { data } = await http.get('/admin/carriers')
  return data as {
    carriers: Array<{ id: string; code: string; name: string; enabled: boolean }>
    models: Array<{ id: string; code: string; name: string; clientWorkflowId: string; enabled: boolean }>
    cosConfigured: boolean
    autodlEnvToken: boolean
  }
}

export async function fetchAdminSettings() {
  const { data } = await http.get<AdminSettings>('/admin/settings')
  return data
}

export async function updateAdminSettings(payload: Partial<AdminSettings> & { smtpPassword?: string }) {
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

export async function updateAdminWorkflow(id: string, payload: { status: string }) {
  const { data } = await http.patch(`/admin/workflows/${id}`, payload)
  return data as { id: string; code: string; status: string }
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
