<script setup lang="ts">
import { Coins, Plus, RefreshCw, X } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import {
  fetchAdminCarriers,
  fetchAdminCreationModes,
  fetchAdminRetailPricing,
  upsertAdminRetailPricing,
  type AdminCreationMode,
  type AdminModel,
  type AdminRetailPricing,
} from '@/api/admin'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'

const app = useAppStore()
const rules = ref<AdminRetailPricing[]>([])
const modes = ref<AdminCreationMode[]>([])
const models = ref<AdminModel[]>([])
const drafts = ref<Record<string, string>>({})
const loading = ref(false)
const saving = ref(false)
const showCreateForm = ref(false)
const createForm = ref({ modeId: '', modelId: '', resolution: '768P', generateAudio: false, pricePerSecond: '0.50' })

const modeOptions = computed(() => modes.value.filter((item) => item.status === 'published').map((item) => ({ label: item.name, value: item.id })))
const creatableModels = computed(() => {
  const policyIDs = new Set(modes.value.find((item) => item.id === createForm.value.modeId)?.policies.map((item) => item.modelId) || [])
  return models.value.filter((item) => item.enabled && policyIDs.has(item.id)).map((item) => ({ label: `${item.carrierCode} / ${item.name}`, value: item.id }))
})

async function load() {
  loading.value = true
  try {
    const [pricingData, modeData, carrierData] = await Promise.all([fetchAdminRetailPricing(), fetchAdminCreationModes(), fetchAdminCarriers()])
    rules.value = pricingData.items
    modes.value = modeData.items
    models.value = carrierData.models
    drafts.value = Object.fromEntries(pricingData.items.map((item) => [item.id, String(item.pricePerSecond)]))
  } catch (error) {
    app.notify(error instanceof Error ? error.message : '加载售价规则失败')
  } finally {
    loading.value = false
  }
}

function syncModelForMode() {
  if (!creatableModels.value.some((item) => item.value === createForm.value.modelId)) {
    createForm.value.modelId = creatableModels.value[0]?.value || ''
  }
}

function openCreateForm() {
  createForm.value = {
    modeId: modeOptions.value[0]?.value || '', modelId: '', resolution: '768P', generateAudio: false, pricePerSecond: '0.50',
  }
  syncModelForMode()
  showCreateForm.value = true
}

async function createRule() {
  const pricePerSecond = Number(createForm.value.pricePerSecond)
  if (!createForm.value.modeId || !createForm.value.modelId || !createForm.value.resolution.trim() || !Number.isFinite(pricePerSecond) || pricePerSecond < 0) {
    app.notify('请完整填写创作模式、模型、分辨率和每秒售价')
    return
  }
  const mode = modes.value.find((item) => item.id === createForm.value.modeId)
  const model = models.value.find((item) => item.id === createForm.value.modelId)
  saving.value = true
  try {
    await upsertAdminRetailPricing({
      modeId: createForm.value.modeId,
      modelId: createForm.value.modelId,
      resolution: createForm.value.resolution.trim(),
      generateAudio: createForm.value.generateAudio,
      name: `${model?.name || '模型'} ${mode?.name || '创作'} / ${createForm.value.resolution.trim()}`,
      pricePerSecond,
      status: 'active',
    })
    app.notify('对客售价已发布，仅影响新报价')
    showCreateForm.value = false
    await load()
  } catch (error) {
    app.notify(error instanceof Error ? error.message : '发布售价规则失败')
  } finally {
    saving.value = false
  }
}

