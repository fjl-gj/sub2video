import { expect, test, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// UI recovery contracts with explicit API fixtures. These checks do not prove
// a real charge, provider response, refund, or delivered video.
const taskID = 'task_experience'
const task = {
  id: taskID, title: '体验验收任务', mode: '文生视频', modeId: 'mode_text', carrierId: 'car_minimax', modelId: 'mdl_h3',
  prompt: '清晨海岸，镜头缓慢向前移动。', sourceMeta: {}, status: '生成中', progress: 30,
  duration: '5 秒', durationSeconds: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false,
  cost: 2.5, thumbUrl: '', savedToProject: false, failureReason: '', executionWaiting: false,
  canCancel: false, reviewRequired: false, refundAmount: 0, refundedAt: null,
  deliveryStatus: 'not_ready', downloadUrl: '', createdAt: '2026-09-25T04:00:00Z', updatedAt: '2026-09-25T04:00:00Z', time: '09-25 12:00',
}

async function seed(page: Page) {
  const fixture = await seedSurfaceData(page)
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'experience-test-access')
    localStorage.setItem('s2v_refresh_token', 'experience-test-refresh')
  })
  await page.route('**/api/v1/auth/me', (route) => route.fulfill({ json: {
    id: 'user_experience', email: 'experience@example.test', displayName: '体验验收', workspaceName: '测试工作室', role: 'user', balance: 256,
  } }))
  return fixture
}

