import { expect, test, type Page, type Route } from '@playwright/test'

// Browser interaction evidence only. Wallet, SQL and provider acceptance need
// the real-environment integration suite; these fixtures never reach UAT.
const supplier = { id: 'sup_1', code: 'supplier_1', name: '供应商 1', enabled: true, hasToken: true }
const execution = {
  id: 'exec_1', executionId: 'exec_1', supplierId: supplier.id, supplierName: supplier.name,
  modelId: 'mdl_a', modelName: '模型 A', carrierName: '承运商 A', modeId: 'mode_text_to_video', modeName: '文生视频',
  name: '分维度成本配置', executionType: 'workflow', workflowCode: 'workflow_a', upstreamWorkflowId: 'workflow_a',
  adapterCode: 'autodl_h3_v1', enabled: true, healthStatus: 'healthy', quotaStatus: 'unknown', priority: 1,
  minDurationSec: 1, maxDurationSec: 10, resolutions: ['480P', '768P'], aspectRatios: ['16:9'], minImages: 0, maxImages: 0, supportsGeneratedAudio: false,
}
const costs = [
  { id: 'cost_480', executionId: execution.id, resolution: '480P', generateAudio: false, peakPerSecond: 0.02, offpeakPerSecond: 0.01, currency: 'CNY', version: 'v1', status: 'active' },
  { id: 'cost_768', executionId: execution.id, resolution: '768P', generateAudio: false, peakPerSecond: 0.03, offpeakPerSecond: 0.02, currency: 'CNY', version: 'v1', status: 'active' },
]
const json = (route: Route, body: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })

async function adminSession(page: Page) {
  await page.addInitScript(() => localStorage.setItem('s2v_access_token', 'admin-experience-fixture'))
  await page.route('**/api/v1/auth/me', (route) => json(route, { id: 'admin', email: 'admin@test', displayName: '管理员', role: 'admin', balance: 0 }))
}

async function supplierFixture(page: Page, options: { failCosts?: boolean; suppliers?: number; health?: string; relations?: number } = {}) {
  await adminSession(page)
  const writes: Record<string, unknown>[] = []
  const edits: Record<string, unknown>[] = []
  const costPages: number[] = []
  const state = { failCosts: options.failCosts ?? false }
  await page.route('**/api/v1/admin/suppliers**', (route) => {
    const current = Number(new URL(route.request().url()).searchParams.get('page') || 1)
    const all = Array.from({ length: options.suppliers ?? 1 }, (_, index) => ({ ...supplier, id: `sup_${index + 1}`, code: `supplier_${index + 1}`, name: `供应商 ${index + 1}` }))
    return json(route, { items: all.slice((current - 1) * 10, current * 10), total: all.length, page: current, pageSize: 10 })
  })
  await page.route('**/api/v1/admin/carriers**', (route) => json(route, { carriers: [{ id: 'car_a', code: 'carrier_a', name: '承运商 A', enabled: true }], models: [{ id: 'mdl_a', carrierId: 'car_a', code: 'model_a', name: '模型 A', enabled: true }] }))
  await page.route('**/api/v1/admin/model-configurations**', (route) => json(route, { items: [{ modelId: 'mdl_a', modelName: '模型 A', carrierId: 'car_a', carrierName: '承运商 A', workflowId: execution.modeId, workflowName: '文生视频', enabled: true }] }))
  await page.route('**/api/v1/admin/offerings**', (route) => {
    if (route.request().method() === 'PATCH') {
      const payload = route.request().postDataJSON()
      edits.push(payload)
      return json(route, { ...execution, ...payload })
    }
    const url = new URL(route.request().url())
    const all = url.searchParams.get('executionType') === 'api' || (url.searchParams.get('supplierId') && url.searchParams.get('supplierId') !== supplier.id) ? [] : Array.from({ length: options.relations ?? 1 }, (_, index) => ({ ...execution, id: `exec_${index + 1}`, executionId: `exec_${index + 1}`, healthStatus: options.health ?? 'healthy' }))
    const current = Number(url.searchParams.get('page') || 1)
    return json(route, { items: all.slice((current - 1) * 10, current * 10), total: all.length, page: current, pageSize: 10 })
  })
  await page.route('**/api/v1/admin/supplier-costs**', (route) => {
    if (route.request().method() === 'PUT') {
      writes.push(route.request().postDataJSON())
      return json(route, { id: 'saved', ...route.request().postDataJSON(), version: 'v2' })
    }
    if (state.failCosts) return json(route, { message: '采购成本服务暂不可用' }, 503)
    const current = Number(new URL(route.request().url()).searchParams.get('page') || 1)
    costPages.push(current)
    // Simulate a server page-size cap. The editor must load both dimensions.
    return json(route, { items: costs.slice(current - 1, current), total: costs.length, page: current, pageSize: 1 })
  })
  await page.goto('/admin/supplier-models')
  await expect(page.getByRole('heading', { name: '供应商 1', exact: true })).toBeVisible()
  return { writes, edits, state, costPages }
}

