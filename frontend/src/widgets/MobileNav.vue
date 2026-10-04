<script setup lang="ts">
import { Clapperboard, LayoutDashboard, ListChecks, UserRound, WalletCards } from 'lucide-vue-next'
import { useRoute } from 'vue-router'

const route = useRoute()
const items = [
  { label: '概览', to: '/app/overview', icon: LayoutDashboard },
  { label: '创作', to: '/app/create', icon: Clapperboard },
  { label: '任务', to: '/app/works', icon: ListChecks },
  { label: '钱包', to: '/app/wallet', icon: WalletCards },
  { label: '资料', to: '/app/profile', icon: UserRound },
]
</script>

<template>
  <nav aria-label="移动导航" class="s2v-mobile-nav fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-s2v-line/80 bg-s2v-surface/95 px-1 py-1.5 backdrop-blur md:hidden">
    <RouterLink v-for="item in items" :key="item.to" :to="item.to" :class="['grid min-h-11 place-items-center gap-0.5 rounded-nav text-[10px] font-semibold', route.path === item.to ? 'text-s2v-primary' : 'text-s2v-muted']">
      <component :is="item.icon" class="h-4 w-4" /><span>{{ item.label }}</span>
    </RouterLink>
  </nav>
</template>
