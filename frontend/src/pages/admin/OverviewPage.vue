<script setup lang="ts">
import { Cpu, RefreshCw } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const router = useRouter()
const nodes = [
  { name: '上海张江 H100 集群', detail: '16 卡在线 · 负载 68% · 延迟 12ms' },
  { name: '北京亦庄 A100 集群', detail: '12 卡在线 · 负载 45% · 延迟 18ms' },
  { name: '前端与多模态网关', detail: '4 节点 · 延迟 24ms · 吞吐 420 音轨/分钟' },
]
const metrics = [
  { label: '全网今日产出视频', value: '1,842', unit: '支', note: '较昨日增长 +18.4%' },
  { label: '今日充值实收流水 (GMV)', value: '¥ 18,920.00', note: '含活动赠送补贴成本 ¥1,420.00', tone: 'success' as const },
  { label: '推理预扣净结算额 (Revenue)', value: '¥ 5,120.00', note: '全网异常退款率 0.42% (原路退还)', tone: 'primary' as const },
  { label: 'GPU 算力集群总并发负载', value: '24 / 32', unit: '卡', note: '综合集群利用率 75.0% · 极佳' },
]
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Dashboard · BI" kicker="全站吞吐、算力消耗与财务收益实时中枢" title="运行大盘">
      <template #actions>
        <S2VButton @click="router.push('/admin/monitoring')"><RefreshCw class="h-3.5 w-3.5" />实时刷新</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <S2VCard v-for="item in metrics" :key="item.label" variant="metric">
        <div class="text-xs font-medium text-s2v-muted">{{ item.label }}</div>
        <strong :class="['s2v-mono mt-2 block text-2xl font-bold tabular-nums', item.tone === 'success' ? 'text-s2v-success' : item.tone === 'primary' ? 'text-indigo-600' : 'text-s2v-ink']">
          {{ item.value }} <span v-if="item.unit" class="text-xs font-normal text-s2v-muted">{{ item.unit }}</span>
        </strong>
        <span class="mt-1 block text-[10px] text-s2v-muted">{{ item.note }}</span>
      </S2VCard>
    </div>

    <S2VCard variant="padded" class="mt-6">
      <div class="flex items-center justify-between gap-3">
        <h2 class="text-xs font-bold uppercase tracking-wider text-s2v-ink">智算中心节点实时链路</h2>
        <button type="button" class="text-xs font-semibold text-s2v-primary hover:underline" @click="router.push('/admin/monitoring')">集群拓扑详情 →</button>
      </div>
      <div class="mt-3 grid gap-3 sm:grid-cols-3">
        <div v-for="node in nodes" :key="node.name" class="rounded-xl border border-s2v-line bg-s2v-canvas p-3">
          <div class="flex items-center justify-between gap-2">
            <strong class="truncate text-xs">{{ node.name }}</strong>
            <S2VBadge tone="success">正常</S2VBadge>
          </div>
          <p class="s2v-mono mt-2 flex items-center gap-1.5 text-[11px] text-s2v-muted"><Cpu class="h-3.5 w-3.5 text-s2v-primary" />{{ node.detail }}</p>
        </div>
      </div>
    </S2VCard>
  </section>
</template>
