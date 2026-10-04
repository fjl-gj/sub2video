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
  await page.route('**/api/v1/client/tasks/estimate', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ available: true, selection: { carrier: { id: 'carrier_minimax', name: 'MINMAX' }, model: { id: 'model_h3', name: 'H3' } }, billing: { currency: 'CNY', pricePerSecond: 0.5, estimatedTotal: 2.5, durationSec: 5, resolution: '1080P', generateAudio: false } }) })
  })
  await page.route(/\/api\/v1\/client\/projects(?:\?.*)?$/, async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [], currentProjectId: '' }) })
  })
  await page.route('**/api/v1/client/storage/usage', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ usedGb: 0, quotaGb: 5, fileCount: 0, projectCount: 0, trashCount: 0 }) })
  })
  await page.route('**/api/v1/client/wallet', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ balance, products: [] }) })
  })
  const submittedTasks: Record<string, string> = {
    task_contract_1: '文生合同任务', task_audio_1: '音频合同任务',
    task_generated_audio_1: '自动音频合同任务', task_reroute: '重路由任务',
  }
  await page.route(/\/api\/v1\/client\/tasks\/task_(?:contract_1|audio_1|generated_audio_1|reroute)$/, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    const id = new URL(route.request().url()).pathname.split('/').pop()!
    await route.fulfill({ json: {
      id, title: submittedTasks[id], mode: '文生视频', status: '排队中', progress: 0,
      duration: '5 秒', durationSeconds: 5, resolution: '1080P', aspectRatio: '16:9', generateAudio: false,
      cost: 2.5, thumbUrl: '', savedToProject: false, failureReason: '', executionWaiting: false,
      deliveryStatus: 'not_ready', downloadUrl: '', createdAt: '', updatedAt: '',
    } })
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

async function prepareTextCreation(page: Page, enabled = true) {
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
  if (enabled) await expect(submit).toBeEnabled()
  return submit
}

test.describe('client creation contracts (mocked frontend only)', () => {

  test('replays the same frozen request after a lost response and page refresh', async ({ page }) => {
    await seedAuthenticatedClient(page)
    const submissions: Array<{ key: string; body: unknown }> = []
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      submissions.push({ key: route.request().headers()['idempotency-key']!, body: route.request().postDataJSON() })
      if (submissions.length === 1) return route.abort('connectionreset')
      await route.fulfill({ json: { task: { id: 'task_contract_1', status: '排队中' }, order: { id: 'ord_contract_1', amount: 2.5 }, wallet: { balance: 97.5 }, idempotentReplay: true } })
    })
    await (await prepareTextCreation(page)).click()
    await expect(page.getByRole('alert')).toContainText('尚未确认提交结果')
    await page.reload({ waitUntil: 'networkidle' })
    const recover = page.getByRole('button', { name: '确认原提交结果', exact: true })
    await expect(recover).toBeVisible()
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue('雨夜的霓虹街头，镜头缓慢推进')
    await expect(page.getByRole('combobox', { name: '模型', exact: true })).toHaveValue('model_h3')
    await expect(page.locator('.s2v-studio-form')).toHaveScreenshot('pending-creation.png', { animations: 'disabled', caret: 'hide' })
    await page.getByLabel('画面描述', { exact: true }).fill('刷新后改写的内容不能替换未确认订单')
    await page.getByRole('combobox', { name: '视频时长', exact: true }).selectOption('10')
    await recover.click()
    await expect(page).toHaveURL(/\/app\/works\/task_contract_1$/)
    expect(submissions).toHaveLength(2)
    expect(submissions[1]).toEqual(submissions[0])
    expect(await page.evaluate(() => localStorage.getItem('sub2video.pending-creation.u_contract'))).toBeNull()
  })

  test('does not send a task when its recovery record cannot be persisted', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.addInitScript(() => {
      const setItem = Storage.prototype.setItem
      Storage.prototype.setItem = function (key, value) {
        if (key.startsWith('sub2video.pending-creation.')) throw new DOMException('Quota exceeded', 'QuotaExceededError')
        setItem.call(this, key, value)
      }
    })
    let posts = 0
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      posts += 1
      await route.abort()
    })
    await (await prepareTextCreation(page)).click()
    await expect(page.getByRole('alert')).toContainText('暂未提交任务')
    expect(posts).toBe(0)
  })

  test('a definitive prompt rejection clears the pending request instead of requiring reconciliation', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      await route.fulfill({ status: 400, json: { code: 'PROMPT_REQUIRED', message: '请先填写画面描述' } })
    })
    await (await prepareTextCreation(page)).click()
    await expect(page.getByRole('alert')).toContainText('请先填写画面描述，再提交任务。')
    await expect(page.getByRole('button', { name: '确认原提交结果', exact: true })).toHaveCount(0)
    expect(await page.evaluate(() => localStorage.getItem('sub2video.pending-creation.u_contract'))).toBeNull()
  })

  test('keeps the request identity when a success response is incomplete', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      await route.fulfill({ json: { task: { id: 'task_contract_1' } } })
    })
    await (await prepareTextCreation(page)).click()
    await expect(page.getByRole('button', { name: '确认原提交结果', exact: true })).toBeVisible()
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.getByRole('button', { name: '确认原提交结果', exact: true })).toBeVisible()
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.pending-creation.u_contract') || '{}').requestId)).toBeTruthy()
  })

  test('distinguishes missing materials and low balance from service failures', async ({ page }) => {
    await seedAuthenticatedClient(page, 1)
    let estimates = 0
    page.on('request', (request) => { if (request.url().endsWith('/client/tasks/estimate')) estimates += 1 })
    await page.goto('/app/create?mode=first_last', { waitUntil: 'networkidle' })
    await expect(page.getByLabel('费用预估', { exact: true })).toContainText('等待素材')
    await expect(page.getByText('请先准备首帧与尾帧各一张图片，再查看费用。', { exact: true })).toBeVisible()
    expect(estimates).toBe(0)
    await page.getByRole('button', { name: '文生视频', exact: true }).click()
    await page.getByLabel('画面描述', { exact: true }).fill('一个测试镜头')
    await expect(page.getByText(/余额不足，请前往钱包/)).toBeVisible()
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
  })

  test('does not restore a local-only image as uploaded after refresh', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.addInitScript(() => {
      localStorage.setItem('sub2video.create-draft.u_contract', JSON.stringify({ version: 4, workflowInputType: 'first_last', prompt: '测试', firstFrame: { uploaded: true, assetId: '', previewUrl: 'blob:expired', fileName: 'local.png' }, lastFrame: { uploaded: true, assetId: '', fileName: 'last.png' } }))
    })
    await page.goto('/app/create?mode=first_last', { waitUntil: 'networkidle' })
    await expect(page.getByLabel('费用预估', { exact: true })).toContainText('等待素材')
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    await expect(page.locator('img[src^="blob:"]')).toHaveCount(0)
  })

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
    await expect(page.getByLabel('创作模式', { exact: true }).getByRole('button')).toHaveText(['文生视频', '多图生成视频', '首尾帧生成视频'])

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

  test('synchronizes the creation-page balance from the wallet API', async ({ page }) => {
    await seedAuthenticatedClient(page, 42.5)

    await page.goto('/app/create', { waitUntil: 'networkidle' })
    await expect(page.getByText('钱包余额: ¥ 42.50', { exact: true })).toBeVisible()
  })

  test('estimates a selected model and submits the complete creation contract', async ({ page }) => {
    await seedAuthenticatedClient(page)
    let estimatePayload: Record<string, unknown> | undefined
    let taskPayload: Record<string, unknown> | undefined
    await page.route('**/api/v1/client/tasks/estimate', async (route) => {
      estimatePayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          available: true,
          selection: { carrier: { id: 'carrier_minimax', name: 'MINMAX' }, model: { id: 'model_h3', name: 'H3' } },
          billing: { currency: 'CNY', pricePerSecond: 0.5, durationSec: 5, estimatedTotal: 2.5 },
        }),
      })
    })
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      taskPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
      body: JSON.stringify({ task: { id: 'task_contract_1', title: '文生合同任务', status: '排队中', cost: 2.5, mode: '文生视频' }, order: { id: 'ord_contract_1', amount: 2.5, status: 'paid' }, wallet: { balance: 97.5 } }),
      })
    })

    const submit = await prepareTextCreation(page)
    await submit.click()
    await expect(page).toHaveURL(/\/app\/works\/task_contract_1$/)

    expect(estimatePayload).toEqual({
      modeId: 'mode_text',
      modelId: 'model_h3',
      durationSec: 5,
      resolution: '1080P',
      aspectRatio: '16:9',
      generateAudio: false,
    })
    expect(taskPayload).toMatchObject({ modeId: 'mode_text', modelId: 'model_h3', prompt: '雨夜的霓虹街头，镜头缓慢推进', durationSec: 5, resolution: '1080P', aspectRatio: '16:9', generateAudio: false })
    expect(estimatePayload).not.toHaveProperty('durationSeconds')
    expect(taskPayload).not.toHaveProperty('durationSeconds')
    expect(taskPayload).not.toHaveProperty('quoteId')
    expect(JSON.stringify({ estimatePayload, taskPayload })).not.toMatch(/supplier|cost|token|workflow|executionType|apiModelCode|workflowCode|upstream/i)
  })

  test('uploads optional reference audio only when the creator selects it', async ({ page }) => {
    await seedAuthenticatedClient(page)
    let audioUploadCalls = 0
    let taskPayload: Record<string, unknown> | undefined
    await page.route('**/api/v1/client/uploads/reference-audio', async (route) => {
      audioUploadCalls += 1
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'upl_audio_1', fileName: 'voice.mp3', mimeType: 'audio/mpeg', sizeBytes: 12, url: '/api/v1/client/uploads/upl_audio_1' }),
      })
    })
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      taskPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ task: { id: 'task_audio_1', title: '音频合同任务', status: '排队中', cost: 2.5, mode: '文生视频' }, order: { id: 'ord_audio_1', amount: 2.5, status: 'paid' }, wallet: { balance: 97.5 } }),
      })
    })

    const submit = await prepareTextCreation(page)
    await page.getByRole('combobox', { name: '音频设置', exact: true }).selectOption('reference')
    await page.getByLabel('参考音频文件', { exact: true }).setInputFiles({
      name: 'voice.mp3', mimeType: 'audio/mpeg', buffer: Buffer.from('audio-contract'),
    })
    await expect(page.getByText('voice.mp3', { exact: true })).toBeVisible()
    await submit.click()

    await expect(page).toHaveURL(/\/app\/works\/task_audio_1$/)
    expect(audioUploadCalls).toBe(1)
    expect(taskPayload).toMatchObject({
      generateAudio: true,
      sourceMeta: { audio: { type: 'upload', id: 'upl_audio_1' } },
    })
  })

  test('generates audio without requiring or uploading a reference audio file', async ({ page }) => {
    await seedAuthenticatedClient(page)
    let audioUploadCalls = 0
    let taskPayload: Record<string, unknown> | undefined
    await page.route('**/api/v1/client/uploads/reference-audio', async (route) => {
      audioUploadCalls += 1
      await route.fulfill({ status: 500 })
    })
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      taskPayload = route.request().postDataJSON() as Record<string, unknown>
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ task: { id: 'task_generated_audio_1', title: '自动音频合同任务', status: '排队中', cost: 2.5, mode: '文生视频' }, order: { id: 'ord_generated_audio_1', amount: 2.5, status: 'paid' }, wallet: { balance: 97.5 } }),
      })
    })

    const submit = await prepareTextCreation(page)
    await page.getByRole('combobox', { name: '音频设置', exact: true }).selectOption('generate')
    await submit.click()

    await expect(page).toHaveURL(/\/app\/works\/task_generated_audio_1$/)
    expect(audioUploadCalls).toBe(0)
    expect(taskPayload).toMatchObject({ generateAudio: true })
    expect(taskPayload).not.toHaveProperty('sourceMeta')
  })

  test('keeps submit disabled until an estimate is available', async ({ page }) => {
    await seedAuthenticatedClient(page)
    await page.unroute('**/api/v1/client/tasks/estimate')
    let createCalls = 0
    await page.route('**/api/v1/client/tasks/estimate', async (route) => {
      await route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ message: '当前参数暂无可用执行配置' }) })
    })
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      createCalls += 1
      await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ task: { id: 'task_reroute', title: '重路由任务', status: '排队中', cost: 2.5, mode: '文生视频' }, order: { id: 'ord_reroute', amount: 2.5, currency: 'CNY', status: 'paid' }, wallet: { balance: 97.5 } }) })
    })

    const submit = await prepareTextCreation(page, false)
    await expect(submit).toBeDisabled()
    await expect(page.getByRole('button', { name: '重新预估费用' })).toBeVisible()
    expect(createCalls).toBe(0)
  })

  test('keeps the creator on the form when precharge reports insufficient balance', async ({ page }) => {
    await seedAuthenticatedClient(page, 100)
    await page.route('**/api/v1/client/tasks', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      await route.fulfill({ status: 402, contentType: 'application/json', body: JSON.stringify({ code: 'INSUFFICIENT_BALANCE', message: '余额不足' }) })
    })

    const submit = await prepareTextCreation(page)
    await submit.click()
    await expect(page.getByRole('alert')).toContainText('余额不足，无法预扣')
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
          items: [{ id: 'task_refund_1', title: '待执行任务', mode: '文生视频', status: '排队中', canCancel: true, progress: 0, duration: '5 秒', resolution: '1080P', cost: 2.5, time: '刚刚', thumbUrl: '' }],
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
