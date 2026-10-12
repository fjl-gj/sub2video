import { randomUUID } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import { expect, test, type BrowserContext, type Page, type TestInfo } from '@playwright/test'
import type { CreationCatalog, CreationOptions } from '../../src/api/client'
import type { DirectorBoard } from '../../src/api/director'

const enabled = process.env.SUB2VIDEO_LIVE === '1' && process.env.SUB2VIDEO_LIVE_DIRECTOR === '1'
const isolated = process.env.SUB2VIDEO_LIVE_ISOLATED_ACCOUNT === '1'
const email = process.env.SUB2VIDEO_LIVE_EMAIL || ''
const password = process.env.SUB2VIDEO_LIVE_PASSWORD || ''
const baseURL = process.env.SUB2VIDEO_LIVE_BASE_URL || 'http://127.0.0.1:4185'

// This suite writes only to a disposable, explicitly isolated account. Real
// responses are never fulfilled or mocked. Task creation is blocked before
// transmission, and authentication is never retained in browser artifacts.
test.use({ trace: 'off', video: 'off', screenshot: 'off' })

async function readClient<T>(page: Page, path: string): Promise<T> {
  const token = await page.evaluate(() => localStorage.getItem('s2v_access_token'))
  expect(Boolean(token), 'Real login must establish a session').toBe(true)
  const response = await page.request.get(`/api/v1/${path}`, { headers: { Authorization: `Bearer ${token}` } })
  expect(response.status(), `The authenticated ${path.split('?')[0]} read must succeed`).toBe(200)
  return await response.json() as T
}

async function accounting(page: Page) {
  const wallet = await readClient<{ balance: number }>(page, 'client/wallet')
  const tasks = await readClient<{ total: number; items: Array<{ id: string }> }>(page, 'client/tasks?page=1&pageSize=100')
  const ledger = await readClient<{ total: number; items: Array<{ id: string; amount: number }> }>(page, 'client/wallet/transactions?page=1&pageSize=100')
  return { balance: wallet.balance, tasks: tasks.total, taskIDs: tasks.items.map((item) => item.id), ledger: ledger.total, entries: ledger.items.map(({ id, amount }) => ({ id, amount })) }
}

async function guardWrites(context: BrowserContext) {
  const projects = new Set<string>()
  const blocked: string[] = []
  await context.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    const method = request.method()
    const board = path.match(/^\/api\/v1\/client\/projects\/([^/]+)\/director-board$/)
    const permitted = ['GET', 'HEAD'].includes(method)
      || (method === 'POST' && ['/api/v1/auth/login', '/api/v1/auth/refresh', '/api/v1/client/projects', '/api/v1/client/uploads', '/api/v1/client/tasks/estimate'].includes(path))
      || (method === 'PUT' && !!board && projects.has(board[1]!))
    if (permitted) return route.continue()
    blocked.push(`${method} ${path}`)
    await route.abort('blockedbyclient')
  })
  return { projects, blocked }
}

async function login(page: Page) {
  expect(isolated, 'Explicit isolated-account authorization is required').toBe(true)
  expect(email.toLowerCase().endsWith('@example.test'), 'Only disposable example.test accounts may run these writes').toBe(true)
  expect(['127.0.0.1', 'localhost', '[::1]'].includes(new URL(baseURL).hostname), 'Run this suite through the isolated loopback frontend, never the shared UAT').toBe(true)
  await page.goto('/auth/login', { waitUntil: 'domcontentloaded' })
  await page.getByLabel('邮箱', { exact: true }).fill(email)
  await page.getByLabel('密码', { exact: true }).fill(password)
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(/\/app\/overview(?:$|\?)/, { timeout: 30_000 })
  const account = await readClient<{ email: string }>(page, 'auth/me')
  expect(account.email === email, 'The authenticated owner must be the explicitly isolated account').toBe(true)
}

