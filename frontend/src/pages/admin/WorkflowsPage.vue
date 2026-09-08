<script setup lang="ts">
import { Pencil, RefreshCw, Settings2 } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { fetchAdminCreationModes, updateAdminCreationMode, type AdminCreationMode } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const items = ref<AdminCreationMode[]>([])
const loading = ref(false)
const saving = ref(false)
const editingID = ref('')
const form = ref({ name: '', description: '', sortOrder: '1' })

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminCreationModes()
    items.value = data.items
  } catch (error) {
    app.notify(apiErrorMessage(error, '加载工作流失败'))
  } finally {
    loading.value = false
  }
}

function edit(item: AdminCreationMode) {
  editingID.value = item.id
  form.value = { name: item.name, description: item.description, sortOrder: String(item.sortOrder) }
}

async function save(item: AdminCreationMode) {
  const sortOrder = Number(form.value.sortOrder)
  if (!form.value.name.trim() || !Number.isInteger(sortOrder)) {
    app.notify('请填写工作流名称和排序')
    return
  }
  saving.value = true
  try {
    await updateAdminCreationMode(item.id, {
      name: form.value.name.trim(),
      description: form.value.description.trim(),
      sortOrder,
    })
    editingID.value = ''
    app.notify('工作流展示已更新')
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新工作流失败'))
  } finally {
    saving.value = false
  }
}

async function toggle(item: AdminCreationMode) {
  saving.value = true
  try {
    await updateAdminCreationMode(item.id, { status: item.status === 'published' ? 'disabled' : 'published' })
    await load()
  } catch (error) {
    app.notify(apiErrorMessage(error, '更新工作流状态失败'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="工作流">
      <template #description>工作流只定义客户端输入模式。系统固定为文生视频、首尾帧生成视频和多图生成视频；模型支持、能力与售价在模型配置中维护。</template>
      <template #actions>
        <S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 lg:grid-cols-3">
      <S2VCard v-for="item in items" :key="item.id" variant="padded">
        <div class="flex items-start justify-between gap-3">
          <span class="s2v-icon-well s2v-icon-well-nav"><Settings2 class="h-4 w-4" /></span>
          <S2VBadge :tone="item.status === 'published' ? 'success' : 'neutral'">{{ item.status === 'published' ? '已上架' : '已停用' }}</S2VBadge>
        </div>

        <template v-if="editingID !== item.id">
          <h2 class="mt-4 text-sm font-bold">{{ item.name }}</h2>
          <p class="mt-1.5 min-h-10 text-xs leading-5 text-s2v-muted">{{ item.description }}</p>
          <p class="mt-4 border-t border-s2v-line pt-3 text-xs text-s2v-muted">固定客户端模式，不关联承运商、模型、供应商或调用配置。</p>
          <div class="mt-4 flex gap-2">
            <S2VButton size="sm" title="编辑工作流展示" @click="edit(item)"><Pencil class="h-3.5 w-3.5" />编辑</S2VButton>
            <S2VButton size="sm" :loading="saving" @click="toggle(item)">{{ item.status === 'published' ? '停用' : '启用' }}</S2VButton>
          </div>
        </template>

        <template v-else>
          <div class="mt-4 grid gap-3">
            <S2VInput v-model="form.name" label="客户端名称" />
            <S2VInput v-model="form.description" label="说明" />
            <S2VInput v-model="form.sortOrder" label="排序" type="number" />
          </div>
          <div class="mt-4 flex gap-2">
            <S2VButton variant="primary" size="sm" :loading="saving" @click="save(item)">保存</S2VButton>
            <S2VButton size="sm" @click="editingID = ''">取消</S2VButton>
          </div>
        </template>
      </S2VCard>
    </div>
  </section>
</template>
