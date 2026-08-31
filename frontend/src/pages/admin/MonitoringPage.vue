<script setup lang="ts">
import { Activity, Cpu, Gauge, HardDrive, RefreshCw, Server, Wifi } from 'lucide-vue-next'
import { providers } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VProgress from '@/shared/ui/S2VProgress.vue'

const app = useAppStore()
const metrics = [
  { label: 'GPU 利用率', value: '75%', icon: Cpu },
  { label: '任务队列', value: '3', icon: Activity },
  { label: '平均响应', value: '15.7s', icon: Gauge },
  { label: '存储健康', value: '99.99%', icon: HardDrive },
]
const trends = [
  { label: 'GPU 负载', value: '75%', percent: 75 },
  { label: '队列吞吐', value: '68%', percent: 68 },
  { label: '存储用量', value: '38%', percent: 38 },
  { label: '错误率', value: '3.6%', percent: 3.6 },
]
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="GPU Clusters" kicker="监控全网智算节点显存占用与实时执行队列" title="运维监控">
      <template #actions>
        <S2VButton @click="app.notify('监控数据已刷新')"><RefreshCw class="h-3.5 w-3.5" />刷新数据</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <S2VCard v-for="item in metrics" :key="item.label" variant="metric">
        <div class="flex items-center justify-between text-xs text-s2v-muted">
          <span>{{ item.label }}</span>
          <component :is="item.icon" class="h-4 w-4 text-s2v-primary" />
        </div>
        <strong class="s2v-stat-value">{{ item.value }}</strong>
        <span class="s2v-stat-note">过去 15 分钟</span>
      </S2VCard>
    </div>

    <div class="mt-5 grid gap-4 lg:grid-cols-2">
      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">供应商健康</h2>
            <p class="mt-1 text-xs text-s2v-muted">每 60 秒自动检查一次</p>
          </div>
          <Wifi class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="divide-y divide-s2v-line/70 px-4">
          <div v-for="provider in providers" :key="provider.name" class="flex items-center gap-3 py-4">
            <span class="s2v-icon-well s2v-icon-well-success"><Server class="h-4 w-4" /></span>
            <div class="min-w-0 flex-1">
              <strong class="block text-xs">{{ provider.name }}</strong>
              <span class="mt-1 block text-[11px] text-s2v-muted">延迟 {{ provider.latency }} · 并发 {{ provider.concurrency }}</span>
            </div>
            <S2VBadge :tone="provider.status === '健康' ? 'success' : 'warning'">{{ provider.status }}</S2VBadge>
          </div>
        </div>
      </S2VCard>

      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">资源趋势</h2>
            <p class="mt-1 text-xs text-s2v-muted">过去 6 小时的标准化指标</p>
          </div>
        </div>
        <div class="s2v-panel-body grid gap-5">
          <div v-for="item in trends" :key="item.label">
            <div class="mb-2 flex justify-between text-xs text-s2v-muted">
              <span>{{ item.label }}</span>
              <strong class="font-mono text-s2v-ink">{{ item.value }}</strong>
            </div>
            <S2VProgress :value="item.percent" />
          </div>
        </div>
      </S2VCard>
    </div>
  </section>
</template>
