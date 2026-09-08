<script setup lang="ts">
import { Coins, RefreshCw } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { fetchAdminRetailPricing, type AdminRetailPricing } from '@/api/admin'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const rules = ref<AdminRetailPricing[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const data = await fetchAdminRetailPricing()
    rules.value = data.items
  } catch (error) {
    const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
    app.notify(err.response?.data?.message || err.response?.data?.error || err.message || '加载价格审计失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="价格审计">
      <template #description>平台售价由承运商与模型中的模型配置维护。时长是每秒售价的计费数量；此页只读，用于核查当前对客规则和版本。</template>
      <template #actions><S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton></template>
    </S2VPageHeader>

    <S2VCard>
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">当前对客售价</h2><p class="mt-1 text-xs text-s2v-muted">供应商、采购成本和上游执行标识不在本页展示。修改售价请进入对应模型的模型配置。</p></div><Coins class="h-4 w-4 text-s2v-primary" /></div>
      <div class="hidden overflow-x-auto md:block"><table class="s2v-table"><thead><tr><th>工作流</th><th>承运商 / 模型</th><th>分辨率</th><th>自动音频</th><th>售价（元/秒）</th><th>状态</th><th>版本</th></tr></thead><tbody><tr v-for="rule in rules" :key="rule.id"><td class="font-semibold text-s2v-ink">{{ rule.modeName }}</td><td>{{ rule.carrierName }} / {{ rule.modelName }}</td><td class="font-mono">{{ rule.resolution }}</td><td>{{ rule.generateAudio ? '支持' : '不支持' }}</td><td class="font-mono">{{ rule.pricePerSecond.toFixed(2) }}</td><td><S2VBadge :tone="rule.status === 'active' ? 'success' : 'neutral'">{{ rule.status === 'active' ? '生效中' : rule.status }}</S2VBadge></td><td class="font-mono">{{ rule.version }}</td></tr><tr v-if="!rules.length"><td colspan="7" class="text-center text-s2v-muted">暂无售价规则。</td></tr></tbody></table></div>
      <div class="grid gap-3 px-4 pb-4 md:hidden"><MobileRecordCard v-for="rule in rules" :key="rule.id" :title="rule.modeName" :meta="`${rule.carrierName} / ${rule.modelName}`"><template #badge><S2VBadge :tone="rule.status === 'active' ? 'success' : 'neutral'">{{ rule.status }}</S2VBadge></template><template #body>{{ rule.resolution }} · {{ rule.generateAudio ? '自动音频' : '无自动音频' }} · ¥ {{ rule.pricePerSecond.toFixed(2) }}/秒</template><template #actions><span class="font-mono text-xs text-s2v-muted">{{ rule.version }}</span></template></MobileRecordCard></div>
    </S2VCard>
  </section>
</template>
