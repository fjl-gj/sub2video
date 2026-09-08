<script setup lang="ts">
import { Bot, Pencil, Plus, Power, RefreshCw, X } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import {
  createAdminOffering,
  createAdminSupplier,
  fetchAdminModelConfigurations,
  fetchAdminOfferings,
  fetchAdminSupplierCosts,
  fetchAdminSuppliers,
  updateAdminOffering,
  updateAdminSupplier,
  upsertAdminSupplierCost,
  type AdminModelConfiguration,
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
const suppliers = ref<AdminSupplier[]>([])
const configurations = ref<AdminModelConfiguration[]>([])
const offerings = ref<AdminOffering[]>([])
const costs = ref<AdminSupplierCost[]>([])
const loading = ref(false)
const saving = ref(false)
const supplierEditor = ref(false)
const supplierEditingID = ref('')
const executionEditor = ref(false)
const executionEditingID = ref('')
const selectedConfigurationID = ref('')

const supplierForm = ref({ code: '', name: '', baseUrl: '', token: '', enabled: true })
const executionForm = ref({
  supplierId: '', name: '', upstreamWorkflowId: '', adapterCode: 'autodl_h3_v1', enabled: true,
  healthStatus: 'healthy', priority: '10', minDurationSec: '1', maxDurationSec: '15',
  resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0',
  supportsGeneratedAudio: false, peakPerSecond: '0', offpeakPerSecond: '0',
})

const supplierOptions = computed(() => suppliers.value.filter((item) => item.enabled).map((item) => ({ label: item.name, value: item.id })))
const modelConfigurationOptions = computed(() => configurations.value.map((item) => ({
  label: `${item.carrierName} / ${item.modelName} - ${item.workflowName}${item.enabled ? '' : '（模型配置已停用）'}`,
  value: configurationKey(item.modelId, item.workflowId),
})))
const selectedConfiguration = computed(() => configurations.value.find((item) => configurationKey(item.modelId, item.workflowId) === selectedConfigurationID.value))
const executionEditing = computed(() => executionEditingID.value !== '')

function configurationKey(modelID: string, workflowID: string) {
  return `${modelID}:${workflowID}`
}

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function splitValues(value: string) {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}

function numberValue(value: string, fallback = 0) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function costsFor(offeringID: string) {
  return costs.value.filter((item) => item.offeringId === offeringID)
}

async function load() {
  loading.value = true
  try {
    const [supplierData, configurationData, offeringData, costData] = await Promise.all([
      fetchAdminSuppliers(), fetchAdminModelConfigurations(), fetchAdminOfferings(), fetchAdminSupplierCosts(),
    ])
    suppliers.value = supplierData.items
    configurations.value = configurationData.items
    offerings.value = offeringData.items
    costs.value = costData.items
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载供应商与模型失败'))
  } finally {
    loading.value = false
  }
}

function openSupplierEditor(item?: AdminSupplier) {
  supplierEditingID.value = item?.id || ''
  supplierForm.value = {
    code: item?.code || '', name: item?.name || '', baseUrl: item?.baseUrl || '', token: '', enabled: item?.enabled ?? true,
  }
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
        name: supplierForm.value.name.trim(), baseUrl: supplierForm.value.baseUrl.trim(),
        token: supplierForm.value.token.trim() || undefined, enabled: supplierForm.value.enabled,
      })
    } else {
      await createAdminSupplier({
        code: supplierForm.value.code.trim(), name: supplierForm.value.name.trim(), baseUrl: supplierForm.value.baseUrl.trim(),
        token: supplierForm.value.token.trim(), enabled: supplierForm.value.enabled,
      })
    }
    supplierEditor.value = false
    app.notify('供应商调用配置已保存')
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

function resetExecutionForm() {
  executionEditingID.value = ''
  selectedConfigurationID.value = configurations.value.find((item) => item.enabled)
    ? configurationKey(configurations.value.find((item) => item.enabled)!.modelId, configurations.value.find((item) => item.enabled)!.workflowId)
    : ''
  executionForm.value = {
    supplierId: supplierOptions.value[0]?.value || '', name: '', upstreamWorkflowId: '', adapterCode: 'autodl_h3_v1', enabled: true,
    healthStatus: 'healthy', priority: '10', minDurationSec: '1', maxDurationSec: '15',
    resolutions: '768P', aspectRatios: '16:9,9:16,1:1', minImages: '0', maxImages: '0',
    supportsGeneratedAudio: false, peakPerSecond: '0', offpeakPerSecond: '0',
  }
}

