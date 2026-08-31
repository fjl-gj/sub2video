<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Menu, Wallet } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VIconButton from '@/shared/ui/S2VIconButton.vue'

const props = defineProps<{ surface: 'client' | 'admin'; onMenu?: () => void }>()
const router = useRouter()
const app = useAppStore()
const isClient = computed(() => props.surface === 'client')
</script>

<template>
  <header class="sticky top-0 z-40 flex h-full items-center justify-between border-b border-s2v-line/80 bg-s2v-surface/95 px-4 shadow-xs backdrop-blur-md lg:px-6">
    <div class="flex min-w-0 items-center gap-3">
      <S2VIconButton class="md:hidden" label="打开菜单" @click="onMenu?.()"><Menu class="h-4 w-4" /></S2VIconButton>
      <RouterLink :to="isClient ? '/app/overview' : '/admin/overview'" class="flex min-w-0 items-center gap-2.5">
        <span class="grid h-8 w-8 shrink-0 place-items-center rounded-xl bg-gradient-to-tr from-s2v-primary to-s2v-primary-hover text-sm font-black text-white shadow-sm">S2V</span>
        <span class="min-w-0 leading-tight">
          <span class="flex items-center gap-1.5">
            <span class="truncate text-sm font-bold tracking-tight text-s2v-ink">Sub2Video</span>
            <span class="s2v-mono hidden rounded border border-s2v-primary/25 bg-s2v-primary-soft px-1.5 py-0.5 text-[10px] font-semibold text-s2v-primary sm:inline">{{ isClient ? 'Creator Studio' : 'Admin Ops · v1.5' }}</span>
          </span>
          <span class="mt-0.5 hidden text-[10px] font-normal text-s2v-muted sm:block">{{ isClient ? '智能视频创作与资产管理系统' : '运营管理后台与 AI 算力调度中枢' }}</span>
        </span>
      </RouterLink>
    </div>
    <div class="flex items-center gap-3">
      <div class="hidden items-center gap-2 rounded-full border border-emerald-200/50 bg-emerald-50/80 px-2.5 py-1 text-[11px] font-medium text-emerald-800 md:flex">
        <span class="s2v-badge-pulse h-2 w-2 rounded-full bg-emerald-500" />
        <span class="s2v-mono">{{ isClient ? '算力集群' : '智算中心' }}: <strong>{{ isClient ? '24/32 卡在线' : '24/32' }}</strong>{{ isClient ? '' : ' 卡' }}</span>
        <span class="text-emerald-300">|</span>
        <span class="s2v-mono text-emerald-700">{{ isClient ? '基准 ¥0.20/秒' : '平均负载 75%' }}</span>
      </div>
      <button v-if="isClient" type="button" class="s2v-mono hidden items-center gap-1.5 rounded-lg border border-s2v-primary/25 bg-s2v-primary-soft/80 px-3 py-1 text-xs font-bold text-s2v-primary-hover transition hover:bg-s2v-primary-soft lg:flex" @click="router.push('/app/wallet')">
        <Wallet class="h-3.5 w-3.5" />¥ {{ app.balance.toFixed(2) }}
      </button>
      <button type="button" class="flex items-center gap-2.5 border-l border-s2v-line pl-2" :aria-label="isClient ? '打开个人资料' : '管理员账户'" @click="isClient ? router.push('/app/profile') : undefined">
        <span :class="['grid h-7 w-7 place-items-center rounded-full text-xs font-bold text-white', isClient ? 'bg-slate-900' : 'bg-emerald-900']">{{ isClient ? '李' : '管' }}</span>
        <span class="hidden text-left sm:block">
          <strong class="block text-xs font-semibold leading-none text-s2v-ink">{{ isClient ? '李维工作室' : '超级管理员' }}</strong>
          <span :class="['s2v-mono mt-0.5 block text-[10px]', isClient ? 'text-s2v-muted' : 'text-s2v-success']">{{ isClient ? 'Pro 认证' : 'Ops & Billing Center' }}</span>
        </span>
      </button>
    </div>
  </header>
</template>