test.describe('task and aftercare experience (mocked frontend only)', () => {
  for (const entry of ['detail', 'list'] as const) {
    test(`downloads an attachment instead of opening the preview from the ${entry}`, async ({ page }) => {
      const { unmatchedRequests } = await seed(page)
      const ready = { ...task, status: '已完成', deliveryStatus: 'ready', downloadUrl: '/__experience-preview.mp4' }
      let downloadRequests = 0
      await page.route('**/__experience-preview.mp4', (route) => route.fulfill({ status: 503, body: 'Preview intentionally unavailable' }))
      await page.route('https://media.experience.test/attachment.mp4', (route) => route.fulfill({
        contentType: 'application/octet-stream', headers: { 'Content-Disposition': 'attachment; filename="experience-contract.mp4"' },
        body: 'Explicit download contract fixture, not a playable or generated video.',
      }))
      await page.route(`**/api/v1/client/tasks/${taskID}`, (route) => route.fulfill({ json: ready }))
      await page.route(`**/api/v1/client/tasks/${taskID}?download=true`, (route) => {
        downloadRequests++
        return route.fulfill({ json: { ...ready, downloadUrl: 'https://media.experience.test/attachment.mp4' } })
      })
      await page.route('**/api/v1/client/tasks?*', (route) => route.fulfill({ json: { items: [ready], total: 1, counts: { 全部: 1, 已完成: 1 } } }))
      const path = entry === 'detail' ? `/app/works/${taskID}` : '/app/works'
      await page.goto(path, { waitUntil: 'networkidle' })
      const downloadPromise = page.waitForEvent('download')
      await page.getByRole('button', { name: entry === 'detail' ? '下载成片' : '下载结果', exact: true }).filter({ visible: true }).click()
      const download = await downloadPromise
      expect(download.suggestedFilename()).toBe('experience-contract.mp4')
      expect(downloadRequests).toBe(1)
      await expect(page).toHaveURL((url) => url.pathname === path)
      expect(unmatchedRequests).toEqual([])
    })
  }

  test('retains an attachment preparation failure and lets the creator retry', async ({ page }) => {
    const { unmatchedRequests } = await seed(page)
    await page.route(`**/api/v1/client/tasks/${taskID}`, (route) => route.fulfill({ json: { ...task, status: '已完成', deliveryStatus: 'ready', downloadUrl: '/__experience-preview.mp4' } }))
    await page.route('**/__experience-preview.mp4', (route) => route.fulfill({ status: 503, body: 'Preview intentionally unavailable' }))
    await page.route(`**/api/v1/client/tasks/${taskID}?download=true`, (route) => route.fulfill({ status: 503, json: { message: 'Attachment signing unavailable' } }))
    await page.goto(`/app/works/${taskID}`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '下载成片', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('下载暂时不可用')
    await expect(page.getByRole('button', { name: '下载成片', exact: true })).toBeEnabled()
    expect(unmatchedRequests).toEqual([])
  })

  test('explains a submission under review and carries its task to support without a resubmit action', async ({ page }) => {
    const { unmatchedRequests } = await seed(page)
    await page.route(`**/api/v1/client/tasks/${taskID}`, (route) => route.fulfill({ json: { ...task, reviewRequired: true } }))
    await page.goto(`/app/works/${taskID}`, { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: '提交结果正在核对' })).toBeVisible()
    await expect(page.getByText('待核对', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '按原参数再来一次' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '继续创作' })).toHaveCount(0)
    await page.getByRole('link', { name: '关联此任务联系支持' }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/tickets' && url.searchParams.get('taskId') === taskID)
    await expect(page.getByLabel('关联任务编号')).toHaveValue(taskID)
    expect(unmatchedRequests).toEqual([])
  })

  test('opens the saved project folder after reloading task detail', async ({ page }) => {
    const { unmatchedRequests } = await seed(page)
    await page.route('**/api/v1/client/projects/prj_visual_coast/files/asset_visual_coast?*', (route) => route.fulfill({ status: 503, body: 'Deterministic unavailable preview' }))
    await page.route(`**/api/v1/client/tasks/${taskID}`, (route) => route.fulfill({ json: {
      ...task, status: '已完成', savedToProject: true,
      savedDestination: { projectId: 'prj_visual_coast', folderPath: ['资产', '场景'] },
    } }))
    await page.goto(`/app/works/${taskID}`, { waitUntil: 'networkidle' })
    await page.reload({ waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '查看作品', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/assets' && url.searchParams.get('project') === 'prj_visual_coast' && url.searchParams.get('folder') === '资产/场景')
    await expect(page.getByText('海岸参考图.svg', { exact: true })).toBeVisible()
    expect(unmatchedRequests).toEqual([])
  })

  test('only offers a server-approved cancellation and retains a raced rejection in the dialog', async ({ page }) => {
    const { unmatchedRequests } = await seed(page)
    let canCancel = true
    let cancelCalls = 0
    await page.route('**/api/v1/client/tasks?*', (route) => route.fulfill({ json: {
      items: [
        { ...task, id: 'task_queued', title: '可取消任务', status: '排队中', canCancel },
        { ...task, id: 'task_accepted', title: '已进入执行' },
        { ...task, id: 'task_review', title: '待核对任务', reviewRequired: true },
      ], total: 3, counts: { 全部: 3, 排队中: 1, 生成中: 2 },
    } }))
    await page.route('**/api/v1/client/tasks/task_queued/cancel', async (route) => {
      cancelCalls++
      canCancel = false
      await route.fulfill({ status: 409, json: { message: '任务已进入执行，暂不可取消。' } })
    })
    await page.goto('/app/works', { waitUntil: 'networkidle' })
    const visibleCancel = page.getByRole('button', { name: '终止任务并退款', exact: true }).filter({ visible: true })
    await expect(visibleCancel).toHaveCount(1)
    await visibleCancel.click()
    const dialog = page.getByRole('dialog', { name: '终止渲染任务确认' })
    await expect(dialog).not.toContainText('即时全额')
    await dialog.getByRole('button', { name: '终止并退款', exact: true }).click()
    await expect(dialog.getByRole('alert')).toContainText('任务已进入执行')
    await expect(dialog.getByRole('button', { name: '终止并退款', exact: true })).toHaveCount(0)
    await expect(dialog.getByRole('button', { name: '查看最新任务状态' })).toBeVisible()
    expect(cancelCalls).toBe(1)
    expect(unmatchedRequests).toEqual([])
  })

  test('distinguishes a task loading failure from an empty list and recovers explicitly', async ({ page }) => {
    const { unmatchedRequests } = await seed(page)
    let failing = true
    await page.route('**/api/v1/client/tasks?*', async (route) => {
      if (failing) await route.fulfill({ status: 503, json: { message: 'Unavailable' } })
      else await route.fulfill({ json: { items: [], total: 0, counts: {} } })
    })
    await page.goto('/app/works', { waitUntil: 'networkidle' })
    await expect(page.getByRole('alert')).toContainText('任务列表暂时无法读取')
    await expect(page.getByText('暂无符合条件的任务')).toHaveCount(0)
    failing = false
    await page.getByRole('button', { name: '重新加载任务' }).click()
    await expect(page.getByText('暂无符合条件的任务')).toBeVisible()
    await expect(page.getByRole('alert')).toHaveCount(0)
    expect(unmatchedRequests).toEqual([])
  })

  test('keeps wallet errors and a rejected redemption visible without pretending the account is empty', async ({ page }) => {
    const { unmatchedRequests } = await seed(page)
    let failing = true
    await page.route('**/api/v1/client/wallet', async (route) => {
      if (failing) await route.fulfill({ status: 503, json: { message: 'Unavailable' } })
      else await route.fulfill({ json: { balance: 21.5, products: [] } })
    })
    await page.route('**/api/v1/client/wallet/transactions?*', async (route) => {
      if (failing) await route.fulfill({ status: 503, json: { message: 'Unavailable' } })
      else await route.fulfill({ json: { items: [], total: 0 } })
    })
    await page.route('**/api/v1/client/wallet/redeem', (route) => route.fulfill({ status: 400, json: { message: '兑换码无效，请检查后重试。' } }))
    await page.goto('/app/wallet', { waitUntil: 'networkidle' })
    await expect(page.getByText('余额暂时无法读取，请重新加载。', { exact: true })).toBeVisible()
    await expect(page.getByText('余额流水暂时无法读取，请重新加载。', { exact: true })).toBeVisible()
    await expect(page.getByText('暂无流水记录', { exact: true })).toHaveCount(0)
    await expect(page.getByText('选择充值档位，赠送金额会一起到账。')).toHaveCount(0)
    failing = false
    await page.getByRole('button', { name: '重新加载余额' }).click()
    await page.getByRole('button', { name: '重新加载流水' }).click()
    await expect(page.getByRole('main').getByText('¥ 21.50', { exact: true })).toBeVisible()
    await expect(page.getByText('暂无流水记录', { exact: true })).toBeVisible()
    await page.getByLabel('兑换码', { exact: true }).fill('INVALID-EXPERIENCE')
    await page.getByRole('button', { name: '确认兑换' }).click()
    await expect(page.getByRole('alert')).toContainText('兑换码无效')
    await expect(page.getByLabel('兑换码', { exact: true })).toHaveValue('INVALID-EXPERIENCE')
    expect(unmatchedRequests).toEqual([])
  })

  test('shows a real ticket detail and preserves task context and input after submission fails', async ({ page }) => {
    const { unmatchedRequests } = await seed(page)
    const ticket = { id: 'ticket_experience', title: '请核对这条任务', category: '生成异常', status: '处理中', task: taskID, updated: '09-25 12:00', detail: '我已提交一次，请核对结果。\n不要重复生成。' }
    let failing = true
    let submitted: unknown
    await page.route('**/api/v1/client/tickets?*', async (route) => {
      if (failing) await route.fulfill({ status: 503, json: { message: 'Unavailable' } })
      else await route.fulfill({ json: { items: [ticket], total: 1 } })
    })
    await page.route('**/api/v1/client/tickets', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      submitted = route.request().postDataJSON()
      await route.fulfill({ status: 503, json: { message: '提交暂不可用，请稍后重试。' } })
    })
    await page.goto(`/app/tickets?taskId=${taskID}`, { waitUntil: 'networkidle' })
    await expect(page.getByLabel('关联任务编号')).toHaveValue(taskID)
    await expect(page.getByText('暂无工单记录。')).toHaveCount(0)
    failing = false
    await page.getByRole('button', { name: '重新加载工单' }).click()
    await page.getByRole('button', { name: /请核对这条任务/ }).click()
    const dialog = page.getByRole('dialog', { name: ticket.title })
    await expect(dialog.getByText(ticket.detail, { exact: true })).toBeVisible()
    await expect(dialog.getByRole('link', { name: taskID })).toHaveAttribute('href', `/app/works/${taskID}`)
    await dialog.getByRole('button', { name: '关闭' }).click()
    await page.getByLabel('工单标题').fill('生成任务需要帮助')
    await page.getByLabel('问题描述').fill('请保留我的问题描述。')
    await page.getByRole('button', { name: '提交', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('提交暂不可用')
    await expect(page.getByLabel('工单标题')).toHaveValue('生成任务需要帮助')
    await expect(page.getByLabel('问题描述')).toHaveValue('请保留我的问题描述。')
    expect(submitted).toMatchObject({ taskId: taskID, title: '生成任务需要帮助', detail: '请保留我的问题描述。' })
    expect(unmatchedRequests).toEqual([])
  })
})
