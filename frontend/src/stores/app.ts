import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type {
  AdminTask,
  AdminTicket,
  Announcement,
  AuditLog,
  ClientTicket,
  LedgerEntry,
  PlatformSettings,
  PricingRule,
  ProviderItem,
  RechargeProduct,
  Task,
  UserAccount,
  WorkflowItem,
} from '@/entities/types'
import {
  cloneFolders,
  countProjectFiles,
  createStandardFolders,
  DEFAULT_PROJECT_ID,
  flattenProject,
  initialProjects,
  type WorkProject,
} from '@/data/projects'

export type { AdminTask, AdminTicket, UserAccount }
export type DeviceSession = { id: string; name: string; meta: string; current: boolean }

const NOT_CONNECTED = '功能尚未接入后端'

export const useAppStore = defineStore('app', () => {
  const balance = ref(0)
  const draftPrompt = ref('')
  const tasks = ref<Task[]>([])
  const adminTasks = ref<AdminTask[]>([])
  const adminTickets = ref<AdminTicket[]>([])
  const tickets = ref<ClientTicket[]>([])
  const projects = ref<WorkProject[]>(initialProjects.map((project) => ({
    ...project,
    folders: cloneFolders(project.folders),
  })))
  const currentProjectId = ref(DEFAULT_PROJECT_ID)
  const transactions = ref<LedgerEntry[]>([])
  const users = ref<UserAccount[]>([])
  const usedCodes = ref<string[]>([])
  const profile = ref({
    name: '',
    email: '',
    notifyTask: true,
    notifyBalance: true,
    notifyTicket: false,
  })
  const devices = ref<DeviceSession[]>([])
  const workflows = ref<WorkflowItem[]>([])
  const providers = ref<ProviderItem[]>([])
  const products = ref<RechargeProduct[]>([])
  const pricingRules = ref<PricingRule[]>([])
  const announcements = ref<Announcement[]>([])
  const settings = ref<PlatformSettings>({
    inputRetention: '7',
    resultRetention: '30',
    notify: true,
    audit: true,
  })
  const auditLogs = ref<AuditLog[]>([])
  const toast = ref('')

  let toastTimer: ReturnType<typeof setTimeout> | undefined
  let projectSeq = 2

  const openTicketCount = computed(() => adminTickets.value.filter((item) => item.status !== '已解决').length)
  const openClientTicketCount = computed(() => tickets.value.filter((item) => item.status !== '已解决').length)
  const activeTasks = computed(() => tasks.value.filter((task) => task.status === '生成中' || task.status === '排队中'))
  const completedTasks = computed(() => tasks.value.filter((task) => task.status === '已完成'))
  const activeProjects = computed(() => projects.value.filter((project) => !project.trashed))
  const trashedProjects = computed(() => projects.value.filter((project) => project.trashed))
  const currentProject = computed(() => activeProjects.value.find((project) => project.id === currentProjectId.value) ?? activeProjects.value[0] ?? null)
  const assets = computed(() => (currentProject.value ? flattenProject(currentProject.value) : []))
  const fileCount = computed(() => activeProjects.value.reduce((sum, project) => sum + countProjectFiles(project), 0))
  const storageUsed = computed(() => 0)
  const storageQuota = 5
  const storagePercent = computed(() => Math.min(100, Math.round((storageUsed.value / storageQuota) * 100)))

  function notify(message: string) {
    toast.value = message
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toast.value = '' }, 2800)
  }

  function notifyUnavailable() {
    notify(NOT_CONNECTED)
    return false
  }

  function selectProject(id: string) {
    const project = activeProjects.value.find((item) => item.id === id)
    if (!project) return false
    currentProjectId.value = id
    return true
  }

  function ensureCurrentProject() {
    if (currentProject.value) {
      currentProjectId.value = currentProject.value.id
      return currentProject.value
    }
    currentProjectId.value = ''
    return null
  }

  function createProject(name: string) {
    const trimmed = name.trim() || `未命名作品 ${projectSeq}`
    const project: WorkProject = {
      id: `prj_${projectSeq++}`,
      name: trimmed,
      desc: '空白工程',
      createdAt: '刚刚',
      trashed: false,
      folders: cloneFolders(createStandardFolders()),
    }
    projects.value.unshift(project)
    currentProjectId.value = project.id
    notify(`已创建空白作品「${project.name}」`)
    return project.id
  }

  function trashProject(id: string) {
    if (id === DEFAULT_PROJECT_ID) {
      notify('默认作品不能移入回收站')
      return false
    }
    const project = projects.value.find((item) => item.id === id)
    if (!project || project.trashed) {
      notify('该作品无法移入回收站')
      return false
    }
    project.trashed = true
    project.trashedAt = '刚刚'
    if (currentProjectId.value === id) ensureCurrentProject()
    notify(`「${project.name}」已移入回收站`)
    return true
  }

  function restoreProject(id: string) {
    const project = projects.value.find((item) => item.id === id)
    if (!project || !project.trashed) {
      notify('该作品不在回收站')
      return false
    }
    project.trashed = false
    project.trashedAt = undefined
    if (!currentProjectId.value) currentProjectId.value = project.id
    notify(`已恢复「${project.name}」`)
    return true
  }

  function purgeProject(id: string) {
    const index = projects.value.findIndex((item) => item.id === id && item.trashed)
    if (index < 0) {
      notify('只能从回收站彻底删除作品')
      return false
    }
    const project = projects.value[index]
    if (!project) return false
    projects.value.splice(index, 1)
    if (currentProjectId.value === id) ensureCurrentProject()
    notify(`已彻底删除「${project.name}」`)
    return true
  }

  function createTask(_cost: number, _options?: {
    title?: string
    workflow?: string
    durationSec?: number
    aspectRatio?: string
  }) {
    return notifyUnavailable()
  }

  function recharge(_amount: number) {
    notify(NOT_CONNECTED)
  }

  function redeem(_code: string) {
    return notifyUnavailable()
  }

  function cancelTask(_id: string) {
    return notifyUnavailable()
  }

  function retryTask(_id: string) {
    return notifyUnavailable()
  }

  function saveWorkToAsset(_id: string) {
    return notifyUnavailable()
  }

  function isWorkSaved(_taskId: string) {
    return false
  }

  function uploadAsset(_projectId?: string, _path: string[] = []) {
    return notifyUnavailable()
  }

  function removeAsset(_id: string) {
    return notifyUnavailable()
  }

  function createTicket(_title: string, _category: string, _task = '-') {
    notify(NOT_CONNECTED)
    return ''
  }

  function approveTicketRefund(_id: string) {
    return notifyUnavailable()
  }

  function resolveTicket(_id: string) {
    return notifyUnavailable()
  }

  function toggleUserStatus(_id: string) {
    return notifyUnavailable()
  }

  function revokeDevice(_id: string) {
    return notifyUnavailable()
  }

  function saveProfile() {
    return notifyUnavailable()
  }

  function toggleWorkflow(_id: string) {
    return notifyUnavailable()
  }

  function createWorkflow() {
    return notifyUnavailable()
  }

  function checkProvider(_name: string) {
    return notifyUnavailable()
  }

  function addProduct() {
    return notifyUnavailable()
  }

  function savePricing() {
    return notifyUnavailable()
  }

  function addPricingRule() {
    return notifyUnavailable()
  }

  function publishAnnouncement(_id: string) {
    return notifyUnavailable()
  }

  function createAnnouncement() {
    return notifyUnavailable()
  }

  function saveSettings() {
    return notifyUnavailable()
  }

  function refreshMonitoring() {
    return notifyUnavailable()
  }

  return {
    balance,
    draftPrompt,
    tasks,
    adminTasks,
    adminTickets,
    tickets,
    projects,
    currentProjectId,
    currentProject,
    activeProjects,
    trashedProjects,
    fileCount,
    assets,
    transactions,
    users,
    usedCodes,
    profile,
    devices,
    toast,
    activeTasks,
    completedTasks,
    storageUsed,
    storageQuota,
    storagePercent,
    notify,
    createTask,
    recharge,
    redeem,
    cancelTask,
    retryTask,
    saveWorkToAsset,
    isWorkSaved,
    uploadAsset,
    removeAsset,
    createProject,
    selectProject,
    trashProject,
    restoreProject,
    purgeProject,
    createTicket,
    approveTicketRefund,
    resolveTicket,
    toggleUserStatus,
    revokeDevice,
    saveProfile,
    workflows,
    providers,
    products,
    pricingRules,
    announcements,
    settings,
    auditLogs,
    openTicketCount,
    openClientTicketCount,
    toggleWorkflow,
    createWorkflow,
    checkProvider,
    addProduct,
    savePricing,
    addPricingRule,
    publishAnnouncement,
    createAnnouncement,
    saveSettings,
    refreshMonitoring,
  }
})
