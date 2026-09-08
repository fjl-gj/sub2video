import { expect, test, type Page } from '@playwright/test'

const catalog = {
  modes: [
    { id: 'mode_text', inputType: 'text', name: '文生视频', description: '通过文字描述生成视频', enabled: true },
    { id: 'mode_keyframes', inputType: 'first_last', name: '首尾帧生成视频', description: '使用首帧和尾帧', enabled: true },
    { id: 'mode_multi', inputType: 'multi_image', name: '多图生成视频', description: '使用参考图', enabled: true },
  ],
  carriers: [{ id: 'carrier_minimax', name: 'MINMAX' }],
  models: [{ id: 'model_h3', carrierId: 'carrier_minimax', name: 'H3', description: '视频模型', inputTypes: ['text', 'first_last', 'multi_image'] }],
}

const options = {
  durations: [5, 10],
  resolutions: ['1080P'],
  aspectRatios: ['16:9'],
  supportsGeneratedAudio: true,
  minImages: 1,
  maxImages: 9,
  defaultDuration: 5,
  defaultResolution: '1080P',
}

async function seedAuthenticatedClient(page: Page, balance = 100) {
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'contract-access-token')
    localStorage.setItem('s2v_refresh_token', 'contract-refresh-token')
  })
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: 'u_contract', email: 'contract@sub2video.test', displayName: '合同测试', role: 'user', workspaceName: '合同测试作品', balance }),
    })
  })
  await page.route('**/api/v1/client/create/catalog', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(catalog) })
  })
  await page.route('**/api/v1/client/create/options**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(options) })
  })
  await page.route(/\/api\/v1\/client\/projects(?:\?.*)?$/, async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [], currentProjectId: '' }) })
  })
  await page.route('**/api/v1/client/storage/usage', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ usedGb: 0, quotaGb: 5, fileCount: 0, projectCount: 0, trashCount: 0 }) })
  })
  await page.route(/\/api\/v1\/client\/tasks(?:\?.*)?$/, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, pageSize: 20, counts: { 全部: 0, 排队中: 0, 生成中: 0, 已完成: 0, 失败: 0 } }),
    })
  })
}

async function prepareTextCreation(page: Page) {
  await page.goto('/app/create', { waitUntil: 'networkidle' })
  const textMode = page.getByRole('button', { name: '文生视频', exact: true })
  await expect(textMode).toBeVisible()
  await textMode.click()
  const model = page.getByRole('combobox', { name: '模型', exact: true })
  await expect(model).toBeVisible()
  await model.selectOption('model_h3')
  const prompt = page.getByPlaceholder('描述主体动作、镜头运动与环境变化…\n例如：主体缓缓低头，微风吹拂发丝，光影柔和自然')
  await prompt.fill('雨夜的霓虹街头，镜头缓慢推进')
  const submit = page.getByRole('button', { name: '提交生成任务', exact: true })
  await expect(submit).toBeEnabled()
  return submit
}

