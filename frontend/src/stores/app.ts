import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  adminTasks as seedAdminTasks,
  adminTickets as seedAdminTickets,
  assets as seedAssets,
  initialTasks,
  redeemCodes,
  tickets as seedTickets,
  transactions as seedTransactions,
  users as seedUsers,
  type Asset,
  type ClientTicket,
  type LedgerEntry,
  type Task,
  type TicketStatus,
} from '@/data/mock'

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
  const assets = ref<Asset[]>(seedAssets.map((item) => ({ ...item })))
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
  const toast = ref('')
  let toastTimer: ReturnType<typeof setTimeout> | undefined
  let assetSeq = seedAssets.length + 1
  let ticketSeq = 1030
  let ledgerSeq = 90200

  const activeTasks = computed(() => tasks.value.filter((task) => task.status === '生成中' || task.status === '排队中'))
  const completedTasks = computed(() => tasks.value.filter((task) => task.status === '已完成'))
  const storageUsed = computed(() => Number((1.6 + assets.value.length * 0.05).toFixed(1)))
  const storageQuota = 5
  const storagePercent = computed(() => Math.min(100, Math.round((storageUsed.value / storageQuota) * 100)))

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
    return assets.value.some((item) => item.sourceTask === taskId)
  }

  function createTask(title: string, cost: number) {
    if (balance.value < cost) {
      notify('钱包余额不足，请先充值')
      return false
    }
    balance.value -= cost
    syncCreatorBalance()
    const task: Task = {
      id: `tsk_${89000 + tasks.value.length}`,
      title: title || '未命名创作',
      workflow: 'H3 图生视频',
      status: '排队中',
      progress: 0,
      duration: '15 秒',
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
      notify('只能保存已完成的作品')
      return false
    }
    if (isWorkSaved(id)) {
      notify('该作品已经在资产库中')
      return false
    }
    assets.value.unshift({
      id: `ast-${assetSeq++}`,
      name: task.title,
      type: '视频',
      size: '12.0 MB',
      folder: '视频',
      color: '164e63',
      accent: '67e8f9',
      updated: '刚刚',
      sourceTask: task.id,
    })
    notify(`已将「${task.title}」保存到资产`)
    return true
  }

  function uploadAsset() {
    assets.value.unshift({
      id: `ast-${assetSeq++}`,
      name: `本地上传素材 ${assetSeq - 1}`,
      type: '图片',
      size: '1.2 MB',
      folder: '场景',
      color: '1e293b',
      accent: '818cf8',
      updated: '刚刚',
    })
    notify('已添加本地素材（原型）')
    return true
  }

  function removeAsset(id: string) {
    const index = assets.value.findIndex((item) => item.id === id)
    if (index < 0) return false
    const item = assets.value[index]
    if (!item) return false
    assets.value.splice(index, 1)
    notify(`已从资产库移除「${item.name}」`)
    return true
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

  return {
    balance,
    draftPrompt,
    tasks,
    adminTasks,
    adminTickets,
    tickets,
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
    createTicket,
    approveTicketRefund,
    resolveTicket,
    toggleUserStatus,
    revokeDevice,
    saveProfile,
  }
})
