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

export type AdminModel = {
  id: string
  carrierId: string
  code: string
  name: string
  description?: string
  enabled: boolean
  carrierCode?: string
}

export type AdminCarrier = {
  id: string
  code: string
  name: string
  description?: string
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

export async function createAdminCarrier(payload: {
  code: string
  name: string
  description?: string
}) {
  const { data } = await http.post('/admin/carriers', payload)
  return data as AdminCarrier
}

export async function updateAdminCarrier(id: string, payload: {
  name?: string
  description?: string
  enabled?: boolean
}) {
  const { data } = await http.patch(`/admin/carriers/${id}`, payload)
  return data as AdminCarrier
}

export async function createAdminModel(payload: {
  carrierId: string
  code: string
  name: string
  description?: string
}) {
  const { data } = await http.post('/admin/models', payload)
  return data as AdminModel
}

export async function updateAdminModel(id: string, payload: {
  name?: string
  description?: string
  enabled?: boolean
}) {
  const { data } = await http.patch(`/admin/models/${id}`, payload)
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

export type AdminTaskItem = {
  id: string
  title: string
  mode: string
  modeId: string
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

export type AdminCreationMode = {
  id: string
  code: string
  name: string
  description: string
  status: 'published' | 'disabled'
  sortOrder: number
}

export type AdminOffering = {
  id: string
  supplierId: string
  supplierName: string
  modelId: string
  modelName: string
  carrierName: string
  modeId: string
  modeName: string
  name: string
  upstreamWorkflowId: string
  adapterCode: string
  enabled: boolean
  healthStatus: string
  priority: number
  minDurationSec: number
  maxDurationSec: number
  resolutions: string[]
  aspectRatios: string[]
  minImages: number
  maxImages: number
  supportsGeneratedAudio: boolean
}

export type AdminSupplierCost = {
  id: string
  offeringId: string
  resolution: string
  generateAudio: boolean
  peakPerSecond: number
  offpeakPerSecond: number
  currency: string
  version: string
  status: string
}

export type AdminRetailPricing = {
  id: string
  modeId: string
  modeName: string
  modelId: string
  modelName: string
  carrierName: string
  resolution: string
  generateAudio: boolean
  name: string
  pricePerSecond: number
  version: string
  status: string
}

export type AdminModelConfiguration = {
  modelId: string
  modelCode: string
  modelName: string
  carrierId: string
  carrierName: string
  workflowId: string
  workflowName: string
  enabled: boolean
  recommendationRank: number
  displayBadge: string
  defaultDuration: number
  defaultResolution: string
}

export async function fetchAdminCreationModes() {
  const { data } = await http.get('/admin/creation-modes')
  return data as { items: AdminCreationMode[] }
}

export async function updateAdminCreationMode(id: string, payload: Partial<Pick<AdminCreationMode, 'name' | 'description' | 'status' | 'sortOrder'>>) {
  const { data } = await http.patch(`/admin/creation-modes/${id}`, payload)
  return data as AdminCreationMode
}

export async function fetchAdminModelConfigurations() {
  const { data } = await http.get('/admin/model-configurations')
  return data as { items: AdminModelConfiguration[] }
}

export async function upsertAdminModelConfiguration(payload: Pick<AdminModelConfiguration, 'modelId' | 'workflowId' | 'enabled' | 'recommendationRank' | 'displayBadge' | 'defaultDuration' | 'defaultResolution'>) {
  const { data } = await http.put('/admin/model-configurations', payload)
  return data as { ok: boolean }
}

export async function fetchAdminOfferings() {
  const { data } = await http.get('/admin/offerings')
  return data as { items: AdminOffering[] }
}

export async function createAdminOffering(payload: Omit<AdminOffering, 'id' | 'supplierName' | 'modelName' | 'carrierName' | 'modeName'>) {
  const { data } = await http.post('/admin/offerings', payload)
  return data as AdminOffering
}

export async function updateAdminOffering(id: string, payload: Omit<AdminOffering, 'id' | 'supplierName' | 'modelName' | 'carrierName' | 'modeName'>) {
  const { data } = await http.patch(`/admin/offerings/${id}`, payload)
  return data as AdminOffering
}

export async function fetchAdminSupplierCosts() {
  const { data } = await http.get('/admin/supplier-costs')
  return data as { items: AdminSupplierCost[] }
}

export async function upsertAdminSupplierCost(payload: Omit<AdminSupplierCost, 'id' | 'version'>) {
  const { data } = await http.put('/admin/supplier-costs', payload)
  return data as AdminSupplierCost
}

export async function fetchAdminRetailPricing() {
  const { data } = await http.get('/admin/retail-pricing')
  return data as { items: AdminRetailPricing[] }
}

export async function upsertAdminRetailPricing(payload: Pick<AdminRetailPricing, 'modeId' | 'modelId' | 'resolution' | 'generateAudio' | 'name' | 'pricePerSecond' | 'status'>) {
  const { data } = await http.put('/admin/retail-pricing', payload)
  return data as Pick<AdminRetailPricing, 'id' | 'modeId' | 'modelId' | 'pricePerSecond' | 'version' | 'status'>
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

export type AdminAuditLog = {
  id: string
  time: string
  operator: string
  operatorEmail: string
  action: string
  targetType: string
  target: string
  details: Record<string, unknown>
}

export async function fetchAdminAuditLogs(limit = 100) {
  const { data } = await http.get('/admin/audit-logs', { params: { limit } })
  return data as { items: AdminAuditLog[] }
}