async function createProjectAndShot(page: Page, allowedProjects: Set<string>, label: string) {
  const name = `导演台浏览器验收-${label}-${randomUUID().slice(0, 8)}`
  await page.goto('/app/assets', { waitUntil: 'networkidle' })
  await page.getByRole('button', { name: '新建作品', exact: true }).first().click()
  const dialog = page.getByRole('dialog', { name: '新建作品', exact: true })
  await dialog.getByLabel('作品名称', { exact: true }).fill(name)
  const created = page.waitForResponse((response) => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/client/projects')
  await dialog.getByRole('button', { name: '创建作品', exact: true }).click()
  const response = await created
  expect(response.status()).toBe(201)
  const { project } = await response.json() as { project: { id: string } }
  allowedProjects.add(project.id)
  await expect(page).toHaveURL((url) => url.pathname === '/app/assets' && url.searchParams.get('project') === project.id)
  await page.getByRole('button', { name: '导演台', exact: true }).click()
  const directorPath = `/app/projects/${project.id}/director`
  await expect(page).toHaveURL(directorPath)
  await expect(page.getByRole('heading', { name: '从第一镜开始组织故事', exact: true })).toBeVisible()
  await page.getByRole('button', { name: '新增镜头', exact: true }).click()
  await page.getByLabel('镜头名称', { exact: true }).fill('海岸开场')
  await page.getByLabel('画面描述', { exact: true }).fill('清晨，山路通向海岸，镜头缓慢推进。')
  await page.getByRole('button', { name: '保存镜头板', exact: true }).click()
  await expect(page.getByText('镜头板已保存', { exact: true })).toBeVisible()
  await page.reload({ waitUntil: 'networkidle' })
  await expect(page.getByLabel('镜头名称', { exact: true })).toHaveValue('海岸开场')
  await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue('清晨，山路通向海岸，镜头缓慢推进。')
  const board = await readClient<DirectorBoard>(page, `client/projects/${project.id}/director-board`)
  expect(board.shots).toHaveLength(1)
  return { projectId: project.id, directorPath, shotId: board.shots[0]!.id, revision: board.revision }
}

async function inspectLayout(page: Page, testInfo: TestInfo, surface: string) {
  for (const width of (testInfo.project.name === 'live-mobile' ? [320, 390] : [768, 1024, 1440, 1920])) {
    await page.setViewportSize({ width, height: 900 })
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), { message: `${surface} must fit ${width}px without horizontal overflow` }).toBe(true)
    const control = page.getByRole('button', { name: surface === 'director' ? '前往创作' : '保存到镜头', exact: true })
    await control.scrollIntoViewIfNeeded()
    await expect(control).toBeVisible()
    const box = await control.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.x).toBeGreaterThanOrEqual(0)
    expect(box!.x + box!.width).toBeLessThanOrEqual(width + 1)
    expect(box!.height).toBeGreaterThanOrEqual(40)
    const mobileNav = page.getByRole('navigation', { name: '移动导航', exact: true })
    if (await mobileNav.isVisible()) {
      const navBox = await mobileNav.boundingBox()
      expect(box!.y + box!.height, 'The primary workflow control must not be hidden behind mobile navigation').toBeLessThanOrEqual(navBox!.y)
    }
    // Intentional screenshots contain only the workspace after login. Trace,
    // request headers, response bodies and login-page images remain disabled.
    await page.screenshot({ path: testInfo.outputPath(`${surface}-${width}.png`), fullPage: true, animations: 'disabled' })
  }
}

async function chooseImage(page: Page, slot: string, name: string, buffer: Buffer) {
  await page.getByRole('button', { name: slot, exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '添加或选取素材', exact: true })
  await dialog.getByLabel('参考图片文件', { exact: true }).setInputFiles({ name, mimeType: 'image/jpeg', buffer })
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('button', { name: slot, exact: true }).getByRole('img', { name, exact: true })).toBeVisible()
}

