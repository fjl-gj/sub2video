import type { Page } from '@playwright/test'

// Populated, deterministic screenshot fixtures only. No response below is
// evidence of a supplier call, a real payment, or an accepted delivery.
const mediaRoot = 'https://surface-fixtures.sub2video.test'
const thumb = (name: string) => `${mediaRoot}/${name}.svg`

const tasks = [
  { id: 'tsk_visual_coast', title: '海岸清晨 · 镜头 03', mode: '多图生成视频', modeId: 'mode_multi_image', status: '已完成', progress: 100, duration: '10 秒', durationSec: 10, resolution: '768P', cost: 5, time: '09-25 11:42', thumbUrl: thumb('coast'), savedToProject: true },
  { id: 'tsk_visual_product', title: '香水产品 · 柔光特写', mode: '首尾帧生成视频', modeId: 'mode_first_last', status: '生成中', progress: 65, duration: '5 秒', durationSec: 5, resolution: '768P', cost: 2.5, time: '09-25 11:50', thumbUrl: thumb('product'), savedToProject: false },
  { id: 'tsk_visual_mountain', title: '山间公路 · 清晨航拍', mode: '文生视频', modeId: 'mode_text_to_video', status: '排队中', progress: 0, duration: '15 秒', durationSec: 15, resolution: '768P', cost: 7.5, time: '09-25 11:55', thumbUrl: thumb('mountain'), savedToProject: false },
  { id: 'tsk_visual_city', title: '城市漫游 · 雨夜街角', mode: '文生视频', modeId: 'mode_text_to_video', status: '已完成', progress: 100, duration: '10 秒', durationSec: 10, resolution: '768P', cost: 5, time: '09-24 19:20', thumbUrl: thumb('city'), savedToProject: true },
  { id: 'tsk_visual_failed', title: '静物光影 · 镜头 02', mode: '多图生成视频', modeId: 'mode_multi_image', status: '失败', progress: 0, duration: '5 秒', durationSec: 5, resolution: '768P', cost: 2.5, time: '09-24 18:10', thumbUrl: thumb('product'), savedToProject: false },
]

const projects = [
  { id: 'prj_visual_coast', name: '海岸清晨', desc: '海边生活方式短片 · 人物、场景与成片', createdAt: '2026-09-20', trashed: false, isDefault: true, fileCount: 6 },
  { id: 'prj_visual_product', name: '秋日香氛', desc: '产品展示短片 · 包装、质地与细节', createdAt: '2026-09-22', trashed: false, isDefault: false, fileCount: 4 },
  { id: 'prj_visual_city', name: '城市漫游', desc: '街头氛围镜头 · 霓虹与雨夜', createdAt: '2026-09-23', trashed: false, isDefault: false, fileCount: 3 },
]
const trash = [{ id: 'prj_visual_draft', name: '初版镜头草稿', desc: '早期素材与未采用的镜头', createdAt: '2026-09-15', trashed: true, trashedAt: '2026-09-24', fileCount: 2 }]
const tickets = [
  { id: 'tic_visual_1024', title: '已完成任务的素材保存方式', category: '售后支持', status: '处理中', task: 'tsk_visual_coast', updated: '2026-09-25 10:30' },
  { id: 'tic_visual_1023', title: '失败任务的费用已经退回', category: '生成异常', status: '已解决', task: 'tsk_visual_failed', updated: '2026-09-24 18:20' },
]

function paged<T>(items: T[], params: URLSearchParams) {
  const page = Math.max(1, Number(params.get('page') || 1))
  const pageSize = Math.max(1, Number(params.get('pageSize') || 20))
  return { items: items.slice((page - 1) * pageSize, page * pageSize), total: items.length, page, pageSize }
}

function imageFixture(name: string) {
  const palettes: Record<string, string[]> = {
    coast: ['#d9ddd3', '#8ea9a5', '#526f72', '#f3ead4'],
    mountain: ['#d6ded8', '#9bac9a', '#536756', '#e9e6d7'],
    product: ['#e7ded3', '#c9b99f', '#796d5b', '#f5eee5'],
    city: ['#ccd3d6', '#82949c', '#3c505e', '#e3b892'],
  }
  const [sky, distant, front, light] = palettes[name] || palettes.coast
  // A test illustration, intentionally not a production thumbnail or an AI
  // generation sample. Defined here to avoid variable external image content.
  return `<svg xmlns="http://www.w3.org/2000/svg" width="960" height="540" viewBox="0 0 960 540"><rect width="960" height="540" fill="${sky}"/><circle cx="730" cy="115" r="52" fill="${light}"/><path d="M0 285L180 160L320 285L490 180L760 310L960 210V540H0Z" fill="${distant}"/><path d="M0 380L240 280L440 390L640 285L960 390V540H0Z" fill="${front}"/><path d="M430 540Q440 430 555 375Q640 335 700 350" fill="none" stroke="${light}" stroke-width="18" opacity=".8"/></svg>`
}

