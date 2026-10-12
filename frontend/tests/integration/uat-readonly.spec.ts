import { expect, test, type Page } from '@playwright/test'

// This suite is selected by playwright.readonly.config.ts, whose fail-fast
// validation prevents an all-skipped run from masquerading as UAT acceptance.
const email = process.env.SUB2VIDEO_LIVE_EMAIL || ''
const password = process.env.SUB2VIDEO_LIVE_PASSWORD || ''
const profile = process.env.SUB2VIDEO_UAT_PROFILE || ''
test.use({ serviceWorkers: 'block', trace: 'off', screenshot: 'off', video: 'off' })

type Catalog = {
  modes: { id: string; name: string; inputType: string; enabled: boolean }[]
  models: { id: string; carrierId: string; inputTypes: string[] }[]
}

type ReadPath = 'wallet' | 'tasks?page=1&pageSize=1' | 'wallet/transactions?page=1&pageSize=1' | 'create/catalog'
type Estimate = { code?: string; available?: boolean; billing?: { estimatedTotal?: number } }

async function clientFetch(page: Page, path: ReadPath | 'tasks/estimate', payload?: Record<string, unknown>) {
  // Keep tokens inside the browser. APIRequestContext errors include request
  // headers in their call logs, even when tracing is disabled.
  const result = await page.evaluate(async ({ path, payload }) => {
    const token = localStorage.getItem('s2v_access_token')
    if (!token) return { status: 0, data: null, failure: 'missing-session' }
    if (payload && path !== 'tasks/estimate') return { status: 0, data: null, failure: 'write-not-allowed' }
    try {
      const response = await fetch(`/api/v1/client/${path}`, {
        method: payload ? 'POST' : 'GET', redirect: 'error', credentials: 'same-origin',
        headers: { Authorization: `Bearer ${token}`, ...(payload ? { 'Content-Type': 'application/json' } : {}) },
        body: payload ? JSON.stringify(payload) : undefined, signal: AbortSignal.timeout(20_000),
      })
      try { return { status: response.status, data: await response.json() as unknown, failure: '' } }
      catch { return { status: response.status, data: null, failure: 'invalid-json' } }
    } catch { return { status: 0, data: null, failure: 'network' } }
  }, { path, payload })
  expect(result.failure, 'Authenticated read-only request completed safely').toBe('')
  expect(hasInternalFields(result.data), 'Client DTO excludes internal execution fields').toBe(false)
  return result
}

async function loginWithoutCredentialArtifacts(page: Page) {
  await page.goto('/auth/login', { waitUntil: 'domcontentloaded' })
  await expect(page.getByLabel('密码', { exact: true })).toHaveAttribute('type', 'password')
  try {
    for (const [label, value] of [['邮箱', email], ['密码', password]]) {
      const input = page.getByLabel(label, { exact: true })
      await expect(input).toBeEditable()
      // locator.fill(secret) embeds its argument in a failed action's call log.
      // Native input events preserve Vue's normal form data flow without it.
      const set = await input.evaluate((element, secret) => {
        if (!(element instanceof HTMLInputElement)) return false
        const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
        if (!setter) return false
        setter.call(element, secret)
        element.dispatchEvent(new Event('input', { bubbles: true }))
        element.dispatchEvent(new Event('change', { bubbles: true }))
        return true
      }, value)
      expect(set, 'Authentication input accepted native form events').toBe(true)
    }
    await page.getByRole('button', { name: '登录', exact: true }).click()
    await expect(page).toHaveURL(/\/app\/overview/, { timeout: 30_000 })
  } finally {
    // Also remove values if login fails while the form remains mounted.
    await page.evaluate(() => {
      const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
      for (const input of document.querySelectorAll('input[name="email"], input[name="password"]')) {
        if (!(input instanceof HTMLInputElement)) continue
        setter?.call(input, '')
        input.dispatchEvent(new Event('input', { bubbles: true }))
        input.dispatchEvent(new Event('change', { bubbles: true }))
      }
    }).catch(() => undefined)
  }
}

