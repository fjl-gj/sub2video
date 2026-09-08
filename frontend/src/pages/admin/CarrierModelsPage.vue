<script setup lang="ts">
import { Pencil, Plus, Power, RefreshCw, SlidersHorizontal, X } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import {
  createAdminCarrier,
  createAdminModel,
  fetchAdminCarriers,
  fetchAdminCreationModes,
  fetchAdminModelConfigurations,
  fetchAdminRetailPricing,
  updateAdminCarrier,
  updateAdminModel,
  upsertAdminModelConfiguration,
  upsertAdminRetailPricing,
  type AdminCarrier,
  type AdminCreationMode,
  type AdminModel,
  type AdminModelConfiguration,
  type AdminRetailPricing,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'

const app = useAppStore()
const carriers = ref<AdminCarrier[]>([])
const models = ref<AdminModel[]>([])
const workflows = ref<AdminCreationMode[]>([])
const configurations = ref<AdminModelConfiguration[]>([])
const prices = ref<AdminRetailPricing[]>([])
const loading = ref(false)
const saving = ref(false)
const catalogEditor = ref<'carrier' | 'model' | null>(null)
const editingID = ref('')
const configModel = ref<AdminModel | null>(null)
const configEditing = ref<AdminModelConfiguration | null>(null)

const carrierForm = ref({ code: '', name: '', description: '', enabled: true })
const modelForm = ref({ carrierId: '', code: '', name: '', description: '', enabled: true })
const configForm = ref({ workflowId: '', enabled: true, recommendationRank: '10', defaultDuration: '5', defaultResolution: '768P', resolution: '768P', generateAudio: false, pricePerSecond: '0.50' })

const carrierOptions = computed(() => carriers.value.filter((item) => item.enabled).map((item) => ({ label: item.name, value: item.id })))
const workflowOptions = computed(() => workflows.value.filter((item) => item.status === 'published').map((item) => ({ label: item.name, value: item.id })))
const activeCarrierCount = computed(() => carriers.value.filter((item) => item.enabled).length)
const activeModelCount = computed(() => models.value.filter((item) => item.enabled).length)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function numberValue(value: string, fallback: number) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function configurationsFor(modelID: string) {
  return configurations.value.filter((item) => item.modelId === modelID)
}

function pricesFor(config: AdminModelConfiguration) {
  return prices.value.filter((item) => item.modelId === config.modelId && item.modeId === config.workflowId)
}

function modelsFor(carrierID: string) {
  return models.value.filter((item) => item.carrierId === carrierID)
}

async function load() {
  loading.value = true
  try {
    const [carrierData, workflowData, configData, priceData] = await Promise.all([
      fetchAdminCarriers(), fetchAdminCreationModes(), fetchAdminModelConfigurations(), fetchAdminRetailPricing(),
    ])
    carriers.value = carrierData.carriers
    models.value = carrierData.models
    workflows.value = workflowData.items
    configurations.value = configData.items
    prices.value = priceData.items
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载承运商与模型失败'))
  } finally {
    loading.value = false
  }
}

function openCatalogEditor(kind: 'carrier' | 'model', item?: AdminCarrier | AdminModel) {
  catalogEditor.value = kind
  editingID.value = item?.id || ''
  if (kind === 'carrier') {
    const carrier = item as AdminCarrier | undefined
    carrierForm.value = { code: carrier?.code || '', name: carrier?.name || '', description: carrier?.description || '', enabled: carrier?.enabled ?? true }
  } else {
    const model = item as AdminModel | undefined
    modelForm.value = {
      carrierId: model?.carrierId || carrierOptions.value[0]?.value || '', code: model?.code || '',
      name: model?.name || '', description: model?.description || '', enabled: model?.enabled ?? true,
    }
  }
}

async function saveCatalog() {
  if (!catalogEditor.value) return
  saving.value = true
  try {
    if (catalogEditor.value === 'carrier') {
      if (editingID.value) await updateAdminCarrier(editingID.value, { name: carrierForm.value.name, description: carrierForm.value.description, enabled: carrierForm.value.enabled })
      else await createAdminCarrier({ code: carrierForm.value.code, name: carrierForm.value.name, description: carrierForm.value.description })
    } else if (editingID.value) {
      await updateAdminModel(editingID.value, { name: modelForm.value.name, description: modelForm.value.description, enabled: modelForm.value.enabled })
    } else {
      await createAdminModel({ carrierId: modelForm.value.carrierId, code: modelForm.value.code, name: modelForm.value.name, description: modelForm.value.description })
    }
    catalogEditor.value = null
    editingID.value = ''
    app.notify('承运商目录已保存')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存承运商目录失败'))
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
    app.notify(apiErrorMessage(error, '更新承运商状态失败'))
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
    app.notify(apiErrorMessage(error, '更新模型状态失败'))
  } finally {
    saving.value = false
  }
}

function openConfiguration(model: AdminModel, item?: AdminModelConfiguration) {
  configModel.value = model
  configEditing.value = item || null
  const currentPrice = item ? pricesFor(item)[0] : undefined
  configForm.value = {
    workflowId: item?.workflowId || workflowOptions.value[0]?.value || '', enabled: item?.enabled ?? true,
    recommendationRank: String(item?.recommendationRank || configurationsFor(model.id).length + 1),
    defaultDuration: String(item?.defaultDuration || 5), defaultResolution: item?.defaultResolution || '768P',
    resolution: currentPrice?.resolution || item?.defaultResolution || '768P', generateAudio: currentPrice?.generateAudio || false,
    pricePerSecond: String(currentPrice?.pricePerSecond ?? 0.5),
  }
}

async function saveConfiguration() {
  if (!configModel.value || !configForm.value.workflowId) {
    app.notify('请选择工作流')
    return
  }
  const price = numberValue(configForm.value.pricePerSecond, -1)
  if (!configForm.value.resolution.trim() || price < 0) {
    app.notify('请填写分辨率和有效每秒售价')
    return
  }
  const workflow = workflows.value.find((item) => item.id === configForm.value.workflowId)
  saving.value = true
  try {
    await upsertAdminModelConfiguration({
      modelId: configModel.value.id,
      workflowId: configForm.value.workflowId,
      enabled: configForm.value.enabled,
      recommendationRank: numberValue(configForm.value.recommendationRank, 1),
      displayBadge: '',
      defaultDuration: numberValue(configForm.value.defaultDuration, 5),
      defaultResolution: configForm.value.defaultResolution.trim() || '768P',
    })
    await upsertAdminRetailPricing({
      modeId: configForm.value.workflowId,
      modelId: configModel.value.id,
      resolution: configForm.value.resolution.trim(),
      generateAudio: configForm.value.generateAudio,
      name: `${configModel.value.name} ${workflow?.name || '工作流'} / ${configForm.value.resolution.trim()}`,
      pricePerSecond: price,
      status: 'active',
    })
    configModel.value = null
    configEditing.value = null
    app.notify('模型配置与对客售价已保存')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存模型配置失败'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="承运商与模型">
      <template #description>承运商是模型原厂，模型归属承运商。模型配置由模型侧声明支持哪些固定工作流，并在同处维护对客售价。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton size="sm" @click="openCatalogEditor('carrier')"><Plus class="h-3.5 w-3.5" />新增承运商</S2VButton>
        <S2VButton variant="primary" size="sm" @click="openCatalogEditor('model')"><Plus class="h-3.5 w-3.5" />新增模型系列</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-5 grid gap-3 sm:grid-cols-3">
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">启用承运商</span><strong class="mt-3 block font-mono text-xl">{{ activeCarrierCount }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">启用模型</span><strong class="mt-3 block font-mono text-xl">{{ activeModelCount }}</strong></S2VCard>
      <S2VCard variant="metric"><span class="text-xs text-s2v-muted">模型配置</span><strong class="mt-3 block font-mono text-xl">{{ configurations.filter((item) => item.enabled).length }}</strong></S2VCard>
    </div>

    <S2VCard v-if="catalogEditor" class="mb-5" variant="padded">
      <div class="s2v-panel-header">
        <div><h2 class="s2v-section-title">{{ editingID ? '编辑' : '新增' }}{{ catalogEditor === 'carrier' ? '承运商' : '模型系列' }}</h2><p class="mt-1 text-xs text-s2v-muted">编码为内部稳定标识，显示名称可独立调整。</p></div>
        <S2VButton size="sm" title="关闭编辑器" @click="catalogEditor = null"><X class="h-3.5 w-3.5" /></S2VButton>
      </div>
      <div v-if="catalogEditor === 'carrier'" class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VInput v-model="carrierForm.code" label="内部编码" placeholder="volcengine" :disabled="Boolean(editingID)" />
        <S2VInput v-model="carrierForm.name" label="承运商名称" placeholder="火山引擎" />
        <S2VInput v-model="carrierForm.description" label="说明" class="sm:col-span-2" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="carrierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">允许新增模型和参与客户端目录</label>
      </div>
      <div v-else class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VSelect v-model="modelForm.carrierId" label="承运商" :options="carrierOptions" :disabled="Boolean(editingID)" />
        <S2VInput v-model="modelForm.code" label="内部编码" placeholder="seedance_2_5" :disabled="Boolean(editingID)" />
        <S2VInput v-model="modelForm.name" label="模型系列名称" placeholder="Seedance 2.5" />
        <S2VInput v-model="modelForm.description" label="说明" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="modelForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">允许配置工作流和参与客户端目录</label>
      </div>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveCatalog">保存</S2VButton></div>
    </S2VCard>

    <S2VCard v-if="configModel" class="mb-5" variant="padded">
      <div class="s2v-panel-header">
        <div><h2 class="s2v-section-title">{{ configEditing ? '编辑' : '新增' }}模型配置</h2><p class="mt-1 text-xs text-s2v-muted">{{ configModel.carrierCode }} / {{ configModel.name }}：声明该模型可支持的固定工作流，并维护对应的对客售价。</p></div>
        <S2VButton size="sm" title="关闭模型配置" @click="configModel = null"><X class="h-3.5 w-3.5" /></S2VButton>
      </div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VSelect v-model="configForm.workflowId" label="支持工作流" :options="workflowOptions" :disabled="Boolean(configEditing)" />
        <S2VInput v-model="configForm.recommendationRank" label="智能推荐顺序" type="number" />
        <S2VInput v-model="configForm.defaultDuration" label="默认时长（秒）" type="number" />
        <S2VInput v-model="configForm.defaultResolution" label="默认分辨率" placeholder="768P" />
        <S2VInput v-model="configForm.resolution" label="计价分辨率" placeholder="1080P" />
        <S2VInput v-model="configForm.pricePerSecond" label="平台售价（元/秒）" type="number" />
        <label class="flex items-center gap-2 text-xs"><input v-model="configForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">该模型支持此工作流</label>
        <label class="flex items-center gap-2 text-xs"><input v-model="configForm.generateAudio" type="checkbox" class="h-4 w-4 accent-s2v-primary">此售价支持自动生成音频</label>
      </div>
      <div class="mt-4"><S2VButton variant="primary" size="sm" :loading="saving" @click="saveConfiguration">保存模型配置</S2VButton></div>
    </S2VCard>

    <div class="grid gap-4">
      <S2VCard v-for="carrier in carriers" :key="carrier.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0"><div class="flex items-center gap-2"><h2 class="text-sm font-bold">{{ carrier.name }}</h2><S2VBadge :tone="carrier.enabled ? 'success' : 'neutral'">{{ carrier.enabled ? '启用' : '停用' }}</S2VBadge></div><p class="mt-1 text-xs text-s2v-muted">{{ carrier.description || '暂无说明' }}</p></div>
          <div class="flex shrink-0 gap-1"><S2VButton size="sm" variant="ghost" title="编辑承运商" @click="openCatalogEditor('carrier', carrier)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="carrier.enabled ? '停用承运商' : '启用承运商'" :loading="saving" @click="toggleCarrier(carrier)"><Power class="h-3.5 w-3.5" /></S2VButton></div>
        </div>
        <div class="mt-4 grid gap-3 border-t border-s2v-line pt-4 lg:grid-cols-2">
          <div v-for="model in modelsFor(carrier.id)" :key="model.id" class="s2v-inset">
            <div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex items-center gap-2"><strong class="truncate text-xs">{{ model.name }}</strong><S2VBadge :tone="model.enabled ? 'success' : 'neutral'">{{ model.enabled ? '启用' : '停用' }}</S2VBadge></div><p class="mt-1 text-[11px] text-s2v-muted">{{ model.description || '暂无说明' }}</p></div><div class="flex shrink-0 gap-1"><S2VButton size="sm" variant="ghost" title="编辑模型" @click="openCatalogEditor('model', model)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="model.enabled ? '停用模型' : '启用模型'" :loading="saving" @click="toggleModel(model)"><Power class="h-3.5 w-3.5" /></S2VButton></div></div>
            <div class="mt-3 grid gap-2 border-t border-s2v-line pt-3">
              <div v-for="config in configurationsFor(model.id)" :key="`${config.modelId}:${config.workflowId}`" class="flex items-center justify-between gap-3 text-xs"><div class="min-w-0"><span class="font-medium text-s2v-ink">{{ config.workflowName }}</span><span class="ml-2 text-s2v-muted">默认 {{ config.defaultDuration }} 秒 · {{ config.defaultResolution }}</span><span v-for="price in pricesFor(config)" :key="price.id" class="ml-2 font-mono text-s2v-muted">¥{{ price.pricePerSecond }}/秒 {{ price.resolution }}{{ price.generateAudio ? ' 音频' : '' }}</span></div><S2VButton size="sm" variant="ghost" title="编辑模型配置" @click="openConfiguration(model, config)"><SlidersHorizontal class="h-3.5 w-3.5" /></S2VButton></div>
              <p v-if="!configurationsFor(model.id).length" class="text-xs text-s2v-muted">尚未声明支持的工作流。</p>
            </div>
            <div class="mt-3"><S2VButton size="sm" @click="openConfiguration(model)"><Plus class="h-3.5 w-3.5" />模型配置</S2VButton></div>
          </div>
          <p v-if="!modelsFor(carrier.id).length" class="s2v-inset-dashed text-xs text-s2v-muted lg:col-span-2">该承运商尚未建立模型系列。</p>
        </div>
      </S2VCard>
      <p v-if="!carriers.length" class="s2v-inset-dashed">暂无承运商，请先新增承运商和模型系列。</p>
    </div>
  </section>
</template>
