<script setup lang="ts">
import { Bot, Pencil, Plus, Power, RefreshCw, X } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import {
  createAdminCarrier,
  createAdminModel,
  createAdminOffering,
  createAdminSupplier,
  fetchAdminCarriers,
  fetchAdminCreationModes,
  fetchAdminOfferings,
  fetchAdminSupplierCosts,
  fetchAdminSuppliers,
  updateAdminCarrier,
  updateAdminModel,
  updateAdminOffering,
  updateAdminSupplier,
  upsertAdminSupplierCost,
  type AdminCarrier,
  type AdminCreationMode,
  type AdminModel,
  type AdminOffering,
  type AdminSupplier,
  type AdminSupplierCost,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'

const app = useAppStore()
const loading = ref(false)
const saving = ref(false)
const catalogEditor = ref<'carrier' | 'model' | 'supplier' | null>(null)
const catalogEditingID = ref('')
const showOfferingForm = ref(false)
const offeringEditingID = ref('')
const suppliers = ref<AdminSupplier[]>([])
const carriers = ref<AdminCarrier[]>([])
const models = ref<AdminModel[]>([])
const modes = ref<AdminCreationMode[]>([])
const offerings = ref<AdminOffering[]>([])
const costs = ref<AdminSupplierCost[]>([])

const carrierForm = ref({ code: '', name: '', description: '', enabled: true })
const modelForm = ref({ carrierId: '', code: '', name: '', description: '', enabled: true })
const supplierForm = ref({ code: '', name: '', baseUrl: '', token: '', enabled: true })
const offeringForm = ref({
  supplierId: '', modelId: '', modeId: '', name: '', executionType: 'workflow' as 'api' | 'workflow', apiModelCode: '', apiVersion: '', workflowCode: '', upstreamWorkflowId: '', adapterCode: 'autodl_h3_v1',
  enabled: true, healthStatus: 'healthy', priority: '10', minDurationSec: '1', maxDurationSec: '15',
  resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0',
  supportsGeneratedAudio: false, peakPerSecond: '0.03', offpeakPerSecond: '0.02',
})

const enabledModels = computed(() => models.value.filter((item) => item.enabled))
const carrierOptions = computed(() => carriers.value.filter((item) => item.enabled).map((item) => ({ label: `${item.name} (${item.code})`, value: item.id })))
const supplierOptions = computed(() => suppliers.value.filter((item) => item.enabled).map((item) => ({ label: `${item.name} (${item.code})`, value: item.id })))
const modelOptions = computed(() => enabledModels.value.map((item) => ({ label: `${item.carrierCode} / ${item.name}`, value: item.id })))
const modeOptions = computed(() => modes.value.filter((item) => item.status === 'published').map((item) => ({ label: item.name, value: item.id })))
const editingOffering = computed(() => offeringEditingID.value !== '')

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function splitValues(value: string) {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}

function toNumber(value: string, fallback = 0) {
  const result = Number(value)
  return Number.isFinite(result) ? result : fallback
}

function resetOfferingForm() {
  offeringEditingID.value = ''
  offeringForm.value = {
    supplierId: supplierOptions.value[0]?.value || '', modelId: modelOptions.value[0]?.value || '', modeId: modeOptions.value[0]?.value || '',
    name: '', executionType: 'workflow', apiModelCode: '', apiVersion: '', workflowCode: '', upstreamWorkflowId: '', adapterCode: 'autodl_h3_v1', enabled: true, healthStatus: 'healthy', priority: '10',
    minDurationSec: '1', maxDurationSec: '15', resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0',
    supportsGeneratedAudio: false, peakPerSecond: '0.03', offpeakPerSecond: '0.02',
  }
}

function openOfferingForm(item?: AdminOffering) {
  if (!item) {
    resetOfferingForm()
  } else {
    const currentCost = costs.value.find((cost) => cost.offeringId === item.id)
    offeringEditingID.value = item.id
    offeringForm.value = {
      supplierId: item.supplierId, modelId: item.modelId, modeId: item.modeId, name: item.name,
      executionType: item.executionType || 'workflow', apiModelCode: item.apiModelCode || '', apiVersion: item.apiVersion || '', workflowCode: item.workflowCode || item.upstreamWorkflowId, upstreamWorkflowId: item.upstreamWorkflowId, adapterCode: item.adapterCode, enabled: item.enabled,
      healthStatus: item.healthStatus, priority: String(item.priority), minDurationSec: String(item.minDurationSec),
      maxDurationSec: String(item.maxDurationSec), resolutions: item.resolutions.join(','), aspectRatios: item.aspectRatios.join(','),
      minImages: String(item.minImages), maxImages: String(item.maxImages), supportsGeneratedAudio: item.supportsGeneratedAudio,
      peakPerSecond: String(currentCost?.peakPerSecond ?? 0), offpeakPerSecond: String(currentCost?.offpeakPerSecond ?? 0),
    }
  }
  showOfferingForm.value = true
}

function openCatalogEditor(kind: 'carrier' | 'model' | 'supplier', item?: AdminCarrier | AdminModel | AdminSupplier) {
  catalogEditor.value = kind
  catalogEditingID.value = item?.id || ''
  if (kind === 'carrier') {
    const carrier = item as AdminCarrier | undefined
    carrierForm.value = { code: carrier?.code || '', name: carrier?.name || '', description: carrier?.description || '', enabled: carrier?.enabled ?? true }
  }
  if (kind === 'model') {
    const model = item as AdminModel | undefined
    modelForm.value = { carrierId: model?.carrierId || carrierOptions.value[0]?.value || '', code: model?.code || '', name: model?.name || '', description: model?.description || '', enabled: model?.enabled ?? true }
  }
  if (kind === 'supplier') {
    const supplier = item as AdminSupplier | undefined
    supplierForm.value = { code: supplier?.code || '', name: supplier?.name || '', baseUrl: supplier?.baseUrl || '', token: '', enabled: supplier?.enabled ?? true }
  }
}

function closeCatalogEditor() {
  catalogEditor.value = null
  catalogEditingID.value = ''
}

async function load() {
  loading.value = true
  try {
    const [supplierData, carrierData, modeData, offeringData, costData] = await Promise.all([
      fetchAdminSuppliers(), fetchAdminCarriers(), fetchAdminCreationModes(), fetchAdminOfferings(), fetchAdminSupplierCosts(),
    ])
    suppliers.value = supplierData.items
    carriers.value = carrierData.carriers
    models.value = carrierData.models
    modes.value = modeData.items
    offerings.value = offeringData.items
    costs.value = costData.items
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载供应商供给失败'))
  } finally {
    loading.value = false
  }
}

async function saveCatalog() {
  if (!catalogEditor.value) return
  saving.value = true
  try {
    if (catalogEditor.value === 'carrier') {
      if (catalogEditingID.value) await updateAdminCarrier(catalogEditingID.value, { name: carrierForm.value.name, description: carrierForm.value.description, enabled: carrierForm.value.enabled })
      else await createAdminCarrier({ code: carrierForm.value.code, name: carrierForm.value.name, description: carrierForm.value.description })
    }
    if (catalogEditor.value === 'model') {
      if (catalogEditingID.value) await updateAdminModel(catalogEditingID.value, { name: modelForm.value.name, description: modelForm.value.description, enabled: modelForm.value.enabled })
      else await createAdminModel({ carrierId: modelForm.value.carrierId, code: modelForm.value.code, name: modelForm.value.name, description: modelForm.value.description })
    }
    if (catalogEditor.value === 'supplier') {
      if (catalogEditingID.value) await updateAdminSupplier(catalogEditingID.value, { name: supplierForm.value.name, baseUrl: supplierForm.value.baseUrl, token: supplierForm.value.token || undefined, enabled: supplierForm.value.enabled })
      else await createAdminSupplier({ code: supplierForm.value.code, name: supplierForm.value.name, baseUrl: supplierForm.value.baseUrl, token: supplierForm.value.token, enabled: supplierForm.value.enabled })
    }
    app.notify('目录配置已保存')
    closeCatalogEditor()
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存目录配置失败'))
  } finally {
    saving.value = false
  }
}

async function toggleCarrier(item: AdminCarrier) {
  saving.value = true
  try {
    await updateAdminCarrier(item.id, { enabled: !item.enabled })
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新承运商失败'))
  } finally {
    saving.value = false
  }
}

async function toggleModel(item: AdminModel) {
  saving.value = true
  try {
    await updateAdminModel(item.id, { enabled: !item.enabled })
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新模型失败'))
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
    app.notify(apiErrorMessage(error, '更新供应商失败'))
  } finally {
    saving.value = false
  }
}

function offeringPayload() {
  return {
    supplierId: offeringForm.value.supplierId,
    modelId: offeringForm.value.modelId,
    modeId: offeringForm.value.modeId,
    name: offeringForm.value.name.trim() || '未命名供给',
    executionType: offeringForm.value.executionType,
    apiModelCode: offeringForm.value.apiModelCode.trim(),
    apiVersion: offeringForm.value.apiVersion.trim(),
    workflowCode: offeringForm.value.workflowCode.trim(),
    upstreamWorkflowId: offeringForm.value.upstreamWorkflowId.trim(),
    adapterCode: offeringForm.value.adapterCode.trim() || 'autodl_h3_v1',
    enabled: offeringForm.value.enabled,
    healthStatus: offeringForm.value.healthStatus || 'healthy',
    priority: toNumber(offeringForm.value.priority, 10),
    minDurationSec: toNumber(offeringForm.value.minDurationSec, 1),
    maxDurationSec: toNumber(offeringForm.value.maxDurationSec, 15),
    resolutions: splitValues(offeringForm.value.resolutions),
    aspectRatios: splitValues(offeringForm.value.aspectRatios),
    minImages: toNumber(offeringForm.value.minImages),
    maxImages: toNumber(offeringForm.value.maxImages),
    supportsGeneratedAudio: offeringForm.value.supportsGeneratedAudio,
  }
}

async function saveOffering() {
  if (!offeringForm.value.supplierId || !offeringForm.value.modelId || !offeringForm.value.modeId || (offeringForm.value.executionType === 'workflow' ? !offeringForm.value.workflowCode.trim() : !offeringForm.value.apiModelCode.trim())) {
    app.notify('请选择供应商、模型和创作模式，并填写执行配置')
    return
  }
  const payload = offeringPayload()
  if (payload.maxDurationSec < payload.minDurationSec || !payload.resolutions.length || !payload.aspectRatios.length) {
    app.notify('请填写有效的时长范围、分辨率和画幅比例')
    return
  }
  saving.value = true
  try {
    const offering = offeringEditingID.value
      ? await updateAdminOffering(offeringEditingID.value, payload)
      : await createAdminOffering(payload)
    for (const resolution of payload.resolutions) {
      for (const generateAudio of payload.supportsGeneratedAudio ? [false, true] : [false]) {
        await upsertAdminSupplierCost({
          offeringId: offering.id,
          resolution,
          generateAudio,
          peakPerSecond: toNumber(offeringForm.value.peakPerSecond),
          offpeakPerSecond: toNumber(offeringForm.value.offpeakPerSecond),
          currency: 'CNY',
          status: 'active',
        })
      }
    }
    showOfferingForm.value = false
    app.notify('供应商供给和采购成本已保存')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存供应商供给失败'))
  } finally {
    saving.value = false
  }
}

function costsFor(offeringId: string) {
  return costs.value.filter((item) => item.offeringId === offeringId)
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="模型与供应商">
      <template #description>承运商和模型描述原厂能力；供应商供给描述实际采购渠道、上游工作流与成本。客户端永不看到此处的供应商或成本信息。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton variant="primary" size="sm" @click="openOfferingForm()"><Plus class="h-3.5 w-3.5" />新建供给</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-5 grid gap-3 sm:grid-cols-3">
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">启用供应商</span><strong class="mt-3 block font-mono text-xl">{{ suppliers.filter((item) => item.enabled).length }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">承运商 / 模型</span><strong class="mt-3 block font-mono text-xl">{{ carriers.length }} / {{ models.length }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">可用供给</span><strong class="mt-3 block font-mono text-xl">{{ offerings.filter((item) => item.enabled).length }}</strong></S2VCard>
    </div>

    <S2VCard class="mb-5" variant="padded">
      <div class="s2v-panel-header">
        <div><h2 class="s2v-section-title">承运商、模型与供应商目录</h2><p class="mt-1 text-xs text-s2v-muted">新增后还需配置模式策略、供给能力、采购成本与对客售价，才会对客户端可用。</p></div>
        <div class="flex flex-wrap gap-2">
          <S2VButton size="sm" @click="openCatalogEditor('carrier')"><Plus class="h-3.5 w-3.5" />承运商</S2VButton>
          <S2VButton size="sm" @click="openCatalogEditor('model')"><Plus class="h-3.5 w-3.5" />模型</S2VButton>
          <S2VButton size="sm" @click="openCatalogEditor('supplier')"><Plus class="h-3.5 w-3.5" />供应商</S2VButton>
        </div>
      </div>

      <div v-if="catalogEditor" class="border-t border-s2v-line pt-4">
        <div class="mb-3 flex items-center justify-between gap-3">
          <strong class="text-xs">{{ catalogEditingID ? '编辑' : '新建' }}{{ catalogEditor === 'carrier' ? '承运商' : catalogEditor === 'model' ? '模型' : '供应商' }}</strong>
          <S2VButton size="sm" title="关闭编辑器" @click="closeCatalogEditor"><X class="h-3.5 w-3.5" /></S2VButton>
        </div>
        <div v-if="catalogEditor === 'carrier'" class="grid gap-3 sm:grid-cols-2">
          <S2VInput v-model="carrierForm.code" label="承运商编码" placeholder="volcengine" :disabled="Boolean(catalogEditingID)" />
          <S2VInput v-model="carrierForm.name" label="承运商名称" placeholder="火山引擎" />
          <S2VInput v-model="carrierForm.description" label="说明" class="sm:col-span-2" />
          <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="carrierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">允许新建模型与客户端展示</label>
        </div>
        <div v-else-if="catalogEditor === 'model'" class="grid gap-3 sm:grid-cols-2">
          <S2VSelect v-model="modelForm.carrierId" label="承运商" :options="carrierOptions" :disabled="Boolean(catalogEditingID)" />
          <S2VInput v-model="modelForm.code" label="模型编码" placeholder="seedance_2_5" :disabled="Boolean(catalogEditingID)" />
          <S2VInput v-model="modelForm.name" label="模型名称" placeholder="Seedance 2.5" />
          <S2VInput v-model="modelForm.description" label="说明" />
          <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="modelForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">允许新建供给与客户端展示</label>
        </div>
        <div v-else class="grid gap-3 sm:grid-cols-2">
          <S2VInput v-model="supplierForm.code" label="供应商编码" placeholder="official_minimax" :disabled="Boolean(catalogEditingID)" />
          <S2VInput v-model="supplierForm.name" label="供应商名称" placeholder="官方 MiniMax" />
          <S2VInput v-model="supplierForm.baseUrl" label="上游服务地址" placeholder="https://api.example.com" class="sm:col-span-2" />
          <S2VInput v-model="supplierForm.token" type="password" :label="catalogEditingID ? '更新凭据（留空则不变）' : '凭据'" placeholder="仅写入，不会回显" class="sm:col-span-2" />
          <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="supplierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">允许参与预估和提交</label>
        </div>
        <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveCatalog">保存目录配置</S2VButton></div>
      </div>

      <div v-else class="mt-4 grid gap-4 lg:grid-cols-3">
        <div>
          <strong class="mb-2 block text-xs">承运商</strong>
          <div class="grid gap-2"><div v-for="carrier in carriers" :key="carrier.id" class="s2v-inset"><div class="flex items-start justify-between gap-2"><div class="min-w-0"><strong class="block truncate text-xs">{{ carrier.name }}</strong><span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ carrier.code }}</span></div><S2VBadge :tone="carrier.enabled ? 'success' : 'neutral'">{{ carrier.enabled ? '启用' : '停用' }}</S2VBadge></div><div class="mt-2 flex gap-1"><S2VButton size="sm" variant="ghost" title="编辑承运商" @click="openCatalogEditor('carrier', carrier)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="carrier.enabled ? '停用承运商' : '启用承运商'" :loading="saving" @click="toggleCarrier(carrier)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div></div>
        </div>
        <div>
          <strong class="mb-2 block text-xs">模型</strong>
          <div class="grid gap-2"><div v-for="model in models" :key="model.id" class="s2v-inset"><div class="flex items-start justify-between gap-2"><div class="min-w-0"><strong class="block truncate text-xs">{{ model.name }}</strong><span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ model.carrierCode }} · {{ model.code }}</span></div><S2VBadge :tone="model.enabled ? 'success' : 'neutral'">{{ model.enabled ? '启用' : '停用' }}</S2VBadge></div><div class="mt-2 flex gap-1"><S2VButton size="sm" variant="ghost" title="编辑模型" @click="openCatalogEditor('model', model)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="model.enabled ? '停用模型' : '启用模型'" :loading="saving" @click="toggleModel(model)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div></div>
        </div>
        <div>
          <strong class="mb-2 block text-xs">供应商</strong>
          <div class="grid gap-2"><div v-for="supplier in suppliers" :key="supplier.id" class="s2v-inset"><div class="flex items-start justify-between gap-2"><div class="min-w-0"><strong class="block truncate text-xs">{{ supplier.name }}</strong><span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ supplier.code }} · {{ supplier.hasToken ? '凭据已配置' : '未配置凭据' }}</span></div><S2VBadge :tone="supplier.enabled ? 'success' : 'neutral'">{{ supplier.enabled ? '启用' : '停用' }}</S2VBadge></div><div class="mt-2 flex gap-1"><S2VButton size="sm" variant="ghost" title="编辑供应商" @click="openCatalogEditor('supplier', supplier)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="supplier.enabled ? '停用供应商' : '启用供应商'" :loading="saving" @click="toggleSupplier(supplier)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div></div>
        </div>
      </div>
    </S2VCard>

    <S2VCard v-if="showOfferingForm" class="mb-5">
      <div class="s2v-panel-header">
        <div><h2 class="s2v-section-title">{{ editingOffering ? '编辑' : '新建' }}供应商供给</h2><p class="mt-1 text-xs text-s2v-muted">供给绑定供应商、模型和固定创作模式；成本归属供给而非模型。</p></div>
        <S2VButton size="sm" title="关闭编辑器" @click="showOfferingForm = false"><X class="h-3.5 w-3.5" /></S2VButton>
      </div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VSelect v-model="offeringForm.supplierId" label="供应商" :options="supplierOptions" :disabled="editingOffering" />
        <S2VSelect v-model="offeringForm.modelId" label="模型" :options="modelOptions" :disabled="editingOffering" />
        <S2VSelect v-model="offeringForm.modeId" label="创作模式" :options="modeOptions" :disabled="editingOffering" />
        <S2VInput v-model="offeringForm.name" label="供给名称" placeholder="H3 文生视频" />
        <S2VSelect v-model="offeringForm.executionType" label="执行类型" :options="[{ label: 'API', value: 'api' }, { label: 'Workflow', value: 'workflow' }]" />
        <S2VInput v-if="offeringForm.executionType === 'workflow'" v-model="offeringForm.workflowCode" label="工作流编码" placeholder="内部执行编码" class="sm:col-span-2" />
        <template v-else>
          <S2VInput v-model="offeringForm.apiModelCode" label="API 模型编码" placeholder="上游模型编码" />
          <S2VInput v-model="offeringForm.apiVersion" label="API 版本" placeholder="可选" />
        </template>
        <S2VInput v-model="offeringForm.adapterCode" label="适配器" />
        <S2VInput v-model="offeringForm.priority" label="同成本优先级" type="number" />
        <S2VInput v-model="offeringForm.resolutions" label="标准分辨率（逗号分隔）" />
        <S2VInput v-model="offeringForm.aspectRatios" label="画幅比例（逗号分隔）" />
        <S2VInput v-model="offeringForm.minDurationSec" label="最小时长（秒）" type="number" />
        <S2VInput v-model="offeringForm.maxDurationSec" label="最大时长（秒）" type="number" />
        <S2VInput v-model="offeringForm.minImages" label="最少参考图" type="number" />
        <S2VInput v-model="offeringForm.maxImages" label="最多参考图" type="number" />
        <S2VInput v-model="offeringForm.peakPerSecond" label="峰时采购成本（元/秒）" type="number" />
        <S2VInput v-model="offeringForm.offpeakPerSecond" label="谷时采购成本（元/秒）" type="number" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="offeringForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">参与预估和提交</label>
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="offeringForm.supportsGeneratedAudio" type="checkbox" class="h-4 w-4 accent-s2v-primary">支持模型自动生成音频</label>
        <div class="sm:col-span-2"><S2VButton variant="primary" :loading="saving" @click="saveOffering">保存供给</S2VButton></div>
      </div>
    </S2VCard>

    <div class="grid gap-4 lg:grid-cols-2">
      <S2VCard v-for="offering in offerings" :key="offering.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-2"><span class="s2v-icon-well"><Bot class="h-4 w-4" /></span><div class="min-w-0"><strong class="block truncate text-xs">{{ offering.name }}</strong><span class="mt-1 block truncate text-[11px] text-s2v-muted">{{ offering.supplierName }} · {{ offering.carrierName }} / {{ offering.modelName }}</span></div></div>
          <div class="flex items-center gap-1"><S2VBadge :tone="offering.enabled ? 'success' : 'neutral'">{{ offering.enabled ? '启用' : '停用' }}</S2VBadge><S2VButton size="sm" variant="ghost" title="编辑供给" @click="openOfferingForm(offering)"><Pencil class="h-3.5 w-3.5" /></S2VButton></div>
        </div>
        <div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-xs text-s2v-muted">
          <div class="flex justify-between gap-3"><span>创作模式</span><strong class="text-right text-s2v-ink">{{ offering.modeName }}</strong></div>
          <div class="flex justify-between gap-3"><span>能力</span><strong class="text-right font-mono text-s2v-ink">{{ offering.minDurationSec }}-{{ offering.maxDurationSec }} 秒 · {{ offering.resolutions.join(', ') }}</strong></div>
          <div class="flex justify-between gap-3"><span>自动音频</span><strong class="text-right text-s2v-ink">{{ offering.supportsGeneratedAudio ? '支持' : '不支持' }}</strong></div>
          <div v-for="cost in costsFor(offering.id)" :key="cost.id" class="flex justify-between gap-3"><span>采购成本 {{ cost.resolution }}</span><strong class="text-right font-mono text-s2v-ink">{{ cost.peakPerSecond }} / {{ cost.offpeakPerSecond }} 元/秒</strong></div>
        </div>
      </S2VCard>
      <p v-if="!offerings.length" class="s2v-inset-dashed lg:col-span-2">暂无供应商供给。先配置供应商、承运商和模型，再新增供给。</p>
    </div>
  </section>
</template>