function hasInternalFields(value: unknown): boolean {
  if (!value || typeof value !== 'object') return false
  // Task `cost` is the user's retail charge; auth tokens belong only to auth
  // exchanges, which are deliberately excluded from this client DTO check.
  const forbidden = /^(supplier(?:Id|Code|Name|Cost)?|costSnapshot|executionSnapshot|executionType|workflowCode|apiModelCode|upstream(?:Url|Address|TaskId)|token)$/i
  return Object.entries(value).some(([key, child]) => forbidden.test(key) || hasInternalFields(child))
}

async function readClient<T>(page: Page, path: ReadPath): Promise<T> {
  const response = await clientFetch(page, path)
  expect(response.status, 'Authenticated client read succeeds').toBe(200)
  return response.data as T
}

async function accounting(page: Page) {
  const wallet = await readClient<{ balance: number }>(page, 'wallet')
  const tasks = await readClient<{ total: number }>(page, 'tasks?page=1&pageSize=1')
  const ledger = await readClient<{ total: number }>(page, 'wallet/transactions?page=1&pageSize=1')
  expect(Number.isFinite(wallet.balance), 'Balance is a finite number').toBe(true)
  expect(Number.isInteger(tasks.total) && tasks.total >= 0, 'Task count is a nonnegative integer').toBe(true)
  expect(Number.isInteger(ledger.total) && ledger.total >= 0, 'Ledger count is a nonnegative integer').toBe(true)
  return { balance: wallet.balance, tasks: tasks.total, ledger: ledger.total }
}

