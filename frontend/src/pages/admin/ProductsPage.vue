<script setup lang="ts">
import { Coins, Edit3, Gift, Plus } from 'lucide-vue-next'
import { rechargeProducts, redeemCodes } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader badge="Catalog" title="充值商品">
      <template #description>配置用户可以买到的余额档位，不在这里直接修改用户余额。</template>
      <template #actions>
        <S2VButton variant="primary" @click="app.notify('充值商品编辑器将在支付接入后开放')"><Plus class="h-3.5 w-3.5" />新增商品</S2VButton>
      </template>
    </S2VPageHeader>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <S2VCard v-for="product in rechargeProducts" :key="product.id" variant="padded">
        <div class="flex items-start justify-between">
          <span class="s2v-icon-well"><Coins class="h-4 w-4" /></span>
          <S2VBadge tone="success">上架中</S2VBadge>
        </div>
        <strong class="mt-4 block text-xl font-bold font-mono">¥{{ product.amount }}</strong>
        <p class="mt-1 text-xs text-s2v-muted">到账 ¥{{ product.amount + product.bonus }}{{ product.bonus ? `（赠 ¥${product.bonus}）` : '' }}</p>
        <div class="mt-4 flex items-center justify-between border-t border-s2v-line pt-3">
          <span class="text-[11px] text-s2v-muted">微信支付</span>
          <button class="text-s2v-primary" aria-label="编辑充值商品" type="button" @click="app.notify(`已打开 ¥${product.amount} 商品配置`)"><Edit3 class="h-3.5 w-3.5" /></button>
        </div>
      </S2VCard>
    </div>

    <S2VCard class="mt-5">
      <div class="s2v-panel-header">
        <div>
          <h2 class="s2v-section-title">兑换码</h2>
          <p class="mt-1 text-xs text-s2v-muted">可在充值商品里发放体验金，不单独做订阅套餐。</p>
        </div>
        <Gift class="h-4 w-4 text-s2v-primary" />
      </div>
      <div class="divide-y divide-s2v-line/70 px-4">
        <div v-for="item in redeemCodes" :key="item.code" class="flex items-center justify-between gap-3 py-3">
          <div>
            <strong class="block font-mono text-xs">{{ item.code }}</strong>
            <span class="mt-1 block text-[11px] text-s2v-muted">{{ item.label }} · ¥{{ item.amount }}</span>
          </div>
          <S2VBadge :tone="app.usedCodes.includes(item.code) ? 'warning' : 'success'">{{ app.usedCodes.includes(item.code) ? '已兑换' : '可兑换' }}</S2VBadge>
        </div>
      </div>
    </S2VCard>
  </section>
</template>