function openExecutionEditor(item?: AdminOffering) {
  if (!item) {
    resetExecutionForm()
  } else {
    const currentCost = costsFor(item.id).find((cost) => cost.resolution === item.resolutions[0] && !cost.generateAudio) || costsFor(item.id)[0]
    executionEditingID.value = item.id
    selectedConfigurationID.value = configurationKey(item.modelId, item.modeId)
    executionForm.value = {
      supplierId: item.supplierId, name: item.name, upstreamWorkflowId: item.upstreamWorkflowId, adapterCode: item.adapterCode,
      enabled: item.enabled, healthStatus: item.healthStatus, priority: String(item.priority), minDurationSec: String(item.minDurationSec),
      maxDurationSec: String(item.maxDurationSec), resolutions: item.resolutions.join(','), aspectRatios: item.aspectRatios.join(','),
      minImages: String(item.minImages), maxImages: String(item.maxImages), supportsGeneratedAudio: item.supportsGeneratedAudio,
      peakPerSecond: String(currentCost?.peakPerSecond ?? 0), offpeakPerSecond: String(currentCost?.offpeakPerSecond ?? 0),
    }
  }
  executionEditor.value = true
}

function executionPayload(configuration: AdminModelConfiguration) {
  return {
    supplierId: executionForm.value.supplierId,
    modelId: configuration.modelId,
    modeId: configuration.workflowId,
    name: executionForm.value.name.trim() || `${configuration.modelName} ${configuration.workflowName}`,
    upstreamWorkflowId: executionForm.value.upstreamWorkflowId.trim(),
    adapterCode: executionForm.value.adapterCode.trim() || 'autodl_h3_v1',
    enabled: executionForm.value.enabled,
    healthStatus: executionForm.value.healthStatus,
    priority: numberValue(executionForm.value.priority, 10),
    minDurationSec: numberValue(executionForm.value.minDurationSec, 1),
    maxDurationSec: numberValue(executionForm.value.maxDurationSec, 15),
    resolutions: splitValues(executionForm.value.resolutions),
    aspectRatios: splitValues(executionForm.value.aspectRatios),
    minImages: numberValue(executionForm.value.minImages),
    maxImages: numberValue(executionForm.value.maxImages),
    supportsGeneratedAudio: executionForm.value.supportsGeneratedAudio,
  }
}

async function saveExecution() {
  const configuration = selectedConfiguration.value
  if (!configuration || !executionForm.value.supplierId || !executionForm.value.upstreamWorkflowId.trim()) {
    app.notify('请选择供应商与模型配置，并填写上游执行标识')
    return
  }
  if (!executionEditing.value && !configuration.enabled) {
    app.notify('请先在承运商与模型中启用该模型配置')
    return
  }
  const payload = executionPayload(configuration)
  if (!payload.resolutions.length || !payload.aspectRatios.length || payload.maxDurationSec < payload.minDurationSec) {
    app.notify('请填写有效的时长范围、分辨率和画幅比例')
    return
  }
  saving.value = true
  try {
    const offering = executionEditing.value
      ? await updateAdminOffering(executionEditingID.value, payload)
      : await createAdminOffering(payload)
    for (const resolution of payload.resolutions) {
      for (const generateAudio of payload.supportsGeneratedAudio ? [false, true] : [false]) {
        await upsertAdminSupplierCost({
          offeringId: offering.id, resolution, generateAudio,
          peakPerSecond: numberValue(executionForm.value.peakPerSecond),
          offpeakPerSecond: numberValue(executionForm.value.offpeakPerSecond), currency: 'CNY', status: 'active',
        })
      }
    }
    executionEditor.value = false
    app.notify('供应商模型执行配置与采购成本已保存')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存供应商模型执行配置失败'))
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
      supplierId: item.supplierId, modelId: item.modelId, modeId: item.modeId, name: item.name,
      upstreamWorkflowId: item.upstreamWorkflowId, adapterCode: item.adapterCode, enabled: !item.enabled,
      healthStatus: item.healthStatus, priority: item.priority, minDurationSec: item.minDurationSec, maxDurationSec: item.maxDurationSec,
      resolutions: item.resolutions, aspectRatios: item.aspectRatios, minImages: item.minImages, maxImages: item.maxImages,
      supportsGeneratedAudio: item.supportsGeneratedAudio,
    })
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新执行配置状态失败'))
  } finally {
    saving.value = false
  }
}

