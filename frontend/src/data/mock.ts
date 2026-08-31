export type TaskStatus = '排队中' | '生成中' | '已完成' | '失败'
export type AssetType = '图片' | '视频' | '音频'
export type TicketStatus = '待处理' | '处理中' | '已解决'

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
}

export interface Asset {
  id: string
  name: string
  type: AssetType
  size: string
  folder: string
  color: string
  accent: string
  updated: string
  sourceTask?: string
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

export const sceneTemplates = [
  { id: 'scene-1', title: '雨夜霓虹漫步', category: '电影感', description: '潮湿街巷、霓虹反射和稳定跟拍，适合人物氛围片。', color: '1e293b', accent: '818cf8' },
  { id: 'scene-2', title: '晨雾山谷航拍', category: '自然', description: '从云层穿行到山谷全景，强调空间和纵深。', color: '164e63', accent: '67e8f9' },
  { id: 'scene-3', title: '产品桌面特写', category: '商业', description: '柔和侧光与微距移动，适合新品展示和电商素材。', color: '3f3f46', accent: 'fbbf24' },
  { id: 'scene-4', title: '城市驶过窗边', category: '生活', description: '车窗视角的城市流动感，节奏轻快自然。', color: '334155', accent: 'fda4af' },
  { id: 'scene-5', title: '纸上故事展开', category: '叙事', description: '手绘纸张逐帧展开，适合品牌故事和知识内容。', color: '713f12', accent: 'fde68a' },
  { id: 'scene-6', title: '静物光影实验', category: '艺术', description: '光影在静物表面缓慢移动，突出材质和细节。', color: '312e81', accent: 'c4b5fd' },
]

export const initialTasks: Task[] = [
  { id: 'tsk_88921', title: '雨夜街道霓虹漫步', workflow: 'H3 图生视频', status: '生成中', progress: 68, duration: '15 秒', resolution: '768P', cost: 4.5, time: '今天 14:20', thumb: 'https://images.unsplash.com/photo-1519608487953-e999c86e7455?auto=format&fit=crop&w=480&q=80' },
  { id: 'tsk_88918', title: '产品桌面柔光展示', workflow: 'H3 图生视频', status: '已完成', progress: 100, duration: '15 秒', resolution: '768P', cost: 4.5, time: '今天 13:42', thumb: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=480&q=80' },
  { id: 'tsk_88840', title: '晨雾山谷全景', workflow: 'H3 图生视频', status: '失败', progress: 0, duration: '15 秒', resolution: '768P', cost: 4.5, time: '昨天 18:09', thumb: 'https://images.unsplash.com/photo-1500534623283-312aade485b7?auto=format&fit=crop&w=480&q=80' },
  { id: 'tsk_88812', title: '咖啡店窗边人物', workflow: 'H3 图生视频', status: '已完成', progress: 100, duration: '15 秒', resolution: '768P', cost: 4.5, time: '昨天 15:31', thumb: 'https://images.unsplash.com/photo-1445116572660-236099ec97a0?auto=format&fit=crop&w=480&q=80' },
]

export const assets: Asset[] = [
  { id: 'ast-1', name: '雨夜街景参考', type: '图片', size: '2.4 MB', folder: '场景', color: '1e293b', accent: '818cf8', updated: '今天 14:18' },
  { id: 'ast-2', name: '产品主视觉', type: '图片', size: '1.8 MB', folder: '产品', color: '3f3f46', accent: 'fbbf24', updated: '今天 13:36' },
  { id: 'ast-3', name: '人物侧脸素材', type: '图片', size: '3.1 MB', folder: '人物', color: '7f1d1d', accent: 'fca5a5', updated: '昨天 19:22' },
  { id: 'ast-4', name: '山谷航拍片段', type: '视频', size: '18.6 MB', folder: '视频', color: '164e63', accent: '67e8f9', updated: '昨天 17:04' },
  { id: 'ast-5', name: '品牌 Logo', type: '图片', size: '84 KB', folder: '品牌', color: '312e81', accent: 'c4b5fd', updated: '8 月 28 日' },
  { id: 'ast-6', name: '咖啡店环境音', type: '音频', size: '4.2 MB', folder: '音频', color: '713f12', accent: 'fde68a', updated: '8 月 27 日' },
]

export const transactions: LedgerEntry[] = [
  { id: 'TX_90182', type: '任务消费', amount: -4.5, desc: 'tsk_88921 雨夜街道霓虹漫步', time: '今天 14:20' },
  { id: 'TX_90177', type: '充值到账', amount: 100, desc: '余额充值 · 微信支付', time: '今天 13:10' },
  { id: 'TX_90120', type: '任务退款', amount: 4.5, desc: 'tsk_88840 生成失败退款', time: '昨天 18:12' },
  { id: 'TX_90096', type: '任务消费', amount: -4.5, desc: 'tsk_88812 咖啡店窗边人物', time: '昨天 15:31' },
]

export const rechargeProducts = [
  { id: 'p-1', amount: 50, bonus: 0, price: 50 },
  { id: 'p-2', amount: 100, bonus: 5, price: 100 },
  { id: 'p-3', amount: 300, bonus: 24, price: 300 },
  { id: 'p-4', amount: 500, bonus: 50, price: 500 },
]

export const tickets: ClientTicket[] = [
  { id: 'TKT-1028', title: '任务生成失败，如何申请退款？', category: '售后支持', status: '处理中', updated: '今天 18:12', task: 'tsk_88840' },
  { id: 'TKT-1019', title: '想了解 2K 分辨率的开放时间', category: '售前咨询', status: '已解决', updated: '8 月 28 日', task: '-' },
]

export const redeemCodes = [
  { code: 'S2V-WELCOME-20', amount: 20, label: '新用户体验金' },
]

export const workflows = [
  { id: 'wf-h3-image', name: 'H3 图生视频', description: '使用一张首帧图片和文字描述生成 768P 视频。', status: '已上架', provider: 'MiniMax H3', price: '0.50 元/秒', updated: '今天 11:28' },
  { id: 'wf-h3-text', name: 'H3 文生视频', description: '从文字提示生成视频，等待供应商能力确认。', status: '草稿', provider: 'MiniMax H3', price: '待定', updated: '昨天 16:04' },
  { id: 'wf-h3-multi', name: 'H3 多图生视频', description: '组合多张参考图控制角色和场景连续性。', status: '未开放', provider: 'MiniMax H3', price: '待定', updated: '8 月 26 日' },
]

export const providers = [
  { name: 'MiniMax H3', channel: '官方 API', status: '健康', latency: '12.4 s', concurrency: '18 / 40', cost: '0.50 元/秒' },
  { name: 'AutoDL ComfyUI', channel: '国内 GPU 节点', status: '健康', latency: '18.9 s', concurrency: '9 / 16', cost: '0.02 元/秒' },
  { name: '备用供应商', channel: '未配置', status: '未启用', latency: '-', concurrency: '-', cost: '-' },
]

export const adminTasks = initialTasks.map((task, index) => ({ ...task, user: ['林默', '陈也', '苏晚', '林默'][index], provider: index % 2 ? 'MiniMax H3' : 'AutoDL ComfyUI' }))

export const users = [
  { id: 'u_98241029', name: '林默', email: 'linmo@example.com', status: '正常', balance: 256.0, tasks: 28, joined: '2026-08-12' },
  { id: 'u_98241011', name: '陈也', email: 'chenye@example.com', status: '正常', balance: 82.5, tasks: 9, joined: '2026-08-19' },
  { id: 'u_98240987', name: '苏晚', email: 'suwan@example.com', status: '待验证', balance: 0, tasks: 0, joined: '2026-08-30' },
  { id: 'u_98240902', name: '赵临', email: 'zhaolin@example.com', status: '已冻结', balance: 14.2, tasks: 5, joined: '2026-08-05' },
]

export const adminTickets = [
  { id: 'TKT-1028', user: '林默', title: '任务生成失败，如何申请退款？', type: '退款申请', status: '待处理', amount: 4.5, updated: '今天 18:12' },
  { id: 'TKT-1024', user: '陈也', title: '视频结果出现黑屏', type: '生成异常', status: '处理中', amount: 4.5, updated: '今天 16:48' },
  { id: 'TKT-1019', user: '苏晚', title: '想了解 2K 分辨率开放时间', type: '售前咨询', status: '已解决', amount: 0, updated: '8 月 28 日' },
]

export const auditLogs = [
  { time: '今天 18:22:04', operator: 'admin', action: '调整定价规则', target: 'H3 图生视频 / 768P', result: '成功' },
  { time: '今天 18:12:36', operator: 'admin', action: '处理工单退款', target: 'TKT-1028', result: '成功' },
  { time: '今天 17:40:09', operator: 'ops', action: '重新调度任务', target: 'tsk_88921', result: '成功' },
  { time: '今天 16:05:17', operator: 'admin', action: '发布工作流', target: 'H3 图生视频', result: '成功' },
]

export const pricingRules = [
  { id: 'pr-1', name: 'H3 图生视频 / 768P', cost: '0.02', price: '0.50', duration: '按秒', status: '生效中', version: 'v1.2' },
  { id: 'pr-2', name: 'H3 图生视频 / 2K', cost: '0.04', price: '0.80', duration: '按秒', status: '生效中', version: 'v1.0' },
  { id: 'pr-3', name: 'H3 再生成 / 768P → 2K', cost: '0.03', price: '0.30', duration: '按秒', status: '草稿', version: 'v0.1' },
]

export const announcements = [
  { id: 'n-1', title: 'H3 图生视频限时价格活动', desc: '活动期间 768P 价格低至 0.02 元/秒，具体以创作页报价为准。', status: '已发布', date: '今天 09:00' },
  { id: 'n-2', title: '服务条款更新说明', desc: '我们更新了视频内容使用规范和素材留存策略。', status: '草稿', date: '昨天 16:30' },
]

export const platformSettings = {
  inputRetention: '7',
  resultRetention: '30',
  notify: true,
  audit: true,
}
