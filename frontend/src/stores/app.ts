import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { initialTasks, type Task } from '@/data/mock'

export const useAppStore = defineStore('app', () => {
  const balance = ref(256.0)
  const draftPrompt = ref('')
  const tasks = ref<Task[]>([...initialTasks])
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
    tasks.value.unshift({
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
    })
    notify(`任务已提交，预扣 ¥${cost.toFixed(2)}`)
    return true
  }

  function recharge(amount: number) {
    balance.value += amount
    notify(`充值成功，余额增加 ¥${amount.toFixed(2)}`)
  }

  return { balance, draftPrompt, tasks, toast, activeTasks, completedTasks, notify, createTask, recharge }
})