test.describe('isolated real director workspace without generation', () => {
  test.skip(!enabled || !isolated || !email || !password, 'Set SUB2VIDEO_LIVE=1, SUB2VIDEO_LIVE_DIRECTOR=1, SUB2VIDEO_LIVE_ISOLATED_ACCOUNT=1 and isolated credentials')

  test('persists real shot material and parameters despite unavailable delivery storage, then restores them from the server', async ({ page, context }, testInfo) => {
    test.setTimeout(180_000)
    const guard = await guardWrites(context)
    await login(page)
    const before = await accounting(page)
    const { projectId, directorPath, shotId, revision } = await createProjectAndShot(page, guard.projects, testInfo.project.name)
    try {
      await inspectLayout(page, testInfo, 'director')
      await page.getByRole('button', { name: '前往创作', exact: true }).click()
      await expect(page).toHaveURL((url) => url.pathname === '/app/create' && url.searchParams.get('directorShot') === shotId)
      await expect(page.getByRole('button', { name: '保存到镜头', exact: true })).toBeDisabled()
      await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
      await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue('清晨，山路通向海岸，镜头缓慢推进。')

      const catalog = await readClient<CreationCatalog>(page, 'client/create/catalog')
      const mode = catalog.modes.find((item) => item.enabled && item.inputType === 'first_last')
      const model = catalog.models.find((item) => item.inputTypes.includes('first_last'))
      expect(Boolean(mode && model), 'The isolated catalog must provide a selectable first/last-frame model').toBe(true)
      const options = await readClient<CreationOptions>(page, `client/create/options?modeId=${encodeURIComponent(mode!.id)}&modelId=${encodeURIComponent(model!.id)}`)
      expect(options.durations.length).toBeGreaterThan(0)
      expect(options.resolutions.length).toBeGreaterThan(0)
      expect(options.aspectRatios.length).toBeGreaterThan(0)
      const durationSec = options.durations.includes(5) ? 5 : options.durations[0]!
      const resolution = options.resolutions.includes('768P') ? '768P' : options.resolutions[0]!
      const aspectRatio = options.aspectRatios.includes('16:9') ? '16:9' : options.aspectRatios[0]!
      await page.getByRole('button', { name: mode!.name, exact: true }).click()
      await page.getByLabel('承运商', { exact: true }).selectOption(model!.carrierId)
      await page.getByLabel('模型', { exact: true }).selectOption(model!.id)
      await page.waitForLoadState('networkidle')
      await page.getByLabel('视频时长', { exact: true }).selectOption(String(durationSec))
      await page.getByLabel('输出分辨率', { exact: true }).selectOption(resolution)
      await page.getByLabel('画幅比例', { exact: true }).selectOption(aspectRatio)
      await expect(page.getByLabel('音频设置', { exact: true })).toHaveValue('none')
      const prompt = '首尾帧衔接海岸晨光，仅保存镜头素材，不提交生成。'
      await page.getByRole('textbox', { name: '画面描述', exact: true }).fill(prompt)
      const source = await readFile(new URL('../../public/brand/auth/iceland-coast.jpg', import.meta.url))
      await chooseImage(page, '选择首帧', 'director-first.jpg', source)
      const estimateResponse = page.waitForResponse((response) => {
        if (response.request().method() !== 'POST' || new URL(response.url()).pathname !== '/api/v1/client/tasks/estimate') return false
        const payload = response.request().postDataJSON() as { sourceMeta?: { first?: unknown; last?: unknown } }
        return !!payload.sourceMeta?.first && !!payload.sourceMeta?.last
      })
      await chooseImage(page, '选择尾帧', 'director-last.jpg', source)
      const unavailable = await estimateResponse
      expect(unavailable.status()).toBe(503)
      expect((await unavailable.json() as { code: string }).code).toBe('STORAGE_UNAVAILABLE')
      await expect(page.getByText('素材与成片存储暂不可用，请稍后重试。本次未提交任务。', { exact: true })).toBeVisible()
      await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
      await expect(page.getByRole('button', { name: '保存到镜头', exact: true })).toBeEnabled()

      const saved = page.waitForResponse((response) => response.request().method() === 'PUT' && new URL(response.url()).pathname === `/api/v1/client/projects/${projectId}/director-board`)
      await page.getByRole('button', { name: '保存到镜头', exact: true }).click()
      const savedResponse = await saved
      expect(savedResponse.status()).toBe(200)
      await expect(page.getByText('镜头素材与参数已保存，未创建任务或扣费。', { exact: true })).toBeVisible()
      const board = await readClient<DirectorBoard>(page, `client/projects/${projectId}/director-board`)
      expect(board.revision).toBe(revision + 1)
      expect(board.shots[0]).toMatchObject({ id: shotId, title: '海岸开场', prompt, modeId: mode!.id, carrierId: model!.carrierId, modelId: model!.id, durationSec, resolution, aspectRatio, generateAudio: false, candidateTaskIds: [], selectedTaskId: '' })
      expect(board.shots[0]!.sourceMeta.first).toMatchObject({ type: 'upload', id: expect.stringMatching(/^upl_/) })
      expect(board.shots[0]!.sourceMeta.last).toMatchObject({ type: 'upload', id: expect.stringMatching(/^upl_/) })
      expect(JSON.stringify(board)).not.toMatch(/supplier|workflowCode|apiModelCode|executionType|access_token|Bearer |https?:/i)
      await inspectLayout(page, testInfo, 'create')
      await page.getByRole('link', { name: '返回导演台', exact: true }).click()
      await expect(page).toHaveURL((url) => url.pathname === directorPath)
      await page.reload({ waitUntil: 'networkidle' })
      await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue(prompt)
      await page.getByRole('button', { name: '前往创作', exact: true }).click()
      // Explicitly choose the persisted board, not the browser-local draft.
      page.once('dialog', (dialog) => dialog.dismiss())
      await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
      await expect(page.getByRole('button', { name: mode!.name, exact: true })).toHaveAttribute('aria-pressed', 'true')
      await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue(prompt)
      await expect(page.getByLabel('模型', { exact: true })).toHaveValue(model!.id)
      await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue(String(durationSec))
      await expect(page.getByLabel('输出分辨率', { exact: true })).toHaveValue(resolution)
      await expect(page.getByLabel('画幅比例', { exact: true })).toHaveValue(aspectRatio)
      for (const slot of ['选择首帧', '选择尾帧']) {
        const image = page.getByRole('button', { name: slot, exact: true }).getByRole('img', { name: '已保存参考图片', exact: true })
        await expect(image).toBeVisible()
        await expect.poll(() => image.evaluate((item: HTMLImageElement) => item.complete && item.naturalWidth > 0), { message: 'The real persisted upload must remain readable after reapplying the server board' }).toBe(true)
      }
      await testInfo.attach('director-real-persistence.json', { body: JSON.stringify({ projectId, shotId, revision: board.revision, sourceIds: [board.shots[0]!.sourceMeta.first!.id, board.shots[0]!.sourceMeta.last!.id], estimate: 'STORAGE_UNAVAILABLE', generationSubmitted: false, accountMutationScope: 'isolated example.test account only' }, null, 2), contentType: 'application/json' })
    } finally {
      expect(await accounting(page), 'Editing and real image upload must leave wallet, tasks and ledger unchanged').toEqual(before)
      expect(guard.blocked, 'No browser action may attempt paid generation or an out-of-scope mutation').toEqual([])
    }
  })

  test('rejects a stale real board revision without losing edits or replacing the other page save', async ({ page, context }, testInfo) => {
    test.setTimeout(120_000)
    const guard = await guardWrites(context)
    await login(page)
    const before = await accounting(page)
    const { projectId, directorPath } = await createProjectAndShot(page, guard.projects, `conflict-${testInfo.project.name}`)
    const second = await context.newPage()
    try {
      await second.goto(directorPath, { waitUntil: 'networkidle' })
      await expect(second.getByLabel('镜头名称', { exact: true })).toHaveValue('海岸开场')
      await page.getByLabel('画面描述', { exact: true }).fill('第一个页面保存的版本。')
      await page.getByRole('button', { name: '保存镜头板', exact: true }).click()
      await expect(page.getByText('镜头板已保存', { exact: true })).toBeVisible()
      await second.getByLabel('画面描述', { exact: true }).fill('第二个页面的本地编辑必须保留。')
      const rejected = second.waitForResponse((response) => response.request().method() === 'PUT' && new URL(response.url()).pathname === `/api/v1/client/projects/${projectId}/director-board`)
      await second.getByRole('button', { name: '保存镜头板', exact: true }).click()
      const response = await rejected
      expect(response.status()).toBe(409)
      expect((await response.json() as { code: string }).code).toBe('DIRECTOR_BOARD_CONFLICT')
      await expect(second.getByRole('alert')).toContainText('你的编辑仍保留')
      await expect(second.getByLabel('画面描述', { exact: true })).toHaveValue('第二个页面的本地编辑必须保留。')
      second.once('dialog', (dialog) => dialog.dismiss())
      await second.getByRole('button', { name: '重新加载镜头板', exact: true }).click()
      await expect(second.getByLabel('画面描述', { exact: true })).toHaveValue('第二个页面的本地编辑必须保留。')
      const board = await readClient<DirectorBoard>(page, `client/projects/${projectId}/director-board`)
      expect(board.shots[0]!.prompt).toBe('第一个页面保存的版本。')
      second.once('dialog', (dialog) => dialog.accept())
      await second.getByRole('button', { name: '重新加载镜头板', exact: true }).click()
      await expect(second.getByLabel('画面描述', { exact: true })).toHaveValue('第一个页面保存的版本。')
      await testInfo.attach('director-real-conflict.json', { body: JSON.stringify({ projectId, revision: board.revision, rejectedStatus: response.status(), localEditsRetained: true, explicitReloadConfirmed: true, generationSubmitted: false }, null, 2), contentType: 'application/json' })
    } finally {
      await second.close()
      expect(await accounting(page), 'Conflicting edits must not create tasks or wallet mutations').toEqual(before)
      expect(guard.blocked).toEqual([])
    }
  })
})
