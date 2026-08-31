<script setup lang="ts">
import { Activity, Bot, KeyRound, Plus, RefreshCw, Server } from 'lucide-vue-next'
import { providers } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const metrics = [
  { label: '已接入供应商', value: '2', icon: Server },
  { label: '可用模型', value: '3', icon: Bot },
  { label: '平均延迟', value: '15.7s', icon: Activity },
]
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Providers" title="模型与供应商">
      <template #description>供应商凭据只在管理端使用，为工作流提供健康的生成能力。</template>
      <template #actions>
        <S2VButton variant="primary" @click="app.notify('供应商接入表单将在后端接入后开放')">
          <template #icon><Plus class="h-4 w-4" /></template>
          添加供应商
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

    <div class="grid gap-4 lg:grid-cols-3">
      <S2VCard v-for="provider in providers" :key="provider.name" variant="padded">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-2">
            <span class="grid h-8 w-8 place-items-center rounded-full bg-s2v-primary-soft text-s2v-primary"><Bot class="h-4 w-4" /></span>
            <div>
              <strong class="block text-xs">{{ provider.name }}</strong>
              <span class="mt-1 block text-[11px] text-s2v-muted">{{ provider.channel }}</span>
            </div>
          </div>
          <S2VBadge :tone="provider.status === '健康' ? 'success' : 'neutral'">{{ provider.status }}</S2VBadge>
        </div>
        <div class="mt-4 grid gap-2 border-t border-s2v-line pt-3 text-xs">
          <div class="flex justify-between text-s2v-muted"><span>并发</span><strong class="font-mono text-s2v-ink">{{ provider.concurrency }}</strong></div>
          <div class="flex justify-between text-s2v-muted"><span>延迟</span><strong class="font-mono text-s2v-ink">{{ provider.latency }}</strong></div>
          <div class="flex justify-between text-s2v-muted"><span>成本</span><strong class="font-mono text-s2v-ink">{{ provider.cost }}</strong></div>
        </div>
        <div class="mt-4 flex gap-2">
          <S2VButton size="sm" class="flex-1" @click="app.notify(`已刷新 ${provider.name} 健康状态`)"><RefreshCw class="h-3.5 w-3.5" />健康检查</S2VButton>
          <S2VButton size="sm" aria-label="凭据设置" @click="app.notify('凭据由管理端安全存储')"><KeyRound class="h-3.5 w-3.5" /></S2VButton>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
