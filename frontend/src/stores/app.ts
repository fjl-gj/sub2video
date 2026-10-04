import { computed, onScopeDispose, ref } from 'vue'
import { CanceledError, isCancel } from 'axios'
import { defineStore } from 'pinia'
import { getAuthSession, onAuthSessionChange } from '@/shared/lib/auth-session'
import { useAuthStore } from '@/stores/auth'
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
  flattenProject,
  type WorkProject,
} from '@/data/projects'
import {
  createProject as createProjectApi,
  deleteProjectFile,
  fetchProjectAssets,
  fetchProjects,
  fetchProjectTree,
  fetchStorageUsage,
  purgeProject as purgeProjectApi,
  restoreProject as restoreProjectApi,
  saveTaskToProject,
  setCurrentProject as setCurrentProjectApi,
  trashProject as trashProjectApi,
  uploadProjectFile,
  type ProjectSummary,
} from '@/api/client'

export type { AdminTask, AdminTicket, UserAccount }
export type DeviceSession = { id: string; name: string; meta: string; current: boolean }

const NOT_CONNECTED = '功能尚未接入后端'

function summaryToProject(item: ProjectSummary): WorkProject {
  return {
    id: item.id,
    name: item.name,
    desc: item.desc,
    createdAt: item.createdAt,
    trashed: item.trashed,
    trashedAt: item.trashedAt,
    isDefault: item.isDefault,
    fileCount: item.fileCount,
    folders: cloneFolders(createStandardFolders()),
  }
}

