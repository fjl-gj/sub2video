<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Bot, KeyRound, Pencil, Plus, RefreshCw, Server, X } from 'lucide-vue-next'
import {
  createAdminModel,
  createAdminRoute,
  createAdminSupplier,
  fetchAdminCarriers,
  fetchAdminRoutes,
  fetchAdminSuppliers,
  updateAdminRoute,
  updateAdminSupplier,
  type AdminCarrier,
  type AdminModel,
  type AdminRoute,
  type AdminSupplier,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const loading = ref(false)
const saving = ref(false)
const suppliers = ref<AdminSupplier[]>([])
const routes = ref<AdminRoute[]>([])
const carriers = ref<AdminCarrier[]>([])
const models = ref<AdminModel[]>([])
const cosConfigured = ref(false)
const autodlEnvToken = ref(false)

const showSupplierForm = ref(false)
const editingSupplierId = ref<string | null>(null)
const supplierForm = ref({ code: '', name: '', baseUrl: 'https://autodl.art', token: '', enabled: true })

const showRouteForm = ref(false)
const editingRouteId = ref<string | null>(null)
const routeForm = ref({
  supplierId: '',
  modelId: '',
  workflowId: '',
  name: '',
  costPerSecPeak: '0.03',
  costPerSecOffpeak: '0.02',
  enabled: true,
})

const showModelForm = ref(false)
const modelForm = ref({ carrierId: '', code: '', name: '', clientWorkflowId: '', description: '' })

const metrics = computed(() => [
  { label: '已接入供应商', value: String(suppliers.value.filter((item) => item.enabled).length), icon: Server },
  { label: '启用路由', value: String(routes.value.filter((item) => item.enabled).length), icon: Bot },
  { label: 'COS', value: cosConfigured.value ? '已配置' : '待接入', icon: KeyRound },
])

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const [s, r, c] = await Promise.all([
      fetchAdminSuppliers(),
      fetchAdminRoutes(),
      fetchAdminCarriers(),
    ])
    suppliers.value = s.items
    routes.value = r.items
    carriers.value = c.carriers
    models.value = c.models
    cosConfigured.value = c.cosConfigured
    autodlEnvToken.value = c.autodlEnvToken
    if (!routeForm.value.supplierId && s.items[0]) routeForm.value.supplierId = s.items[0].id
    if (!routeForm.value.modelId && c.models[0]) routeForm.value.modelId = c.models[0].id
    if (!modelForm.value.carrierId && c.carriers[0]) modelForm.value.carrierId = c.carriers[0].id
  } catch (err) {
    app.notify(apiErrorMessage(err, '加载供应商失败（需 admin 角色）'))
  } finally {
    loading.value = false
  }
}

function openCreateSupplier() {
  editingSupplierId.value = null
  supplierForm.value = { code: '', name: '', baseUrl: 'https://autodl.art', token: '', enabled: true }
  showSupplierForm.value = true
}

function openEditSupplier(sp: AdminSupplier) {
  editingSupplierId.value = sp.id
  supplierForm.value = {
    code: sp.code,
    name: sp.name,
    baseUrl: sp.baseUrl,
    token: '',
    enabled: sp.enabled,
  }
  showSupplierForm.value = true
}

async function saveSupplier() {
  saving.value = true
  try {
    if (editingSupplierId.value) {
      await updateAdminSupplier(editingSupplierId.value, {
        name: supplierForm.value.name.trim(),
        baseUrl: supplierForm.value.baseUrl.trim(),
        enabled: supplierForm.value.enabled,
        token: supplierForm.value.token.trim() || undefined,
      })
      app.notify('供应商已更新')
    } else {
      if (!supplierForm.value.code.trim() || !supplierForm.value.name.trim()) {
        app.notify('请填写供应商编码和名称')
        return
      }
      await createAdminSupplier({
        code: supplierForm.value.code.trim(),
        name: supplierForm.value.name.trim(),
        baseUrl: supplierForm.value.baseUrl.trim(),
        token: supplierForm.value.token.trim() || undefined,
        enabled: supplierForm.value.enabled,
      })
      app.notify('供应商已创建')
    }
    showSupplierForm.value = false
    await load()
  } catch (err) {
    app.notify(apiErrorMessage(err, '保存供应商失败'))
  } finally {
    saving.value = false
  }
}

function openCreateRoute() {
  editingRouteId.value = null
  routeForm.value = {
    supplierId: suppliers.value[0]?.id || '',
    modelId: models.value[0]?.id || '',
    workflowId: '',
    name: '',
    costPerSecPeak: '0.03',
    costPerSecOffpeak: '0.02',
    enabled: true,
  }
  showRouteForm.value = true
}

