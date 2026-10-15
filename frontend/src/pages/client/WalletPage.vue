<script setup lang="ts">
import { ArrowDownLeft, ArrowUpRight, Gift, History } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { fetchWalletTransactions, redeemCode } from '@/api/client'
import type { LedgerEntry } from '@/entities/types'
import { PAGE_SIZE } from '@/shared/lib/paginate'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import S2VButton from '@/shared/ui/S2VButton.vue'
import S2VCard from '@/shared/ui/S2VCard.vue'
import S2VInput from '@/shared/ui/S2VInput.vue'
import S2VPageHeader from '@/shared/ui/S2VPageHeader.vue'
import S2VPagination from '@/shared/ui/S2VPagination.vue'

const app = useAppStore()
const auth = useAuthStore()

const balance = computed(() => auth.balance)
const walletLoading = ref(false)
const walletLoaded = ref(false)
const walletError = ref('')
const displayWalletError = computed(() => walletError.value || auth.balanceSyncError)
const transactionsLoading = ref(false)
const transactionsError = ref('')
const redeemError = ref('')
let transactionRequest = 0
const code = ref('')
const page = ref(1)
const redeeming = ref(false)
const transactions = ref<LedgerEntry[]>([])
const transactionTotal = ref(0)

const pagedTransactions = computed(() => transactions.value)

function apiErrorMessage(error: unknown, fallback: string) {
  const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return err.response?.data?.message || err.response?.data?.error || err.message || fallback
}

async function loadWallet() {
  if (walletLoading.value) return
  walletLoading.value = true
  try {
    await auth.refreshBalance()
    walletLoaded.value = true
    walletError.value = ''
  }
  catch { walletError.value = '余额暂时无法读取，请重新加载。' }
  finally { walletLoading.value = false }
}

async function loadTransactions() {
  const request = ++transactionRequest
  transactionsLoading.value = true
  try {
    const data = await fetchWalletTransactions({ page: page.value, pageSize: PAGE_SIZE })
    if (request !== transactionRequest) return
    transactions.value = data.items
    transactionTotal.value = data.total
    transactionsError.value = ''
  }
  catch { if (request === transactionRequest) transactionsError.value = '余额流水暂时无法读取，请重新加载。' }
  finally { if (request === transactionRequest) transactionsLoading.value = false }
}

async function redeem() {
  if (!code.value.trim() || redeeming.value) return
  redeeming.value = true
  redeemError.value = ''
  const balanceMutation = auth.beginBalanceMutation()
  try {
    const result = await redeemCode(code.value.trim())
    auth.acceptBalanceMutation(balanceMutation, result.wallet.balance)
    walletLoaded.value = true
    walletError.value = ''
    app.notify(`兑换成功，余额增加 ¥${result.credited.toFixed(2)}`)
    code.value = ''
    page.value = 1
    await loadTransactions()
  }
  catch (error) {
    redeemError.value = apiErrorMessage(error, '兑换失败，请检查兑换码后重试。')
  }
  finally {
    auth.finishBalanceMutation(balanceMutation)
    redeeming.value = false
  }
}

watch(page, () => {
  void loadTransactions()
})

onMounted(() => {
  void loadWallet()
  void loadTransactions()
})
</script>

<template>
  <section class="s2v-page">
    <S2VPageHeader title="钱包">
      <template #description>真实余额与流水。不提供订阅或 API 套餐。在线充值尚未开放。</template>
      <template #actions>
        <div class="text-right">
          <span class="text-xs text-s2v-muted">{{ displayWalletError && walletLoaded ? '上次读取余额:' : '可用余额:' }}</span>
          <div class="s2v-mono text-2xl font-bold text-s2v-success">{{ walletLoaded ? `¥ ${balance.toFixed(2)}` : walletLoading ? '读取中…' : '暂不可用' }}</div>
        </div>
      </template>
    </S2VPageHeader>

    <S2VCard v-if="displayWalletError" variant="padded" class="mb-5" role="alert"><p>{{ displayWalletError }}</p><S2VButton class="mt-3" :loading="walletLoading" @click="loadWallet">重新加载余额</S2VButton></S2VCard>

    <div class="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(300px,.8fr)]">
        <S2VCard>
          <div class="s2v-panel-header">
            <div>
              <h2 class="s2v-section-title">余额流水</h2>
              <p class="mt-1 text-xs text-s2v-muted">最近的充值、消费和退款。</p>
            </div>
            <History class="h-4 w-4 text-s2v-muted" />
          </div>
          <div class="divide-y divide-s2v-line/70 px-4">
            <div v-if="transactionsError" class="py-4" role="alert"><p>{{ transactionsError }}{{ transactions.length ? '当前显示上次读取的记录。' : '' }}</p><S2VButton class="mt-3" :loading="transactionsLoading" @click="loadTransactions">重新加载流水</S2VButton></div>
            <p v-if="transactionsLoading && !transactions.length" class="py-4" role="status">正在读取流水…</p>
            <div v-for="item in pagedTransactions" :key="item.id" class="flex items-center gap-3 py-3">
              <span :class="item.amount > 0 ? 's2v-icon-well s2v-icon-well-success' : 's2v-icon-well s2v-icon-well-muted'">
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
            <p v-if="!transactionsLoading && !transactionsError && !transactions.length" class="py-6 text-center text-xs text-s2v-muted">暂无流水记录</p>
          </div>
          <div v-if="transactionTotal" class="border-t border-s2v-line px-4 py-3">
            <S2VPagination v-model="page" :total="transactionTotal" :page-size="PAGE_SIZE" />
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
            <p class="text-sm">在线充值暂未开放。已获得的兑换码可用于增加创作余额。</p>
            <S2VInput v-model="code" label="兑换码" placeholder="输入平台发放的兑换码" :disabled="redeeming" />
            <p v-if="redeemError" role="alert">{{ redeemError }}</p>
            <S2VButton :disabled="!code.trim() || redeeming" :loading="redeeming" @click="redeem">确认兑换</S2VButton>
          </div>
        </S2VCard>
        <S2VCard>
          <div class="s2v-panel-header"><h2 class="s2v-section-title">创作余额与费用规则</h2></div>
          <div class="s2v-panel-body grid gap-3 text-sm">
            <p>估价不扣费。提交生成任务时，按当时的参数与价格预扣费用。</p>
            <p>确认失败或成功取消后退回创作余额，退款金额和时间可在任务详情与流水中查看。</p>
            <p>任务进入执行或提交结果待核对后暂不可取消。结果尚未确认时，请勿重复提交。</p>
            <RouterLink to="/app/tickets">对费用有疑问，前往帮助与工单</RouterLink>
          </div>
        </S2VCard>
      </div>
    </div>
  </section>
</template>