watch(selectedConfigurationID, () => {
  if (!executionEditing.value && selectedConfiguration.value) {
    executionForm.value.name = `${selectedConfiguration.value.modelName} ${selectedConfiguration.value.workflowName}`
  }
})

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="供应商与模型">
      <template #description>供应商保存调用地址和凭据；供应商模型执行配置把供应商连接到已启用的模型配置，并维护上游执行标识、能力和采购成本。客户端不会获得本页信息。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton size="sm" @click="openSupplierEditor()"><Plus class="h-3.5 w-3.5" />新增供应商</S2VButton>
        <S2VButton variant="primary" size="sm" @click="openExecutionEditor()"><Plus class="h-3.5 w-3.5" />新增执行配置</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-5 grid gap-3 sm:grid-cols-3">
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">启用供应商</span><strong class="mt-3 block font-mono text-xl">{{ suppliers.filter((item) => item.enabled).length }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">启用模型配置</span><strong class="mt-3 block font-mono text-xl">{{ configurations.filter((item) => item.enabled).length }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">可用执行配置</span><strong class="mt-3 block font-mono text-xl">{{ offerings.filter((item) => item.enabled).length }}</strong></S2VCard>
    </div>

    <S2VCard v-if="supplierEditor" variant="padded" class="mb-5">
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">{{ supplierEditingID ? '编辑' : '新增' }}供应商</h2><p class="mt-1 text-xs text-s2v-muted">内部编码独立于名称；调用凭据仅能写入或轮换，不会回显。</p></div><S2VButton size="sm" title="关闭供应商编辑器" @click="supplierEditor = false"><X class="h-3.5 w-3.5" /></S2VButton></div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VInput v-model="supplierForm.code" label="内部编码" placeholder="official_minimax" :disabled="Boolean(supplierEditingID)" />
        <S2VInput v-model="supplierForm.name" label="供应商名称" placeholder="官方 MiniMax" />
        <S2VInput v-model="supplierForm.baseUrl" label="调用地址" placeholder="https://api.example.com" class="sm:col-span-2" />
        <S2VInput v-model="supplierForm.token" type="password" :label="supplierEditingID ? '轮换凭据（留空不变）' : '调用凭据'" placeholder="仅写入，不会回显" class="sm:col-span-2" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="supplierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">允许参与新报价</label>
      </div>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveSupplier">保存供应商</S2VButton></div>
    </S2VCard>

    <S2VCard v-if="executionEditor" variant="padded" class="mb-5">
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">{{ executionEditing ? '编辑' : '新增' }}供应商模型执行配置</h2><p class="mt-1 text-xs text-s2v-muted">选择已建立的模型配置，工作流由该配置决定，不能在此任意重新绑定。</p></div><S2VButton size="sm" title="关闭执行配置编辑器" @click="executionEditor = false"><X class="h-3.5 w-3.5" /></S2VButton></div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VSelect v-model="executionForm.supplierId" label="供应商" :options="supplierOptions" :disabled="executionEditing" />
        <S2VSelect v-model="selectedConfigurationID" label="模型配置（模型 / 工作流）" :options="modelConfigurationOptions" :disabled="executionEditing" />
        <S2VInput v-model="executionForm.name" label="执行配置名称" />
        <S2VInput v-model="executionForm.upstreamWorkflowId" label="上游执行标识" placeholder="仅管理端可见" />
        <S2VInput v-model="executionForm.adapterCode" label="执行适配器" />
        <S2VInput v-model="executionForm.priority" label="路由优先级" type="number" />
        <S2VInput v-model="executionForm.minDurationSec" label="最小时长（秒）" type="number" />
        <S2VInput v-model="executionForm.maxDurationSec" label="最大时长（秒）" type="number" />
        <S2VInput v-model="executionForm.resolutions" label="分辨率（逗号分隔）" />
        <S2VInput v-model="executionForm.aspectRatios" label="画幅比例（逗号分隔）" />
        <S2VInput v-model="executionForm.minImages" label="最少图片数" type="number" />
        <S2VInput v-model="executionForm.maxImages" label="最多图片数" type="number" />
        <S2VInput v-model="executionForm.peakPerSecond" label="峰时采购成本（元/秒）" type="number" />
        <S2VInput v-model="executionForm.offpeakPerSecond" label="谷时采购成本（元/秒）" type="number" />
        <label class="flex items-center gap-2 text-xs"><input v-model="executionForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">参与新报价</label>
        <label class="flex items-center gap-2 text-xs"><input v-model="executionForm.supportsGeneratedAudio" type="checkbox" class="h-4 w-4 accent-s2v-primary">支持模型自动生成音频</label>
      </div>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveExecution">保存执行配置</S2VButton></div>
    </S2VCard>

    <S2VCard variant="padded" class="mb-5">
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">供应商目录</h2><p class="mt-1 text-xs text-s2v-muted">禁用供应商会使其所有执行配置退出新报价，不影响已有任务快照。</p></div></div>
      <div class="grid gap-3 lg:grid-cols-3">
        <div v-for="supplier in suppliers" :key="supplier.id" class="s2v-inset">
          <div class="flex items-start justify-between gap-3"><div class="min-w-0"><strong class="block truncate text-xs">{{ supplier.name }}</strong><span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ supplier.code }}</span></div><S2VBadge :tone="supplier.enabled ? 'success' : 'neutral'">{{ supplier.enabled ? '启用' : '停用' }}</S2VBadge></div>
          <p class="mt-3 text-[11px] text-s2v-muted">{{ supplier.hasToken ? '调用凭据已配置' : '尚未配置调用凭据' }}</p>
          <div class="mt-3 flex gap-1"><S2VButton size="sm" variant="ghost" title="编辑供应商" @click="openSupplierEditor(supplier)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="supplier.enabled ? '停用供应商' : '启用供应商'" :loading="saving" @click="toggleSupplier(supplier)"><Power class="h-3.5 w-3.5" /></S2VButton></div>
        </div>
        <p v-if="!suppliers.length" class="s2v-inset-dashed lg:col-span-3">暂无供应商。先新增供应商并写入调用凭据。</p>
      </div>
    </S2VCard>

    <div class="grid gap-4 lg:grid-cols-2">
      <S2VCard v-for="offering in offerings" :key="offering.id" variant="padded">
        <div class="flex items-start justify-between gap-3"><div class="flex min-w-0 items-center gap-2"><span class="s2v-icon-well"><Bot class="h-4 w-4" /></span><div class="min-w-0"><strong class="block truncate text-xs">{{ offering.name }}</strong><span class="mt-1 block truncate text-[11px] text-s2v-muted">{{ offering.supplierName }} · {{ offering.carrierName }} / {{ offering.modelName }} · {{ offering.modeName }}</span></div></div><div class="flex shrink-0 items-center gap-1"><S2VBadge :tone="offering.enabled ? 'success' : 'neutral'">{{ offering.enabled ? '可路由' : '停用' }}</S2VBadge><S2VButton size="sm" variant="ghost" title="编辑执行配置" @click="openExecutionEditor(offering)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="offering.enabled ? '停用执行配置' : '启用执行配置'" :loading="saving" @click="toggleExecution(offering)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div>
        <div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-xs text-s2v-muted"><div class="flex justify-between gap-3"><span>能力</span><strong class="text-right font-mono text-s2v-ink">{{ offering.minDurationSec }}-{{ offering.maxDurationSec }} 秒 · {{ offering.resolutions.join(', ') }}</strong></div><div class="flex justify-between gap-3"><span>自动音频</span><strong class="text-right text-s2v-ink">{{ offering.supportsGeneratedAudio ? '支持' : '不支持' }}</strong></div><div v-for="cost in costsFor(offering.id)" :key="cost.id" class="flex justify-between gap-3"><span>采购成本 {{ cost.resolution }}{{ cost.generateAudio ? ' / 自动音频' : '' }}</span><strong class="text-right font-mono text-s2v-ink">{{ cost.peakPerSecond }} / {{ cost.offpeakPerSecond }} 元/秒</strong></div></div>
      </S2VCard>
      <p v-if="!offerings.length" class="s2v-inset-dashed lg:col-span-2">暂无执行配置。先启用承运商模型中的模型配置，再绑定供应商。</p>
    </div>
  </section>
</template>