function openEditRoute(route: AdminRoute) {
  editingRouteId.value = route.id
  routeForm.value = {
    supplierId: route.supplierId || route.supplierCode,
    modelId: route.modelId || route.modelCode,
    workflowId: route.workflowId,
    name: route.name,
    costPerSecPeak: String(route.costPerSecPeak),
    costPerSecOffpeak: String(route.costPerSecOffpeak),
    enabled: route.enabled,
  }
  showRouteForm.value = true
}

async function saveRoute() {
  saving.value = true
  try {
    const peak = Number(routeForm.value.costPerSecPeak)
    const off = Number(routeForm.value.costPerSecOffpeak)
    if (editingRouteId.value) {
      await updateAdminRoute(editingRouteId.value, {
        name: routeForm.value.name.trim(),
        workflowId: routeForm.value.workflowId.trim(),
        costPerSecPeak: peak,
        costPerSecOffpeak: off,
        enabled: routeForm.value.enabled,
      })
      app.notify('路由已更新')
    } else {
      if (!routeForm.value.supplierId || !routeForm.value.modelId || !routeForm.value.workflowId.trim()) {
        app.notify('请选择供应商、模型并填写上游 Workflow ID')
        return
      }
      await createAdminRoute({
        supplierId: routeForm.value.supplierId,
        modelId: routeForm.value.modelId,
        workflowId: routeForm.value.workflowId.trim(),
        name: routeForm.value.name.trim() || undefined,
        costPerSecPeak: peak,
        costPerSecOffpeak: off,
        enabled: routeForm.value.enabled,
      })
      app.notify('路由已创建')
    }
    showRouteForm.value = false
    await load()
  } catch (err) {
    app.notify(apiErrorMessage(err, '保存路由失败'))
  } finally {
    saving.value = false
  }
}

async function saveModel() {
  saving.value = true
  try {
    if (!modelForm.value.carrierId || !modelForm.value.code.trim() || !modelForm.value.name.trim()) {
      app.notify('请填写承运、模型编码和名称')
      return
    }
    await createAdminModel({
      carrierId: modelForm.value.carrierId,
      code: modelForm.value.code.trim(),
      name: modelForm.value.name.trim(),
      description: modelForm.value.description.trim() || undefined,
      clientWorkflowId: modelForm.value.clientWorkflowId.trim() || undefined,
    })
    app.notify('模型已创建')
    showModelForm.value = false
    modelForm.value = { carrierId: carriers.value[0]?.id || '', code: '', name: '', clientWorkflowId: '', description: '' }
    await load()
  } catch (err) {
    app.notify(apiErrorMessage(err, '创建模型失败'))
  } finally {
    saving.value = false
  }
}

async function toggleSupplier(sp: AdminSupplier) {
  try {
    await updateAdminSupplier(sp.id, { enabled: !sp.enabled })
    app.notify(`${sp.name} 已${!sp.enabled ? '启用' : '停用'}`)
    await load()
  } catch (err) {
    app.notify(apiErrorMessage(err, '更新失败'))
  }
}

