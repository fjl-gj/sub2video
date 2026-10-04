<script setup lang="ts">
import { Pencil, Plus, Power, RefreshCw, Search } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import '@/features/admin/catalog.css'
import {
  createAdminCarrier,
  createAdminModel,
  fetchAdminCarriers,
  updateAdminCarrier,
  updateAdminModel,
  type AdminCarrier,
  type AdminModel,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'

const app = useAppStore()
const carriers = ref<AdminCarrier[]>([])
const models = ref<AdminModel[]>([])
const loading = ref(false)
const saving = ref(false)
const keyword = ref('')
const editor = ref<'carrier' | 'model' | null>(null)
const editorOpen = ref(false)
const editingID = ref('')
const page = ref(1)
const pageSize = 10
const total = ref(0)
const carrierForm = ref({ code: '', name: '', description: '', enabled: true })
const modelForm = ref({ carrierId: '', code: '', name: '', description: '', enabled: true })

const carrierOptions = computed(() => carriers.value
  .filter((item) => item.enabled)
  .map((item) => ({ label: `${item.name} (${item.code})`, value: item.id })))
const normalizedKeyword = computed(() => keyword.value.trim().toLowerCase())

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

function modelsFor(carrierID: string) {
  return models.value.filter((item) => item.carrierId === carrierID)
}

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminCarriers({ page: page.value, pageSize, keyword: normalizedKeyword.value || undefined })
    carriers.value = data.carriers
    models.value = data.models
    total.value = data.total ?? data.carriers.length
    if (page.value > Math.max(1, Math.ceil(total.value / pageSize))) {
      page.value = 1
      return
    }
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载承运商目录失败'))
  } finally {
    loading.value = false
  }
}

function openEditor(kind: 'carrier' | 'model', item?: AdminCarrier | AdminModel, carrierID = '') {
  editor.value = kind
  editorOpen.value = true
  editingID.value = item?.id || ''
  if (kind === 'carrier') {
    const carrier = item as AdminCarrier | undefined
    carrierForm.value = {
      code: carrier?.code || '', name: carrier?.name || '', description: carrier?.description || '', enabled: carrier?.enabled ?? true,
    }
    return
  }
  const model = item as AdminModel | undefined
  modelForm.value = {
    carrierId: model?.carrierId || carrierID || carrierOptions.value[0]?.value || '',
    code: model?.code || '', name: model?.name || '', description: model?.description || '', enabled: model?.enabled ?? true,
  }
}

function closeEditor() {
  editorOpen.value = false
  editor.value = null
  editingID.value = ''
}

async function saveCatalog() {
  if (!editor.value) return
  const savingKind = editor.value
  saving.value = true
  try {
    if (editor.value === 'carrier') {
      if (editingID.value) {
        await updateAdminCarrier(editingID.value, {
          name: carrierForm.value.name, description: carrierForm.value.description, enabled: carrierForm.value.enabled,
        })
      } else {
        const created = await createAdminCarrier({
          code: carrierForm.value.code, name: carrierForm.value.name, description: carrierForm.value.description,
        })
        if (!carrierForm.value.enabled) await updateAdminCarrier(created.id, { enabled: false })
      }
    } else if (editingID.value) {
      await updateAdminModel(editingID.value, {
        name: modelForm.value.name, description: modelForm.value.description, enabled: modelForm.value.enabled,
      })
    } else {
      const created = await createAdminModel({
        carrierId: modelForm.value.carrierId, code: modelForm.value.code, name: modelForm.value.name, description: modelForm.value.description,
      })
      if (!modelForm.value.enabled) await updateAdminModel(created.id, { enabled: false })
    }
    closeEditor()
    app.notify(savingKind === 'carrier' ? '承运商已保存' : '模型已保存')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '保存目录失败'))
  } finally {
    saving.value = false
  }
}

watch(keyword, () => {
  if (page.value !== 1) page.value = 1
  else void load()
})
watch(page, () => { void load() })
watch(editorOpen, (open) => { if (!open) { editor.value = null; editingID.value = '' } })

