import { expect, test, type Page } from '@playwright/test'

const liveEnabled = process.env.SUB2VIDEO_LIVE === '1'
const email = process.env.SUB2VIDEO_LIVE_EMAIL || ''
const password = process.env.SUB2VIDEO_LIVE_PASSWORD || ''
const requestedModel = process.env.SUB2VIDEO_LIVE_MODEL_ID || ''
const imageBuffer = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lN8AAAAASUVORK5CYII=', 'base64')

// No setting in this suite permits paid task creation. Authentication tokens
// must not be retained in traces, videos or failure images.
test.use({ trace: 'off', video: 'off', screenshot: 'off' })

type CreationMode = 'text' | 'first_last' | 'multi_image'
type Catalog = {
  modes: Array<{ id: string; inputType: CreationMode; name: string; enabled: boolean }>
  models: Array<{ id: string; carrierId: string; inputTypes: CreationMode[] }>
}
type Options = { durations: number[]; resolutions: string[]; aspectRatios: string[]; defaultDuration: number; defaultResolution: string }
type Estimate = {
  available: boolean
  selection: { carrier: { id: string }; model: { id: string } }
  billing: { durationSec: number; resolution: string; generateAudio: boolean; estimatedTotal: number; pricePerSecond: number }
}
type Exchange = { payload: Record<string, unknown>; status: number; response: unknown }

function assertSafeClientFields(value: unknown, path = 'response', allowTaskAmount = false) {
  if (Array.isArray(value)) {
    value.forEach((item, index) => assertSafeClientFields(item, `${path}[${index}]`, allowTaskAmount))
    return
  }
  if (!value || typeof value !== 'object') return
  for (const [key, item] of Object.entries(value)) {
    // Task cost is the creator's charge, not a supplier cost snapshot.
    if (!(allowTaskAmount && key === 'cost')) {
      expect(key, `${path} must not expose internal fields`).not.toMatch(/supplier|cost|token|workflow|apiModelCode|executionType|upstream|quoteId/i)
    }
    assertSafeClientFields(item, `${path}.${key}`, allowTaskAmount)
  }
}

async function readClient<T>(page: Page, path: string) {
  const token = await page.evaluate(() => localStorage.getItem('s2v_access_token'))
  expect(Boolean(token), 'Real login must establish a session.').toBe(true)
  const response = await page.request.get(`/api/v1/client/${path}`, { headers: { Authorization: `Bearer ${token}` } })
  expect(response.ok(), `GET /client/${path} must succeed`).toBe(true)
  return await response.json() as T
}

async function snapshotAccounting(page: Page) {
  const wallet = await readClient<{ balance: number }>(page, 'wallet')
  const tasks = await readClient<{ items: Array<{ id: string }>; total: number }>(page, 'tasks?page=1&pageSize=100')
  assertSafeClientFields(wallet)
  assertSafeClientFields(tasks, 'tasks', true)
  return { balance: wallet.balance, taskTotal: tasks.total, taskIDs: tasks.items.map((task) => task.id) }
}

async function selectLocalImage(page: Page, slot: string, name: string) {
  await page.getByRole('button', { name: slot, exact: true }).click()
  const picker = page.getByRole('dialog', { name: '添加或选取素材', exact: true })
  await picker.getByLabel('参考图片文件', { exact: true }).setInputFiles({ name, mimeType: 'image/png', buffer: imageBuffer })
  await expect(picker).toHaveCount(0)
  await expect(page.getByRole('button', { name: slot, exact: true }).getByRole('img', { name, exact: true })).toBeVisible()
}

