<script setup lang="ts">
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

const app = useAppStore()
const suppliers = ref<AdminSupplier[]>([])
const carriers = ref<AdminCarrier[]>([])
const models = ref<AdminModel[]>([])
const configurations = ref<AdminModelConfiguration[]>([])
const offerings = ref<AdminOffering[]>([])
const costs = ref<AdminSupplierCost[]>([])
const loading = ref(false)
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

const supplierForm = ref({ code: '', name: '', baseUrl: '', token: '', enabled: true })
const executionForm = ref({
  supplierId: '', name: '', executionType: 'workflow' as 'api' | 'workflow', apiModelCode: '', apiVersion: '', workflowCode: '', upstreamWorkflowId: '', adapterCode: 'autodl_h3_v1', enabled: true,
  healthStatus: 'healthy', priority: '10', minDurationSec: '1', maxDurationSec: '15',
  resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0',
  supportsGeneratedAudio: false, peakPerSecond: '0', offpeakPerSecond: '0',
})

const tabs: ExecutionTab[] = ['供应商', 'API', 'Workflow']
const tabCounts = computed(() => ({
  供应商: suppliers.value.length,
  API: offerings.value.filter((item) => item.executionType === 'api').length,
  Workflow: offerings.value.filter((item) => item.executionType !== 'api').length,
}))
const supplierOptions = computed(() => suppliers.value.filter((item) => item.enabled).map((item) => ({ label: `${item.name} (${item.code})`, value: item.id })))
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
const activeCarrierCount = computed(() => carriers.value.filter((item) => item.enabled).length)
const activeModelCount = computed(() => models.value.filter((item) => item.enabled).length)
const visibleExecutions = computed(() => offerings.value.filter((item) => {
  const typeMatch = activeTab.value === 'API' ? item.executionType === 'api' : activeTab.value === 'Workflow' ? item.executionType !== 'api' : true
  const carrierMatch = !selectedCarrierID.value || carrierForModelID(item.modelId)?.id === selectedCarrierID.value
  const modelMatch = !selectedModelID.value || item.modelId === selectedModelID.value
  return typeMatch && carrierMatch && modelMatch
}))
const pagedSuppliers = computed(() => suppliers.value.slice((supplierPage.value - 1) * pageSize, supplierPage.value * pageSize))
const pagedExecutions = computed(() => visibleExecutions.value.slice((executionPage.value - 1) * pageSize, executionPage.value * pageSize))

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
  return costs.value.filter((item) => item.offeringId === offeringID)
}

function executionLabel(item: AdminOffering) {
  return item.executionType === 'api' ? item.apiModelCode || item.name : item.workflowCode || item.upstreamWorkflowId || item.name
}

async function load() {
  loading.value = true
  try {
    const [supplierResult, carrierResult, configurationResult, offeringResult, costResult] = await Promise.allSettled([
      fetchAdminSuppliers(), fetchAdminCarriers(), fetchAdminModelConfigurations(), fetchAdminOfferings(), fetchAdminSupplierCosts(),
    ])
    if (supplierResult.status === 'fulfilled') suppliers.value = supplierResult.value.items
    else app.notify(apiErrorMessage(supplierResult.reason, '加载供应商目录失败'))
    if (carrierResult.status === 'fulfilled') {
      carriers.value = carrierResult.value.carriers
      models.value = carrierResult.value.models
    } else app.notify(apiErrorMessage(carrierResult.reason, '加载承运商模型目录失败'))
    if (configurationResult.status === 'fulfilled') configurations.value = configurationResult.value.items
    else app.notify(apiErrorMessage(configurationResult.reason, '加载模型配置失败'))
    if (offeringResult.status === 'fulfilled') offerings.value = offeringResult.value.items
    else app.notify(apiErrorMessage(offeringResult.reason, '加载执行配置失败'))
    if (costResult.status === 'fulfilled') costs.value = costResult.value.items
    else app.notify(apiErrorMessage(costResult.reason, '加载采购成本失败'))
  } finally {
    loading.value = false
  }
}

function openSupplierEditor(item?: AdminSupplier) {
  supplierEditingID.value = item?.id || ''
  supplierForm.value = { code: item?.code || '', name: item?.name || '', baseUrl: item?.baseUrl || '', token: '', enabled: item?.enabled ?? true }
  supplierEditor.value = true
}

async function saveSupplier() {
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
    resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0', supportsGeneratedAudio: false, peakPerSecond: '0', offpeakPerSecond: '0',
  }
}

