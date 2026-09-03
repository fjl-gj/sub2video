export type TaskStatus = '排队中' | '生成中' | '已完成' | '失败'
export type TicketStatus = '待处理' | '处理中' | '已解决'
export type SceneAspectRatio = '16:9' | '9:16' | '1:1'

export interface Task {
  id: string
  title: string
  workflow: string
  status: TaskStatus
  progress: number
  duration: string
  resolution: string
  cost: number
  time: string
  thumb: string
  thumbUrl?: string
  savedToProject?: boolean
}

export interface LedgerEntry {
  id: string
  type: string
  amount: number
  desc: string
  time: string
}

export interface ClientTicket {
  id: string
  title: string
  category: string
  status: TicketStatus
  updated: string
  task: string
}

export interface SceneTemplate {
  id: string
  title: string
  category: string
  description: string
  color: string
  accent: string
  aspectRatio: SceneAspectRatio
  duration: string
  resolution: string
  poster: string
  preview: string
}

export interface AdminTask extends Task {
  user: string
  provider: string
}

export interface AdminTicket {
  id: string
  title: string
  user: string
  type: string
  status: TicketStatus
  amount?: number
  updated: string
}

export interface UserAccount {
  id: string
  name: string
  email: string
  status: string
  balance: number
  tasks: number
  joined: string
}

export interface WorkflowItem {
  id: string
  name: string
  description: string
  status: string
  provider: string
  price: string
  updated: string
}

export interface PricingRule {
  id: string
  name: string
  cost: string
  price: string
  duration: string
  status: string
  version: string
}

export interface Announcement {
  id: string
  title: string
  desc: string
  status: string
  date: string
}

export interface AuditLog {
  time: string
  operator: string
  action: string
  target: string
  result: string
}

export interface PlatformSettings {
  inputRetention: string
  resultRetention: string
  notify: boolean
  audit: boolean
}

export interface ProviderItem {
  name: string
  channel: string
  status: string
  latency: string
  concurrency: string
  cost: string
}

export interface RechargeProduct {
  id: string
  amount: number
  bonus: number
  price: number
}
