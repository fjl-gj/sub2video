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
