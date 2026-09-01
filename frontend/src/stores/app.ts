import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  adminTasks as seedAdminTasks,
  adminTickets as seedAdminTickets,
  announcements as seedAnnouncements,
  auditLogs as seedAuditLogs,
  initialTasks,
  platformSettings as seedSettings,
  pricingRules as seedPricingRules,
  providers as seedProviders,
  rechargeProducts as seedProducts,
  redeemCodes,
  tickets as seedTickets,
  transactions as seedTransactions,
  users as seedUsers,
  workflows as seedWorkflows,
  type ClientTicket,
  type LedgerEntry,
  type Task,
  type TicketStatus,
} from '@/data/mock'
import {
  cloneFolders,
  countProjectFiles,
  createStandardFolders,
  DEFAULT_PROJECT_ID,
  DEFAULT_VIDEO_ALBUM,
  findFileInProject,
  flattenProject,
  initialProjects,
  resolveNode,
  type FsItem,
  type WorkProject,
} from '@/data/projects'

export type AdminTask = (typeof seedAdminTasks)[number]
export type AdminTicket = (typeof seedAdminTickets)[number]
export type UserAccount = (typeof seedUsers)[number]
export type DeviceSession = { id: string; name: string; meta: string; current: boolean }

export const useAppStore = defineStore('app', () => {
  const balance = ref(256.0)
  const draftPrompt = ref('')
  const tasks = ref<Task[]>(initialTasks.map((task) => ({ ...task })))
  const adminTasks = ref<AdminTask[]>(seedAdminTasks.map((task) => ({ ...task })))
  const adminTickets = ref<AdminTicket[]>(seedAdminTickets.map((ticket) => ({ ...ticket })))
  const tickets = ref<ClientTicket[]>(seedTickets.map((ticket) => ({ ...ticket })))
  const projects = ref<WorkProject[]>(initialProjects.map((project) => ({
    ...project,
    folders: cloneFolders(project.folders),
  })))
  const currentProjectId = ref(DEFAULT_PROJECT_ID)
  const transactions = ref<LedgerEntry[]>(seedTransactions.map((item) => ({ ...item })))
  const users = ref<UserAccount[]>(seedUsers.map((item) => ({ ...item })))
  const usedCodes = ref<string[]>([])
  const profile = ref({
    name: '林默',
    email: 'linmo@example.com',
    notifyTask: true,
    notifyBalance: true,
    notifyTicket: false,
  })
  const devices = ref<DeviceSession[]>([
    { id: 'dev-1', name: 'Windows · Chrome', meta: '当前设备 · 今天 09:42', current: true },
    { id: 'dev-2', name: 'iPhone 15', meta: '上海 · 昨天 21:18', current: false },
  ])
  const workflows = ref(seedWorkflows.map((item) => ({ ...item })))
  const providers = ref(seedProviders.map((item) => ({ ...item })))
  const products = ref(seedProducts.map((item) => ({ ...item })))
  const pricingRules = ref(seedPricingRules.map((item) => ({ ...item })))
  const announcements = ref(seedAnnouncements.map((item) => ({ ...item })))
  const settings = ref({ ...seedSettings })
  const auditLogs = ref(seedAuditLogs.map((item) => ({ ...item })))
  const openTicketCount = computed(() => adminTickets.value.filter((item) => item.status !== '已解决').length)
  const openClientTicketCount = computed(() => tickets.value.filter((item) => item.status !== '已解决').length)
  const toast = ref('')
  let toastTimer: ReturnType<typeof setTimeout> | undefined
  let assetSeq = 7
  let projectSeq = 2
  let ticketSeq = 1030
  let ledgerSeq = 90200

  const activeTasks = computed(() => tasks.value.filter((task) => task.status === '生成中' || task.status === '排队中'))
  const completedTasks = computed(() => tasks.value.filter((task) => task.status === '已完成'))
  const activeProjects = computed(() => projects.value.filter((project) => !project.trashed))
  const trashedProjects = computed(() => projects.value.filter((project) => project.trashed))
  const currentProject = computed(() => activeProjects.value.find((project) => project.id === currentProjectId.value) ?? activeProjects.value[0] ?? null)
  const assets = computed(() => (currentProject.value ? flattenProject(currentProject.value) : []))
  const fileCount = computed(() => activeProjects.value.reduce((sum, project) => sum + countProjectFiles(project), 0))
  const storageUsed = computed(() => Number((1.6 + fileCount.value * 0.05).toFixed(1)))
  const storageQuota = 5
  const storagePercent = computed(() => Math.min(100, Math.round((storageUsed.value / storageQuota) * 100)))

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

  function notify(message: string) {
    toast.value = message
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toast.value = '' }, 2800)
  }

  function syncCreatorBalance() {
    const user = users.value.find((item) => item.name === '林默')
    if (user) user.balance = balance.value
  }

  function pushLedger(type: string, amount: number, desc: string) {
    transactions.value.unshift({
      id: `TX_${ledgerSeq++}`,
      type,
      amount,
      desc,
      time: '刚刚',
    })
  }

  function syncTicketStatus(id: string, status: TicketStatus) {
    const client = tickets.value.find((item) => item.id === id)
    if (client) {
      client.status = status
      client.updated = '刚刚'
    }
    const admin = adminTickets.value.find((item) => item.id === id)
    if (admin) {
      admin.status = status
      admin.updated = '刚刚'
    }
  }

  function isWorkSaved(taskId: string) {
    return projects.value.some((project) => flattenProject(project).some((item) => item.sourceTask === taskId))
  }

  function createProject(name: string) {
    const trimmed = name.trim() || `未命名作品 ${projectSeq}`
    const project: WorkProject = {
      id: `prj_${projectSeq++}`,
      name: trimmed,
      desc: '按标准工程模板创建',
      createdAt: '刚刚',
      trashed: false,
      folders: cloneFolders(createStandardFolders()),
    }
    projects.value.unshift(project)
    currentProjectId.value = project.id
    notify(`已按标准模板创建「${project.name}」`)
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

  function targetFolderForSave(project: WorkProject) {
    const video = project.folders['视频']
    return video?.folders[DEFAULT_VIDEO_ALBUM] ?? video
  }

  function createTask(
    cost: number,
    options?: {
      title?: string
      workflow?: string
      durationSec?: number
      aspectRatio?: string
    },
  ) {
    if (balance.value < cost) {
      notify('钱包余额不足，请先充值')
      return false
    }
    const durationSec = options?.durationSec ?? 15
    balance.value -= cost
    syncCreatorBalance()
    const task: Task = {
      id: `tsk_${89000 + tasks.value.length}`,
      title: options?.title || '未命名创作',
      workflow: options?.workflow || 'H3 图生视频',
      status: '排队中',
      progress: 0,
      duration: `${durationSec} 秒`,
      resolution: '768P',
      cost,
      time: '刚刚',
      thumb: 'https://images.unsplash.com/photo-1519608487953-e999c86e7455?auto=format&fit=crop&w=480&q=80',
    }
    tasks.value.unshift(task)
    adminTasks.value.unshift({ ...task, user: '林默', provider: 'MiniMax H3' })
    pushLedger('任务消费', -cost, `${task.id} ${task.title}`)
    notify(`任务已提交，预扣 ¥${cost.toFixed(2)}`)
    return true
  }

  function recharge(amount: number) {
    balance.value += amount
    syncCreatorBalance()
    pushLedger('充值到账', amount, '余额充值 · 微信支付')
    notify(`充值成功，余额增加 ¥${amount.toFixed(2)}`)
  }

  function redeem(code: string) {
    const normalized = code.trim().toUpperCase()
    const found = redeemCodes.find((item) => item.code === normalized)
    if (!found) {
      notify('兑换码无效或已过期')
      return false
    }
    if (usedCodes.value.includes(found.code)) {
      notify('该兑换码已经使用过')
      return false
    }
    usedCodes.value.push(found.code)
    balance.value += found.amount
    syncCreatorBalance()
    pushLedger('兑换到账', found.amount, `${found.code} · ${found.label}`)
    notify(`兑换成功，余额增加 ¥${found.amount.toFixed(2)}`)
    return true
  }

  function cancelTask(id: string) {
    const task = tasks.value.find((item) => item.id === id)
    const admin = adminTasks.value.find((item) => item.id === id)
    const active = task && (task.status === '生成中' || task.status === '排队中')
    const adminActive = admin && (admin.status === '生成中' || admin.status === '排队中')
    if (!active && !adminActive) {
      notify('该任务无法终止')
      return false
    }
    const refund = task?.cost ?? admin?.cost ?? 0
    if (task && active) {
      task.status = '失败'
      task.progress = 0
      balance.value += task.cost
    }
    if (admin && adminActive) {
      admin.status = '失败'
      admin.progress = 0
      if (!task) balance.value += admin.cost
    }
    syncCreatorBalance()
    pushLedger('任务退款', refund, `${id} 终止退款`)
    notify(`已终止 ${id}，预扣费用已退回钱包`)
    return true
  }

  function retryTask(id: string) {
    const task = tasks.value.find((item) => item.id === id)
    if (!task || task.status !== '失败') return false
    task.status = '排队中'
    task.progress = 0
    const admin = adminTasks.value.find((item) => item.id === id)
    if (admin) {
      admin.status = '排队中'
      admin.progress = 0
    }
    notify('任务已重新排队，费用不会重复扣除')
    return true
  }

  function saveWorkToAsset(id: string) {
    const task = tasks.value.find((item) => item.id === id)
    if (!task || task.status !== '已完成') {
      notify('只能保存已完成的任务结果')
      return false
    }
    if (isWorkSaved(id)) {
      notify('该结果已经保存在作品中')
      return false
    }
    const project = ensureCurrentProject()
    if (!project) {
      notify('请先新建或恢复一个作品')
      return false
    }
    const folder = targetFolderForSave(project)
    if (!folder) {
      notify('当前作品缺少视频目录')
      return false
    }
    folder.items.unshift({
      id: `ast-${assetSeq++}`,
      name: `${task.title}.mp4`,
      type: 'video',
      size: '12.0 MB',
      time: '刚刚',
      meta: `${task.resolution} · ${task.duration}`,
      sourceTask: task.id,
    })
    notify(`已将「${task.title}」保存到「${project.name} / 视频」`)
    return true
  }

  function uploadAsset(projectId?: string, path: string[] = []) {
    const project = projects.value.find((item) => item.id === (projectId || currentProject.value?.id) && !item.trashed)
    if (!project) {
      notify('请先进入一个作品再上传')
      return false
    }
    const targetPath = path.length ? path : ['资产', '素材']
    const node = resolveNode(project, targetPath)
    if (!node.folder) {
      notify('当前目录不可上传')
      return false
    }
    const item: FsItem = {
      id: `ast-${assetSeq++}`,
      name: `本地上传素材 ${assetSeq - 1}.png`,
      type: 'image',
      size: '1.2 MB',
      time: '刚刚',
    }
    node.folder.items.unshift(item)
    notify(`已上传到「${targetPath.join(' / ')}」`)
    return true
  }

  function removeAsset(id: string) {
    for (const project of projects.value) {
      const found = findFileInProject(project, id)
      if (!found) continue
      found.folder.items.splice(found.index, 1)
      notify(`已移除「${found.item.name}」`)
      return true
    }
    return false
  }

  function createTicket(title: string, category: string, task = '-') {
    const id = `TKT-${ticketSeq++}`
    tickets.value.unshift({
      id,
      title,
      category,
      status: '待处理',
      updated: '刚刚',
      task: task || '-',
    })
    adminTickets.value.unshift({
      id,
      user: '林默',
      title,
      type: category,
      status: '待处理',
      amount: category === '售后支持' ? 4.5 : 0,
      updated: '刚刚',
    })
    notify(`${id} 已提交，客服会在工作时间内回复`)
    return id
  }

  function approveTicketRefund(id: string) {
    const ticket = adminTickets.value.find((item) => item.id === id)
    if (!ticket || ticket.status === '已解决') return false
    syncTicketStatus(id, '已解决')
    if (ticket.amount) {
      balance.value += ticket.amount
      syncCreatorBalance()
      pushLedger('工单退款', ticket.amount, `${id} ${ticket.title}`)
    }
    notify(`已核准 ${id} 退款${ticket.amount ? `，¥${ticket.amount.toFixed(2)} 已退回钱包` : ''}`)
    pushAudit('处理工单退款', id)
    return true
  }

  function resolveTicket(id: string) {
    const ticket = adminTickets.value.find((item) => item.id === id)
    if (!ticket) return false
    syncTicketStatus(id, '已解决')
    notify(`${id} 已标记为已解决`)
    return true
  }

  function toggleUserStatus(id: string) {
    const user = users.value.find((item) => item.id === id)
    if (!user) return false
    user.status = user.status === '已冻结' ? '正常' : '已冻结'
    pushAudit(user.status === '已冻结' ? '冻结用户' : '解冻用户', user.name)
    notify(`${user.name} 已${user.status === '已冻结' ? '冻结' : '解冻'}`)
    return true
  }

  function revokeDevice(id: string) {
    const device = devices.value.find((item) => item.id === id)
    if (!device || device.current) {
      notify('当前设备不能下线')
      return false
    }
    devices.value = devices.value.filter((item) => item.id !== id)
    notify(`已下线 ${device.name}`)
    return true
  }

  function saveProfile() {
    notify('个人资料已保存')
    return true
  }

  function pushAudit(action: string, target: string) {
    auditLogs.value.unshift({ time: '刚刚', operator: 'admin', action, target, result: '成功' })
  }

  function toggleWorkflow(id: string) {
    const item = workflows.value.find((row) => row.id === id)
    if (!item || item.status === '未开放') {
      notify('该工作流尚未开放')
      return false
    }
    item.status = item.status === '已上架' ? '草稿' : '已上架'
    item.updated = '刚刚'
    pushAudit(item.status === '已上架' ? '发布工作流' : '下架工作流', item.name)
    notify(`${item.name} 已${item.status === '已上架' ? '上架' : '改为草稿'}`)
    return true
  }

  function createWorkflow() {
    const id = `wf-draft-${workflows.value.length + 1}`
    workflows.value.unshift({
      id,
      name: '未命名工作流',
      description: '草稿工作流，确认供应商能力后再上架。',
      status: '草稿',
      provider: 'MiniMax H3',
      price: '待定',
      updated: '刚刚',
    })
    pushAudit('新建工作流', id)
    notify('已创建草稿工作流')
    return true
  }

  function checkProvider(name: string) {
    const provider = providers.value.find((item) => item.name === name)
    if (!provider) return false
    if (provider.status === '未启用') {
      provider.status = '健康'
      provider.latency = '20.0 s'
      provider.concurrency = '1 / 8'
      provider.cost = '0.02 元/秒'
      provider.channel = '备用节点'
    }
    pushAudit('健康检查', name)
    notify(`已刷新 ${name} 健康状态`)
    return true
  }

  function addProduct() {
    const amount = 1000
    products.value.push({ id: `p-${products.value.length + 1}`, amount, bonus: 120, price: amount })
    pushAudit('新增充值商品', `¥${amount}`)
    notify('已新增 ¥1000 档位（赠 ¥120）')
    return true
  }

  function savePricing() {
    pricingRules.value.forEach((rule) => {
      if (rule.status === '生效中') {
        const current = Number(rule.version.replace('v', ''))
        rule.version = `v${(current + 0.1).toFixed(1)}`
      }
    })
    pushAudit('调整定价规则', pricingRules.value[0]?.name || '费率矩阵')
    notify('定价规则已保存，后续新任务将使用新版本')
    return true
  }

  function addPricingRule() {
    pricingRules.value.push({
      id: `pr-${pricingRules.value.length + 1}`,
      name: '未命名规则',
      cost: '0.02',
      price: '0.50',
      duration: '按秒',
      status: '草稿',
      version: 'v0.1',
    })
    notify('已新增草稿定价规则')
    return true
  }

  function publishAnnouncement(id: string) {
    const item = announcements.value.find((row) => row.id === id)
    if (!item) return false
    item.status = item.status === '已发布' ? '草稿' : '已发布'
    item.date = '刚刚'
    pushAudit(item.status === '已发布' ? '发布公告' : '撤回公告', item.title)
    notify(`「${item.title}」已${item.status === '已发布' ? '发布' : '改回草稿'}`)
    return true
  }

  function createAnnouncement() {
    announcements.value.unshift({
      id: `n-${announcements.value.length + 1}`,
      title: '未命名公告',
      desc: '草稿公告，确认文案后再发布给创作者。',
      status: '草稿',
      date: '刚刚',
    })
    notify('已创建草稿公告')
    return true
  }

  function saveSettings() {
    pushAudit('保存系统设置', `输入 ${settings.value.inputRetention} 天 / 结果 ${settings.value.resultRetention} 天`)
    notify('系统设置已保存')
    return true
  }

  function refreshMonitoring() {
    pushAudit('刷新监控', '供应商健康')
    notify('监控数据已刷新')
    return true
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
