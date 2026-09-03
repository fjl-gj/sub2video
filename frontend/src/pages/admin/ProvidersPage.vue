<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Bot, KeyRound, RefreshCw, Server } from 'lucide-vue-next'
import {
  fetchAdminCarriers,
  fetchAdminRoutes,
  fetchAdminSuppliers,
  updateAdminRoute,
  updateAdminSupplier,
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
const suppliers = ref<AdminSupplier[]>([])
const routes = ref<AdminRoute[]>([])
const cosConfigured = ref(false)
const autodlEnvToken = ref(false)
const tokenDraft = ref<Record<string, string>>({})

const metrics = computed(() => [
  { label: '已接入供应商', value: String(suppliers.value.filter((item) => item.enabled).length), icon: Server },
  { label: '启用路由', value: String(routes.value.filter((item) => item.enabled).length), icon: Bot },
  { label: 'COS', value: cosConfigured.value ? '已配置' : '待接入', icon: KeyRound },
])

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
    cosConfigured.value = c.cosConfigured
    autodlEnvToken.value = c.autodlEnvToken
  } catch (err) {
    app.notify(err instanceof Error ? err.message : '加载供应商失败（需 admin 角色）')
  } finally {
    loading.value = false
  }
}

async function saveToken(sp: AdminSupplier) {
  const token = (tokenDraft.value[sp.id] || '').trim()
  try {
    await updateAdminSupplier(sp.id, {
      token: token || undefined,
      enabled: sp.enabled,
    })
    tokenDraft.value[sp.id] = ''
    app.notify(token ? `已保存 ${sp.name} Token` : `已更新 ${sp.name}`)
    await load()
  } catch (err) {
    app.notify(err instanceof Error ? err.message : '保存失败')
  }
}

async function toggleSupplier(sp: AdminSupplier) {
  try {
    await updateAdminSupplier(sp.id, { enabled: !sp.enabled })
    app.notify(`${sp.name} 已${!sp.enabled ? '启用' : '停用'}`)
    await load()
  } catch (err) {
    app.notify(err instanceof Error ? err.message : '更新失败')
  }
}

async function toggleRoute(route: AdminRoute) {
  try {
    await updateAdminRoute(route.id, { enabled: !route.enabled })
    app.notify(`${route.name} 已${!route.enabled ? '启用' : '停用'}`)
    await load()
  } catch (err) {
    app.notify(err instanceof Error ? err.message : '更新失败')
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="模型与供应商">
      <template #description>
        供应商 Token 仅管理端可见；未配置时文生任务自动降级模拟器。COS 就绪后再开通图生。
      </template>
      <template #actions>
        <S2VButton variant="primary" :disabled="loading" @click="load">
          <template #icon><RefreshCw class="h-4 w-4" /></template>
          刷新
        </S2VButton>
      </template>
    </S2VPageHeader>

    <div class="mb-5 grid gap-3 sm:grid-cols-3">
      <S2VCard v-for="item in metrics" :key="item.label" variant="metric">
        <div class="flex items-center justify-between text-xs text-s2v-muted">
          <span>{{ item.label }}</span>
          <component :is="item.icon" class="h-4 w-4 text-s2v-primary" />
        </div>
        <strong class="mt-3 block text-xl font-bold font-mono">{{ item.value }}</strong>
      </S2VCard>
    </div>

    <p v-if="autodlEnvToken" class="mb-4 text-xs text-s2v-muted">环境变量已提供 AutoDL Token 兜底；管理端保存后优先用库内加密值。</p>

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
        <div class="mt-3">
          <S2VInput
            :model-value="tokenDraft[provider.id] || ''"
            type="password"
            size="sm"
            label="更新 Token（留空则保持原值）"
            placeholder="ComfyUI Token"
            @update:model-value="tokenDraft[provider.id] = $event"
          />
        </div>
        <div class="mt-4 flex gap-2">
          <S2VButton size="sm" class="flex-1" @click="saveToken(provider)">
            <KeyRound class="h-3.5 w-3.5" />保存凭据
          </S2VButton>
          <S2VButton size="sm" @click="toggleSupplier(provider)">
            {{ provider.enabled ? '停用' : '启用' }}
          </S2VButton>
        </div>
      </S2VCard>
    </div>

    <S2VCard class="mt-5" variant="padded">
      <strong class="mb-3 block text-xs">供应商路由（最低价自动选路）</strong>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[640px] text-left text-xs">
          <thead class="text-s2v-muted">
            <tr class="border-b border-s2v-line">
              <th class="py-2 font-medium">名称</th>
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
              <td class="py-2.5 font-mono">{{ route.modelCode }}</td>
              <td class="py-2.5 font-mono text-[11px]">{{ route.workflowId }}</td>
              <td class="py-2.5 font-mono">{{ route.costPerSecPeak }} / {{ route.costPerSecOffpeak }}</td>
              <td class="py-2.5">
                <S2VBadge :tone="route.enabled ? 'success' : 'neutral'">{{ route.enabled ? '启用' : '停用' }}</S2VBadge>
              </td>
              <td class="py-2.5 text-right">
                <S2VButton size="sm" @click="toggleRoute(route)">{{ route.enabled ? '停用' : '启用' }}</S2VButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </S2VCard>
  </section>
</template>