export const useAppStore = defineStore('app', () => {
  const auth = useAuthStore()
  const balance = computed(() => auth.balance)
  const draftPrompt = ref('')
  const tasks = ref<Task[]>([])
  const adminTasks = ref<AdminTask[]>([])
  const adminTickets = ref<AdminTicket[]>([])
  const tickets = ref<ClientTicket[]>([])
  const projects = ref<WorkProject[]>([])
  const currentProjectId = ref('')
  const projectsLoaded = ref(false)
  const storageUsed = ref(0)
  const storageQuota = ref(5)
  const storageFileCount = ref(0)
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

  function resetPrivateState() {
    draftPrompt.value = ''
    tasks.value = []; adminTasks.value = []; adminTickets.value = []; tickets.value = []
    projects.value = []; currentProjectId.value = ''; projectsLoaded.value = false
    storageUsed.value = 0; storageQuota.value = 5; storageFileCount.value = 0
    transactions.value = []; users.value = []; usedCodes.value = []
    profile.value = { name: '', email: '', notifyTask: true, notifyBalance: true, notifyTicket: false }
    devices.value = []; workflows.value = []; providers.value = []; products.value = []
    pricingRules.value = []; announcements.value = []; auditLogs.value = []
    settings.value = { inputRetention: '7', resultRetention: '30', notify: true, audit: true }
    toast.value = ''
    if (toastTimer) clearTimeout(toastTimer)
  }
  const unsubscribeSession = onAuthSessionChange((next, previous) => {
    if (next.id !== previous.id || !next.accessToken) resetPrivateState()
  })
  onScopeDispose(() => { unsubscribeSession(); if (toastTimer) clearTimeout(toastTimer) })

  function sessionGuard() {
    const expected = getAuthSession().id
    return () => {
      if (getAuthSession().id !== expected) throw new CanceledError('登录会话已变化，请重试')
    }
  }

  const openTicketCount = computed(() => adminTickets.value.filter((item) => item.status !== '已解决').length)
  const openClientTicketCount = computed(() => tickets.value.filter((item) => item.status !== '已解决').length)
  const activeTasks = computed(() => tasks.value.filter((task) => task.status === '生成中' || task.status === '排队中'))
  const completedTasks = computed(() => tasks.value.filter((task) => task.status === '已完成'))
  const activeProjects = computed(() => projects.value.filter((project) => !project.trashed))
  const trashedProjects = computed(() => projects.value.filter((project) => project.trashed))
  const currentProject = computed(() => activeProjects.value.find((project) => project.id === currentProjectId.value) ?? activeProjects.value[0] ?? null)
  const assets = computed(() => (currentProject.value ? flattenProject(currentProject.value) : []))
  const fileCount = computed(() => storageFileCount.value || activeProjects.value.reduce((sum, project) => sum + countProjectFiles(project), 0))
  const storagePercent = computed(() => {
    if (!storageQuota.value) return 0
    return Math.min(100, Math.round((storageUsed.value / storageQuota.value) * 100))
  })

  function notify(message: string) {
    toast.value = message
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toast.value = '' }, 2800)
  }

  function notifyUnavailable() {
    notify(NOT_CONNECTED)
    return false
  }

  function apiErrorMessage(error: unknown, fallback: string) {
    const err = error as { response?: { data?: { message?: string; error?: string } }; message?: string }
    return err.response?.data?.message || err.response?.data?.error || err.message || fallback
  }

  function upsertProject(next: WorkProject) {
    const index = projects.value.findIndex((item) => item.id === next.id)
    if (index >= 0) {
      projects.value[index] = { ...projects.value[index], ...next, folders: next.folders }
    } else {
      projects.value.unshift(next)
    }
  }

  async function refreshStorage() {
    const assertSession = sessionGuard()
    try {
      const usage = await fetchStorageUsage()
      assertSession()
      storageUsed.value = usage.usedGb
      storageQuota.value = usage.quotaGb
      storageFileCount.value = usage.fileCount
    } catch {
      /* ignore */
    }
  }

  async function loadProjects(options?: { includeTrash?: boolean }) {
    const assertSession = sessionGuard()
    const includeTrash = options?.includeTrash ?? true
    const active = await fetchProjects(false)
    assertSession()
    const mapped = active.items.map(summaryToProject)
    if (includeTrash) {
      const trash = await fetchProjects(true)
      assertSession()
      mapped.push(...trash.items.map(summaryToProject))
    }
    projects.value = mapped
    currentProjectId.value = active.currentProjectId || mapped.find((item) => !item.trashed)?.id || ''
    projectsLoaded.value = true
    await refreshStorage()
    assertSession()
    return mapped
  }

  async function loadProjectTree(id: string) {
    const assertSession = sessionGuard()
    const tree = await fetchProjectTree(id)
    assertSession()
    const project: WorkProject = {
      id: tree.id,
      name: tree.name,
      desc: tree.desc,
      createdAt: tree.createdAt,
      trashed: tree.trashed,
      trashedAt: tree.trashedAt,
      isDefault: tree.isDefault,
      fileCount: tree.fileCount,
      folders: tree.folders || cloneFolders(createStandardFolders()),
    }
    upsertProject(project)
    return project
  }

  async function selectProject(id: string) {
    const assertSession = sessionGuard()
    const project = activeProjects.value.find((item) => item.id === id)
    if (!project) return false
    currentProjectId.value = id
    try {
      await setCurrentProjectApi(id)
      assertSession()
      await loadProjectTree(id)
      assertSession()
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '切换作品失败'))
      return false
    }
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

  async function createProject(name: string) {
    const assertSession = sessionGuard()
    try {
      const { project } = await createProjectApi(name)
      assertSession()
      const next = summaryToProject(project)
      upsertProject(next)
      currentProjectId.value = next.id
      await loadProjectTree(next.id)
      assertSession()
      await refreshStorage()
      assertSession()
      notify(`已创建空白作品「${next.name}」`)
      return next.id
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '创建作品失败'))
      return ''
    }
  }

  async function trashProject(id: string) {
    const assertSession = sessionGuard()
    try {
      const { project } = await trashProjectApi(id)
      assertSession()
      const existing = projects.value.find((item) => item.id === id)
      upsertProject({
        ...(existing || summaryToProject(project)),
        ...summaryToProject(project),
        folders: existing?.folders || cloneFolders(createStandardFolders()),
      })
      if (currentProjectId.value === id) ensureCurrentProject()
      await refreshStorage()
      assertSession()
      notify(`「${project.name}」已移入回收站`)
      return true
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '移入回收站失败'))
      return false
    }
  }

  async function restoreProject(id: string) {
    const assertSession = sessionGuard()
    try {
      const { project } = await restoreProjectApi(id)
      assertSession()
      const existing = projects.value.find((item) => item.id === id)
      upsertProject({
        ...(existing || summaryToProject(project)),
        ...summaryToProject(project),
        folders: existing?.folders || cloneFolders(createStandardFolders()),
      })
      if (!currentProjectId.value) currentProjectId.value = project.id
      await refreshStorage()
      assertSession()
      notify(`已恢复「${project.name}」`)
      return true
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '恢复失败'))
      return false
    }
  }

  async function purgeProject(id: string) {
    const assertSession = sessionGuard()
    try {
      await purgeProjectApi(id)
      assertSession()
      projects.value = projects.value.filter((item) => item.id !== id)
      if (currentProjectId.value === id) ensureCurrentProject()
      await refreshStorage()
      assertSession()
      notify('已彻底删除作品')
      return true
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '彻底删除失败'))
      return false
    }
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

  async function saveWorkToAsset(id: string) {
    const assertSession = sessionGuard()
    try {
      await saveTaskToProject(id, currentProjectId.value || undefined)
      assertSession()
      if (currentProjectId.value) await loadProjectTree(currentProjectId.value)
      assertSession()
      await refreshStorage()
      assertSession()
      notify('已保存到当前作品')
      return true
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '保存到作品失败'))
      return false
    }
  }

  function isWorkSaved(taskId: string) {
    return assets.value.some((asset) => asset.sourceTask === taskId)
  }

  async function uploadAsset(projectId?: string, path: string[] = [], file?: File) {
    const assertSession = sessionGuard()
    const id = projectId || currentProjectId.value
    if (!id || !file) {
      notify('请选择要上传的文件')
      return false
    }
    try {
      await uploadProjectFile(id, path.length ? path : ['资产', '素材'], file)
      assertSession()
      await loadProjectTree(id)
      assertSession()
      await refreshStorage()
      assertSession()
      notify('素材已上传')
      return true
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '上传失败'))
      return false
    }
  }

  async function removeAsset(id: string) {
    const assertSession = sessionGuard()
    const projectId = currentProjectId.value
    if (!projectId) return false
    try {
      await deleteProjectFile(projectId, id)
      assertSession()
      await loadProjectTree(projectId)
      assertSession()
      await refreshStorage()
      assertSession()
      notify('已删除文件')
      return true
    } catch (e: unknown) {
      if (!isCancel(e)) notify(apiErrorMessage(e, '删除失败'))
      return false
    }
  }

  async function loadCurrentAssets(type?: string) {
    const assertSession = sessionGuard()
    const id = currentProjectId.value
    if (!id) return []
    const { items } = await fetchProjectAssets(id, type)
    assertSession()
    return items
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
    projectsLoaded,
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
    loadProjects,
    loadProjectTree,
    refreshStorage,
    loadCurrentAssets,
    createTask,
    recharge,
    redeem,
    cancelTask,
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