async function toggleRoute(route: AdminRoute) {
  try {
    await updateAdminRoute(route.id, { enabled: !route.enabled })
    app.notify(`${route.name} 已${!route.enabled ? '启用' : '停用'}`)
    await load()
  } catch (err) {
    app.notify(apiErrorMessage(err, '更新失败'))
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="模型与供应商">
      <template #description>
        供应商凭据、模型与路由都在这里维护。客户端只看到上架工作流，看不到供应商。
      </template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton size="sm" @click="showModelForm = !showModelForm"><Plus class="h-3.5 w-3.5" />新模型</S2VButton>
        <S2VButton size="sm" @click="openCreateRoute"><Plus class="h-3.5 w-3.5" />新路由</S2VButton>
        <S2VButton variant="primary" size="sm" @click="openCreateSupplier"><Plus class="h-3.5 w-3.5" />新供应商</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-5 grid gap-3 sm:grid-cols-3">
      <S2VCard v-for="item in metrics" :key="item.label" variant="metric">
        <div class="flex items-center justify-between text-xs text-s2v-muted">
          <span>{{ item.label }}</span>
          <component :is="item.icon" class="h-4 w-4 text-s2v-primary" />
        </div>
        <strong class="mt-3 block font-mono text-xl font-bold">{{ item.value }}</strong>
      </S2VCard>
    </div>

    <p v-if="autodlEnvToken" class="mb-4 text-xs text-s2v-muted">环境变量已提供 AutoDL Token 兜底；管理端保存后优先用库内加密值。</p>

    <S2VCard v-if="showSupplierForm" class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">{{ editingSupplierId ? '编辑供应商' : '新增供应商' }}</h2>
          <p class="mt-1 text-xs text-s2v-muted">Token 加密存储；编码创建后不可改。</p>
        </div>
        <S2VButton size="sm" @click="showSupplierForm = false"><X class="h-3.5 w-3.5" />关闭</S2VButton>
      </div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VInput v-model="supplierForm.code" label="编码" placeholder="autodl" :disabled="!!editingSupplierId" />
        <S2VInput v-model="supplierForm.name" label="名称" placeholder="AutoDL" />
        <S2VInput v-model="supplierForm.baseUrl" label="Base URL" placeholder="https://autodl.art" class="sm:col-span-2" />
        <S2VInput v-model="supplierForm.token" label="Token" type="password" :placeholder="editingSupplierId ? '留空保持原 Token' : 'ComfyUI Token'" class="sm:col-span-2" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2">
          <input v-model="supplierForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary" />
          启用供应商
        </label>
        <div class="sm:col-span-2">
          <S2VButton variant="primary" :loading="saving" @click="saveSupplier">保存供应商</S2VButton>
        </div>
      </div>
    </S2VCard>

    <S2VCard v-if="showModelForm" class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">新增模型</h2>
          <p class="mt-1 text-xs text-s2v-muted">模型挂在承运能力下，再被路由和工作流引用。</p>
        </div>
        <S2VButton size="sm" @click="showModelForm = false"><X class="h-3.5 w-3.5" />关闭</S2VButton>
      </div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70">
          <span>承运</span>
          <select v-model="modelForm.carrierId" class="rounded-control border border-s2v-line bg-s2v-canvas px-3 py-2 text-sm font-normal text-s2v-ink">
            <option v-for="c in carriers" :key="c.id" :value="c.id">{{ c.name }} ({{ c.code }})</option>
          </select>
        </label>
        <S2VInput v-model="modelForm.code" label="模型编码" placeholder="text" />
        <S2VInput v-model="modelForm.name" label="名称" placeholder="文生视频" />
        <S2VInput v-model="modelForm.clientWorkflowId" label="关联平台工作流编码" placeholder="wf-h3-text" />
        <S2VInput v-model="modelForm.description" label="说明" class="sm:col-span-2" />
        <div class="sm:col-span-2">
          <S2VButton variant="primary" :loading="saving" @click="saveModel">创建模型</S2VButton>
        </div>
      </div>
    </S2VCard>

    <S2VCard v-if="showRouteForm" class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">{{ editingRouteId ? '编辑路由' : '新增路由' }}</h2>
          <p class="mt-1 text-xs text-s2v-muted">模型 → 供应商上游 workflow_id；成本用于自动选路。</p>
        </div>
        <S2VButton size="sm" @click="showRouteForm = false"><X class="h-3.5 w-3.5" />关闭</S2VButton>
      </div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70">
          <span>供应商</span>
          <select v-model="routeForm.supplierId" class="rounded-control border border-s2v-line bg-s2v-canvas px-3 py-2 text-sm font-normal text-s2v-ink" :disabled="!!editingRouteId">
            <option v-for="sp in suppliers" :key="sp.id" :value="sp.id">{{ sp.name }} ({{ sp.code }})</option>
          </select>
        </label>
        <label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70">
          <span>模型</span>
          <select v-model="routeForm.modelId" class="rounded-control border border-s2v-line bg-s2v-canvas px-3 py-2 text-sm font-normal text-s2v-ink" :disabled="!!editingRouteId">
            <option v-for="m in models" :key="m.id" :value="m.id">{{ m.name }} ({{ m.code }})</option>
          </select>
        </label>
        <S2VInput v-model="routeForm.workflowId" label="上游 Workflow ID" placeholder="minimax_h3_lightx2v" class="sm:col-span-2" />
        <S2VInput v-model="routeForm.name" label="显示名称" placeholder="H3文生" />
        <div class="grid grid-cols-2 gap-3">
          <S2VInput v-model="routeForm.costPerSecPeak" label="峰时成本/秒" type="number" />
          <S2VInput v-model="routeForm.costPerSecOffpeak" label="谷时成本/秒" type="number" />
        </div>
        <label class="flex items-center gap-2 text-xs sm:col-span-2">
          <input v-model="routeForm.enabled" type="checkbox" class="h-4 w-4 accent-s2v-primary" />
          启用路由
        </label>
        <div class="sm:col-span-2">
          <S2VButton variant="primary" :loading="saving" @click="saveRoute">保存路由</S2VButton>
        </div>
      </div>
    </S2VCard>

    <div class="grid gap-4 lg:grid-cols-2">
      <S2VCard v-for="provider in suppliers" :key="provider.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <div class="flex items-center gap-2">
            <span class="s2v-icon-well"><Bot class="h-4 w-4" /></span>
            <div>
              <strong class="block text-xs">{{ provider.name }}</strong>
              <span class="mt-1 block text-[11px] text-s2v-muted">{{ provider.code }} · {{ provider.baseUrl }}</span>
            </div>
          </div>
          <S2VBadge :tone="provider.hasToken && provider.enabled ? 'success' : 'neutral'">{{ provider.status }}</S2VBadge>
        </div>
        <div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-xs text-s2v-muted">
          <div class="flex justify-between"><span>Token</span><strong class="font-mono text-s2v-ink">{{ provider.hasToken ? '已配置' : '未配置' }}</strong></div>
        </div>
        <div class="mt-4 flex gap-2">
          <S2VButton size="sm" class="flex-1" @click="openEditSupplier(provider)"><Pencil class="h-3.5 w-3.5" />编辑</S2VButton>
          <S2VButton size="sm" @click="toggleSupplier(provider)">{{ provider.enabled ? '停用' : '启用' }}</S2VButton>
        </div>
      </S2VCard>
      <p v-if="!suppliers.length" class="s2v-inset-dashed lg:col-span-2">暂无供应商，点击「新供应商」添加。</p>
    </div>

    <S2VCard class="mt-5" variant="padded">
      <div class="mb-3 flex items-center justify-between gap-2">
        <strong class="block text-xs">供应商路由（最低成本自动选路）</strong>
        <S2VButton size="sm" @click="openCreateRoute"><Plus class="h-3.5 w-3.5" />新增</S2VButton>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[720px] text-left text-xs">
          <thead class="text-s2v-muted">
            <tr class="border-b border-s2v-line">
              <th class="py-2 font-medium">名称</th>
              <th class="py-2 font-medium">供应商</th>
              <th class="py-2 font-medium">模型</th>
              <th class="py-2 font-medium">上游 Workflow</th>
              <th class="py-2 font-medium">峰/谷 元每秒</th>
              <th class="py-2 font-medium">状态</th>
              <th class="py-2 font-medium" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="route in routes" :key="route.id" class="border-b border-s2v-line/70">
              <td class="py-2.5">{{ route.name }}</td>
              <td class="py-2.5 font-mono">{{ route.supplierCode }}</td>
              <td class="py-2.5 font-mono">{{ route.modelCode }}</td>
              <td class="py-2.5 font-mono text-[11px]">{{ route.workflowId }}</td>
              <td class="py-2.5 font-mono">{{ route.costPerSecPeak }} / {{ route.costPerSecOffpeak }}</td>
              <td class="py-2.5">
                <S2VBadge :tone="route.enabled ? 'success' : 'neutral'">{{ route.enabled ? '启用' : '停用' }}</S2VBadge>
              </td>
              <td class="py-2.5 text-right">
                <div class="flex justify-end gap-1">
                  <S2VButton size="sm" @click="openEditRoute(route)"><Pencil class="h-3.5 w-3.5" />编辑</S2VButton>
                  <S2VButton size="sm" @click="toggleRoute(route)">{{ route.enabled ? '停用' : '启用' }}</S2VButton>
                </div>
              </td>
            </tr>
            <tr v-if="!routes.length">
              <td colspan="7" class="py-4 text-s2v-muted">暂无路由。</td>
            </tr>
          </tbody>
        </table>
      </div>
    </S2VCard>

    <S2VCard class="mt-5" variant="padded">
      <strong class="mb-3 block text-xs">模型清单</strong>
      <div class="grid gap-2 sm:grid-cols-3">
        <div v-for="m in models" :key="m.id" class="s2v-inset">
          <strong class="block text-xs">{{ m.name }}</strong>
          <span class="mt-1 block font-mono text-[10px] text-s2v-muted">{{ m.code }} · {{ m.clientWorkflowId || '-' }}</span>
        </div>
        <p v-if="!models.length" class="text-xs text-s2v-muted">暂无模型。</p>
      </div>
    </S2VCard>
  </section>
</template>
