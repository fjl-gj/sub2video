<script setup lang="ts">
import { Download, FileText, Search } from 'lucide-vue-next'
import { ref } from 'vue'
import { transactions } from '@/data/mock'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
const app = useAppStore(); const search = ref('')
</script>
<template><section class="s2v-page"><header class="s2v-page-header"><div><p class="s2v-eyebrow">计费管理 / 订单管理</p><h1 class="s2v-page-title">订单管理</h1><p class="s2v-subcopy">充值、任务扣费和退款统一留痕，方便按用户和订单追溯。</p></div><S2VIconButton label="导出订单" @click="app.notify('已生成订单导出文件（原型）')"><Download class="h-4 w-4" /></S2VIconButton></header><div class="mb-4 flex items-center gap-2 rounded-[7px] border border-s2v-line bg-s2v-surface px-3 py-2 sm:max-w-xs"><Search class="h-3.5 w-3.5 text-s2v-muted" /><input v-model="search" class="min-w-0 flex-1 bg-transparent text-xs outline-none" placeholder="订单号 / 用户" /></div><div class="s2v-table-wrap"><table class="s2v-table"><thead><tr><th>流水号</th><th>类型</th><th>说明</th><th>金额</th><th>状态</th><th>时间</th><th>凭证</th></tr></thead><tbody><tr v-for="item in transactions.filter((row) => !search || `${row.id}${row.desc}`.includes(search))" :key="item.id"><td class="font-mono text-s2v-ink">{{ item.id }}</td><td>{{ item.type }}</td><td>{{ item.desc }}</td><td :class="item.amount > 0 ? 'font-mono text-emerald-600' : 'font-mono'">{{ item.amount > 0 ? '+' : '' }}¥{{ item.amount.toFixed(2) }}</td><td><S2VBadge tone="success">已完成</S2VBadge></td><td>{{ item.time }}</td><td><button class="flex items-center gap-1 text-xs text-s2v-primary" @click="app.notify('凭证详情已打开')"><FileText class="h-3.5 w-3.5" />查看</button></td></tr></tbody></table></div></section></template>
