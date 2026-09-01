<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Activity, Bot, CircleHelp, Coins, FileClock, FolderOpen, LayoutDashboard, Library, ListChecks, Megaphone, ReceiptText, Settings2, Sparkles, Ticket, Trash2, UsersRound, WalletCards, Workflow } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VNavItem from '@/shared/ui/S2VNavItem.vue'
import S2VProgress from '@/shared/ui/S2VProgress.vue'
import { cn } from '@/shared/lib/cn'

interface NavItem { label: string; to: string; icon: string; badge?: string; badgeTone?: 'primary' | 'success' | 'warning' | 'danger'; trailing?: string }
interface NavGroup { label: string; items: NavItem[] }
const props = defineProps<{ surface: 'client' | 'admin'; overlay?: boolean }>()
const route = useRoute()
const app = useAppStore()
const iconMap = { Activity, Bot, CircleHelp, Coins, FileClock, FolderOpen, LayoutDashboard, Library, ListChecks, Megaphone, ReceiptText, Settings2, Sparkles, Ticket, Trash2, UsersRound, WalletCards, Workflow }

const clientGroups = computed<NavGroup[]>(() => [
  { label: '', items: [{ label: '首页概览', to: '/app/overview', icon: 'LayoutDashboard' }] },
  { label: '创作空间', items: [
    { label: '创作', to: '/app/create', icon: 'Sparkles' },
    { label: '广场', to: '/app/garden', icon: 'Library' },
    { label: '任务', to: '/app/works', icon: 'ListChecks', trailing: app.activeTasks.length ? `${app.activeTasks.length}` : undefined },
    { label: '资产', to: '/app/assets', icon: 'FolderOpen' },
    { label: '回收站', to: '/app/trash', icon: 'Trash2', trailing: app.trashedProjects.length ? `${app.trashedProjects.length}` : undefined },
  ] },
  { label: '账户服务', items: [
    { label: '钱包', to: '/app/wallet', icon: 'WalletCards' },
    { label: '帮助与工单', to: '/app/tickets', icon: 'CircleHelp', badge: app.openClientTicketCount ? `${app.openClientTicketCount}` : undefined, badgeTone: 'danger' },
  ] },
])

const adminGroups = computed<NavGroup[]>(() => [
  { label: '', items: [{ label: '首页概览', to: '/admin/overview', icon: 'LayoutDashboard' }] },
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
    { label: '工单中心', to: '/admin/tickets', icon: 'Ticket', badge: app.openTicketCount ? `${app.openTicketCount}` : undefined, badgeTone: 'danger' },
    { label: '公告', to: '/admin/announcements', icon: 'Megaphone' },
  ] },
  { label: '系统', items: [
    { label: '运维监控', to: '/admin/monitoring', icon: 'Activity' },
    { label: '操作日志', to: '/admin/audit-log', icon: 'FileClock' },
    { label: '系统设置', to: '/admin/settings', icon: 'Settings2' },
  ] },
])

const groups = computed(() => props.surface === 'client' ? clientGroups.value : adminGroups.value)
function active(to: string) { return route.path === to }
</script>

<template>
  <aside :class="cn(overlay ? 'fixed inset-y-14 left-0 z-40 flex h-auto w-60 flex-col border-r border-s2v-line/80 bg-s2v-surface shadow-subtle md:hidden' : 's2v-sidebar')">
    <nav class="grid gap-5 overflow-y-auto p-3" aria-label="主导航">
      <div v-for="(group, index) in groups" :key="group.label || index" class="grid gap-1">
        <div v-if="group.label" class="px-2.5 pb-1 text-[11px] font-bold uppercase tracking-wider text-s2v-label">{{ group.label }}</div>
        <S2VNavItem v-for="item in group.items" :key="item.to" :to="item.to" :active="active(item.to)" :badge="item.badge" :badge-tone="item.badgeTone">
          <template #icon><component :is="iconMap[item.icon as keyof typeof iconMap]" class="h-4 w-4" /></template>
          {{ item.label }}
          <template v-if="item.trailing" #trailing>
            <span :class="['s2v-mono shrink-0 text-[10px] leading-none', item.to === '/app/works' ? 'rounded border border-amber-200/60 bg-amber-50 px-1.5 py-0.5 font-bold text-amber-700' : 'text-s2v-label']">{{ item.trailing }}</span>
          </template>
        </S2VNavItem>
      </div>
    </nav>
    <div v-if="props.surface === 'client'" class="mt-auto space-y-2 border-t border-s2v-nav-hover bg-s2v-canvas/50 p-3">
      <div class="flex items-center justify-between text-[11px]"><span class="font-medium text-s2v-muted">空间资源占用</span><span class="s2v-mono font-semibold text-s2v-ink">{{ app.storageUsed }} / {{ app.storageQuota }} GB</span></div>
      <S2VProgress :value="app.storagePercent" />
      <div class="s2v-mono flex justify-end text-[10px] text-s2v-label">
        <RouterLink to="/app/assets" class="text-s2v-primary hover:underline">管理 →</RouterLink>
      </div>
    </div>
  </aside>
</template>
