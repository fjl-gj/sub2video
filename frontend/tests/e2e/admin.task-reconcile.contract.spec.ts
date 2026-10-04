import { expect, test, type Page } from '@playwright/test'

// These mocked browser tests verify the operator confirmation flow only.
// Real provider reconciliation, audit writes and refunds require backend tests.
const uncertainTask = {
  id: 'task_reconcile_1', title: '海边公路镜头', mode: '文生视频', modeId: 'mode_text', status: '生成中',
  progress: 0, resolution: '768P', durationSec: 5, cost: 2.5, user: '创作者', userEmail: 'creator@example.test',
  provider: '测试供应商', upstreamTaskId: '', upstreamStatus: 'SUBMIT_UNKNOWN', failReason: '', time: '09-28 12:00',
  pricePerSec: 0.5, costPerSec: 0.02, supplierCode: 'test-provider', upstreamWorkflowId: 'internal_workflow',
  resultUrl: '', resultOssUrl: '', priceSnapshot: {}, routeSnapshot: {}, createdAt: '', updatedAt: '',
}

async function seedTask(page: Page, upstreamStatus = 'SUBMIT_UNKNOWN', providerRequestId = '') {
  await page.addInitScript(() => localStorage.setItem('s2v_access_token', 'admin-reconciliation-fixture'))
  const task = { ...uncertainTask, upstreamStatus, providerRequestId }
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (route.request().method() !== 'GET') {
      await route.fulfill({ status: 503, json: { message: '未配置的测试操作' } })
      return
    }
    if (path === '/api/v1/auth/me') {
      await route.fulfill({ json: { id: 'admin_reconcile', email: 'admin@example.test', displayName: '管理员', role: 'admin', balance: 0 } })
    } else if (path === '/api/v1/admin/tasks') {
      await route.fulfill({ json: { items: [task], total: 1, page: 1, pageSize: 50 } })
    } else if (path === `/api/v1/admin/tasks/${task.id}`) {
      await route.fulfill({ json: task })
    } else {
      await route.fulfill({ status: 404, json: { message: '未配置的测试读取' } })
    }
  })
  await page.goto('/admin/tasks', { waitUntil: 'networkidle' })
  await page.getByRole('button', { name: '详情', exact: true }).filter({ visible: true }).click()
  await expect(page.getByRole('heading', { name: '任务详情', exact: true })).toBeVisible()
  return task
}