async function save() {
  saving.value = true
  try {
    for (const rule of rules.value) {
      const pricePerSecond = Number(drafts.value[rule.id])
      if (!Number.isFinite(pricePerSecond) || Math.abs(pricePerSecond - rule.pricePerSecond) < 0.0001) continue
      await upsertAdminRetailPricing({
        modeId: rule.modeId,
        modelId: rule.modelId,
        resolution: rule.resolution,
        generateAudio: rule.generateAudio,
        name: rule.name,
        pricePerSecond,
        status: rule.status,
      })
    }
    app.notify('对客售价已发布，仅影响新报价')
    await load()
  } catch (error) {
    app.notify(error instanceof Error ? error.message : '保存售价失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="定价规则">
      <template #description>售价按创作模式、模型、分辨率和自动音频维护；时长是每秒单价的计费数量，采购成本只在供应商供给中维护。</template>
      <template #actions>
        <S2VButton :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton variant="primary" size="sm" @click="openCreateForm"><Plus class="h-3.5 w-3.5" />新增售价</S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard v-if="showCreateForm" class="mb-5" variant="padded">
      <div class="s2v-panel-header">
        <div><h2 class="s2v-section-title">新增对客售价</h2><p class="mt-1 text-xs text-s2v-muted">先将模型加入创作模式策略，再发布该组合的售价。时长按每秒单价计算。</p></div>
        <S2VButton size="sm" title="关闭编辑器" @click="showCreateForm = false"><X class="h-3.5 w-3.5" /></S2VButton>
      </div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VSelect v-model="createForm.modeId" label="创作模式" :options="modeOptions" @update:model-value="syncModelForMode" />
        <S2VSelect v-model="createForm.modelId" label="模型" :options="creatableModels" />
        <S2VInput v-model="createForm.resolution" label="分辨率" placeholder="1080P" />
        <S2VInput v-model="createForm.pricePerSecond" label="售价（元/秒）" type="number" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="createForm.generateAudio" type="checkbox" class="h-4 w-4 accent-s2v-primary">支持模型自动生成音频</label>
        <div class="sm:col-span-2"><S2VButton variant="primary" size="sm" :loading="saving" @click="createRule">发布售价</S2VButton></div>
      </div>
    </S2VCard>

    <S2VCard>
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">平台对客售价</h2>
          <p class="mt-1 text-xs text-s2v-muted">此处不展示供应商成本，调价会使未使用的旧报价失效。</p>
        </div>
        <Coins class="h-4 w-4 text-s2v-primary" />
      </div>
      <div class="hidden overflow-x-auto md:block">
        <table class="s2v-table">
          <thead>
            <tr>
              <th>创作模式</th>
              <th>承运商 / 模型</th>
              <th>分辨率</th>
              <th>自动音频</th>
              <th>售价（元/秒）</th>
              <th>状态</th>
              <th>版本</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="rule in rules" :key="rule.id">
              <td class="font-semibold text-s2v-ink">{{ rule.modeName }}</td>
              <td>{{ rule.carrierName }} / {{ rule.modelName }}</td>
              <td class="font-mono">{{ rule.resolution }}</td>
              <td>{{ rule.generateAudio ? '支持' : '不支持' }}</td>
              <td><S2VInput :model-value="drafts[rule.id]" size="sm" class="w-24" @update:model-value="drafts[rule.id] = $event" /></td>
              <td><S2VBadge :tone="rule.status === 'active' ? 'success' : 'neutral'">{{ rule.status === 'active' ? '生效中' : rule.status }}</S2VBadge></td>
              <td class="font-mono">{{ rule.version }}</td>
            </tr>
            <tr v-if="!rules.length"><td colspan="7" class="text-center text-s2v-muted">暂无售价规则。</td></tr>
          </tbody>
        </table>
      </div>
      <div class="grid gap-3 px-4 pb-4 md:hidden">
        <MobileRecordCard v-for="rule in rules" :key="rule.id" :title="rule.modeName" :meta="`${rule.carrierName} / ${rule.modelName}`">
          <template #badge><S2VBadge :tone="rule.status === 'active' ? 'success' : 'neutral'">{{ rule.status }}</S2VBadge></template>
          <template #body>{{ rule.resolution }} · {{ rule.generateAudio ? '自动音频' : '无自动音频' }} · {{ rule.version }}</template>
          <template #actions><S2VInput :model-value="drafts[rule.id]" size="sm" class="max-w-28" @update:model-value="drafts[rule.id] = $event" /></template>
        </MobileRecordCard>
      </div>
      <div class="flex justify-end border-t border-s2v-line px-4 py-3">
        <S2VButton variant="primary" size="sm" :loading="saving" @click="save">发布调价</S2VButton>
      </div>
    </S2VCard>
  </section>
</template>
