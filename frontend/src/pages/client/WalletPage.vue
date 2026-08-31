<script setup lang="ts">
import { ArrowDownLeft, ArrowUpRight, Check, CircleHelp, Coins, CreditCard, Gift, History } from 'lucide-vue-next'
import { ref } from 'vue'
import { rechargeProducts, transactions } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VChoice from '@/shared/ui/S2VChoice.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const selected = ref(rechargeProducts[1].id)
const code = ref('')

function recharge() {
  const product = rechargeProducts.find((item) => item.id === selected.value)
  if (product) app.recharge(product.amount + product.bonus)
}
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="RMB 钱包中心">
      <template #description>支持 RMB 在线充值、余额实时抵扣、生成预扣与账本流水</template>
      <template #actions>
        <div class="text-right">
          <span class="text-xs text-s2v-muted">可用余额:</span>
          <div class="s2v-mono text-2xl font-bold text-s2v-success">¥ {{ app.balance.toFixed(2) }}</div>
        </div>
      </template>
    </S2VPageHeader>

    <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(300px,.8fr)]">
      <S2VCard>
        <div class="s2v-panel-header">
          <div>
            <h2 class="s2v-section-title">充值余额</h2>
            <p class="mt-1 text-xs text-s2v-muted">选择充值档位，赠送金额会一起到账。</p>
          </div>
          <Coins class="h-4 w-4 text-s2v-primary" />
        </div>
        <div class="s2v-panel-body grid grid-cols-2 gap-2 sm:grid-cols-4">
          <S2VChoice v-for="product in rechargeProducts" :key="product.id" variant="tile" :selected="selected === product.id" @click="selected = product.id">
            <strong class="block text-lg text-s2v-ink">¥{{ product.amount }}</strong>
            <span class="mt-1 block text-[11px] font-normal text-s2v-muted">{{ product.bonus ? `赠 ¥${product.bonus}` : '无额外赠送' }}</span>
            <Check v-if="selected === product.id" class="mt-2 h-3.5 w-3.5" />
          </S2VChoice>
        </div>
        <div class="flex items-center justify-between gap-3 border-t border-s2v-line px-4 py-3">
          <span class="text-xs text-s2v-muted">支付方式：微信支付（原型）</span>
          <S2VButton variant="primary" @click="recharge"><CreditCard class="h-3.5 w-3.5" />立即充值</S2VButton>
        </div>
      </S2VCard>

      <div class="grid gap-5">
        <S2VCard>
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">兑换余额</h2>
              <p class="mt-1 text-xs text-s2v-muted">输入平台发放的兑换码。</p>
            </div>
            <Gift class="h-4 w-4 text-s2v-primary" />
          </div>
          <div class="s2v-panel-body grid gap-3">
            <S2VInput v-model="code" placeholder="例如 S2V-XXXX-XXXX" />
            <S2VButton :disabled="!code" @click="app.notify('兑换码校验成功（原型）')">确认兑换</S2VButton>
          </div>
        </S2VCard>

        <S2VCard>
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">余额流水</h2>
              <p class="mt-1 text-xs text-s2v-muted">最近的充值、消费和退款。</p>
            </div>
            <History class="h-4 w-4 text-s2v-muted" />
          </div>
          <div class="divide-y divide-s2v-line/70 px-4">
            <div v-for="item in transactions" :key="item.id" class="flex items-center gap-3 py-3">
              <span :class="['grid h-7 w-7 place-items-center rounded-full', item.amount > 0 ? 'bg-emerald-50 text-s2v-success' : 'bg-s2v-canvas text-s2v-muted']">
                <ArrowDownLeft v-if="item.amount > 0" class="h-3.5 w-3.5" />
                <ArrowUpRight v-else class="h-3.5 w-3.5" />
              </span>
              <div class="min-w-0 flex-1">
                <strong class="block truncate text-xs">{{ item.type }}</strong>
                <span class="mt-1 block truncate text-[11px] text-s2v-muted">{{ item.desc }}</span>
              </div>
              <div class="text-right">
                <strong :class="['block font-mono text-xs', item.amount > 0 ? 'text-s2v-success' : 'text-s2v-ink']">{{ item.amount > 0 ? '+' : '' }}¥{{ item.amount.toFixed(2) }}</strong>
                <span class="mt-1 block text-[10px] text-s2v-muted">{{ item.time }}</span>
              </div>
            </div>
          </div>
          <div class="border-t border-s2v-line px-4 py-3">
            <button class="flex items-center gap-1 text-xs font-semibold text-s2v-primary" type="button" @click="app.notify('流水详情将在后端接入后开放')"><CircleHelp class="h-3.5 w-3.5" />了解余额扣费规则</button>
          </div>
        </S2VCard>
      </div>
    </div>
  </section>
</template>