export async function seedSurfaceData(page: Page) {
  const unmatchedRequests: string[] = []
  await page.clock.setFixedTime(new Date('2026-09-25T04:00:00Z'))
  await page.route(`${mediaRoot}/**`, async (route) => {
    const name = new URL(route.request().url()).pathname.split('/').pop()?.replace('.svg', '') || 'coast'
    await route.fulfill({ contentType: 'image/svg+xml', body: imageFixture(name) })
  })
  await page.route('https://images.unsplash.com/**', async (route) => {
    await route.fulfill({ contentType: 'image/svg+xml', body: imageFixture('coast') })
  })
  // Plaza screenshots cover static posters; external demo video downloads are
  // explicitly blocked. Playback is covered by separate interaction contracts.
  await page.route('https://interactive-examples.mdn.mozilla.net/**', async (route) => { await route.abort('blockedbyclient') })
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname.replace('/api/v1', '')
    let response: unknown
    if (path === '/client/tasks/estimate' && request.method() === 'POST') {
      const payload = request.postDataJSON() as { durationSec: number; resolution: string; generateAudio: boolean }
      response = {
        available: true,
        selection: { carrier: { id: 'car_minimax', name: 'MINMAX' }, model: { id: 'mdl_h3', name: 'H3' } },
        billing: { ...payload, basis: 'per_second', currency: 'CNY', pricePerSecond: 0.5, estimatedTotal: payload.durationSec * 0.5 },
      }
    } else if (request.method() === 'GET') {
      switch (path) {
        case '/settings/public':
          response = { emailVerifyEnabled: true, passwordResetEnabled: true, registrationEnabled: true }
          break
        case '/client/overview':
          response = {
            workspace: { displayName: '李维工作室' }, wallet: { balance: 256 },
            billing: { totalSpent: 44, monthSpent: 16 }, pricing: { available: true, referencePricePerSecond: 0.5, referenceNote: '按请求时长计费' },
            storage: { usedGb: 1.2, quotaGb: 5, fileCount: 13 },
            tasks: { activeCount: 2, activeTask: tasks[1], recentCompleted: tasks.filter((task) => task.status === '已完成') },
          }
          break
        case '/client/wallet':
          response = { balance: 256, products: [{ id: 'product_visual_50', amount: 50, bonus: 0, price: 50 }, { id: 'product_visual_100', amount: 100, bonus: 5, price: 100 }, { id: 'product_visual_200', amount: 200, bonus: 15, price: 200 }, { id: 'product_visual_500', amount: 500, bonus: 50, price: 500 }] }
          break
        case '/client/wallet/transactions':
          response = paged([
            { id: 'ledger_visual_3', type: '退款', amount: 2.5, desc: '静物光影 · 镜头 02 失败退款', time: '2026-09-24 18:12' },
            { id: 'ledger_visual_2', type: '消费', amount: -5, desc: '海岸清晨 · 镜头 03', time: '2026-09-24 17:30' },
            { id: 'ledger_visual_1', type: '充值', amount: 100, desc: '测试账本记录', time: '2026-09-20 09:00' },
          ], url.searchParams)
          break
        case '/client/storage/usage':
          response = { usedGb: 1.2, quotaGb: 5, fileCount: 13, projectCount: 3, trashCount: 1 }
          break
        case '/client/projects':
          response = { items: url.searchParams.get('trashed') === 'true' ? trash : projects, currentProjectId: projects[0].id }
          break
        case '/client/tasks': {
          const status = url.searchParams.get('status')
          response = { ...paged(tasks.filter((task) => !status || task.status === status), url.searchParams), counts: { 全部: 5, 排队中: 1, 生成中: 1, 已完成: 2, 失败: 1 } }
          break
        }
        case '/client/tickets':
          response = paged(tickets, url.searchParams)
          break
        case '/admin/tickets':
          response = paged(tickets.map((ticket) => ({ ...ticket, user: '林默', userEmail: 'creator@example.test' })), url.searchParams)
          break
        case '/admin/tasks': {
          const status = url.searchParams.get('status')
          response = paged(tasks.filter((task) => !status || task.status === status).map((task) => ({
            ...task, user: '林默', userEmail: 'creator@example.test', provider: 'AutoDL',
            upstreamTaskId: `upstream_${task.id}`, upstreamStatus: task.status === '已完成' ? 'SUCCESS' : task.status === '失败' ? 'FAILED' : 'RUNNING',
            failReason: task.status === '失败' ? '生成超时，已退款' : '',
          })), url.searchParams)
          break
        }
      }
      if (path.startsWith('/client/tasks/')) {
        const task = tasks.find((item) => path === `/client/tasks/${item.id}`)
        if (task) response = {
          ...task, durationSeconds: task.durationSec, aspectRatio: '16:9', generateAudio: false,
          failureReason: task.status === '失败' ? '生成超时，已退款' : '', executionWaiting: false,
          // The fixture has a poster, not a playable video. Do not fabricate a
          // successful media download to make the preview look delivered.
          deliveryStatus: task.status === '已完成' ? 'pending_storage' : 'not_ready', downloadUrl: '',
          createdAt: '2026-09-25T03:30:00Z', updatedAt: '2026-09-25T03:42:00Z',
        }
      }
      const projectMatch = path.match(/^\/client\/projects\/([^/]+)\/(tree|assets)$/)
      if (projectMatch) {
        const project = [...projects, ...trash].find((item) => item.id === projectMatch[1])
        const items = [{ id: 'asset_visual_coast', name: '海岸参考图.svg', type: 'image', size: '24 KB', time: '2026-09-24', url: thumb('coast') }]
        if (project) response = projectMatch[2] === 'assets' ? { items, projectId: project.id } : {
          ...project, folders: {
            资产: { isSystem: true, desc: '参考素材', items: [], folders: { 场景: { isSystem: true, desc: '场景参考', items, folders: {} } } },
            视频: { isSystem: true, desc: '生成成片', items: [], folders: {} },
          },
        }
      }
    }
    if (response === undefined) {
      unmatchedRequests.push(`${request.method()} ${path}`)
      await route.fulfill({ status: 501, json: { message: 'Missing deterministic visual fixture' } })
      return
    }
    await route.fulfill({ json: response })
  })
  return { unmatchedRequests }
}