test.describe('client creation contracts (mocked frontend only)', () => {

  test('redirects a creator account away from the protected admin routes', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.route('**/api/v1/client/overview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          workspace: { displayName: '合同测试作品' }, billing: { totalSpent: 0, monthSpent: 0 }, wallet: { balance: 100 },
          pricing: { available: true, referencePricePerSecond: 0.5, referenceNote: '按秒计费' }, storage: { usedGb: 0, quotaGb: 5, fileCount: 0 },
          tasks: { activeCount: 0, activeTask: null, recentCompleted: [] },
        }),
      })
    })

    await page.goto('/admin/overview', { waitUntil: 'networkidle' })
    await expect(page).toHaveURL(/\/app\/overview$/)
  })

  test('renders only enabled catalog modes in the client-defined order', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.route('**/api/v1/client/create/catalog', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...catalog,
          modes: [
            { id: 'mode_keyframes', inputType: 'first_last', name: '首尾帧生成视频', description: '使用首帧和尾帧', enabled: true },
            { id: 'mode_multi', inputType: 'multi_image', name: '多图生成视频', description: '使用参考图', enabled: true },
            { id: 'mode_text', inputType: 'text', name: '文生视频', description: '通过文字描述生成视频', enabled: true },
          ],
        }),
      })
    })

    await page.goto('/app/create', { waitUntil: 'networkidle' })
    await expect(page.locator('.s2v-wb-segment')).toHaveText(['文生视频', '多图生成视频', '首尾帧生成视频'])

    await page.route('**/api/v1/client/create/catalog', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...catalog,
          modes: catalog.modes.map((item) => item.inputType === 'first_last' ? { ...item, enabled: false } : item),
        }),
      })
    })
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.getByRole('button', { name: '首尾帧生成视频', exact: true })).toHaveCount(0)
  })

  test('quotes a selected model and creates a task only from quoteId', async ({ page }) => {
    await seedAuthenticatedClient(page)
    let quotePayload: Record<string, unknown> | undefined
    let taskPayload: Record<string, unknown> | undefined
    await page.route('**/api/v1/client/tasks/quote', async (route) => {
      quotePayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          quoteId: 'quote_contract_1',
          expiresAt: '2030-01-01T00:00:00Z',
          selection: { carrier: { id: 'carrier_minimax', name: 'MINMAX' }, model: { id: 'model_h3', name: 'H3' } },
          price: { currency: 'CNY', pricePerSecond: 0.5, durationSeconds: 5, total: 2.5 },
        }),
      })
    })
    await page.route('**/api/v1/client/tasks', async (route) => {
      taskPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ task: { id: 'task_contract_1', title: '文生合同任务', status: '排队中', cost: 2.5, mode: '文生视频' }, wallet: { balance: 97.5 } }),
      })
    })

    const submit = await prepareTextCreation(page)
    await submit.click()
    await expect(page).toHaveURL(/\/app\/works$/)

    expect(quotePayload).toEqual({
      modeId: 'mode_text',
      modelId: 'model_h3',
      durationSeconds: 5,
      resolution: '1080P',
      aspectRatio: '16:9',
      generateAudio: false,
    })
    expect(taskPayload).toMatchObject({ quoteId: 'quote_contract_1', prompt: '雨夜的霓虹街头，镜头缓慢推进' })
    expect(Object.keys(taskPayload ?? {}).sort()).toEqual(['prompt', 'quoteId', 'title'])
    expect(JSON.stringify({ quotePayload, taskPayload })).not.toMatch(/supplier|workflow|costSnapshot|upstream/i)
  })

  test('does not create a task after an expired quote', async ({ page }) => {
    await seedAuthenticatedClient(page)
    let createCalls = 0
    await page.route('**/api/v1/client/tasks/quote', async (route) => {
      await route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ message: '报价已过期，请重新获取报价' }) })
    })
    await page.route('**/api/v1/client/tasks', async (route) => {
      createCalls += 1
      await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ message: '不应创建任务' }) })
    })

    const submit = await prepareTextCreation(page)
    await submit.click()
    await expect(page.getByText('报价已过期，请重新获取报价', { exact: true })).toBeVisible()
    expect(createCalls).toBe(0)
    await expect(page).toHaveURL(/\/app\/create$/)
  })

  test('keeps the creator on the form when precharge reports insufficient balance', async ({ page }) => {
    await seedAuthenticatedClient(page, 2)
    await page.route('**/api/v1/client/tasks/quote', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          quoteId: 'quote_balance_1',
          expiresAt: '2030-01-01T00:00:00Z',
          selection: { carrier: { id: 'carrier_minimax', name: 'MINMAX' }, model: { id: 'model_h3', name: 'H3' } },
          price: { currency: 'CNY', pricePerSecond: 0.5, durationSeconds: 5, total: 2.5 },
        }),
      })
    })
    await page.route('**/api/v1/client/tasks', async (route) => {
      await route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ message: '余额不足，无法预扣' }) })
    })

    const submit = await prepareTextCreation(page)
    await submit.click()
    await expect(page.getByText('余额不足，无法预扣', { exact: true })).toBeVisible()
    await expect(page).toHaveURL(/\/app\/create$/)
  })

  test('refunds a queued task through the cancel contract', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.route(/\/api\/v1\/client\/tasks(?:\?.*)?$/, async (route) => {
      if (route.request().method() !== 'GET') return route.fallback()
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [{ id: 'task_refund_1', title: '待执行任务', mode: '文生视频', status: '排队中', progress: 0, duration: '5 秒', resolution: '1080P', cost: 2.5, time: '刚刚', thumbUrl: '' }],
          total: 1,
          page: 1,
          pageSize: 20,
          counts: { 全部: 1, 排队中: 1, 生成中: 0, 已完成: 0, 失败: 0 },
        }),
      })
    })
    let cancelMethod = ''
    await page.route('**/api/v1/client/tasks/task_refund_1/cancel', async (route) => {
      cancelMethod = route.request().method()
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ task: { id: 'task_refund_1', status: '已取消' }, refundAmount: 2.5, wallet: { balance: 102.5 } }),
      })
    })

    await page.goto('/app/works', { waitUntil: 'networkidle' })
    const cancel = page.getByRole('button', { name: '终止任务并退款', exact: true })
    await expect(cancel).toBeVisible()
    await cancel.click()
    const confirm = page.getByRole('button', { name: '终止并退款', exact: true })
    await expect(confirm).toBeVisible()
    await confirm.click()

    await expect(page.getByText('已终止 task_refund_1，预扣费用已退回钱包', { exact: true })).toBeVisible()
    expect(cancelMethod).toBe('POST')
  })

  test('does not render supplier or upstream details from a task-detail response', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.route('**/api/v1/client/tasks/task_safe_1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'task_safe_1', title: '安全交付任务', mode: '文生视频', status: '已完成', progress: 100,
          duration: '5 秒', durationSeconds: 5, resolution: '1080P', aspectRatio: '16:9', generateAudio: false,
          cost: 2.5, thumbUrl: '', savedToProject: false, failureReason: '', executionWaiting: false,
          deliveryStatus: 'ready', downloadUrl: 'https://storage.example.test/task_safe_1.mp4', createdAt: '', updatedAt: '',
          supplierName: 'AutoDL-不应显示', upstreamWorkflowId: 'secret-upstream-不应显示', costSnapshot: { amount: 0.01 },
        }),
      })
    })

    await page.goto('/app/works/task_safe_1', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: '安全交付任务', exact: true })).toBeVisible()
    await expect(page.getByText('AutoDL-不应显示', { exact: true })).toHaveCount(0)
    await expect(page.getByText('secret-upstream-不应显示', { exact: true })).toHaveCount(0)
  })

  test('shows unavailable state instead of accepting a task without configured generation capacity', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.route('**/api/v1/client/create/catalog', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ modes: [], carriers: [], models: [] }) })
    })

    await page.goto('/app/create', { waitUntil: 'networkidle' })
    await expect(page.getByText('当前没有可用生成能力，暂不可提交', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
  })
})