test.describe('Sub2Video live three-mode estimate preflight', () => {
  test.skip(!liveEnabled, 'set SUB2VIDEO_LIVE=1 to run against an isolated real frontend and backend')
  test.skip(!email || !password, 'set SUB2VIDEO_LIVE_EMAIL and SUB2VIDEO_LIVE_PASSWORD for the isolated test account')

  test('estimates three explicit mode/model selections without creating or charging a task', async ({ page }, testInfo) => {
    test.setTimeout(180_000)
    const blockedWrites: string[] = []
    const exchanges: Exchange[] = []
    const summary: Array<Record<string, string | number>> = []
    // All allowed requests reach the real service. This guard prevents writes
    // before transmission and never fabricates a business response.
    await page.route('**/api/v1/**', async (route) => {
      const request = route.request()
      const path = new URL(request.url()).pathname
      const allowed = ['GET', 'HEAD'].includes(request.method())
        || (request.method() === 'POST' && ['/api/v1/auth/login', '/api/v1/auth/refresh', '/api/v1/client/tasks/estimate'].includes(path))
      if (allowed) return route.continue()
      blockedWrites.push(`${request.method()} ${path}`)
      await route.abort('blockedbyclient')
    })
    page.on('response', async (response) => {
      if (response.request().method() !== 'POST' || new URL(response.url()).pathname !== '/api/v1/client/tasks/estimate') return
      let body: unknown = null
      try { body = await response.json() } catch { /* Missing JSON must fail the estimate assertion. */ }
      exchanges.push({ payload: response.request().postDataJSON() as Record<string, unknown>, status: response.status(), response: body })
    })
    await page.goto('/auth/login', { waitUntil: 'domcontentloaded' })
    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByLabel('密码', { exact: true }).fill(password)
    await page.getByRole('button', { name: '登录', exact: true }).click()
    await expect(page).toHaveURL(/\/app\/overview(?:$|\?)/, { timeout: 30_000 })
    const before = await snapshotAccounting(page)
    const catalog = await readClient<Catalog>(page, 'create/catalog')
    assertSafeClientFields(catalog)
    try {
      for (const inputType of ['text', 'first_last', 'multi_image'] as const) {
        const mode = catalog.modes.find((item) => item.inputType === inputType && item.enabled)
        expect(mode, `The real catalog must enable ${inputType}`).toBeTruthy()
        const model = catalog.models.find((item) => item.inputTypes.includes(inputType) && (!requestedModel || item.id === requestedModel))
        expect(model, `The real catalog must offer the requested model for ${inputType}`).toBeTruthy()
        const options = await readClient<Options>(page, `create/options?modeId=${encodeURIComponent(mode!.id)}&modelId=${encodeURIComponent(model!.id)}`)
        assertSafeClientFields(options)
        expect(options.durations.length).toBeGreaterThan(0)
        expect(options.resolutions.length).toBeGreaterThan(0)
        expect(options.aspectRatios.length).toBeGreaterThan(0)
        const durationSec = options.durations.includes(options.defaultDuration) ? options.defaultDuration : options.durations[0]
        const resolution = options.resolutions.includes(options.defaultResolution) ? options.defaultResolution : options.resolutions[0]
        const aspectRatio = options.aspectRatios[0]
        const startIndex = exchanges.length
        await page.goto(`/app/create?mode=${inputType}`, { waitUntil: 'networkidle' })
        await expect(page.getByRole('button', { name: mode!.name, exact: true })).toHaveAttribute('aria-pressed', 'true')
        await page.getByLabel('承运商', { exact: true }).selectOption(model!.carrierId)
        await page.getByLabel('模型', { exact: true }).selectOption(model!.id)
        await page.waitForLoadState('networkidle')
        await page.getByLabel('视频时长', { exact: true }).selectOption(String(durationSec))
        await page.getByLabel('输出分辨率', { exact: true }).selectOption(resolution)
        await page.getByLabel('画幅比例', { exact: true }).selectOption(aspectRatio)
        const audio = page.getByLabel('音频设置', { exact: true })
        if (await audio.isEnabled()) await audio.selectOption('none')
        await expect(audio).toHaveValue('none')
        await page.getByRole('textbox', { name: '画面描述', exact: true }).fill('测试场景：海边公路，镜头平稳推进，仅验证参数与预估。')
        // The app uploads at task submission. Local selection and preview
        // therefore do not imply acceptance by the object-storage service.
        if (inputType === 'first_last') {
          await selectLocalImage(page, '选择首帧', 'preflight-first.png')
          await selectLocalImage(page, '选择尾帧', 'preflight-last.png')
        } else if (inputType === 'multi_image') {
          await selectLocalImage(page, '选择参考图 1', 'preflight-reference.png')
        }
        const matchesSelection = (exchange: Exchange) => {
          const payload = exchange.payload
          const materials = payload.sourceMeta as { first?: unknown; last?: unknown; refs?: unknown[] } | undefined
          return payload.modeId === mode!.id && payload.carrierId === model!.carrierId && payload.modelId === model!.id
            && payload.durationSec === durationSec && payload.resolution === resolution && payload.aspectRatio === aspectRatio && payload.generateAudio === false
            && (inputType === 'text' || (inputType === 'first_last' ? !!materials?.first && !!materials?.last : materials?.refs?.length === 1))
        }
        await expect.poll(() => exchanges.slice(startIndex).some(matchesSelection), { timeout: 30_000, message: `${mode!.name} must estimate the final model, parameters and materials` }).toBe(true)
        const exchange = exchanges.slice(startIndex).filter(matchesSelection).at(-1)!
        assertSafeClientFields(exchange.payload, 'request')
        expect(exchange.payload).not.toHaveProperty('durationSeconds')
        expect(exchange.status, `${mode!.name} final estimate must succeed`).toBe(200)
        assertSafeClientFields(exchange.response)
        const estimate = exchange.response as Estimate
        expect(estimate.available).toBe(true)
        expect(estimate.selection.carrier.id).toBe(model!.carrierId)
        expect(estimate.selection.model.id).toBe(model!.id)
        expect(estimate.billing.durationSec).toBe(durationSec)
        expect(estimate.billing.resolution).toBe(resolution)
        expect(estimate.billing.generateAudio).toBe(false)
        expect(estimate.billing.estimatedTotal).toBeGreaterThan(0)
        await expect(page.getByLabel('费用预估', { exact: true })).toContainText(estimate.billing.estimatedTotal.toFixed(2))
        await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeEnabled()
        expect(await page.locator('main').innerText()).not.toMatch(/supplier|workflowCode|apiModelCode|executionType|autodl|供应商|采购成本/i)
        summary.push({ modeId: mode!.id, carrierId: model!.carrierId, modelId: model!.id, durationSec, resolution, aspectRatio, amount: estimate.billing.estimatedTotal })
        await page.evaluate(() => document.fonts.ready)
        await page.screenshot({ path: testInfo.outputPath(`live-creation-${inputType}.png`), fullPage: true, animations: 'disabled' })
        const submit = page.getByRole('button', { name: '提交生成任务', exact: true })
        await submit.scrollIntoViewIfNeeded()
        await expect(submit).toBeVisible()
        const navigation = page.getByRole('navigation', { name: '移动导航', exact: true })
        if (await navigation.isVisible()) {
          const submitBox = await submit.boundingBox()
          const navigationBox = await navigation.boundingBox()
          expect(submitBox!.y + submitBox!.height, 'The submission control must not be covered by mobile navigation').toBeLessThanOrEqual(navigationBox!.y)
        }
        expect(await page.locator('main').evaluate((main) => main.scrollWidth <= main.clientWidth + 1), 'The real creation page must not overflow horizontally').toBe(true)
        await page.screenshot({ path: testInfo.outputPath(`live-creation-${inputType}-estimate.png`), fullPage: true, animations: 'disabled' })
      }
    } finally {
      expect(await snapshotAccounting(page), 'Estimate preflight must leave wallet balance and task list unchanged').toEqual(before)
      expect(blockedWrites, 'The UI must not attempt task creation, uploads or other writes during estimate preflight').toEqual([])
    }
    expect(summary).toHaveLength(3)
    await testInfo.attach('three-mode-estimates.json', { body: JSON.stringify(summary, null, 2), contentType: 'application/json' })
  })
})
