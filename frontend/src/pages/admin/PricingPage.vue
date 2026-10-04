<script setup lang="ts">
import { Coins, Pencil, RefreshCw } from 'lucide-vue-next'
import { onMounted, ref, watch } from 'vue'
import { fetchAdminCreationModes, fetchAdminRetailPricing, upsertAdminRetailPricing, type AdminCreationMode, type AdminRetailPricing } from '@/api/admin'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VDialog from '@/shared/ui/S2VDialog.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'
import S2VSelect from '@/shared/ui/S2VSelect.vue'

const app = useAppStore()
const rules = ref<AdminRetailPricing[]>([])
const modes = ref<AdminCreationMode[]>([])
const loading = ref(false)
const loadError = ref('')
const saveError = ref('')
let loadSequence = 0
const saving = ref(false)
const page = ref(1)
const pageSize = 10
const total = ref(0)
const editorOpen = ref(false)
const editing = ref<AdminRetailPricing | null>(null)
const form = ref({ modeId: '', resolution: '768P', generateAudio: false, name: '', pricePerSecond: '0.03', status: 'active' })

const modeOptions = () => modes.value.filter((item) => item.status === 'published' || item.id === editing.value?.modeId).map((item) => ({ label: item.name, value: item.id }))
const resolutionOptions = ['480P', '736P', '768P', '1080P'].map((value) => ({ label: value, value }))

