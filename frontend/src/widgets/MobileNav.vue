<script setup lang="ts">
import { computed } from 'vue'
import { FolderOpen, LayoutDashboard, PanelsTopLeft, Sparkles, WalletCards } from 'lucide-vue-next'
import { useRoute } from 'vue-router'
const route = useRoute()
const props = defineProps<{ surface: 'client' | 'admin' }>()
const clientItems = [
  { label: '概览', to: '/app/overview', icon: LayoutDashboard },
  { label: '创作', to: '/app/create', icon: Sparkles },
  { label: '作品', to: '/app/works', icon: PanelsTopLeft },
  { label: '资产', to: '/app/assets', icon: FolderOpen },
  { label: '钱包', to: '/app/wallet', icon: WalletCards },
]
const adminItems = [
  { label: '概览', to: '/admin/overview', icon: LayoutDashboard },
  { label: '工作流', to: '/admin/workflows', icon: Sparkles },
  { label: '任务', to: '/admin/tasks', icon: PanelsTopLeft },
  { label: '用户', to: '/admin/users', icon: FolderOpen },
]
const items = computed(() => props.surface === 'client' ? clientItems : adminItems)
</script>
<template>
  <nav class="fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-s2v-line bg-s2v-surface/95 px-1 py-1.5 backdrop-blur md:hidden">
    <RouterLink v-for="item in items" :key="item.to" :to="item.to" :class="['grid min-h-11 place-items-center gap-0.5 rounded-[7px] text-[10px] font-semibold', route.path === item.to ? 'text-s2v-primary' : 'text-s2v-muted']">
      <component :is="item.icon" class="h-4 w-4" /><span>{{ item.label }}</span>
    </RouterLink>
  </nav>
</template>
