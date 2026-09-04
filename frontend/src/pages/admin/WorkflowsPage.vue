<script setup lang="ts">
import { Plus, RefreshCw, Settings2 } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import {
  fetchAdminCarriers,
  fetchAdminCreationModes,
  updateAdminCreationMode,
  upsertAdminModePolicy,
  type AdminCreationMode,
  type AdminModel,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'

const app = useAppStore()
const items = ref<AdminCreationMode[]>([])
const models = ref<AdminModel[]>([])
const policyDrafts = ref<Record<string, string>>({})
const loading = ref(false)
const saving = ref<string | null>(null)

const enabledModels = computed(() => models.value.filter((item) => item.enabled))

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const [modeData, carrierData] = await Promise.all([fetchAdminCreationModes(), fetchAdminCarriers()])
    items.value = modeData.items
    models.value = carrierData.models
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载创作模式失败'))
  } finally {
    loading.value = false
  }
}

async function toggleMode(item: AdminCreationMode) {
  saving.value = item.id
  try {
    await updateAdminCreationMode(item.id, { status: item.status === 'published' ? 'disabled' : 'published' })
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新创作模式失败'))
  } finally {
    saving.value = null
  }
}

async function savePolicy(mode: AdminCreationMode, policy: AdminCreationMode['policies'][number]) {
  saving.value = `${mode.id}:${policy.modelId}`
  try {
    await upsertAdminModePolicy(mode.id, policy)
    app.notify('模型策略已保存')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存模型策略失败'))
  } finally {
    saving.value = null
  }
}

function policyOptions(mode: AdminCreationMode) {
  const assigned = new Set(mode.policies.map((item) => item.modelId))
  return enabledModels.value
    .filter((item) => !assigned.has(item.id))
    .map((item) => ({ label: `${item.carrierCode} / ${item.name}`, value: item.id }))
}

async function addPolicy(mode: AdminCreationMode) {
  const modelID = policyDrafts.value[mode.id] || policyOptions(mode)[0]?.value
  const model = models.value.find((item) => item.id === modelID)
  if (!model) return
  saving.value = `${mode.id}:new`
  try {
    await upsertAdminModePolicy(mode.id, {
      modelId: model.id,
      modelName: model.name,
      carrierName: model.carrierCode || '',
      enabled: true,
      recommendationRank: mode.policies.length + 1,
      displayBadge: '',
      defaultDuration: 5,
      defaultResolution: '768P',
    })
    app.notify('模型已加入该创作模式')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '添加模型策略失败'))
  } finally {
    saving.value = null
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="创作模式">
      <template #description>客户端固定为文生视频、首尾帧生成视频和多图生成视频；此处仅维护上架状态及模式内模型推荐策略。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 lg:grid-cols-3">
      <S2VCard v-for="item in items" :key="item.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <span class="s2v-icon-well s2v-icon-well-nav"><Settings2 class="h-4 w-4" /></span>
          <S2VBadge :tone="item.status === 'published' ? 'success' : 'neutral'">{{ item.status === 'published' ? '已上架' : '已下架' }}</S2VBadge>
        </div>
        <h2 class="mt-4 text-sm font-bold">{{ item.name }}</h2>
        <p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ item.description }}</p>
        <div class="mt-4 border-t border-s2v-line pt-3">
          <p class="mb-2 text-[11px] font-semibold text-s2v-muted">可选模型与推荐顺序</p>
          <div v-for="policy in item.policies" :key="policy.modelId" class="grid grid-cols-[1fr_56px_auto] items-center gap-2 py-1.5 text-xs">
            <span class="min-w-0 truncate">{{ policy.carrierName }} / {{ policy.modelName }}</span>
            <S2VInput
              :model-value="String(policy.recommendationRank)"
              size="sm"
              @update:model-value="policy.recommendationRank = Number($event) || 1"
            />
            <div class="flex items-center justify-end gap-1">
              <input v-model="policy.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">
              <S2VButton size="sm" :loading="saving === `${item.id}:${policy.modelId}`" @click="savePolicy(item, policy)">保存</S2VButton>
            </div>
          </div>
          <div v-if="policyOptions(item).length" class="mt-2 flex items-end gap-2">
            <S2VSelect v-model="policyDrafts[item.id]" class="min-w-0 flex-1" :options="policyOptions(item)" />
            <S2VButton size="sm" title="加入模型" :loading="saving === `${item.id}:new`" @click="addPolicy(item)"><Plus class="h-3.5 w-3.5" /></S2VButton>
          </div>
          <p v-if="!item.policies.length" class="text-xs text-s2v-muted">尚未配置可用模型。</p>
        </div>
        <div class="mt-4">
          <S2VButton size="sm" :loading="saving === item.id" @click="toggleMode(item)">{{ item.status === 'published' ? '下架模式' : '上架模式' }}</S2VButton>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