test('UAT-RO-01 real login, creation states and wallet stay consistent without business writes', async ({ page, context, baseURL }, testInfo) => {
  test.skip(!['ready', 'unconfigured'].includes(profile), 'Use pnpm test:uat:readonly with an explicit profile')
  test.setTimeout(90_000)
  const blocked: string[] = []
  const privateResponses: Promise<void>[] = []
  const responseFailures: string[] = []
  let uiEstimate: { payload: Record<string, unknown>; status: number; data: Estimate } | undefined
  const targetOrigin = new URL(baseURL!).origin
  await context.route('**/api/v1/**', async route => {
    const req = route.request()
    const url = new URL(req.url())
    const path = url.pathname
    const allowed = url.origin === targetOrigin && (['GET', 'HEAD'].includes(req.method())
      || (req.method() === 'POST' && ['/api/v1/auth/login', '/api/v1/auth/refresh', '/api/v1/client/tasks/estimate'].includes(path)))
    if (allowed) return route.continue()
    blocked.push(`${req.method()} ${path}`)
    return route.abort('blockedbyclient')
  })
  page.on('response', response => {
    if (!new URL(response.url()).pathname.startsWith('/api/v1/client/')) return
    privateResponses.push((async () => {
      if (!(response.headers()['content-type'] || '').includes('application/json')) return
      const data: unknown = await response.json()
      if (hasInternalFields(data)) responseFailures.push('Client response contains internal execution fields')
      if (new URL(response.url()).pathname === '/api/v1/client/tasks/estimate' && response.request().method() === 'POST') {
        uiEstimate = { payload: response.request().postDataJSON() as Record<string, unknown>, status: response.status(), data: data as Estimate }
      }
    })().catch(() => {
      // Collect failures immediately: event callbacks must not leave rejected
      // promises unhandled, and diagnostics must not retain response bodies.
      responseFailures.push('Client response could not be inspected')
    }))
  })
  await loginWithoutCredentialArtifacts(page)
  const before = await accounting(page)
  const catalog = await readClient<Catalog>(page, 'create/catalog')
  for (const inputType of ['text', 'first_last', 'multi_image']) {
    expect(catalog.modes.some(mode => mode.inputType === inputType), `Catalog retains the ${inputType} mode`).toBe(true)
  }
  const mode = catalog.modes.find(item => item.inputType === 'text')
  const model = catalog.models.find(item => item.inputTypes.includes('text'))
  expect(Boolean(mode), 'The public text-mode entry can be inspected').toBe(true)
  if (profile === 'ready') {
    expect(Boolean(mode?.enabled && model), 'Ready requires a usable explicit model, not an empty catalog').toBe(true)
  }
  await page.goto('/app/create', { waitUntil: 'networkidle' })
  await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toBeVisible()
  await page.getByRole('textbox', { name: '画面描述', exact: true }).fill('无扣费验收：清晨的海岸，镜头平稳推进。')
  let uiReady = false
  if (profile === 'ready') {
    await page.getByRole('button', { name: mode!.name, exact: true }).click()
    await page.getByRole('combobox', { name: '承运商', exact: true }).selectOption(model!.carrierId)
    await page.getByRole('combobox', { name: '模型', exact: true }).selectOption(model!.id)
    await page.getByRole('combobox', { name: '视频时长', exact: true }).selectOption('5')
    await page.getByRole('combobox', { name: '输出分辨率', exact: true }).selectOption('768P')
    await page.getByRole('combobox', { name: '画幅比例', exact: true }).selectOption('16:9')
    const audio = page.getByRole('combobox', { name: '音频设置', exact: true })
    if (await audio.isEnabled()) await audio.selectOption('none')
    await expect(audio).toHaveValue('none')
    await expect(page.getByRole('combobox', { name: '模型', exact: true })).toHaveValue(model!.id)
    await expect.poll(() => Boolean(uiEstimate && uiEstimate.status === 200 && uiEstimate.data.available === true
      && uiEstimate.payload.modeId === mode!.id && uiEstimate.payload.carrierId === model!.carrierId && uiEstimate.payload.modelId === model!.id
      && uiEstimate.payload.durationSec === 5 && uiEstimate.payload.resolution === '768P' && uiEstimate.payload.aspectRatio === '16:9'
      && uiEstimate.payload.generateAudio === false && Number.isFinite(uiEstimate.data.billing?.estimatedTotal)),
    { message: 'Ready requires the actual UI estimate for the explicitly selected parameters', timeout: 30_000 }).toBe(true)
    await expect(page.getByLabel('费用预估', { exact: true })).toContainText(`¥ ${uiEstimate!.data.billing!.estimatedTotal!.toFixed(2)}`)
    await expect(page.getByLabel('费用预估', { exact: true })).toHaveAttribute('aria-busy', 'false')
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeEnabled()
    uiReady = true
  }
  if (profile === 'unconfigured') {
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    await expect(page.getByRole('alert').first()).toBeVisible()
  }
  // The same browser whitelist covers this explicit, side-effect-free POST.
  const response = await clientFetch(page, 'tasks/estimate', {
    // An unconfigured catalog correctly filters out models with no usable
    // supply. Empty selection asks for a recommendation and must be refused;
    // never invent a model ID or relax the positive ready-profile assertion.
    modeId: mode!.id, carrierId: model?.carrierId || '', modelId: model?.id || '',
      durationSec: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false,
      sourceMeta: {},
  })
  const estimate = response.data as Estimate
  if (profile === 'ready') {
    expect.soft(response.status, 'Ready requires a positive real estimate').toBe(200)
    expect.soft(estimate.available).toBe(true)
  } else {
    // Keep collecting viewport and accounting evidence on a public-contract
    // failure. Soft assertions still fail the test; a 500 is never accepted.
    expect.soft([400, 503], 'Unconfigured must use a public availability error').toContain(response.status)
    expect.soft(['UNAVAILABLE_COMBINATION', 'SUPPLY_UNAVAILABLE', 'STORAGE_UNAVAILABLE']).toContain(estimate.code)
  }
  const widths = testInfo.project.name === 'live-mobile' ? [320, 390] : [768, 1024, 1440, 1920]
  for (const width of widths) {
    await page.setViewportSize({ width, height: 900 })
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
    await page.getByRole('button', { name: '提交生成任务', exact: true }).scrollIntoViewIfNeeded()
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeVisible()
  }
  const after = await accounting(page)
  expect(after, 'Read-only checks do not change balance, tasks or ledger').toEqual(before)
  await Promise.all(privateResponses)
  expect(responseFailures, 'Every rendered client response was inspected safely').toEqual([])
  expect(blocked, 'No unexpected business write was attempted').toEqual([])
  // Only counts and public error codes are retained, never credentials or DTOs.
  await testInfo.attach('readonly-summary.json', { contentType: 'application/json', body: JSON.stringify({
    profile, viewportWidths: widths, estimateStatus: response.status, publicCode: estimate.code,
    explicitModelAvailable: Boolean(model),
    accountingUnchanged: true, generationReady: uiReady && response.status === 200 && estimate.available === true,
  }) })
})
