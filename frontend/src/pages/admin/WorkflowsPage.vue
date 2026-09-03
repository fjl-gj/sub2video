<script setup lang="ts">
import { Edit3, Plus, RefreshCw, Workflow, X } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import {
  createAdminWorkflow,
  fetchAdminCarriers,
  fetchAdminWorkflows,
  updateAdminWorkflow,
  type AdminModel,
  type AdminWorkflow,
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VTextarea from '@/shared/ui/S2VTextarea.vue'

const app = useAppStore()
const items = ref<AdminWorkflow[]>([])
const models = ref<AdminModel[]>([])
const loading = ref(false)
const saving = ref(false)
const showForm = ref(false)
const editingId = ref<string | null>(null)
const form = ref({
  code: '',
  name: '',
  description: '',
  modelId: '',
  status: 'draft',
  badge: '',
})

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const [data, carriers] = await Promise.all([fetchAdminWorkflows(), fetchAdminCarriers()])
    items.value = data.items
    models.value = carriers.models
    if (!form.value.modelId && carriers.models[0]) form.value.modelId = carriers.models[0].id
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '加载工作流失败'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = {
    code: '',
    name: '',
    description: '',
    modelId: models.value[0]?.id || '',
    status: 'draft',
    badge: '',
  }
  showForm.value = true
}

function openEdit(item: AdminWorkflow) {
  editingId.value = item.id
  form.value = {
    code: item.code,
    name: item.name,
    description: item.description,
    modelId: item.modelId || models.value.find((m) => m.code === item.modelCode)?.id || '',
    status: item.statusRaw || 'draft',
    badge: item.badge || '',
  }
  showForm.value = true
}

async function save() {
  saving.value = true
  try {
    if (editingId.value) {
      await updateAdminWorkflow(editingId.value, {
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        modelId: form.value.modelId,
        status: form.value.status,
        badge: form.value.badge.trim(),
      })
      app.notify('工作流已更新')
    } else {
      if (!form.value.code.trim() || !form.value.name.trim() || !form.value.modelId) {
        app.notify('请填写编码、名称并选择模型')
        return
      }
      await createAdminWorkflow({
        code: form.value.code.trim(),
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        modelId: form.value.modelId,
        status: form.value.status,
        badge: form.value.badge.trim() || undefined,
      })
      app.notify('工作流已创建')
    }
    showForm.value = false
    await load()
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '保存失败'))
  } finally {
    saving.value = false
  }
}

async function toggle(item: AdminWorkflow) {
  const next = item.statusRaw === 'published' ? 'draft' : 'published'
  try {
    await updateAdminWorkflow(item.id, { status: next })
    app.notify(`${item.name} 已${next === 'published' ? '上架' : '下架'}`)
    await load()
  } catch (e: unknown) {
    app.notify(apiErrorMessage(e, '更新失败'))
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="工作流">
      <template #description>平台工作流对客户端上架；经绑定模型走供应商路由到上游能力。售价在「定价规则」配置。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
        <S2VButton variant="primary" size="sm" @click="openCreate"><Plus class="h-3.5 w-3.5" />新建工作流</S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard v-if="showForm" class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">{{ editingId ? '编辑工作流' : '新建工作流' }}</h2>
          <p class="mt-1 text-xs text-s2v-muted">编码创建后固定；上架前请确保对应模型有启用路由和定价。</p>
        </div>
        <S2VButton size="sm" @click="showForm = false"><X class="h-3.5 w-3.5" />关闭</S2VButton>
      </div>
      <div class="s2v-panel-body grid gap-3 sm:grid-cols-2">
        <S2VInput v-model="form.code" label="编码" placeholder="wf-h3-text" :disabled="!!editingId" />
        <S2VInput v-model="form.name" label="名称" placeholder="文生视频" />
        <label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70">
          <span>绑定模型</span>
          <select v-model="form.modelId" class="rounded-control border border-s2v-line bg-s2v-canvas px-3 py-2 text-sm font-normal text-s2v-ink">
            <option v-for="m in models" :key="m.id" :value="m.id">{{ m.name }} ({{ m.code }})</option>
          </select>
        </label>
        <label class="grid gap-1.5 text-xs font-semibold text-s2v-ink/70">
          <span>状态</span>
          <select v-model="form.status" class="rounded-control border border-s2v-line bg-s2v-canvas px-3 py-2 text-sm font-normal text-s2v-ink">
            <option value="draft">草稿</option>
            <option value="published">已上架</option>
            <option value="disabled">未开放</option>
          </select>
        </label>
        <S2VInput v-model="form.badge" label="角标（可选）" placeholder="即将开放" />
        <div class="sm:col-span-2">
          <S2VTextarea v-model="form.description" label="说明" placeholder="客户端看到的能力描述" :rows="3" />
        </div>
        <div class="sm:col-span-2">
          <S2VButton variant="primary" :loading="saving" @click="save">保存工作流</S2VButton>
        </div>
      </div>
    </S2VCard>

    <div class="grid gap-4 lg:grid-cols-3">
      <S2VCard v-for="item in items" :key="item.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <span class="s2v-icon-well s2v-icon-well-nav"><Workflow class="h-4 w-4" /></span>
          <S2VBadge :tone="item.status === '已上架' ? 'success' : item.status === '草稿' ? 'warning' : 'neutral'">{{ item.status }}</S2VBadge>
        </div>
        <h2 class="mt-4 text-sm font-bold">{{ item.name }}</h2>
        <p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ item.description }}</p>
        <div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-[11px] text-s2v-muted">
          <div class="flex justify-between"><span>承运 / 模型</span><strong class="text-s2v-ink">{{ item.carrier }} / {{ item.modelCode }}</strong></div>
          <div class="flex justify-between"><span>售价</span><strong class="font-mono text-s2v-ink">{{ item.price }}</strong></div>
          <div class="flex justify-between"><span>编码</span><span class="font-mono">{{ item.code }}</span></div>
        </div>
        <div class="mt-4 flex gap-2">
          <S2VButton class="flex-1" size="sm" @click="openEdit(item)"><Edit3 class="h-3.5 w-3.5" />编辑</S2VButton>
          <S2VButton size="sm" @click="toggle(item)">{{ item.statusRaw === 'published' ? '下架' : '上架' }}</S2VButton>
        </div>
      </S2VCard>
    </div>
    <p v-if="!items.length" class="s2v-inset-dashed">暂无平台工作流，点击「新建工作流」添加。</p>
  </section>
</template>
