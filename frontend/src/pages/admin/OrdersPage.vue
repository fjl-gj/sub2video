<script setup lang="ts">
import { Download, FileText, Search } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import MobileRecordCard from '@/features/lists/MobileRecordCard.vue'
import { useAppStore } from '@/stores/app'
import S2VBadge from '@/shared/ui/S2VBadge.vue'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'

const app = useAppStore()
const search = ref('')
const list = computed(() => app.transactions.filter((row) => !search.value || `${row.id}${row.desc}${row.type}`.includes(search.value)))
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="订单管理">
      <template #description>充值、任务扣费和退款统一留痕，方便按用户和订单追溯。</template>
      <template #actions>
        <S2VIconButton label="导出订单" @click="app.notify('已生成订单导出文件（原型）')"><Download class="h-4 w-4" /></S2VIconButton>
      </template>
    </S2VPageHeader>

    <div class="mb-4 max-w-xs">
      <S2VInput v-model="search" size="sm" placeholder="订单号 / 用户">
        <template #prefix><Search class="h-3.5 w-3.5 text-s2v-muted" /></template>
      </S2VInput>
    </div>

    <div class="s2v-table-wrap hidden md:block">
      <table class="s2v-table">
        <thead>
          <tr>
            <th>流水号</th>
            <th>类型</th>
            <th>说明</th>
            <th>金额</th>
            <th>状态</th>
            <th>时间</th>
            <th>凭证</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in list" :key="item.id">
            <td class="font-mono text-s2v-ink">{{ item.id }}</td>
            <td>{{ item.type }}</td>
            <td>{{ item.desc }}</td>
            <td :class="item.amount > 0 ? 'font-mono text-s2v-success' : 'font-mono'">{{ item.amount > 0 ? '+' : '' }}¥{{ item.amount.toFixed(2) }}</td>
            <td><S2VBadge tone="success">已完成</S2VBadge></td>
            <td>{{ item.time }}</td>
            <td><button class="flex items-center gap-1 text-xs text-s2v-primary" type="button" @click="app.notify(`${item.id} 凭证：${item.type}`)"><FileText class="h-3.5 w-3.5" />查看</button></td>
          </tr>
          <tr v-if="!list.length">
            <td colspan="7" class="text-center text-s2v-muted">暂无订单数据，等待后端数据接入。</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="grid gap-3 md:hidden">
      <MobileRecordCard v-for="item in list" :key="item.id" :title="item.type" :meta="item.id">
        <template #badge><S2VBadge tone="success">已完成</S2VBadge></template>
        <template #body>{{ item.desc }} · {{ item.amount > 0 ? '+' : '' }}¥{{ item.amount.toFixed(2) }}</template>
        <template #actions>
          <button class="flex items-center gap-1 text-xs text-s2v-primary" type="button" @click="app.notify(`${item.id} 凭证：${item.type}`)"><FileText class="h-3.5 w-3.5" />查看</button>
        </template>
      </MobileRecordCard>
      <p v-if="!list.length" class="text-xs text-s2v-muted">暂无订单数据。</p>
    </div>
  </section>
</template>
