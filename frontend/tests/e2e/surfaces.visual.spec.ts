import { expect, test, type Page } from '@playwright/test'

const cases = [
  { name: 'client-overview', path: '/app/overview', heading: '欢迎回来，李维工作室', auth: true },
  { name: 'client-create', path: '/app/create', heading: '创作', auth: true },
  { name: 'client-garden', path: '/app/garden', heading: '广场', auth: true },
  { name: 'client-works', path: '/app/works', heading: '任务', auth: true },
  { name: 'client-assets', path: '/app/assets', heading: '资产', auth: true },
  { name: 'client-trash', path: '/app/trash', heading: '回收站', auth: true },
  { name: 'client-wallet', path: '/app/wallet', heading: '钱包', auth: true },
  { name: 'client-tickets', path: '/app/tickets', heading: '帮助与工单', auth: true },
  { name: 'client-profile', path: '/app/profile', heading: '个人资料', auth: true },
  { name: 'admin-overview', path: '/admin/overview', heading: '运行大盘', auth: true, role: 'admin' },
  { name: 'admin-workflows', path: '/admin/workflows', heading: '工作流', auth: true, role: 'admin' },
  { name: 'admin-carriers-models', path: '/admin/carriers-models', heading: '承运商与模型', auth: true, role: 'admin' },
  { name: 'admin-supplier-models', path: '/admin/supplier-models', heading: '供应商与模型', auth: true, role: 'admin' },
  { name: 'admin-pricing', path: '/admin/pricing', heading: '价格审计', auth: true, role: 'admin' },
  { name: 'admin-tasks', path: '/admin/tasks', heading: '任务中心', auth: true, role: 'admin' },
  { name: 'admin-tickets', path: '/admin/tickets', heading: '工单中心', auth: true, role: 'admin' },
]

async function seedClientAuth(page: Page, role = 'user') {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'u_visual',
        email: 'demo@sub2video.local',
        displayName: '林默',
        role,
        workspaceName: '李维工作室',
        balance: 256,
      }),
    })
  })
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'visual-test-access-token')
    localStorage.setItem('s2v_refresh_token', 'visual-test-refresh-token')
  })
}

async function seedCreationAndSupply(page: Page) {
  await page.route('**/api/v1/client/create/catalog', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
      modes: [
        { id: 'mode_text_to_video', inputType: 'text', name: '文生视频', description: '通过文字描述生成视频', enabled: true },
        { id: 'mode_first_last', inputType: 'first_last', name: '首尾帧生成视频', description: '使用首帧、尾帧和提示词生成视频', enabled: true },
        { id: 'mode_multi_image', inputType: 'multi_image', name: '多图生成视频', description: '使用多张参考图和提示词生成视频', enabled: true },
      ],
      carriers: [{ id: 'car_minimax', name: 'MINMAX' }],
      models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', description: 'MiniMax H3 视频模型', inputTypes: ['text', 'first_last', 'multi_image'] }],
    }) })
  })
  await page.route('**/api/v1/client/create/options*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
      durations: [5, 10, 15], resolutions: ['768P'], aspectRatios: ['16:9', '9:16', '1:1'],
      supportsGeneratedAudio: false, minImages: 0, maxImages: 9, defaultDuration: 5, defaultResolution: '768P',
    }) })
  })
  await page.route('**/api/v1/admin/creation-modes', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [
      { id: 'mode_text_to_video', code: 'text_to_video', name: '文生视频', description: '通过文字描述生成视频', status: 'published', sortOrder: 1 },
      { id: 'mode_first_last', code: 'first_last_frame_to_video', name: '首尾帧生成视频', description: '首帧、尾帧和提示词', status: 'published', sortOrder: 2 },
      { id: 'mode_multi_image', code: 'multi_image_to_video', name: '多图生成视频', description: '多张参考图和提示词', status: 'published', sortOrder: 3 },
    ] }) })
  })
  await page.route('**/api/v1/admin/carriers', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ carriers: [{ id: 'car_minimax', code: 'minimax', name: 'MINMAX', enabled: true }], models: [{ id: 'mdl_h3', carrierId: 'car_minimax', code: 'h3', name: 'H3', carrierCode: 'minimax', enabled: true }], cosConfigured: false, autodlEnvToken: false }) })
  })
  await page.route('**/api/v1/admin/model-configurations', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [
      { modelId: 'mdl_h3', modelCode: 'h3', modelName: 'H3', carrierId: 'car_minimax', carrierName: 'MINMAX', workflowId: 'mode_text_to_video', workflowName: '文生视频', enabled: true, recommendationRank: 1, displayBadge: '', defaultDuration: 5, defaultResolution: '768P' },
    ] }) })
  })
  await page.route('**/api/v1/admin/suppliers', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'sup_autodl', code: 'autodl', name: 'AutoDL', baseUrl: 'https://autodl.art', enabled: true, hasToken: true, status: '就绪' }] }) })
  })
  await page.route('**/api/v1/admin/offerings', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'off_h3_text_b99', supplierId: 'sup_autodl', supplierName: 'AutoDL', modelId: 'mdl_h3', modelName: 'H3', carrierName: 'MINMAX', modeId: 'mode_text_to_video', modeName: '文生视频', name: 'H3 文生视频', upstreamWorkflowId: 'minimax_h3_b99_001', adapterCode: 'autodl_h3_v1', enabled: true, healthStatus: 'healthy', priority: 10, minDurationSec: 1, maxDurationSec: 15, resolutions: ['768P'], aspectRatios: ['16:9', '9:16', '1:1'], minImages: 0, maxImages: 0, supportsGeneratedAudio: false }] }) })
  })
  await page.route('**/api/v1/admin/supplier-costs', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'cost_h3_text', offeringId: 'off_h3_text_b99', resolution: '768P', generateAudio: false, peakPerSecond: 0.03, offpeakPerSecond: 0.02, currency: 'CNY', version: 'v1', status: 'active' }] }) })
  })
  await page.route('**/api/v1/admin/retail-pricing', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'ret_h3_text', modeId: 'mode_text_to_video', modeName: '文生视频', modelId: 'mdl_h3', modelName: 'H3', carrierName: 'MINMAX', resolution: '768P', generateAudio: false, name: 'H3 文生 / 768P', pricePerSecond: 0.5, version: 'v1', status: 'active' }] }) })
  })
}

for (const item of cases) {
  test(`${item.name} matches the surface baseline`, async ({ page }) => {
    if (item.auth) await seedClientAuth(page, item.role)
    await seedCreationAndSupply(page)
    await page.goto(item.path, { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: item.heading, exact: true })).toBeVisible({ timeout: 15_000 })
    if (item.name === 'admin-carriers-models') await expect(page.getByText('H3', { exact: true })).toBeVisible()
    await expect(page).toHaveScreenshot(`${item.name}.png`, {
      fullPage: true,
      animations: 'disabled',
      caret: 'hide',
      scale: 'css',
      maxDiffPixelRatio: 0.02,
    })
  })
}