test('editing execution metadata preserves different cost dimensions', async ({ page }) => {
  const fixture = await supplierFixture(page)
  await page.getByRole('button', { name: '编辑执行配置', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByLabel('480P / 无自动音频 峰时成本')).toHaveValue('0.02')
  await expect(dialog.getByLabel('768P / 无自动音频 峰时成本')).toHaveValue('0.03')
  expect(fixture.costPages).toContain(2)
  await dialog.getByLabel('执行配置名称', { exact: true }).fill('仅修改名称')
  await dialog.getByRole('button', { name: '保存执行配置', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  expect(fixture.edits).toHaveLength(1)
  expect(fixture.writes).toEqual([])
})

test('editing one cost saves only that resolution and audio dimension', async ({ page }) => {
  const fixture = await supplierFixture(page)
  await page.getByRole('button', { name: '编辑执行配置', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('768P / 无自动音频 峰时成本').fill('0.04')
  await dialog.getByRole('button', { name: '保存执行配置', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  expect(fixture.writes).toEqual([{ executionId: 'exec_1', resolution: '768P', generateAudio: false, peakPerSecond: 0.04, offpeakPerSecond: 0.02, currency: 'CNY', status: 'active' }])
})

test('cost loading failure blocks save and retry restores original amounts', async ({ page }) => {
  const fixture = await supplierFixture(page, { failCosts: true })
  await page.getByRole('button', { name: '编辑执行配置', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByRole('alert')).toContainText('采购成本服务暂不可用')
  await expect(dialog.getByRole('button', { name: '保存执行配置', exact: true })).toBeDisabled()
  expect(fixture.writes).toEqual([])
  fixture.state.failCosts = false
  await dialog.getByRole('button', { name: '重试加载配置' }).click()
  await expect(dialog.getByLabel('768P / 无自动音频 峰时成本')).toHaveValue('0.03')
  await dialog.getByLabel('支持模型自动生成音频').check()
  await expect(dialog.getByLabel('768P / 自动音频 峰时成本')).toHaveValue('')
  await dialog.getByRole('button', { name: '保存执行配置', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  expect(fixture.writes).toEqual([])
})

test('new executions can select suppliers beyond the first page and degraded is truthful', async ({ page }) => {
  await supplierFixture(page, { suppliers: 12, health: 'degraded' })
  await expect(page.getByText('健康降级', { exact: true })).toBeVisible()
  await expect(page.getByText('可路由', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: 'Workflow 1', exact: true }).click()
  await page.getByRole('button', { name: '新增Workflow配置', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByLabel('供应商', { exact: true })).toBeEnabled()
  await dialog.getByLabel('供应商', { exact: true }).selectOption('sup_12')
  await expect(dialog.getByLabel('供应商', { exact: true })).toHaveValue('sup_12')
})

test('supplier relationship pagination retrieves later executions from the server', async ({ page }) => {
  await supplierFixture(page, { relations: 11 })
  await expect(page.getByText('exec_11', { exact: true })).toHaveCount(0)
  const requested = page.waitForRequest((request) => {
    const url = new URL(request.url())
    return url.pathname.endsWith('/admin/offerings') && url.searchParams.get('supplierId') === supplier.id && url.searchParams.get('page') === '2'
  })
  await page.getByRole('button', { name: '下一页', exact: true }).filter({ hasNot: page.locator('[disabled]') }).first().click()
  await requested
  await expect(page.getByText('exec_11', { exact: true })).toBeVisible()
  await expect(page.getByText('exec_1', { exact: true })).toHaveCount(0)
})

test('partial cost failure stays open and retries only unsaved dimensions', async ({ page }) => {
  await supplierFixture(page)
  const attempts: string[] = []
  let failSecond = true
  await page.route('**/api/v1/admin/supplier-costs**', async (route) => {
    if (route.request().method() !== 'PUT') return route.fallback()
    const payload = route.request().postDataJSON()
    attempts.push(payload.resolution)
    if (payload.resolution === '768P' && failSecond) return json(route, { message: '该维度成本暂未保存' }, 503)
    return json(route, { id: 'saved', ...payload, version: 'v2' })
  })
  await page.getByRole('button', { name: '编辑执行配置', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('480P / 无自动音频 峰时成本').fill('0.04')
  await dialog.getByLabel('768P / 无自动音频 峰时成本').fill('0.05')
  await dialog.getByRole('button', { name: '保存执行配置', exact: true }).click()
  await expect(dialog.getByRole('alert')).toContainText('该维度成本暂未保存')
  failSecond = false
  await dialog.getByRole('button', { name: '保存执行配置', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  expect(attempts).toEqual(['480P', '768P', '768P'])
})

test('retail edit locks identity and rejects an empty amount', async ({ page }) => {
  await adminSession(page)
  const writes: unknown[] = []
  const rule = { id: 'price_a', modeId: execution.modeId, modeName: '文生视频', modelId: '', modelName: '', carrierName: '', resolution: '768P', generateAudio: false, name: '常规售价', pricePerSecond: 0.05, version: 'v1', status: 'active' }
  await page.route('**/api/v1/admin/creation-modes', (route) => json(route, { items: [{ id: execution.modeId, name: '文生视频', status: 'published' }] }))
  await page.route('**/api/v1/admin/retail-pricing**', (route) => {
    if (route.request().method() === 'PUT') { writes.push(route.request().postDataJSON()); return json(route, rule) }
    return json(route, { items: [rule], total: 1, page: 1, pageSize: 10 })
  })
  await page.goto('/admin/pricing')
  await page.getByRole('button', { name: '编辑售价', exact: true }).filter({ visible: true }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByLabel('创作模式', { exact: true })).toBeDisabled()
  await expect(dialog.getByLabel('分辨率', { exact: true })).toBeDisabled()
  await expect(dialog.getByLabel('包含自动生成音频')).toBeDisabled()
  await dialog.getByLabel('售价（元/秒）').fill('')
  await dialog.getByRole('button', { name: '保存售价' }).click()
  await expect(dialog.getByRole('alert')).toContainText('有效售价')
  expect(writes).toEqual([])
  await dialog.getByLabel('售价（元/秒）').fill('0.06')
  await dialog.getByRole('button', { name: '保存售价' }).click()
  await expect(dialog).not.toBeVisible()
  expect(writes).toEqual([{ modeId: execution.modeId, resolution: '768P', generateAudio: false, name: '常规售价', pricePerSecond: 0.06, status: 'active' }])
})

test('tasks paginate and search on the server; accepted tasks cannot be ordinarily cancelled', async ({ page }) => {
  await adminSession(page)
  const requests: URL[] = []
  await page.route('**/api/v1/admin/tasks**', (route) => {
    const url = new URL(route.request().url()); requests.push(url)
    const current = Number(url.searchParams.get('page') || 1)
    const keyword = url.searchParams.get('keyword')
    const id = keyword ? 'task_found' : `task_page_${current}`
    return json(route, { items: [{ id, title: keyword ? '已搜索的任务' : `第 ${current} 页任务`, user: '用户', userEmail: 'creator@test', provider: '内部供应商', status: '生成中', cost: 0.5, time: '09-29 10:00', upstreamTaskId: 'upstream_1', upstreamStatus: 'RUNNING', mode: '文生视频' }], total: keyword ? 1 : 21, page: current, pageSize: 20 })
  })
  await page.goto('/admin/tasks')
  await page.getByRole('button', { name: '下一页', exact: true }).click()
  await expect(page.getByText('第 2 页任务', { exact: true }).filter({ visible: true })).toBeVisible()
  await page.getByRole('textbox', { name: '搜索任务' }).fill('creator@test')
  await expect(page.getByText('已搜索的任务', { exact: true }).filter({ visible: true })).toBeVisible()
  expect(requests.some((url) => url.searchParams.get('page') === '2' && url.searchParams.get('pageSize') === '20')).toBeTruthy()
  expect(requests.at(-1)?.searchParams.get('keyword')).toBe('creator@test')
  expect(requests.at(-1)?.searchParams.get('page')).toBe('1')
  await expect(page.getByRole('button', { name: '终止退款' })).toHaveCount(0)
})

test('stale task search cannot replace newer results and load errors remain actionable', async ({ page }) => {
  await adminSession(page)
  let releaseOld: (() => Promise<void>) | undefined
  await page.route('**/api/v1/admin/tasks**', async (route) => {
    const keyword = new URL(route.request().url()).searchParams.get('keyword') || 'initial'
    const result = { items: [{ id: `task_${keyword}`, title: `搜索结果 ${keyword}`, user: '用户', userEmail: 'creator@test', provider: '内部供应商', status: '已完成', cost: 0.5, time: '09-29 10:00', upstreamTaskId: 'upstream_1', upstreamStatus: 'SUCCESS', mode: '文生视频' }], total: 1, page: 1, pageSize: 20 }
    if (keyword === 'old') { releaseOld = () => json(route, result); return }
    if (keyword === 'error') return json(route, { message: '任务服务暂不可用' }, 503)
    return json(route, result)
  })
  await page.goto('/admin/tasks')
  const search = page.getByRole('textbox', { name: '搜索任务' })
  await search.fill('old')
  await expect.poll(() => Boolean(releaseOld)).toBeTruthy()
  await search.fill('new')
  await expect(page.getByText('搜索结果 new', { exact: true }).filter({ visible: true })).toBeVisible()
  const oldResponse = page.waitForResponse((response) => new URL(response.url()).searchParams.get('keyword') === 'old')
  await releaseOld!()
  await oldResponse
  await expect(page.getByText('搜索结果 new', { exact: true }).filter({ visible: true })).toBeVisible()
  await search.fill('error')
  await expect(page.getByRole('alert')).toContainText('任务服务暂不可用')
  await expect(page.getByRole('button', { name: '重试加载' })).toBeVisible()
  await expect(page.getByText('暂无任务。', { exact: true })).toHaveCount(0)
})

test('rejected cancellation keeps its confirmation open with the server reason', async ({ page }) => {
  await adminSession(page)
  await page.route('**/api/v1/admin/tasks**', (route) => {
    if (route.request().method() === 'POST') return json(route, { message: '任务已被上游受理，请等待执行结果' }, 409)
    return json(route, { items: [{ id: 'task_queue', title: '排队任务', user: '用户', userEmail: 'creator@test', provider: '内部供应商', status: '排队中', cost: 0.5, time: '09-29 10:00', upstreamTaskId: '', upstreamStatus: '', mode: '文生视频' }], total: 1, page: 1, pageSize: 20 })
  })
  await page.goto('/admin/tasks')
  await page.getByRole('button', { name: '终止退款', exact: true }).filter({ visible: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name: '终止并退款' }).click()
  await expect(dialog.getByRole('alert')).toContainText('任务已被上游受理')
  await expect(dialog.getByRole('button', { name: '终止并退款' })).toBeEnabled()
})