async function load() {
  const sequence = ++loadSequence
  loading.value = true
  loadError.value = ''
  try {
    const [data, modeData] = await Promise.all([fetchAdminRetailPricing({ page: page.value, pageSize }), fetchAdminCreationModes()])
    if (sequence !== loadSequence) return
    rules.value = data.items
    total.value = data.total ?? data.items.length
    modes.value = modeData.items
    if (page.value > Math.max(1, Math.ceil(total.value / pageSize))) {
      page.value = 1
      return
    }
  } catch (error) {
    if (sequence !== loadSequence) return
    const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
    loadError.value = err.response?.data?.message || err.response?.data?.error || err.message || '加载价格审计失败'
    rules.value = []
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

function openEditor(rule?: AdminRetailPricing) {
  if (rule?.modelId) {
    app.notify('历史模型售价仅供查阅，请维护模式级平台售价')
    return
  }
  saveError.value = ''
  editing.value = rule || null
  form.value = {
    modeId: rule?.modeId || modeOptions()[0]?.value || '',
    resolution: rule?.resolution || '768P',
    generateAudio: rule?.generateAudio ?? false,
    name: rule?.name || '',
    pricePerSecond: String(rule?.pricePerSecond ?? 0.03),
    status: rule?.status || 'active',
  }
  editorOpen.value = true
}

async function save() {
  if (saving.value) return
  const price = Number(form.value.pricePerSecond)
  if (!form.value.modeId || !form.value.resolution || !form.value.pricePerSecond.trim() || !Number.isFinite(price) || price < 0) {
    saveError.value = '请填写模式、分辨率和有效售价'
    return
  }
  saving.value = true
  saveError.value = ''
  try {
    await upsertAdminRetailPricing({
      modeId: editing.value?.modeId ?? form.value.modeId,
      resolution: editing.value?.resolution ?? form.value.resolution,
      generateAudio: editing.value?.generateAudio ?? form.value.generateAudio,
      name: form.value.name.trim() || '模式售价',
      pricePerSecond: price,
      status: form.value.status,
    })
    editorOpen.value = false
    app.notify('平台售价已保存')
    await load()
  } catch (error) {
    const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
    saveError.value = err.response?.data?.message || err.response?.data?.error || err.message || '保存平台售价失败'
  } finally {
    saving.value = false
  }
}

watch(page, () => { void load() })

onMounted(load)
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="价格审计">
      <template #description>平台售价按创作模式、分辨率和自动音频维护。时长是每秒售价的计费数量。</template>
      <template #actions><S2VButton size="sm" :disabled="loading" @click="load"><RefreshCw class="h-3.5 w-3.5" />刷新</S2VButton><S2VButton variant="primary" size="sm" @click="openEditor()"><Pencil class="h-3.5 w-3.5" />新增售价</S2VButton></template>
    </S2VPageHeader>

    <S2VDialog v-model:open="editorOpen" :title="`${editing ? '编辑' : '新增'}平台售价`" description="平台售价只绑定创作模式、分辨率和自动音频，不绑定模型或供应商。" hide-actions>
      <div class="grid gap-3 sm:grid-cols-2">
        <S2VSelect v-model="form.modeId" label="创作模式" :options="modeOptions()" :disabled="Boolean(editing)" />
        <S2VSelect v-model="form.resolution" label="分辨率" :options="resolutionOptions" :disabled="Boolean(editing)" />
        <S2VInput v-model="form.pricePerSecond" label="售价（元/秒）" type="number" min="0" step="0.01" />
        <S2VInput v-model="form.name" label="规则名称" placeholder="多图生成视频 · 768P" />
        <label class="flex items-center gap-2 text-xs sm:col-span-2"><input v-model="form.generateAudio" type="checkbox" :disabled="Boolean(editing)" class="h-4 w-4 accent-s2v-primary">包含自动生成音频</label>
      </div>
      <p v-if="saveError" role="alert" class="s2v-note mt-4">{{ saveError }}</p>
      <div class="mt-5 flex justify-end gap-2"><S2VButton :disabled="saving" @click="editorOpen = false">取消</S2VButton><S2VButton variant="primary" :loading="saving" @click="save">保存售价</S2VButton></div>
    </S2VDialog>

    <p v-if="loading" role="status" class="s2v-note">正在加载平台售价…</p>
    <div v-else-if="loadError" role="alert" class="s2v-note grid gap-3"><p>{{ loadError }}</p><S2VButton @click="load">重试加载</S2VButton></div>
    <S2VCard v-else>
      <div class="s2v-panel-header"><div><h2 class="s2v-section-title">当前对客售价</h2><p class="mt-1 text-xs text-s2v-muted">供应商、采购成本和上游执行标识不在本页展示；售价只绑定模式、分辨率和自动音频。</p></div><Coins class="h-4 w-4 text-s2v-primary" /></div>
      <div class="hidden overflow-x-auto md:block"><table class="s2v-table"><thead><tr><th>创作模式</th><th>模型绑定</th><th>分辨率</th><th>自动音频</th><th>售价（元/秒）</th><th>状态</th><th>版本</th><th></th></tr></thead><tbody><tr v-for="rule in rules" :key="rule.id"><td class="font-semibold text-s2v-ink">{{ rule.modeName }}</td><td>{{ rule.modelId ? `${rule.carrierName} / ${rule.modelName}` : '全模式统一售价' }}</td><td class="font-mono">{{ rule.resolution }}</td><td>{{ rule.generateAudio ? '支持' : '不支持' }}</td><td class="font-mono">{{ rule.pricePerSecond.toFixed(2) }}</td><td><S2VBadge :tone="rule.status === 'active' ? 'success' : 'neutral'">{{ rule.status === 'active' ? '生效中' : rule.status }}</S2VBadge></td><td class="font-mono">{{ rule.version }}</td><td class="text-right"><S2VButton size="sm" variant="ghost" :title="rule.modelId ? '历史模型售价仅供查阅' : '编辑售价'" :disabled="Boolean(rule.modelId)" @click="openEditor(rule)"><Pencil class="h-3.5 w-3.5" /></S2VButton></td></tr><tr v-if="!rules.length"><td colspan="8" class="text-center text-s2v-muted">暂无售价规则。</td></tr></tbody></table></div>
      <div class="grid gap-3 px-4 pb-4 md:hidden"><MobileRecordCard v-for="rule in rules" :key="rule.id" :title="rule.modeName" :meta="rule.modelId ? `${rule.carrierName} / ${rule.modelName}` : '全模式统一售价'"><template #badge><S2VBadge :tone="rule.status === 'active' ? 'success' : 'neutral'">{{ rule.status }}</S2VBadge></template><template #body>{{ rule.resolution }} · {{ rule.generateAudio ? '自动音频' : '无自动音频' }} · ¥ {{ rule.pricePerSecond.toFixed(2) }}/秒</template><template #actions><S2VButton size="sm" variant="ghost" :title="rule.modelId ? '历史模型售价仅供查阅' : '编辑售价'" :disabled="Boolean(rule.modelId)" @click="openEditor(rule)"><Pencil class="h-3.5 w-3.5" /></S2VButton></template></MobileRecordCard></div>
      <div class="px-4 pb-4"><S2VPagination v-if="total" v-model="page" :total="total" :page-size="pageSize" /></div>
    </S2VCard>
  </section>
</template>
