export type TaskStatus = '排队中' | '生成中' | '已完成' | '失败'
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

export type SceneAspectRatio = '16:9' | '9:16' | '1:1'

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

const scenePreviewSamples = [
  'https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4',
  'https://interactive-examples.mdn.mozilla.net/media/cc0-videos/beach.mp4',
  'https://interactive-examples.mdn.mozilla.net/media/cc0-videos/fruits.mp4',
] as const

export const sceneTemplates: SceneTemplate[] = [
  { id: 'scene-1', title: '雨夜霓虹漫步', category: '电影感', description: '潮湿街巷、霓虹反射和稳定跟拍，适合人物氛围片。', color: '1e293b', accent: '818cf8', aspectRatio: '16:9', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1519608487953-e999c86e7455?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[0] },
  { id: 'scene-2', title: '晨雾山谷航拍', category: '自然', description: '从云层穿行到山谷全景，强调空间和纵深。', color: '164e63', accent: '67e8f9', aspectRatio: '16:9', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1500534623283-312aade485b7?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[1] },
  { id: 'scene-3', title: '产品桌面特写', category: '商业', description: '柔和侧光与微距移动，适合新品展示和电商素材。', color: '3f3f46', accent: 'fbbf24', aspectRatio: '1:1', duration: '10 秒', resolution: '1080P', poster: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=720&h=720&q=80', preview: scenePreviewSamples[2] },
  { id: 'scene-4', title: '城市驶过窗边', category: '生活', description: '车窗视角的城市流动感，节奏轻快自然。', color: '334155', accent: 'fda4af', aspectRatio: '9:16', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1449824913935-59a10b8d2000?auto=format&fit=crop&w=540&h=960&q=80', preview: scenePreviewSamples[0] },
  { id: 'scene-5', title: '纸上故事展开', category: '叙事', description: '手绘纸张逐帧展开，适合品牌故事和知识内容。', color: '713f12', accent: 'fde68a', aspectRatio: '16:9', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1456513080510-7bf3a84b82f8?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[1] },
  { id: 'scene-6', title: '静物光影实验', category: '艺术', description: '光影在静物表面缓慢移动，突出材质和细节。', color: '312e81', accent: 'c4b5fd', aspectRatio: '1:1', duration: '10 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=720&h=720&q=80', preview: scenePreviewSamples[2] },
  { id: 'scene-7', title: '赛博街区穿行', category: '电影感', description: '低角度跟拍与霓虹灯带，强调未来都市质感。', color: '0f172a', accent: '22d3ee', aspectRatio: '9:16', duration: '15 秒', resolution: '1080P', poster: 'https://images.unsplash.com/photo-1514525253161-7a46d19cd819?auto=format&fit=crop&w=540&h=960&q=80', preview: scenePreviewSamples[0] },
  { id: 'scene-8', title: '海岸日落延时', category: '自然', description: '长镜头海面与天际线渐变，适合治愈系内容。', color: '0c4a6e', accent: 'fdba74', aspectRatio: '16:9', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[1] },
  { id: 'scene-9', title: '美妆质地微距', category: '商业', description: '膏体拉丝与高光反射，突出产品质感。', color: '44403c', accent: 'f9a8d4', aspectRatio: '9:16', duration: '10 秒', resolution: '1080P', poster: 'https://images.unsplash.com/photo-1596462502278-27bfdc403348?auto=format&fit=crop&w=540&h=960&q=80', preview: scenePreviewSamples[2] },
  { id: 'scene-10', title: '咖啡馆窗边', category: '生活', description: '人物侧脸与窗外街景虚化，日常感叙事。', color: '57534e', accent: 'fcd34d', aspectRatio: '16:9', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1445116572660-236099ec97a0?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[0] },
  { id: 'scene-11', title: '品牌宣言快切', category: '叙事', description: '大字标题与节奏剪辑，适合活动开场。', color: '1e1b4b', accent: 'a78bfa', aspectRatio: '9:16', duration: '10 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1557804506-669a67965ba0?auto=format&fit=crop&w=540&h=960&q=80', preview: scenePreviewSamples[1] },
  { id: 'scene-12', title: '流体色彩实验', category: '艺术', description: '颜料在水中扩散，抽象视觉与慢节奏。', color: '4c1d95', accent: 'e879f9', aspectRatio: '16:9', duration: '10 秒', resolution: '1080P', poster: 'https://images.unsplash.com/photo-1541701494587-46f7f5a4df22?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[2] },
  { id: 'scene-13', title: '复古胶片人像', category: '电影感', description: '颗粒感与暖色偏移，适合情绪短片。', color: '292524', accent: 'fb923c', aspectRatio: '9:16', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1529626455594-4ff0802cfb7e?auto=format&fit=crop&w=540&h=960&q=80', preview: scenePreviewSamples[0] },
  { id: 'scene-14', title: '森林小径漫步', category: '自然', description: '第一人称穿行林间，光影斑驳。', color: '14532d', accent: '86efac', aspectRatio: '16:9', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1441974231531-c6227db76b6e?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[1] },
  { id: 'scene-15', title: '手表细节旋转', category: '商业', description: '金属表圈反光与慢速旋转展示。', color: '27272a', accent: 'fde047', aspectRatio: '1:1', duration: '10 秒', resolution: '1080P', poster: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=720&h=720&q=80', preview: scenePreviewSamples[2] },
  { id: 'scene-16', title: '地铁通勤一瞥', category: '生活', description: '车厢内人物与窗外掠影，城市节奏。', color: '1f2937', accent: '93c5fd', aspectRatio: '9:16', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1474487548417-781cb71495f3?auto=format&fit=crop&w=540&h=960&q=80', preview: scenePreviewSamples[0] },
  { id: 'scene-17', title: '历史档案翻阅', category: '叙事', description: '旧照片与文字叠化，纪录片式开场。', color: '422006', accent: 'fbbf24', aspectRatio: '16:9', duration: '15 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1457369804613-52ec61db2eba?auto=format&fit=crop&w=960&h=540&q=80', preview: scenePreviewSamples[1] },
  { id: 'scene-18', title: '几何构图实验', category: '艺术', description: '建筑线条与对称构图，极简视觉。', color: '18181b', accent: '38bdf8', aspectRatio: '1:1', duration: '10 秒', resolution: '768P', poster: 'https://images.unsplash.com/photo-1487958449943-2429d8be8625?auto=format&fit=crop&w=720&h=720&q=80', preview: scenePreviewSamples[2] },
]

export const initialTasks: Task[] = [
  { id: 'tsk_88921', title: '雨夜街道霓虹漫步', workflow: 'H3 图生视频', status: '生成中', progress: 68, duration: '15 秒', resolution: '768P', cost: 4.5, time: '今天 14:20', thumb: 'https://images.unsplash.com/photo-1519608487953-e999c86e7455?auto=format&fit=crop&w=480&q=80' },
  { id: 'tsk_88918', title: '产品桌面柔光展示', workflow: 'H3 图生视频', status: '已完成', progress: 100, duration: '15 秒', resolution: '768P', cost: 4.5, time: '今天 13:42', thumb: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=480&q=80' },
  { id: 'tsk_88840', title: '晨雾山谷全景', workflow: 'H3 图生视频', status: '失败', progress: 0, duration: '15 秒', resolution: '768P', cost: 4.5, time: '昨天 18:09', thumb: 'https://images.unsplash.com/photo-1500534623283-312aade485b7?auto=format&fit=crop&w=480&q=80' },
  { id: 'tsk_88812', title: '咖啡店窗边人物', workflow: 'H3 图生视频', status: '已完成', progress: 100, duration: '15 秒', resolution: '768P', cost: 4.5, time: '昨天 15:31', thumb: 'https://images.unsplash.com/photo-1445116572660-236099ec97a0?auto=format&fit=crop&w=480&q=80' },
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
