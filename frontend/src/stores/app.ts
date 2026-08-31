import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { adminTasks as seedAdminTasks, adminTickets as seedAdminTickets, initialTasks, type Task } from '@/data/mock'

export type AdminTask = (typeof seedAdminTasks)[number]
export type AdminTicket = (typeof seedAdminTickets)[number]

export const useAppStore = defineStore('app', () => {
  const balance = ref(256.0)
  const draftPrompt = ref('')
  const tasks = ref<Task[]>(initialTasks.map((task) => ({ ...task })))
  const adminTasks = ref<AdminTask[]>(seedAdminTasks.map((task) => ({ ...task })))
  const adminTickets = ref<AdminTicket[]>(seedAdminTickets.map((ticket) => ({ ...ticket })))
  const toast = ref('')
  let toastTimer: ReturnType<typeof setTimeout> | undefined

  const activeTasks = computed(() => tasks.value.filter((task) => task.status === '生成中' || task.status === '排队中'))
  const completedTasks = computed(() => tasks.value.filter((task) => task.status === '已完成'))

  function notify(message: string) {
    toast.value = message
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toast.value = '' }, 2800)
  }

  function createTask(title: string, cost: number) {
    if (balance.value < cost) {
      notify('钱包余额不足，请先充值')
      return false
    }
    balance.value -= cost
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
    notify(`任务已提交，预扣 ¥${cost.toFixed(2)}`)
    return true
  }

  function recharge(amount: number) {
    balance.value += amount
    notify(`充值成功，余额增加 ¥${amount.toFixed(2)}`)
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

  function approveTicketRefund(id: string) {
    const ticket = adminTickets.value.find((item) => item.id === id)
    if (!ticket || ticket.status === '已解决') return false
    ticket.status = '已解决'
    if (ticket.amount) balance.value += ticket.amount
    notify(`已核准 ${id} 退款${ticket.amount ? `，¥${ticket.amount.toFixed(2)} 已退回钱包` : ''}`)
    return true
  }

  function resolveTicket(id: string) {
    const ticket = adminTickets.value.find((item) => item.id === id)
    if (!ticket) return false
    ticket.status = '已解决'
    notify(`${id} 已标记为已解决`)
    return true
  }

  return {
    balance,
    draftPrompt,
    tasks,
    adminTasks,
    adminTickets,
    toast,
    activeTasks,
    completedTasks,
    notify,
    createTask,
    recharge,
    cancelTask,
    retryTask,
    approveTicketRefund,
    resolveTicket,
  }
})
