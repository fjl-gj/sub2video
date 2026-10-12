<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Activity, Bot, CircleHelp, Clapperboard, Coins, FileClock, FolderOpen, LayoutDashboard, Library, ListChecks, Settings2, Ticket, Trash2, WalletCards, Workflow } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import S2VNavItem from '@/shared/ui/S2VNavItem.vue'
import S2VProgress from '@/shared/ui/S2VProgress.vue'
import { cn } from '@/shared/lib/cn'

interface NavItem { label: string; to: string; icon: string; badge?: string; badgeTone?: 'primary' | 'success' | 'warning' | 'danger'; trailing?: string }
interface NavGroup { label: string; items: NavItem[] }
const props = defineProps<{ surface: 'client' | 'admin'; overlay?: boolean }>()
const route = useRoute()
const app = useAppStore()
const iconMap = { Activity, Bot, CircleHelp, Clapperboard, Coins, FileClock, FolderOpen, LayoutDashboard, Library, ListChecks, Settings2, Ticket, Trash2, WalletCards, Workflow }

const clientGroups = computed<NavGroup[]>(() => [
  { label: '', items: [{ label: '首页概览', to: '/app/overview', icon: 'LayoutDashboard' }] },
  { label: '创作空间', items: [
    { label: '创作', to: '/app/create', icon: 'Clapperboard' },
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
    { label: '承运商', to: '/admin/carriers-models', icon: 'Bot' },
    { label: '供应商', to: '/admin/supplier-models', icon: 'Bot' },
  ] },
  { label: '计费管理', items: [
    { label: '价格审计', to: '/admin/pricing', icon: 'Coins' },
  ] },
  { label: '任务管理', items: [
    { label: '任务中心', to: '/admin/tasks', icon: 'ListChecks' },
  ] },
  { label: '运营管理', items: [
    { label: '工单中心', to: '/admin/tickets', icon: 'Ticket', badge: app.openTicketCount ? `${app.openTicketCount}` : undefined, badgeTone: 'danger' },
  ] },
  { label: '系统', items: [
    { label: '运维监控', to: '/admin/monitoring', icon: 'Activity' },
    { label: '操作日志', to: '/admin/audit-log', icon: 'FileClock' },
    { label: '系统设置', to: '/admin/settings', icon: 'Settings2' },
  ] },
])

const groups = computed(() => props.surface === 'client' ? clientGroups.value : adminGroups.value)
function active(to: string) { return route.path === to || (to === '/app/assets' && /^\/app\/projects\/[^/]+\/director$/.test(route.path)) }
</script>

<template>
  <aside :class="cn(overlay ? 's2v-sidebar-overlay' : 's2v-sidebar')">
    <nav class="s2v-sidebar-nav" aria-label="主导航">
      <div v-for="(group, index) in groups" :key="group.label || index" class="grid gap-1">
        <div v-if="group.label" class="s2v-sidebar-group">{{ group.label }}</div>
        <S2VNavItem v-for="item in group.items" :key="item.to" :to="item.to" :active="active(item.to)" :badge="item.badge" :badge-tone="item.badgeTone">
          <template #icon><component :is="iconMap[item.icon as keyof typeof iconMap]" class="h-4 w-4" /></template>
          {{ item.label }}
          <template v-if="item.trailing" #trailing>
            <span class="s2v-sidebar-count">{{ item.trailing }}</span>
          </template>
        </S2VNavItem>
      </div>
    </nav>
    <div v-if="props.surface === 'client'" class="s2v-sidebar-storage space-y-3">
      <div class="flex items-center justify-between text-[11px]"><span class="text-s2v-muted">创作存储</span><span class="s2v-mono text-s2v-body">{{ app.storageUsed }} / {{ app.storageQuota }} GB</span></div>
      <S2VProgress :value="app.storagePercent" />
      <div class="s2v-mono flex justify-end text-[10px] text-s2v-label">
        <RouterLink to="/app/assets" class="text-s2v-primary hover:underline">管理 →</RouterLink>
      </div>
    </div>
  </aside>
</template>