test.describe('admin uncertain submission reconciliation (mocked frontend only)', () => {
  test('requires a verified upstream task and explicit confirmation before resuming', async ({ page }) => {
    const task = await seedTask(page)
    const requests: Record<string, unknown>[] = []
    await page.route(`**/api/v1/admin/tasks/${task.id}/reconcile`, async (route) => {
      requests.push(route.request().postDataJSON() as Record<string, unknown>)
      task.upstreamStatus = 'SUBMITTED'
      task.upstreamTaskId = 'provider_task_22'
      await route.fulfill({ json: { task: { id: task.id, status: task.status, upstreamStatus: task.upstreamStatus }, refund: 0, alreadyResolved: false } })
    })
    await expect(page.getByRole('button', { name: '终止退款', exact: true })).toHaveCount(0)
    await page.getByRole('button', { name: '关联已受理任务', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '关联已受理任务并继续查询', exact: true })
    const confirm = dialog.getByRole('button', { name: '确认关联并继续查询', exact: true })
    await expect(confirm).toBeDisabled()
    await dialog.getByLabel('已核实的上游任务号', { exact: true }).fill('provider_task_22')
    await dialog.getByLabel('核对记录（10–1000 字）', { exact: true }).fill('已在供应商后台核实任务创建时间和输入素材，属于当前平台任务。')
    await expect(confirm).toBeDisabled()
    expect(requests).toHaveLength(0)
    await dialog.getByRole('checkbox', { name: '我已核实该上游任务属于当前平台任务', exact: true }).check()
    await confirm.click()
    await expect(dialog).toHaveCount(0)
    expect(requests).toEqual([{ action: 'resume', upstreamTaskId: 'provider_task_22', evidence: '已在供应商后台核实任务创建时间和输入素材，属于当前平台任务。', confirmed: true }])
    await expect(page.getByText('provider_task_22', { exact: true }).filter({ visible: true }).first()).toBeVisible()
    await expect(page.getByRole('button', { name: '关联已受理任务', exact: true })).toHaveCount(0)
  })

  test('requires evidence of nonacceptance before refund and does not submit another task', async ({ page }) => {
    const task = await seedTask(page)
    const mutations: Array<{ path: string; payload: unknown }> = []
    page.on('request', (request) => {
      if (request.method() === 'POST') mutations.push({ path: new URL(request.url()).pathname, payload: request.postDataJSON() })
    })
    await page.route(`**/api/v1/admin/tasks/${task.id}/reconcile`, async (route) => {
      task.status = '失败'
      task.upstreamStatus = 'NOT_ACCEPTED'
      await route.fulfill({ json: { task: { id: task.id, status: task.status, upstreamStatus: task.upstreamStatus }, refund: 2.5, alreadyResolved: false } })
    })
    await page.getByRole('button', { name: '核实未受理并退款', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '确认未受理并退款', exact: true })
    const confirm = dialog.getByRole('button', { name: '确认未受理并退款', exact: true })
    await expect(dialog.getByLabel('已核实的上游任务号', { exact: true })).toHaveCount(0)
    await dialog.getByRole('checkbox', { name: '我已获得供应商确认：当前请求未受理且未创建任务', exact: true }).check()
    await dialog.getByLabel('核对记录（10–1000 字）', { exact: true }).fill('已核实')
    await expect(confirm).toBeDisabled()
    await dialog.getByLabel('核对记录（10–1000 字）', { exact: true }).fill('供应商客服已确认该请求未受理且未创建任何任务，记录已核对。')
    await confirm.click()
    await expect(dialog).toHaveCount(0)
    expect(mutations).toEqual([{ path: `/api/v1/admin/tasks/${task.id}/reconcile`, payload: { action: 'refund', evidence: '供应商客服已确认该请求未受理且未创建任何任务，记录已核对。', confirmed: true } }])
  })

  test('keeps evidence and the dialog when reconciliation fails', async ({ page }) => {
    const task = await seedTask(page)
    await page.route(`**/api/v1/admin/tasks/${task.id}/reconcile`, async (route) => {
      await route.fulfill({ status: 409, json: { code: 'TASK_ALREADY_RESOLVED', message: '任务已被其他管理员处理，请重新读取任务状态' } })
    })
    await page.getByRole('button', { name: '核实未受理并退款', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '确认未受理并退款', exact: true })
    await dialog.getByLabel('核对记录（10–1000 字）', { exact: true }).fill('供应商客服明确确认该请求未受理且未创建任务。')
    await dialog.getByRole('checkbox').check()
    await dialog.getByRole('button', { name: '确认未受理并退款', exact: true }).click()
    await expect(dialog.getByRole('alert')).toHaveText('任务已被其他管理员处理，请重新读取任务状态')
    await expect(dialog.getByLabel('核对记录（10–1000 字）', { exact: true })).toHaveValue('供应商客服明确确认该请求未受理且未创建任务。')
    await expect(dialog).toBeVisible()
  })

  test('does not expose reconciliation actions on a known running task', async ({ page }) => {
    await seedTask(page, 'RUNNING')
    await expect(page.getByText('提交结果待核对', { exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '关联已受理任务', exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '核实未受理并退款', exact: true })).toHaveCount(0)
  })

  test('keeps a captured provider task id fixed and offers no unaccepted refund', async ({ page }) => {
    await seedTask(page, 'SUBMIT_UNKNOWN', 'provider_task_captured')
    await expect(page.getByText('provider_task_captured', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '核实未受理并退款', exact: true })).toHaveCount(0)
    await page.getByRole('button', { name: '关联已受理任务', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '关联已受理任务并继续查询', exact: true })
    await expect(dialog.getByLabel('已核实的上游任务号', { exact: true })).toHaveValue('provider_task_captured')
    await expect(dialog.getByLabel('已核实的上游任务号', { exact: true })).toBeDisabled()
    await expect(dialog.getByText('平台已记录该上游任务号，只能核实并继续此任务。', { exact: true })).toBeVisible()
  })

  test('reconciliation dialog uses the existing admin surface on desktop and mobile', async ({ page }) => {
    await seedTask(page)
    await page.getByRole('button', { name: '核实未受理并退款', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '确认未受理并退款', exact: true })
    await expect(dialog).toBeVisible()
    const box = await dialog.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.x).toBeGreaterThanOrEqual(0)
    expect(box!.x + box!.width).toBeLessThanOrEqual(page.viewportSize()!.width)
    await page.evaluate(() => document.fonts.ready)
    await expect(page).toHaveScreenshot('admin-task-reconcile.png', { fullPage: true, animations: 'disabled', caret: 'hide', scale: 'css', maxDiffPixelRatio: 0.02 })
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
  })
})
