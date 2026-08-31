<script setup lang="ts">
import { Coins, Edit3, Plus } from 'lucide-vue-next'
import { rechargeProducts } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
const app = useAppStore()
</script>
<template><section class="s2v-page"><header class="s2v-page-header"><div><p class="s2v-eyebrow">计费管理 / 充值商品</p><h1 class="s2v-page-title">充值商品</h1><p class="s2v-subcopy">配置用户可以买到的余额档位，不在这里直接修改用户余额。</p></div><S2VButton variant="primary" @click="app.notify('充值商品编辑器将在支付接入后开放')"><Plus class="h-3.5 w-3.5" />新增商品</S2VButton></header><div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4"><S2VCard v-for="product in rechargeProducts" :key="product.id" class="p-4"><div class="flex items-start justify-between"><span class="grid h-8 w-8 place-items-center rounded-full bg-emerald-50 text-emerald-600"><Coins class="h-4 w-4" /></span><S2VBadge tone="success">上架中</S2VBadge></div><strong class="mt-4 block text-xl font-bold font-mono">¥{{ product.amount }}</strong><p class="mt-1 text-xs text-s2v-muted">到账 ¥{{ product.amount + product.bonus }}{{ product.bonus ? `（赠 ¥${product.bonus}）` : '' }}</p><div class="mt-4 flex items-center justify-between border-t border-s2v-line pt-3"><span class="text-[11px] text-s2v-muted">微信支付</span><button class="text-s2v-primary" aria-label="编辑充值商品" @click="app.notify(`已打开 ¥${product.amount} 商品配置`)"><Edit3 class="h-3.5 w-3.5" /></button></div></S2VCard></div></section></template>
