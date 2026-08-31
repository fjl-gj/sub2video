<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bell, ChevronDown, Film, Menu, ShieldCheck, WalletCards } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'

const props = defineProps<{ surface: 'client' | 'admin'; onMenu?: () => void }>()
const route = useRoute()
const router = useRouter()
const app = useAppStore()
const pageTitle = computed(() => (route.meta.title as string | undefined) ?? '首页概览')
const isClient = computed(() => props.surface === 'client')

function switchSurface() {
  router.push(isClient.value ? '/admin/overview' : '/app/overview')
}
</script>

<template>
  <header class="flex h-full items-center justify-between gap-4 border-b border-s2v-line bg-s2v-surface px-4 sm:px-5">
    <div class="flex min-w-0 items-center gap-3">
      <S2VIconButton class="md:hidden" label="打开菜单" @click="onMenu?.()"><Menu class="h-4 w-4" /></S2VIconButton>
      <RouterLink to="/app/overview" class="flex min-w-0 items-center gap-2.5 font-bold text-s2v-ink">
        <span class="grid h-8 w-8 shrink-0 place-items-center rounded-[8px] bg-s2v-primary text-white shadow-subtle"><Film class="h-4 w-4" /></span>
        <span class="truncate text-base">Sub2Video</span>
      </RouterLink>
      <span class="hidden h-5 w-px bg-s2v-line sm:block" />
      <span class="hidden truncate text-xs text-s2v-muted sm:block">{{ isClient ? pageTitle : `管理端 / ${pageTitle}` }}</span>
    </div>
    <div class="flex items-center gap-2">
      <button v-if="isClient" type="button" class="hidden items-center gap-1.5 rounded-full border border-s2v-line px-2.5 py-1.5 text-xs text-s2v-muted transition hover:border-s2v-primary hover:text-s2v-primary sm:flex" @click="router.push('/app/wallet')">
        <WalletCards class="h-3.5 w-3.5" /><span>余额</span><strong class="font-mono text-s2v-ink">¥ {{ app.balance.toFixed(2) }}</strong>
      </button>
      <button type="button" class="hidden items-center gap-1.5 rounded-[7px] border border-s2v-line px-2.5 py-1.5 text-xs font-semibold text-s2v-muted transition hover:border-s2v-primary hover:text-s2v-primary md:flex" @click="switchSurface">
        <ShieldCheck class="h-3.5 w-3.5" />{{ isClient ? '管理端' : '客户端' }}<ChevronDown class="h-3.5 w-3.5" />
      </button>
      <S2VIconButton label="通知"><Bell class="h-4 w-4" /><span class="absolute mt-[-15px] ml-4 h-1.5 w-1.5 rounded-full bg-rose-500" /></S2VIconButton>
      <button type="button" class="grid h-8 w-8 place-items-center rounded-full bg-slate-700 text-xs font-bold text-white" aria-label="打开个人资料" @click="router.push('/app/profile')">林</button>
    </div>
  </header>
</template>