function openExecutionEditor(item?: AdminOffering, type: 'api' | 'workflow' = activeTab.value === 'API' ? 'api' : 'workflow') {
  if (!item) {
    resetExecutionForm(type)
  } else {
    const currentCost = costsFor(item.id).find((cost) => cost.resolution === item.resolutions[0] && !cost.generateAudio) || costsFor(item.id)[0]
    executionEditingID.value = item.id
    selectedConfigurationID.value = configurationKey(item.modelId, item.modeId)
    executionForm.value = {
      supplierId: item.supplierId, name: item.name, executionType: item.executionType || 'workflow', apiModelCode: item.apiModelCode || '', apiVersion: item.apiVersion || '', workflowCode: item.workflowCode || item.upstreamWorkflowId, upstreamWorkflowId: item.upstreamWorkflowId, adapterCode: item.adapterCode, enabled: item.enabled,
      healthStatus: item.healthStatus, priority: String(item.priority), minDurationSec: String(item.minDurationSec), maxDurationSec: String(item.maxDurationSec), resolutions: item.resolutions.join(','), aspectRatios: item.aspectRatios.join(','),
      minImages: String(item.minImages), maxImages: String(item.maxImages), supportsGeneratedAudio: item.supportsGeneratedAudio, peakPerSecond: String(currentCost?.peakPerSecond ?? 0), offpeakPerSecond: String(currentCost?.offpeakPerSecond ?? 0),
    }
  }
  executionEditor.value = true
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
  saving.value = true
  try {
    const offering = executionEditing.value ? await updateAdminOffering(executionEditingID.value, payload) : await createAdminOffering(payload)
    for (const resolution of payload.resolutions) {
      for (const generateAudio of payload.supportsGeneratedAudio ? [false, true] : [false]) {
        await upsertAdminSupplierCost({
          offeringId: offering.id, resolution, generateAudio, peakPerSecond: numberValue(executionForm.value.peakPerSecond), offpeakPerSecond: numberValue(executionForm.value.offpeakPerSecond), currency: 'CNY', status: 'active',
        })
      }
    }
    executionEditor.value = false
    app.notify(`${executionForm.value.executionType === 'api' ? 'API' : 'Workflow'} 执行配置与成本已保存`)
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存执行配置失败'))
  } finally {
    saving.value = false
  }
}

