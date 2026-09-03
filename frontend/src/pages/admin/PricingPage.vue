<script setup lang="ts">
import { Coins, RefreshCw } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { fetchAdminPricing, updateAdminPricing, type AdminPricingRule } from '@/api/admin'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const rules = ref<AdminPricingRule[]>([])
const drafts = ref<Record<string, string>>({})
const changed = ref(false)
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminPricing()
    rules.value = data.items
    drafts.value = Object.fromEntries(data.items.map((r) => [r.id, r.price]))
    changed.value = false
  } catch (e: unknown) {
    app.notify(e instanceof Error ? e.message : '加载定价失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    for (const rule of rules.value) {
      const next = Number(drafts.value[rule.id])
      if (!Number.isFinite(next) || Math.abs(next - rule.pricePerSec) < 0.0001) continue
      await updateAdminPricing(rule.id, { pricePerSec: next })
    }
    app.notify('定价已保存，仅影响新任务')
    await load()
  } catch (e: unknown) {
    app.notify(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="定价规则">
      <template #description>售价挂在平台工作流上；成本在供应商路由中，任务创建时固化价格快照。</template>
      <template #actions>
        <S2VButton :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">秒级费率矩阵</h2>
          <p class="mt-1 text-xs text-s2v-muted">成本列为参考进货价；刊例价为对客售价。</p>
        </div>
        <Coins class="h-4 w-4 text-s2v-primary" />
      </div>
      <div class="hidden overflow-x-auto md:block">
        <table class="s2v-table">
          <thead>
            <tr>
              <th>工作流 / 分辨率</th>
              <th>成本参考（元/秒）</th>
              <th>刊例价（元/秒）</th>
              <th>计费规则</th>
              <th>状态</th>
              <th>版本</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="rule in rules" :key="rule.id">
              <td class="font-semibold text-s2v-ink">{{ rule.name }}</td>
              <td class="font-mono">{{ rule.cost }}</td>
              <td>
                <S2VInput
                  :model-value="drafts[rule.id] || rule.price"
                  class="w-24"
                  size="sm"
                  @update:model-value="drafts[rule.id] = $event; changed = true"
                />
              </td>
              <td>{{ rule.duration }}</td>
              <td><S2VBadge :tone="rule.status === '生效中' ? 'success' : 'warning'">{{ rule.status }}</S2VBadge></td>
              <td class="font-mono">{{ rule.version }}</td>
            </tr>
            <tr v-if="!rules.length">
              <td colspan="6" class="text-center text-s2v-muted">暂无定价规则。</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="grid gap-3 px-4 pb-4 md:hidden">
        <MobileRecordCard v-for="rule in rules" :key="rule.id" :title="rule.name" :meta="rule.version">
          <template #badge>
            <S2VBadge :tone="rule.status === '生效中' ? 'success' : 'warning'">{{ rule.status }}</S2VBadge>
          </template>
          <template #body>成本参考 ¥{{ rule.cost }}/秒 · {{ rule.duration }}</template>
          <template #actions>
            <S2VInput
              :model-value="drafts[rule.id] || rule.price"
              class="max-w-28"
              size="sm"
              @update:model-value="drafts[rule.id] = $event; changed = true"
            />
          </template>
        </MobileRecordCard>
        <p v-if="!rules.length" class="text-xs text-s2v-muted">暂无定价规则。</p>
      </div>
      <div v-if="changed" class="flex items-center justify-between border-t border-s2v-line bg-s2v-warning/10 px-4 py-3 text-xs text-s2v-warning">
        <span>有未保存的价格调整</span>
        <S2VButton size="sm" variant="primary" :loading="saving" @click="save">保存规则</S2VButton>
      </div>
    </S2VCard>
  </section>
</template>