async function toggleCarrier(item: AdminCarrier) {
  const action = item.enabled ? '停用' : '启用'
  if (!window.confirm(`确认${action}承运商“${item.name}”？`)) return
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
  const action = item.enabled ? '停用' : '启用'
  if (!window.confirm(`确认${action}模型“${item.name}”？`)) return
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

onMounted(load)
</script>

<template>
  <section class="s2v-page s2v-catalog">
    <S2VPageHeader title="承运商">
      <template #description>维护模型原厂及其模型目录，管理启用状态。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton variant="primary" size="sm" @click="openEditor('carrier')"><Plus class="h-3.5 w-3.5" />新增承运商</S2VButton>
      </template>
    </S2VPageHeader>

    <S2VDialog v-model:open="editorOpen" :title="`${editingID ? '编辑' : '新增'}${editor === 'carrier' ? '承运商' : '模型'}`" description="内部编码是稳定标识，显示名称可独立调整。" hide-actions size="wide">
      <div v-if="editor === 'carrier'" class="grid gap-3 sm:grid-cols-2">
        <S2VInput v-model="carrierForm.code" label="承运商编码" placeholder="minimax" :disabled="Boolean(editingID)" />
        <S2VInput v-model="carrierForm.name" label="承运商名称" placeholder="MiniMax" />
        <S2VInput v-model="carrierForm.description" label="说明" class="sm:col-span-2" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="carrierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">启用承运商</label>
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2">
        <S2VSelect v-model="modelForm.carrierId" label="归属承运商" :options="carrierOptions" :disabled="Boolean(editingID)" />
        <S2VInput v-model="modelForm.code" label="模型编码" placeholder="h3" :disabled="Boolean(editingID)" />
        <S2VInput v-model="modelForm.name" label="模型名称" placeholder="H3" />
        <S2VInput v-model="modelForm.description" label="说明" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="modelForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary">启用模型</label>
      </div>
      <div class="mt-5 flex justify-end gap-2"><S2VButton size="sm" @click="closeEditor">取消</S2VButton><S2VButton variant="primary" size="sm" :loading="saving" @click="saveCatalog">保存</S2VButton></div>
    </S2VDialog>

    <div class="mb-5 max-w-md">
      <S2VInput v-model="keyword" label="查询目录" placeholder="按 ID、名称或编码搜索承运商和模型">
        <template #prefix><Search class="h-4 w-4 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="grid gap-4">
      <S2VCard v-for="carrier in carriers" :key="carrier.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0"><div class="flex items-center gap-2"><h2 class="text-sm font-bold">{{ carrier.name }}</h2><S2VBadge :tone="carrier.enabled ? 'success' : 'neutral'">{{ carrier.enabled ? '启用' : '停用' }}</S2VBadge></div><p class="s2v-mono mt-1 text-[10px] text-s2v-muted">{{ carrier.id }} · {{ carrier.code }}</p><p class="mt-1 text-xs text-s2v-muted">{{ carrier.description || '暂无说明' }}</p></div>
          <div class="flex shrink-0 gap-1"><S2VButton size="sm" title="新增模型" @click="openEditor('model', undefined, carrier.id)"><Plus class="h-3.5 w-3.5" />模型</S2VButton><S2VButton size="sm" variant="ghost" title="编辑承运商" @click="openEditor('carrier', carrier)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="carrier.enabled ? '停用承运商' : '启用承运商'" :loading="saving" @click="toggleCarrier(carrier)"><Power class="h-3.5 w-3.5" /></S2VButton></div>
        </div>
        <div class="mt-4 overflow-x-auto border-t border-s2v-line pt-4">
          <table class="w-full text-left text-xs">
            <thead class="text-s2v-muted"><tr><th class="pb-2 pr-3 font-medium">模型 ID</th><th class="pb-2 pr-3 font-medium">模型名称</th><th class="pb-2 pr-3 font-medium">模型编码</th><th class="pb-2 pr-3 font-medium">承运商 ID</th><th class="pb-2 pr-3 font-medium">承运商编码</th><th class="pb-2 pr-3 font-medium">状态</th><th class="pb-2 text-right font-medium">操作</th></tr></thead>
            <tbody>
              <tr v-for="model in modelsFor(carrier.id)" :key="model.id" class="border-t border-s2v-line/80"><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ model.id }}</td><td class="py-3 pr-3 font-medium text-s2v-ink">{{ model.name }}</td><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ model.code }}</td><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ carrier.id }}</td><td class="s2v-mono py-3 pr-3 text-[10px] text-s2v-muted">{{ carrier.code }}</td><td class="py-3 pr-3"><S2VBadge :tone="model.enabled ? 'success' : 'neutral'">{{ model.enabled ? '启用' : '停用' }}</S2VBadge></td><td class="py-3 text-right"><div class="flex justify-end gap-1"><S2VButton size="sm" variant="ghost" title="编辑模型" @click="openEditor('model', model)"><Pencil class="h-3.5 w-3.5" /></S2VButton><S2VButton size="sm" variant="ghost" :title="model.enabled ? '停用模型' : '启用模型'" :loading="saving" @click="toggleModel(model)"><Power class="h-3.5 w-3.5" /></S2VButton></div></td></tr>
              <tr v-if="!modelsFor(carrier.id).length"><td colspan="7" class="py-4 text-s2v-muted">该承运商尚未建立模型。</td></tr>
            </tbody>
          </table>
        </div>
      </S2VCard>
      <p v-if="!carriers.length" class="s2v-inset-dashed">未找到承运商或模型。</p>
      <S2VPagination v-if="total" v-model="page" :total="total" :page-size="pageSize" />
    </div>
  </section>
</template>