async function toggleExecution(item: AdminOffering) {
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
watch(suppliers, () => { if (supplierPage.value > Math.max(1, Math.ceil(suppliers.value.length / pageSize))) supplierPage.value = 1 })

watch(selectedConfigurationID, () => {
  if (!executionEditing.value && selectedConfiguration.value) executionForm.value.name = `${selectedConfiguration.value.modelName} ${selectedConfiguration.value.workflowName}`
})

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="供应商">
      <template #description>供应商保存调用信息；供应商—承运商—模型路由由 API 与 Workflow 两类执行配置承载，并维护各自采购成本。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton v-if="activeTab === '供应商'" variant="primary" size="sm" @click="openSupplierEditor()"><Plus class="h-3.5 w-3.5" />新增供应商</S2VButton>
        <S2VButton v-else variant="primary" size="sm" @click="openExecutionEditor()"><Plus class="h-3.5 w-3.5" />新增{{ activeTab }}配置</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-5 grid gap-3 sm:grid-cols-3">
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">启用供应商</span><strong class="mt-3 block font-mono text-xl">{{ suppliers.filter((item) => item.enabled).length }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">启用模型配置</span><strong class="mt-3 block font-mono text-xl">{{ configurations.filter((item) => item.enabled).length }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">可用执行配置</span><strong class="mt-3 block font-mono text-xl">{{ offerings.filter((item) => item.enabled).length }}</strong></S2VCard>
    </div>

    <S2VSegmented v-model="activeTab" :options="tabs" :counts="tabCounts" class="mb-5" />

    <S2VDialog v-model:open="supplierEditor" :title="`${supplierEditingID ? '编辑' : '新增'}供应商`" description="调用凭据仅能写入或轮换，不会回显。" hide-actions>
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">{{ supplierEditingID ? '编辑' : '新增' }}供应商</h2><p class="mt-1 text-xs text-s2v-muted">内部编码独立于名称；调用凭据仅能写入或轮换，不会回显。</p></div><S2VButton size="sm" title="关闭供应商编辑器" @click="supplierEditor = false"><X class="h-3.5 w-3.5" /></S2VButton></div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2"><S2VInput v-model="supplierForm.code" label="供应商编码" placeholder="official_minimax" :disabled="Boolean(supplierEditingID)" /><S2VInput v-model="supplierForm.name" label="供应商名称" placeholder="官方 MiniMax" /><S2VInput v-model="supplierForm.baseUrl" label="调用地址" placeholder="https://api.example.com" class="sm:col-span-2" /><S2VInput v-model="supplierForm.token" type="password" :label="supplierEditingID ? '轮换凭据（留空不变）' : '调用凭据'" placeholder="仅写入，不会回显" class="sm:col-span-2" /><label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="supplierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">启用供应商</label></div>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveSupplier">保存供应商</S2VButton></div>
    </S2VDialog>

    <S2VDialog v-model:open="executionEditor" :title="`${executionEditing ? '编辑' : '新增'}${executionForm.executionType === 'api' ? 'API' : 'Workflow'} 执行配置`" description="执行配置是供应商与承运商模型的中间路由记录。" hide-actions size="wide">
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">{{ executionEditing ? '编辑' : '新增' }}{{ executionForm.executionType === 'api' ? 'API' : 'Workflow' }} 执行配置</h2><p class="mt-1 text-xs text-s2v-muted">执行配置是供应商与承运商模型的中间路由记录；一个记录只能属于 API 或 Workflow 之一。</p></div><S2VButton size="sm" title="关闭执行配置编辑器" @click="executionEditor = false"><X class="h-3.5 w-3.5" /></S2VButton></div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2"><S2VSelect v-model="executionForm.supplierId" label="供应商" :options="supplierOptions" :disabled="executionEditing" /><S2VSelect v-model="selectedConfigurationID" label="承运商 / 模型 / 工作流" :options="modelConfigurationOptions" :disabled="executionEditing" /><S2VInput v-model="executionForm.name" label="执行配置名称" /><S2VSelect v-model="executionForm.executionType" label="执行类型" :options="[{ label: 'API', value: 'api' }, { label: 'Workflow', value: 'workflow' }]" :disabled="executionEditing" /><template v-if="executionForm.executionType === 'api'"><S2VInput v-model="executionForm.apiModelCode" label="API 模型编码" placeholder="上游模型编码" /><S2VInput v-model="executionForm.apiVersion" label="API 版本" placeholder="可选" /></template><S2VInput v-else v-model="executionForm.workflowCode" label="Workflow 编码" placeholder="内部工作流编码" /><S2VInput v-model="executionForm.adapterCode" label="执行适配器" /><S2VInput v-model="executionForm.priority" label="路由优先级" type="number" /><S2VInput v-model="executionForm.minDurationSec" label="最小时长（秒）" type="number" /><S2VInput v-model="executionForm.maxDurationSec" label="最大时长（秒）" type="number" /><S2VInput v-model="executionForm.resolutions" label="分辨率（逗号分隔）" /><S2VInput v-model="executionForm.aspectRatios" label="画幅比例（逗号分隔）" /><S2VInput v-model="executionForm.minImages" label="最少图片数" type="number" /><S2VInput v-model="executionForm.maxImages" label="最多图片数" type="number" /><S2VInput v-model="executionForm.peakPerSecond" label="峰时采购成本（元/秒）" type="number" /><S2VInput v-model="executionForm.offpeakPerSecond" label="谷时采购成本（元/秒）" type="number" /><label class="flex items-center gap-2 text-xs"><input v-model="executionForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">参与预估和提交</label><label class="flex items-center gap-2 text-xs"><input v-model="executionForm.supportsGeneratedAudio" type="checkbox" class="h-4 w-4 accent-s2v-primary">支持模型自动生成音频</label></div>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveExecution">保存执行配置</S2VButton></div>
    </S2VDialog>

    <div v-if="activeTab === '供应商'" class="grid gap-4">
      <S2VCard v-for="supplier in pagedSuppliers" :key="supplier.id" variant="padded">
        <div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex items-center gap-2"><h2 class="text-sm font-bold">{{ supplier.name }}</h2><S2VBadge :tone="supplier.enabled ? 'success' : 'neutral'">{{ supplier.enabled ? '启用' : '停用' }}</S2VBadge></div><p class="s2v-mono mt-1 text-[10px] text-s2v-muted">{{ supplier.id }} · {{ supplier.code }}</p><p class="mt-1 text-xs text-s2v-muted">{{ supplier.hasToken ? '调用凭据已配置' : '尚未配置调用凭据' }}</p></div><div class="flex shrink-0 gap-1"><S2VButton size="sm" variant="ghost" title="编辑供应商" @click="openSupplierEditor(supplier)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="supplier.enabled ? '停用供应商' : '启用供应商'" :loading="saving" @click="toggleSupplier(supplier)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div>
        <div class="mt-4 overflow-x-auto border-t border-s2v-line pt-4"><table class="w-full text-left text-xs"><thead class="text-s2v-muted"><tr><th class="pb-2 pr-3 font-medium">执行 ID</th><th class="pb-2 pr-3 font-medium">承运商 / 模型</th><th class="pb-2 pr-3 font-medium">工作流</th><th class="pb-2 pr-3 font-medium">执行类型</th><th class="pb-2 pr-3 font-medium">执行编码</th><th class="pb-2 pr-3 font-medium">状态</th><th class="pb-2 text-right font-medium">操作</th></tr></thead><tbody><tr v-for="item in offerings.filter((offering) => offering.supplierId === supplier.id)" :key="item.id" class="border-t border-s2v-line/80"><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ item.id }}</td><td class="py-3 pr-3"><strong class="font-medium text-s2v-ink">{{ carrierForModelID(item.modelId)?.name || item.carrierName }} / {{ item.modelName }}</strong></td><td class="py-3 pr-3 text-s2v-muted">{{ item.modeName }}</td><td class="py-3 pr-3"><S2VBadge :tone="item.executionType === 'api' ? 'primary' : 'warning'">{{ item.executionType === 'api' ? 'API' : 'Workflow' }}</S2VBadge></td><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ executionLabel(item) }}</td><td class="py-3 pr-3"><S2VBadge :tone="item.enabled ? 'success' : 'neutral'">{{ item.enabled ? '可路由' : '停用' }}</S2VBadge></td><td class="py-3 text-right"><div class="flex justify-end gap-1"><S2VButton size="sm" variant="ghost" title="编辑执行配置" @click="openExecutionEditor(item)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="item.enabled ? '停用执行配置' : '启用执行配置'" :loading="saving" @click="toggleExecution(item)"><Power class="h-3.5 w-3.5" /></S2VButton></div></td></tr><tr v-if="!offerings.some((item) => item.supplierId === supplier.id)"><td colspan="7" class="py-4 text-s2v-muted">该供应商尚未建立承运商模型路由。</td></tr></tbody></table></div>
      </S2VCard>
      <p v-if="!suppliers.length" class="s2v-inset-dashed">暂无供应商。请先新增供应商并配置调用凭据。</p>
      <S2VPagination v-if="suppliers.length" v-model="supplierPage" :total="suppliers.length" :page-size="pageSize" />
    </div>

    <template v-else>
      <div class="mb-5 grid gap-3 sm:grid-cols-2"><S2VSelect v-model="selectedCarrierID" label="筛选承运商" :options="carrierOptions" /><S2VSelect v-model="selectedModelID" label="筛选模型" :options="filteredModelOptions" /></div>
      <div class="grid gap-4 lg:grid-cols-2"><S2VCard v-for="item in pagedExecutions" :key="item.id" variant="padded"><div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex items-center gap-2"><h2 class="truncate text-sm font-bold">{{ item.name }}</h2><S2VBadge :tone="item.enabled ? 'success' : 'neutral'">{{ item.enabled ? '可路由' : '停用' }}</S2VBadge></div><p class="mt-1 text-xs text-s2v-muted">{{ carrierForModelID(item.modelId)?.name || item.carrierName }} / {{ item.modelName }} · {{ item.modeName }}</p></div><div class="flex shrink-0 gap-1"><S2VButton size="sm" variant="ghost" title="编辑执行配置" @click="openExecutionEditor(item)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="item.enabled ? '停用执行配置' : '启用执行配置'" :loading="saving" @click="toggleExecution(item)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div><div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-xs text-s2v-muted"><div class="flex justify-between gap-3"><span>{{ activeTab }} 编码</span><strong class="s2v-mono break-all text-right text-[10px] text-s2v-ink">{{ executionLabel(item) }}</strong></div><div class="flex justify-between gap-3"><span>能力</span><strong class="text-right font-mono text-s2v-ink">{{ item.minDurationSec }}-{{ item.maxDurationSec }} 秒 · {{ item.resolutions.join(', ') }}</strong></div><div class="flex justify-between gap-3"><span>自动音频</span><strong class="text-right text-s2v-ink">{{ item.supportsGeneratedAudio ? '支持' : '不支持' }}</strong></div><div v-for="cost in costsFor(item.id)" :key="cost.id" class="flex justify-between gap-3"><span>成本 {{ cost.resolution }}{{ cost.generateAudio ? ' / 自动音频' : '' }}</span><strong class="font-mono text-right text-s2v-ink">{{ cost.peakPerSecond }} / {{ cost.offpeakPerSecond }} 元/秒</strong></div></div></S2VCard><p v-if="!visibleExecutions.length" class="s2v-inset-dashed lg:col-span-2">当前筛选下没有{{ activeTab }}执行配置。</p></div>
      <S2VPagination v-if="visibleExecutions.length" v-model="executionPage" :total="visibleExecutions.length" :page-size="pageSize" class="mt-4" />
    </template>
  </section>
</template>
