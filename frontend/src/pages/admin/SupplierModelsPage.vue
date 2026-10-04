<script setup lang="ts">
import '@/features/admin/catalog.css'
import { Pencil, Plus, Power, RefreshCw, X } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import {
  createAdminOffering,
  createAdminSupplier,
  fetchAdminCarriers,
  fetchAdminModelConfigurations,
  fetchAdminOfferings,
  fetchAdminSupplierCosts,
  fetchAdminSuppliers,
  updateAdminOffering,
  updateAdminSupplier,
  upsertAdminSupplierCost,
  type AdminCarrier,
  type AdminModel,
  type AdminModelConfiguration,
  type AdminOffering,
  type AdminPage,
  type AdminSupplier,
  type AdminSupplierCost,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VSegmented from '@/shared/ui/S2VSegmented.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'

type ExecutionTab = '供应商' | 'API' | 'Workflow'
type CostDraft = { resolution: string; generateAudio: boolean; peak: string; offpeak: string; currency: string; status: string }

const app = useAppStore()
const suppliers = ref<AdminSupplier[]>([])
const carriers = ref<AdminCarrier[]>([])
const models = ref<AdminModel[]>([])
const configurations = ref<AdminModelConfiguration[]>([])
const offerings = ref<AdminOffering[]>([])
const costs = ref<AdminSupplierCost[]>([])
const loading = ref(false)
const loadError = ref('')
const costError = ref('')
const saveExecutionError = ref('')
const editorError = ref('')
const editorLoading = ref(false)
const editorSuppliers = ref<AdminSupplier[]>([])
const costDrafts = ref<CostDraft[]>([])
const costBaseline = new Map<string, string>()
const supplierExecutionPages = ref<Record<string, number>>({})
const supplierExecutionTotals = ref<Record<string, number>>({})
let loadSequence = 0
let editorSequence = 0
let editingExecutionId = ''
const saving = ref(false)
const activeTab = ref<ExecutionTab>('供应商')
const supplierEditor = ref(false)
const supplierEditingID = ref('')
const executionEditor = ref(false)
const executionEditingID = ref('')
const selectedConfigurationID = ref('')
const selectedCarrierID = ref('')
const selectedModelID = ref('')
const supplierPage = ref(1)
const executionPage = ref(1)
const pageSize = 10
const supplierTotal = ref(0)
const executionTotal = ref(0)
const apiTotal = ref(0)
const workflowTotal = ref(0)

const supplierForm = ref({ code: '', name: '', baseUrl: '', token: '', enabled: true })
const executionForm = ref({
  supplierId: '', name: '', executionType: 'workflow' as 'api' | 'workflow', apiModelCode: '', apiVersion: '', workflowCode: '', upstreamWorkflowId: '', adapterCode: 'autodl_h3_v1', enabled: true,
  healthStatus: 'healthy', priority: '10', minDurationSec: '1', maxDurationSec: '15',
  resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0',
  supportsGeneratedAudio: false,
})

const tabs: ExecutionTab[] = ['供应商', 'API', 'Workflow']
const tabCounts = computed(() => ({
  供应商: supplierTotal.value,
  API: apiTotal.value,
  Workflow: workflowTotal.value,
}))
const supplierOptions = computed(() => editorSuppliers.value.filter((item) => item.enabled || item.id === executionForm.value.supplierId).map((item) => ({ label: `${item.name} (${item.code})${item.enabled ? '' : '（已停用）'}`, value: item.id })))
const carrierOptions = computed(() => [{ label: '全部承运商', value: '' }, ...carriers.value.map((item) => ({ label: `${item.name} (${item.code})`, value: item.id }))])
const filteredModelOptions = computed(() => [{ label: '全部模型', value: '' }, ...models.value
  .filter((item) => !selectedCarrierID.value || item.carrierId === selectedCarrierID.value)
  .map((item) => ({ label: `${carrierForModel(item)?.code || item.carrierCode || '-'} / ${item.name} (${item.code})`, value: item.id }))])
const modelConfigurationOptions = computed(() => configurations.value.map((item) => ({
  label: `${carrierForModelID(item.modelId)?.name || item.carrierName} / ${item.modelName} / ${item.workflowName}${item.enabled ? '' : '（模型配置已停用）'}`,
  value: configurationKey(item.modelId, item.workflowId),
})))
const selectedConfiguration = computed(() => configurations.value.find((item) => configurationKey(item.modelId, item.workflowId) === selectedConfigurationID.value))
const executionEditing = computed(() => executionEditingID.value !== '')
const visibleExecutions = computed(() => offerings.value)
const pagedSuppliers = computed(() => suppliers.value)
const pagedExecutions = computed(() => offerings.value)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function configurationKey(modelID: string, workflowID: string) {
  return `${modelID}:${workflowID}`
}

function splitValues(value: string) {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}

function numberValue(value: string, fallback = 0) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function carrierForModel(model: AdminModel) {
  return carriers.value.find((item) => item.id === model.carrierId)
}

function carrierForModelID(modelID: string) {
  const model = models.value.find((item) => item.id === modelID)
  return model ? carrierForModel(model) : undefined
}

function costsFor(offeringID: string) {
  const offering = offerings.value.find((item) => item.id === offeringID)
  return costs.value.filter((item) => (offering?.executionId && item.executionId === offering.executionId) || item.offeringId === offeringID)
}

function executionLabel(item: AdminOffering) {
  return item.executionType === 'api' ? item.apiModelCode || item.name : item.workflowCode || item.upstreamWorkflowId || item.name
}

function executionStatus(item: AdminOffering) {
  if (!item.enabled) return '停用'
  if (item.healthStatus === 'degraded') return '健康降级'
  if (item.healthStatus === 'unavailable') return '健康异常'
  if (['unavailable', 'exhausted'].includes(item.quotaStatus || '')) return '额度不可用'
  if (item.executionType === 'api' && item.adapterCode === 'autodl_h3_v1') return '适配器未开放'
  if (item.healthStatus !== 'healthy') return '健康待确认'
  return '启用'
}

async function fetchAll<T extends { id: string }>(fetchPage: (page: number) => Promise<AdminPage<T>>) {
  const items: T[] = []
  const seen = new Set<string>()
  for (let page = 1; ; page++) {
    const result = await fetchPage(page)
    const fresh = result.items.filter((item) => !seen.has(item.id))
    items.push(...fresh)
    fresh.forEach((item) => seen.add(item.id))
    if (items.length >= (result.total ?? items.length)) return items
    if (!fresh.length) throw new Error('分页数据不完整，请重试加载')
  }
}

function fetchCosts(executionIds: string[]) {
  if (!executionIds.length) return Promise.resolve([] as AdminSupplierCost[])
  return fetchAll((page) => fetchAdminSupplierCosts({ page, pageSize: 100, executionIds: executionIds.join(',') }))
}

async function load() {
  const sequence = ++loadSequence
  loading.value = true
  loadError.value = ''
  costError.value = ''
  try {
    const executionType = activeTab.value === 'API' ? 'api' : activeTab.value === 'Workflow' ? 'workflow' : undefined
    const [supplierResult, carrierResult, configurationResult, apiCountResult, workflowCountResult] = await Promise.all([
      fetchAdminSuppliers({ page: supplierPage.value, pageSize }),
      fetchAdminCarriers(),
      fetchAdminModelConfigurations(),
      fetchAdminOfferings({ page: 1, pageSize: 1, executionType: 'api', carrierId: selectedCarrierID.value || undefined, modelId: selectedModelID.value || undefined }),
      fetchAdminOfferings({ page: 1, pageSize: 1, executionType: 'workflow', carrierId: selectedCarrierID.value || undefined, modelId: selectedModelID.value || undefined }),
    ])
    if (sequence !== loadSequence) return
    suppliers.value = supplierResult.items
    supplierTotal.value = supplierResult.total ?? supplierResult.items.length
    carriers.value = carrierResult.carriers
    models.value = carrierResult.models
    configurations.value = configurationResult.items
    apiTotal.value = apiCountResult.total ?? apiCountResult.items.length
    workflowTotal.value = workflowCountResult.total ?? workflowCountResult.items.length
    if (activeTab.value === '供应商' && supplierPage.value > Math.max(1, Math.ceil(supplierTotal.value / pageSize))) {
      supplierPage.value = 1
      return
    }
    if (activeTab.value === '供应商') {
      const results = await Promise.all(supplierResult.items.map(async (supplier) => ({
        id: supplier.id,
        result: await fetchAdminOfferings({ supplierId: supplier.id, page: supplierExecutionPages.value[supplier.id] || 1, pageSize }),
      })))
      if (sequence !== loadSequence) return
      offerings.value = results.flatMap(({ result }) => result.items)
      results.forEach(({ id, result }) => { supplierExecutionTotals.value[id] = result.total ?? result.items.length })
      costs.value = []
    } else {
      const result = await fetchAdminOfferings({ page: executionPage.value, pageSize, executionType, carrierId: selectedCarrierID.value || undefined, modelId: selectedModelID.value || undefined })
      if (sequence !== loadSequence) return
      offerings.value = result.items
      executionTotal.value = result.total ?? result.items.length
      if (executionPage.value > Math.max(1, Math.ceil(executionTotal.value / pageSize))) {
        executionPage.value = 1
        return
      }
      try {
        const resultCosts = await fetchCosts(result.items.map((item) => item.executionId || item.id))
        if (sequence !== loadSequence) return
        costs.value = resultCosts
      } catch (error) {
        if (sequence !== loadSequence) return
        costs.value = []
        costError.value = apiErrorMessage(error, '加载采购成本失败')
      }
    }
  } catch (error) {
    if (sequence !== loadSequence) return
    loadError.value = apiErrorMessage(error, '加载供应商与执行配置失败')
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

function changeSupplierExecutionPage(id: string, page: number) {
  supplierExecutionPages.value[id] = page
  void load()
}

function openSupplierEditor(item?: AdminSupplier) {
  supplierEditingID.value = item?.id || ''
  supplierForm.value = { code: item?.code || '', name: item?.name || '', baseUrl: item?.baseUrl || '', token: '', enabled: item?.enabled ?? true }
  supplierEditor.value = true
}

async function saveSupplier() {
  if (saving.value) return
  if (!supplierForm.value.name.trim() || (!supplierEditingID.value && !supplierForm.value.code.trim())) {
    app.notify('请填写供应商名称和内部编码')
    return
  }
  saving.value = true
  try {
    if (supplierEditingID.value) {
      await updateAdminSupplier(supplierEditingID.value, {
        name: supplierForm.value.name.trim(), baseUrl: supplierForm.value.baseUrl.trim(), token: supplierForm.value.token.trim() || undefined, enabled: supplierForm.value.enabled,
      })
    } else {
      await createAdminSupplier({
        code: supplierForm.value.code.trim(), name: supplierForm.value.name.trim(), baseUrl: supplierForm.value.baseUrl.trim(), token: supplierForm.value.token.trim(), enabled: supplierForm.value.enabled,
      })
    }
    supplierEditor.value = false
    app.notify('供应商已保存')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存供应商失败'))
  } finally {
    saving.value = false
  }
}

async function toggleSupplier(item: AdminSupplier) {
  if (saving.value) return
  const action = item.enabled ? '停用' : '启用'
  if (!window.confirm(`确认${action}供应商“${item.name}”？`)) return
  saving.value = true
  try {
    await updateAdminSupplier(item.id, { enabled: !item.enabled })
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新供应商状态失败'))
  } finally {
    saving.value = false
  }
}

function resetExecutionForm(type: 'api' | 'workflow') {
  executionEditingID.value = ''
  const configured = configurations.value.find((item) => item.enabled)
  selectedConfigurationID.value = configured ? configurationKey(configured.modelId, configured.workflowId) : ''
  executionForm.value = {
    supplierId: supplierOptions.value[0]?.value || '', name: '', executionType: type, apiModelCode: '', apiVersion: '', workflowCode: '', upstreamWorkflowId: '', adapterCode: 'autodl_h3_v1', enabled: true,
    healthStatus: 'healthy', priority: '10', minDurationSec: '1', maxDurationSec: '15',
    resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0', supportsGeneratedAudio: false,
  }
}

function costKey(row: Pick<CostDraft, 'resolution' | 'generateAudio'>) {
  return `${row.resolution}:${row.generateAudio}`
}

function costValue(row: CostDraft) {
  return JSON.stringify([row.peak.trim() ? Number(row.peak) : null, row.offpeak.trim() ? Number(row.offpeak) : null, row.currency, row.status])
}

function syncCostDrafts() {
  if (!executionEditor.value || editorLoading.value || editorError.value) return
  const previous = new Map(costDrafts.value.map((row) => [costKey(row), row]))
  costDrafts.value = [...new Set(splitValues(executionForm.value.resolutions))].flatMap((resolution) =>
    (executionForm.value.supportsGeneratedAudio ? [false, true] : [false]).map((generateAudio) =>
      previous.get(costKey({ resolution, generateAudio })) || { resolution, generateAudio, peak: '', offpeak: '', currency: 'CNY', status: 'active' }))
}

async function loadEditorData() {
  const sequence = ++editorSequence
  editorLoading.value = true
  editorError.value = ''
  try {
    const [supplierRows, costRows] = await Promise.all([
      fetchAll((page) => fetchAdminSuppliers({ page, pageSize: 100 })),
      editingExecutionId ? fetchCosts([editingExecutionId]) : Promise.resolve([] as AdminSupplierCost[]),
    ])
    if (sequence !== editorSequence || !executionEditor.value) return
    editorSuppliers.value = supplierRows
    if (!executionForm.value.supplierId) executionForm.value.supplierId = supplierOptions.value[0]?.value || ''
    costBaseline.clear()
    costDrafts.value = costRows.map((row) => {
      const draft = { resolution: row.resolution, generateAudio: row.generateAudio, peak: String(row.peakPerSecond), offpeak: String(row.offpeakPerSecond), currency: row.currency, status: row.status }
      costBaseline.set(costKey(draft), costValue(draft))
      return draft
    })
  } catch (error) {
    if (sequence !== editorSequence || !executionEditor.value) return
    editorError.value = apiErrorMessage(error, '加载执行配置的供应商与成本失败，未进行任何保存')
  } finally {
    if (sequence === editorSequence) {
      editorLoading.value = false
      syncCostDrafts()
    }
  }
}

async function openExecutionEditor(item?: AdminOffering, type: 'api' | 'workflow' = activeTab.value === 'API' ? 'api' : 'workflow') {
  costDrafts.value = []
  costBaseline.clear()
  editingExecutionId = item?.executionId || item?.id || ''
  if (!item) {
    resetExecutionForm(type)
  } else {
    executionEditingID.value = item.id
    selectedConfigurationID.value = configurationKey(item.modelId, item.modeId)
    executionForm.value = {
      supplierId: item.supplierId, name: item.name, executionType: item.executionType || 'workflow', apiModelCode: item.apiModelCode || '', apiVersion: item.apiVersion || '', workflowCode: item.workflowCode || item.upstreamWorkflowId || '', upstreamWorkflowId: item.upstreamWorkflowId || '', adapterCode: item.adapterCode, enabled: item.enabled,
      healthStatus: item.healthStatus, priority: String(item.priority), minDurationSec: String(item.minDurationSec), maxDurationSec: String(item.maxDurationSec), resolutions: item.resolutions.join(','), aspectRatios: item.aspectRatios.join(','),
      minImages: String(item.minImages), maxImages: String(item.maxImages), supportsGeneratedAudio: item.supportsGeneratedAudio,
    }
  }
  executionEditor.value = true
  await loadEditorData()
}

function executionPayload(configuration: AdminModelConfiguration) {
  return {
    supplierId: executionForm.value.supplierId, modelId: configuration.modelId, modeId: configuration.workflowId,
    name: executionForm.value.name.trim() || `${configuration.modelName} ${configuration.workflowName}`,
    executionType: executionForm.value.executionType, apiModelCode: executionForm.value.apiModelCode.trim(), apiVersion: executionForm.value.apiVersion.trim(), workflowCode: executionForm.value.workflowCode.trim(), upstreamWorkflowId: executionForm.value.upstreamWorkflowId.trim(), adapterCode: executionForm.value.adapterCode.trim() || 'autodl_h3_v1',
    enabled: executionForm.value.enabled, healthStatus: executionForm.value.healthStatus, priority: numberValue(executionForm.value.priority, 10), minDurationSec: numberValue(executionForm.value.minDurationSec, 1), maxDurationSec: numberValue(executionForm.value.maxDurationSec, 15), resolutions: splitValues(executionForm.value.resolutions), aspectRatios: splitValues(executionForm.value.aspectRatios), minImages: numberValue(executionForm.value.minImages), maxImages: numberValue(executionForm.value.maxImages), supportsGeneratedAudio: executionForm.value.supportsGeneratedAudio,
  }
}

async function saveExecution() {
  if (saving.value || editorLoading.value || editorError.value) return
  saveExecutionError.value = ''
  const configuration = selectedConfiguration.value
  if (!configuration || !executionForm.value.supplierId || (executionForm.value.executionType === 'workflow' ? !executionForm.value.workflowCode.trim() : !executionForm.value.apiModelCode.trim())) {
    app.notify('请选择供应商与模型配置，并填写执行编码')
    return
  }
  if (!executionEditing.value && !configuration.enabled) {
    app.notify('请先启用对应模型配置')
    return
  }
  const payload = executionPayload(configuration)
  if (!payload.resolutions.length || !payload.aspectRatios.length || payload.maxDurationSec < payload.minDurationSec) {
    app.notify('请填写有效的时长范围、分辨率和画幅比例')
    return
  }
  const changedCosts = costDrafts.value.filter((row) => costBaseline.get(costKey(row)) !== costValue(row))
    .filter((row) => costBaseline.has(costKey(row)) || row.peak.trim() || row.offpeak.trim())
  if (changedCosts.some((row) => !row.peak.trim() || !row.offpeak.trim() || !Number.isFinite(Number(row.peak)) || !Number.isFinite(Number(row.offpeak)) || Number(row.peak) < 0 || Number(row.offpeak) < 0)) {
    app.notify('请为每个修改的成本维度填写有效的峰时和谷时金额；留空不会自动设为零')
    return
  }
  saving.value = true
  try {
    const offering = executionEditing.value ? await updateAdminOffering(executionEditingID.value, payload) : await createAdminOffering(payload)
    executionEditingID.value = offering.id
    editingExecutionId = offering.executionId || offering.id
    for (const row of changedCosts) {
      await upsertAdminSupplierCost({
        executionId: editingExecutionId, resolution: row.resolution, generateAudio: row.generateAudio,
        peakPerSecond: Number(row.peak), offpeakPerSecond: Number(row.offpeak), currency: row.currency, status: row.status,
      })
      costBaseline.set(costKey(row), costValue(row))
    }
    executionEditor.value = false
    app.notify(`${executionForm.value.executionType === 'api' ? 'API' : 'Workflow'} 执行配置与成本已保存`)
    await load()
  } catch (error) {
    saveExecutionError.value = apiErrorMessage(error, '保存执行配置或部分成本失败。已保存项会保留，请重试剩余修改')
  } finally {
    saving.value = false
  }
}

async function toggleExecution(item: AdminOffering) {
  if (saving.value) return
  const action = item.enabled ? '停用' : '启用'
  if (!window.confirm(`确认${action}执行配置“${item.name}”？`)) return
  const configuration = configurations.value.find((candidate) => candidate.modelId === item.modelId && candidate.workflowId === item.modeId)
  if (!configuration) {
    app.notify('此执行配置的模型配置已不存在，无法更新')
    return
  }
  saving.value = true
  try {
    await updateAdminOffering(item.id, {
      supplierId: item.supplierId, modelId: item.modelId, modeId: item.modeId, name: item.name, executionType: item.executionType || 'workflow', apiModelCode: item.apiModelCode || '', apiVersion: item.apiVersion || '', workflowCode: item.workflowCode || item.upstreamWorkflowId, upstreamWorkflowId: item.upstreamWorkflowId, adapterCode: item.adapterCode, enabled: !item.enabled,
      healthStatus: item.healthStatus, priority: item.priority, minDurationSec: item.minDurationSec, maxDurationSec: item.maxDurationSec, resolutions: item.resolutions, aspectRatios: item.aspectRatios, minImages: item.minImages, maxImages: item.maxImages, supportsGeneratedAudio: item.supportsGeneratedAudio,
    })
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新执行配置状态失败'))
  } finally {
    saving.value = false
  }
}

watch(selectedCarrierID, () => {
  if (selectedModelID.value && models.value.find((item) => item.id === selectedModelID.value)?.carrierId !== selectedCarrierID.value) selectedModelID.value = ''
  executionPage.value = 1
})
watch(selectedModelID, () => { executionPage.value = 1 })
watch(activeTab, () => { supplierPage.value = 1; executionPage.value = 1 })
watch([selectedCarrierID, selectedModelID, activeTab, supplierPage, executionPage], () => { void load() })

watch(selectedConfigurationID, () => {
  if (!executionEditing.value && selectedConfiguration.value) executionForm.value.name = `${selectedConfiguration.value.modelName} ${selectedConfiguration.value.workflowName}`
})

watch([() => executionForm.value.resolutions, () => executionForm.value.supportsGeneratedAudio], syncCostDrafts)
watch(executionEditor, (open) => {
  if (!open) editorSequence++
  saveExecutionError.value = ''
})

onMounted(load)
</script>

<template>
  <section class="s2v-page s2v-catalog">
    <S2VPageHeader title="供应商">
      <template #description>管理供应商接入、模型关联与执行配置。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton v-if="activeTab === '供应商'" variant="primary" size="sm" @click="openSupplierEditor()"><Plus class="h-3.5 w-3.5" />新增供应商</S2VButton>
        <S2VButton v-else variant="primary" size="sm" @click="openExecutionEditor()"><Plus class="h-3.5 w-3.5" />新增{{ activeTab }}配置</S2VButton>
      </template>
    </S2VPageHeader>

    <S2VSegmented v-model="activeTab" :options="tabs" :counts="tabCounts" class="mb-5" />

    <S2VDialog v-model:open="supplierEditor" :title="`${supplierEditingID ? '编辑' : '新增'}供应商`" description="调用凭据仅能写入或轮换，不会回显。" hide-actions>
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">{{ supplierEditingID ? '编辑' : '新增' }}供应商</h2><p class="mt-1 text-xs text-s2v-muted">内部编码独立于名称；调用凭据仅能写入或轮换，不会回显。</p></div><S2VButton size="sm" title="关闭供应商编辑器" @click="supplierEditor = false"><X class="h-3.5 w-3.5" /></S2VButton></div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2"><S2VInput v-model="supplierForm.code" label="供应商编码" placeholder="official_minimax" :disabled="Boolean(supplierEditingID)" /><S2VInput v-model="supplierForm.name" label="供应商名称" placeholder="官方 MiniMax" /><S2VInput v-model="supplierForm.baseUrl" label="调用地址" placeholder="https://api.example.com" class="sm:col-span-2" /><S2VInput v-model="supplierForm.token" type="password" :label="supplierEditingID ? '轮换凭据（留空不变）' : '调用凭据'" placeholder="仅写入，不会回显" class="sm:col-span-2" /><label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="supplierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">启用供应商</label></div>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveSupplier">保存供应商</S2VButton></div>
    </S2VDialog>

    <S2VDialog v-model:open="executionEditor" :title="`${executionEditing ? '编辑' : '新增'}${executionForm.executionType === 'api' ? 'API' : 'Workflow'} 执行配置`" description="执行配置是供应商与承运商模型的中间路由记录。" hide-actions size="wide">
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">{{ executionEditing ? '编辑' : '新增' }}{{ executionForm.executionType === 'api' ? 'API' : 'Workflow' }} 执行配置</h2><p class="mt-1 text-xs text-s2v-muted">执行配置是供应商与承运商模型的中间路由记录；一个记录只能属于 API 或 Workflow 之一。</p></div><S2VButton size="sm" title="关闭执行配置编辑器" @click="executionEditor = false"><X class="h-3.5 w-3.5" /></S2VButton></div>
      <p v-if="editorLoading" role="status" class="s2v-note">正在加载供应商与完整成本…</p>
      <div v-else-if="editorError" role="alert" class="s2v-note grid gap-3"><p>{{ editorError }}</p><S2VButton @click="loadEditorData">重试加载配置</S2VButton></div>
      <template v-else>
        <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
          <S2VSelect v-model="executionForm.supplierId" label="供应商" :options="supplierOptions" :disabled="executionEditing" />
          <S2VSelect v-model="selectedConfigurationID" label="承运商 / 模型 / 工作流" :options="modelConfigurationOptions" :disabled="executionEditing" />
          <S2VInput v-model="executionForm.name" label="执行配置名称" />
          <S2VSelect v-model="executionForm.executionType" label="执行类型" :options="[{ label: 'API', value: 'api' }, { label: 'Workflow', value: 'workflow' }]" :disabled="executionEditing" />
          <template v-if="executionForm.executionType === 'api'"><S2VInput v-model="executionForm.apiModelCode" label="API 模型编码" placeholder="上游模型编码" /><S2VInput v-model="executionForm.apiVersion" label="API 版本" placeholder="可选" /></template>
          <S2VInput v-else v-model="executionForm.workflowCode" label="Workflow 编码" placeholder="内部工作流编码" />
          <S2VInput v-model="executionForm.adapterCode" label="执行适配器" />
          <S2VInput v-model="executionForm.priority" label="路由优先级" type="number" />
          <S2VInput v-model="executionForm.minDurationSec" label="最小时长（秒）" type="number" />
          <S2VInput v-model="executionForm.maxDurationSec" label="最大时长（秒）" type="number" />
          <S2VInput v-model="executionForm.resolutions" label="分辨率（逗号分隔）" />
          <S2VInput v-model="executionForm.aspectRatios" label="画幅比例（逗号分隔）" />
          <S2VInput v-model="executionForm.minImages" label="最少图片数" type="number" />
          <S2VInput v-model="executionForm.maxImages" label="最多图片数" type="number" />
          <label class="flex items-center gap-2 text-xs"><input v-model="executionForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">参与预估和提交</label>
          <label class="flex items-center gap-2 text-xs"><input v-model="executionForm.supportsGeneratedAudio" type="checkbox" class="h-4 w-4 accent-s2v-primary">支持模型自动生成音频</label>
        </div>
        <section class="mt-5 grid gap-3" aria-label="逐维度采购成本">
          <h3 class="s2v-section-title">逐维度采购成本</h3>
          <p class="text-xs text-s2v-muted">每种分辨率与音频组合单独维护，只保存修改的成本。未配置的组合留空，不会按零成本路由。</p>
          <div v-for="row in costDrafts" :key="costKey(row)" class="grid gap-3 sm:grid-cols-2">
            <S2VInput v-model="row.peak" :label="`${row.resolution} / ${row.generateAudio ? '自动音频' : '无自动音频'} 峰时成本`" type="number" min="0" step="0.001" placeholder="未配置" />
            <S2VInput v-model="row.offpeak" :label="`${row.resolution} / ${row.generateAudio ? '自动音频' : '无自动音频'} 谷时成本`" type="number" min="0" step="0.001" placeholder="未配置" />
          </div>
        </section>
      </template>
      <p v-if="saveExecutionError" role="alert" class="s2v-note mt-4">{{ saveExecutionError }}</p>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" :disabled="editorLoading || Boolean(editorError)" @click="saveExecution">保存执行配置</S2VButton></div>
    </S2VDialog>

    <p v-if="loading" role="status" class="s2v-note">正在加载供应商与执行配置…</p>
    <div v-else-if="loadError" role="alert" class="s2v-note grid gap-3"><p>{{ loadError }}</p><S2VButton @click="load">重试加载</S2VButton></div>
    <div v-else-if="activeTab === '供应商'" class="grid gap-4">
      <S2VCard v-for="supplier in pagedSuppliers" :key="supplier.id" variant="padded">
        <div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex items-center gap-2"><h2 class="text-sm font-bold">{{ supplier.name }}</h2><S2VBadge :tone="supplier.enabled ? 'success' : 'neutral'">{{ supplier.enabled ? '启用' : '停用' }}</S2VBadge></div><p class="s2v-mono mt-1 text-[10px] text-s2v-muted">{{ supplier.id }} · {{ supplier.code }}</p><p class="mt-1 text-xs text-s2v-muted">{{ supplier.hasToken ? '调用凭据已配置' : '尚未配置调用凭据' }}</p></div><div class="flex shrink-0 gap-1"><S2VButton size="sm" variant="ghost" title="编辑供应商" @click="openSupplierEditor(supplier)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="supplier.enabled ? '停用供应商' : '启用供应商'" :loading="saving" @click="toggleSupplier(supplier)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div>
        <div class="mt-4 overflow-x-auto border-t border-s2v-line pt-4"><table class="w-full text-left text-xs"><thead class="text-s2v-muted"><tr><th class="pb-2 pr-3 font-medium">执行 ID</th><th class="pb-2 pr-3 font-medium">承运商 / 模型</th><th class="pb-2 pr-3 font-medium">工作流</th><th class="pb-2 pr-3 font-medium">执行类型</th><th class="pb-2 pr-3 font-medium">执行编码</th><th class="pb-2 pr-3 font-medium">状态</th><th class="pb-2 text-right font-medium">操作</th></tr></thead><tbody><tr v-for="item in offerings.filter((offering) => offering.supplierId === supplier.id)" :key="item.id" class="border-t border-s2v-line/80"><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ item.id }}</td><td class="py-3 pr-3"><strong class="font-medium text-s2v-ink">{{ carrierForModelID(item.modelId)?.name || item.carrierName }} / {{ item.modelName }}</strong></td><td class="py-3 pr-3 text-s2v-muted">{{ item.modeName }}</td><td class="py-3 pr-3"><S2VBadge :tone="item.executionType === 'api' ? 'primary' : 'warning'">{{ item.executionType === 'api' ? 'API' : 'Workflow' }}</S2VBadge></td><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ executionLabel(item) }}</td><td class="py-3 pr-3"><S2VBadge :tone="executionStatus(item) === '启用' ? 'success' : item.enabled ? 'warning' : 'neutral'">{{ executionStatus(item) }}</S2VBadge></td><td class="py-3 text-right"><div class="flex justify-end gap-1"><S2VButton size="sm" variant="ghost" title="编辑执行配置" @click="openExecutionEditor(item)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="item.enabled ? '停用执行配置' : '启用执行配置'" :loading="saving" @click="toggleExecution(item)"><Power class="h-3.5 w-3.5" /></S2VButton></div></td></tr><tr v-if="!offerings.some((item) => item.supplierId === supplier.id)"><td colspan="7" class="py-4 text-s2v-muted">该供应商尚未建立承运商模型路由。</td></tr></tbody></table></div>
        <S2VPagination v-if="supplierExecutionTotals[supplier.id]" :model-value="supplierExecutionPages[supplier.id] || 1" :total="supplierExecutionTotals[supplier.id] || 0" :page-size="pageSize" class="mt-3" @update:model-value="changeSupplierExecutionPage(supplier.id, $event)" />
      </S2VCard>
      <p v-if="!suppliers.length" class="s2v-inset-dashed">暂无供应商。请先新增供应商并配置调用凭据。</p>
      <S2VPagination v-if="supplierTotal" v-model="supplierPage" :total="supplierTotal" :page-size="pageSize" />
    </div>

    <template v-else>
      <div class="mb-5 grid gap-3 sm:grid-cols-2"><S2VSelect v-model="selectedCarrierID" label="筛选承运商" :options="carrierOptions" /><S2VSelect v-model="selectedModelID" label="筛选模型" :options="filteredModelOptions" /></div>
      <div v-if="costError" role="alert" class="s2v-note mb-4 grid gap-3"><p>采购成本加载失败：{{ costError }}。此时不显示成本数据。</p><S2VButton @click="load">重试加载成本</S2VButton></div>
      <div class="grid gap-4 lg:grid-cols-2"><S2VCard v-for="item in pagedExecutions" :key="item.id" variant="padded"><div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex items-center gap-2"><h2 class="truncate text-sm font-bold">{{ item.name }}</h2><S2VBadge :tone="executionStatus(item) === '启用' ? 'success' : item.enabled ? 'warning' : 'neutral'">{{ executionStatus(item) }}</S2VBadge></div><p class="mt-1 text-xs text-s2v-muted">{{ carrierForModelID(item.modelId)?.name || item.carrierName }} / {{ item.modelName }} · {{ item.modeName }}</p></div><div class="flex shrink-0 gap-1"><S2VButton size="sm" variant="ghost" title="编辑执行配置" @click="openExecutionEditor(item)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="item.enabled ? '停用执行配置' : '启用执行配置'" :loading="saving" @click="toggleExecution(item)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div><div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-xs text-s2v-muted"><div class="flex justify-between gap-3"><span>{{ activeTab }} 编码</span><strong class="s2v-mono break-all text-right text-[10px] text-s2v-ink">{{ executionLabel(item) }}</strong></div><div class="flex justify-between gap-3"><span>能力</span><strong class="text-right font-mono text-s2v-ink">{{ item.minDurationSec }}-{{ item.maxDurationSec }} 秒 · {{ item.resolutions.join(', ') }}</strong></div><div class="flex justify-between gap-3"><span>自动音频</span><strong class="text-right text-s2v-ink">{{ item.supportsGeneratedAudio ? '支持' : '不支持' }}</strong></div><div v-for="cost in costsFor(item.id)" :key="cost.id" class="flex justify-between gap-3"><span>成本 {{ cost.resolution }}{{ cost.generateAudio ? ' / 自动音频' : '' }}</span><strong class="font-mono text-right text-s2v-ink">{{ cost.peakPerSecond }} / {{ cost.offpeakPerSecond }} 元/秒</strong></div></div></S2VCard><p v-if="!visibleExecutions.length" class="s2v-inset-dashed lg:col-span-2">当前筛选下没有{{ activeTab }}执行配置。</p></div>
      <S2VPagination v-if="executionTotal" v-model="executionPage" :total="executionTotal" :page-size="pageSize" class="mt-4" />
    </template>
  </section>
</template>
