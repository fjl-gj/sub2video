<script setup lang="ts">
import { Coins, History, Plus } from 'lucide-vue-next'
import { ref } from 'vue'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const changed = ref(false)

function save() {
  changed.value = false
  app.savePricing()
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="定价规则">
      <template #description>成本价和销售价分开维护，任务创建时会固化价格快照。</template>
      <template #actions>
        <S2VButton @click="app.notify(`当前 ${app.pricingRules.length} 条规则`)"><History class="h-3.5 w-3.5" />版本记录</S2VButton>
        <S2VButton variant="primary" @click="app.addPricingRule(); changed = true"><Plus class="h-3.5 w-3.5" />新增规则</S2VButton>
      </template>
    </S2VPageHeader>

    <S2VCard class="mb-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">秒级费率矩阵</h2>
          <p class="mt-1 text-xs text-s2v-muted">调整后只影响新创建的任务，历史任务不回溯。</p>
        </div>
        <Coins class="h-4 w-4 text-s2v-primary" />
      </div>
      <div class="hidden overflow-x-auto md:block">
        <table class="s2v-table">
          <thead>
            <tr>
              <th>模型 / 接口</th>
              <th>成本价（元/秒）</th>
              <th>刊例价（元/秒）</th>
              <th>计费规则</th>
              <th>状态</th>
              <th>版本</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="rule in app.pricingRules" :key="rule.id">
              <td class="font-semibold text-s2v-ink">{{ rule.name }}</td>
              <td class="font-mono">{{ rule.cost }}</td>
              <td>
                <S2VInput v-model="rule.price" class="w-24" size="sm" @update:model-value="changed = true" />
              </td>
              <td>{{ rule.duration }}</td>
              <td><S2VBadge :tone="rule.status === '生效中' ? 'success' : 'warning'">{{ rule.status }}</S2VBadge></td>
              <td class="font-mono">{{ rule.version }}</td>
            </tr>
            <tr v-if="!app.pricingRules.length">
              <td colspan="6" class="text-center text-s2v-muted">暂无定价规则，等待后端数据接入。</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="grid gap-3 px-4 pb-4 md:hidden">
        <MobileRecordCard v-for="rule in app.pricingRules" :key="rule.id" :title="rule.name" :meta="rule.version">
          <template #badge>
            <S2VBadge :tone="rule.status === '生效中' ? 'success' : 'warning'">{{ rule.status }}</S2VBadge>
          </template>
          <template #body>成本 ¥{{ rule.cost }}/秒 · {{ rule.duration }}</template>
          <template #actions>
            <S2VInput v-model="rule.price" class="max-w-28" size="sm" @update:model-value="changed = true" />
          </template>
        </MobileRecordCard>
        <p v-if="!app.pricingRules.length" class="text-xs text-s2v-muted">暂无定价规则。</p>
      </div>
      <div v-if="changed" class="flex items-center justify-between border-t border-s2v-line bg-s2v-warning/10 px-4 py-3 text-xs text-s2v-warning">
        <span>有未保存的价格调整</span>
        <S2VButton size="sm" variant="primary" @click="save">保存规则</S2VButton>
      </div>
    </S2VCard>
  </section>
</template>
