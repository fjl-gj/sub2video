<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Activity, Bot, CircleHelp, Coins, FileClock, FolderOpen, LayoutDashboard, Library, ListChecks, Megaphone, PanelsTopLeft, ReceiptText, Settings2, Sparkles, Ticket, UsersRound, WalletCards, Workflow } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'

interface NavItem { label: string; to: string; icon: string; badge?: string }
interface NavGroup { label: string; items: NavItem[] }
const props = defineProps<{ surface: 'client' | 'admin' }>()
const route = useRoute()
const app = useAppStore()
const iconMap = { Activity, Bot, CircleHelp, Coins, FileClock, FolderOpen, LayoutDashboard, Library, ListChecks, Megaphone, PanelsTopLeft, ReceiptText, Settings2, Sparkles, Ticket, UsersRound, WalletCards, Workflow }

const clientGroups: NavGroup[] = [
  { label: '概览', items: [{ label: '首页概览', to: '/app/overview', icon: 'LayoutDashboard' }] },
  { label: '创作空间', items: [
    { label: '创作', to: '/app/create', icon: 'Sparkles' },
    { label: '创意园', to: '/app/garden', icon: 'Library' },
    { label: '作品', to: '/app/works', icon: 'PanelsTopLeft', badge: String(app.activeTasks.length) },
    { label: '资产', to: '/app/assets', icon: 'FolderOpen' },
  ] },
  { label: '账户服务', items: [
    { label: '钱包', to: '/app/wallet', icon: 'WalletCards' },
    { label: '帮助与工单', to: '/app/tickets', icon: 'CircleHelp' },
  ] },
]

const adminGroups: NavGroup[] = [
  { label: '概览', items: [{ label: '首页概览', to: '/admin/overview', icon: 'LayoutDashboard' }] },
  { label: '创作管理', items: [
    { label: '工作流', to: '/admin/workflows', icon: 'Workflow' },
    { label: '模型与供应商', to: '/admin/providers', icon: 'Bot' },
  ] },
  { label: '计费管理', items: [
    { label: '定价规则', to: '/admin/pricing', icon: 'Coins' },
    { label: '充值商品', to: '/admin/products', icon: 'Coins' },
    { label: '订单管理', to: '/admin/orders', icon: 'ReceiptText' },
  ] },
  { label: '任务管理', items: [
    { label: '任务中心', to: '/admin/tasks', icon: 'ListChecks' },
  ] },
  { label: '运营管理', items: [
    { label: '用户管理', to: '/admin/users', icon: 'UsersRound' },
    { label: '工单中心', to: '/admin/tickets', icon: 'Ticket', badge: '2' },
    { label: '公告', to: '/admin/announcements', icon: 'Megaphone' },
  ] },
  { label: '系统', items: [
    { label: '运维监控', to: '/admin/monitoring', icon: 'Activity' },
    { label: '操作日志', to: '/admin/audit-log', icon: 'FileClock' },
    { label: '系统设置', to: '/admin/settings', icon: 'Settings2' },
  ] },
]

const groups = computed(() => props.surface === 'client' ? clientGroups : adminGroups)
function active(to: string) { return route.path === to }
</script>

<template>
  <aside class="s2v-sidebar">
    <nav class="grid gap-3 overflow-y-auto p-3" aria-label="主导航">
      <div v-for="group in groups" :key="group.label" class="grid gap-1">
        <div class="px-2 py-1 text-[11px] font-bold tracking-wide text-s2v-muted/75">{{ group.label }}</div>
        <RouterLink v-for="item in group.items" :key="item.to" :to="item.to" :class="['flex min-h-9 items-center gap-2.5 rounded-[7px] border px-2.5 text-xs font-semibold transition', active(item.to) ? 'border-s2v-primary/25 bg-s2v-primary-soft text-s2v-primary' : 'border-transparent text-s2v-muted hover:bg-s2v-canvas hover:text-s2v-ink']">
          <component :is="iconMap[item.icon as keyof typeof iconMap]" class="h-4 w-4 shrink-0" />
          <span class="truncate">{{ item.label }}</span>
          <span v-if="item.badge" class="ml-auto rounded-full bg-s2v-primary-soft px-1.5 text-[10px] font-mono text-s2v-primary">{{ item.badge }}</span>
        </RouterLink>
      </div>
    </nav>
    <div v-if="props.surface === 'client'" class="mt-auto hidden border-t border-s2v-line p-3 lg:block">
      <div class="grid gap-2 rounded-[8px] bg-s2v-canvas p-3 text-[11px]">
        <div class="flex justify-between text-s2v-muted"><span>资产空间</span><span class="font-mono text-s2v-ink">1.9 / 5 GB</span></div>
        <div class="h-1 overflow-hidden rounded-full bg-slate-200"><span class="block h-full w-[38%] rounded-full bg-s2v-primary" /></div>
        <span class="text-s2v-muted">已保存 6 个资产</span>
      </div>
    </div>
  </aside>
</template>
