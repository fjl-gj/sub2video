import { expect, test, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Frontend contracts only. All APIs and reference-video requests are mocked;
// these tests do not exercise suppliers, wallet accounting, or real delivery.
const user = { id: 'u_public_home', email: 'home@example.test', displayName: '首页测试', workspaceName: '首页测试工作室', role: 'user', balance: 256 }
const modes = [
  { name: '文生视频', inputType: 'text', id: 'mode_text' },
  { name: '多图生成视频', inputType: 'multi_image', id: 'mode_multi' },
  { name: '首尾帧生成视频', inputType: 'first_last', id: 'mode_frames' },
] as const
const options = { durations: [5, 10, 15], resolutions: ['768P', '1080P'], aspectRatios: ['16:9', '9:16'], supportsGeneratedAudio: true, minImages: 1, maxImages: 9, defaultDuration: 5, defaultResolution: '768P' }
const emptySlot = { uploaded: false, assetId: '', picking: false }
const draft = {
  version: 3, workflowInputType: 'first_last', prompt: '这是我尚未完成的山路镜头，请保留。',
  ratio: '9:16', durationSec: '10', resolution: '1080P',
  firstFrame: emptySlot, lastFrame: emptySlot, extraFrames: [], multiFrames: [emptySlot],
}

async function seedPublicHome(page: Page) {
  const fixture = await seedSurfaceData(page)
  // Set the preference explicitly for deterministic poster-first contracts.
  // The desktop automatic-playback case opts back in after this fixture.
  await page.emulateMedia({ reducedMotion: 'reduce' })
  const privateRequests: string[] = []
  const mutations: string[] = []
  const videoRequests: string[] = []
  page.on('request', (request) => {
    const path = new URL(request.url()).pathname
    if (/^\/api\/v1\/(client|admin)\//.test(path)) privateRequests.push(`${request.method()} ${path}`)
    if (request.method() !== 'GET' && request.method() !== 'HEAD' && path.startsWith('/api/v1/')) mutations.push(`${request.method()} ${path}`)
  })
  await page.route(/\.(mp4|webm)(\?|$)/, async (route) => {
    videoRequests.push(route.request().url())
    await route.fulfill({ status: 503, contentType: 'text/plain', body: 'Reference video unavailable in deterministic browser fixture.' })
  })
  await page.route('**/api/v1/auth/me', async (route) => { await route.fulfill({ json: user }) })
  await page.route('**/api/v1/auth/login', async (route) => {
    await route.fulfill({ json: { user, token: { accessToken: 'public-home-fixture-access', refreshToken: 'public-home-fixture-refresh', expiresIn: 3600, tokenType: 'Bearer' } } })
  })
  await page.route('**/api/v1/settings/public', async (route) => {
    await route.fulfill({ json: { emailVerifyEnabled: false, passwordResetEnabled: true, registrationEnabled: true } })
  })
  await page.route('**/api/v1/client/create/catalog', async (route) => {
    await route.fulfill({ json: {
      modes: modes.map((mode) => ({ ...mode, enabled: true, description: mode.name })),
      carriers: [{ id: 'car_minimax', name: 'MINMAX' }],
      models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', description: '视频模型', inputTypes: modes.map((mode) => mode.inputType) }],
    } })
  })
  await page.route('**/api/v1/client/create/options**', async (route) => { await route.fulfill({ json: options }) })
  return { ...fixture, privateRequests, mutations, videoRequests }
}

async function login(page: Page) {
  await page.getByLabel('邮箱', { exact: true }).fill(user.email)
  await page.getByLabel('密码', { exact: true }).fill('fixture-password-only')
  await page.getByRole('button', { name: '登录', exact: true }).click()
}

async function expectPosterOnly(page: Page, videoRequests: string[]) {
  const film = page.locator('.s2v-home-film')
  await expect(film).toBeVisible()
  await expect(film.getByRole('img', { name: '冰岛黑沙海岸实拍参考，浪潮沿着山脚涌向岸边', exact: true })).toBeVisible()
  await expect(film.locator('video[src], source[src]')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '播放首页参考影像', exact: true })).toBeVisible()
  expect(videoRequests).toEqual([])
}

test.describe('public cinematic homepage contracts (mocked frontend only)', () => {
  test('is public and does not read private data, estimate, or create a task', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations, videoRequests } = await seedPublicHome(page)
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: /把一个想法，\s*拍成你的下一幕。/, level: 1 })).toBeVisible()
    await expect(page.locator('a[href="/guides/coast-letter"]').first()).toBeVisible()
    for (const mode of modes) await expect(page.getByRole('link', { name: `尝试${mode.name}`, exact: true })).toHaveAttribute('href', `/app/create?mode=${mode.inputType}`)
    await expectPosterOnly(page, videoRequests)
    await expect(page.locator('.s2v-home-film')).toContainText('实拍影像参考 · 冰岛海岸 · 非生成成片')
    await expect(page.locator('.s2v-home-film').getByRole('link', { name: 'CC BY 3.0', exact: true })).toHaveAttribute('href', 'https://creativecommons.org/licenses/by/3.0/')
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(await page.evaluate(() => localStorage.getItem('sub2video.create-draft'))).toBeNull()
    expect(await page.evaluate(() => localStorage.getItem('s2v_access_token'))).toBeNull()
    expect(unmatchedRequests).toEqual([])
  })

  test('uses a safe creation destination even when the public URL contains an external redirect', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.goto('/?redirect=https%3A%2F%2Fexternal.example.test%2Faccount', { waitUntil: 'networkidle' })
    const origin = new URL(page.url()).origin
    await page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true }).click()
    await expect(page).toHaveURL((url) => url.origin === origin && url.pathname === '/auth/login' && url.searchParams.get('redirect') === '/app/create')
    expect(privateRequests).toEqual([])
    await login(page)
    await expect(page).toHaveURL((url) => url.origin === origin && url.pathname === '/app/create' && !url.searchParams.has('redirect'))
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toBeVisible()
    expect(mutations).not.toContain('POST /api/v1/client/tasks')
    expect(unmatchedRequests).toEqual([])
  })

  for (const mode of modes) {
    test(`${mode.inputType} entry survives login and changes only the selected mode of an existing draft`, async ({ page }) => {
      const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
      await page.addInitScript((saved) => { localStorage.setItem('sub2video.create-draft', JSON.stringify(saved)) }, draft)
      await page.goto('/', { waitUntil: 'networkidle' })
      expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft') || '{}'))).toEqual(draft)
      await page.getByRole('link', { name: `尝试${mode.name}`, exact: true }).click()
      const destination = `/app/create?mode=${mode.inputType}`
      await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === destination)
      expect(privateRequests).toEqual([])
      expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft') || '{}'))).toEqual(draft)
      await login(page)
      await expect(page).toHaveURL((url) => `${url.pathname}${url.search}` === destination)
      await expect(page.getByRole('button', { name: mode.name, exact: true })).toHaveAttribute('aria-pressed', 'true')
      await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue(draft.prompt)
      await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue(draft.durationSec)
      await expect(page.getByLabel('输出分辨率', { exact: true })).toHaveValue(draft.resolution)
      await expect(page.getByLabel('画幅比例', { exact: true })).toHaveValue(draft.ratio)
      expect(mutations).not.toContain('POST /api/v1/client/tasks')
      expect(mutations).not.toContain('POST /api/v1/client/uploads')
      expect(unmatchedRequests).toEqual([])
    })
  }

  test('opens the tutorial section and toggles every FAQ with the keyboard', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.goto('/', { waitUntil: 'networkidle' })
    await page.getByRole('link', { name: '浏览镜头教程', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/' && url.hash === '#guides')
    await expect(page.locator('#guides')).toBeInViewport()
    const questions = page.locator('main details')
    expect(await questions.count()).toBeGreaterThanOrEqual(3)
    await expect(questions.first().locator('summary')).toHaveText('第一次使用，需要准备什么？')
    for (const detail of await questions.all()) {
      const wasOpen = await detail.evaluate((element) => (element as HTMLDetailsElement).open)
      const summary = detail.locator('summary')
      await summary.focus()
      await expect(summary).toBeFocused()
      await summary.press('Enter')
      await expect(detail).toHaveJSProperty('open', !wasOpen)
      await summary.press('Space')
      await expect(detail).toHaveJSProperty('open', wasOpen)
    }
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('does not fetch reference video automatically with reduced motion', async ({ page }) => {
    const { unmatchedRequests, videoRequests } = await seedPublicHome(page)
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expectPosterOnly(page, videoRequests)
    expect(unmatchedRequests).toEqual([])
  })

  test('keeps mobile visitors on the poster even when motion is allowed', async ({ page }) => {
    const { unmatchedRequests, videoRequests } = await seedPublicHome(page)
    await page.setViewportSize({ width: 390, height: 844 })
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expectPosterOnly(page, videoRequests)
    expect(unmatchedRequests).toEqual([])
  })

  test('respects data saving without downloading a video', async ({ page }) => {
    const { unmatchedRequests, videoRequests } = await seedPublicHome(page)
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.addInitScript(() => { Object.defineProperty(navigator, 'connection', { value: { saveData: true }, configurable: true }) })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expectPosterOnly(page, videoRequests)
    expect(unmatchedRequests).toEqual([])
  })

  test('keeps the poster, creation, and tutorials available after an explicit video playback fails', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations, videoRequests } = await seedPublicHome(page)
    await page.goto('/', { waitUntil: 'networkidle' })
    await expectPosterOnly(page, videoRequests)
    await page.getByRole('button', { name: '播放首页参考影像', exact: true }).click()
    await expect.poll(() => videoRequests.length).toBeGreaterThan(0)
    await expect(page.getByRole('status')).toContainText('参考影像暂不可播放，创作和教程仍可使用。')
    await expect(page.locator('.s2v-home-film')).toBeVisible()
    await expect(page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true })).toBeVisible()
    await page.getByRole('link', { name: '浏览镜头教程', exact: true }).click()
    await expect(page).toHaveURL((url) => url.hash === '#guides')
    await page.locator('a[href="/guides/coast-letter"]').first().click()
    await expect(page).toHaveURL((url) => url.pathname === '/guides/coast-letter')
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('attempts desktop playback with motion allowed and retains the poster when retry also fails', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations, videoRequests } = await seedPublicHome(page)
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect.poll(() => videoRequests.length).toBeGreaterThan(0)
    await expect(page.getByRole('status')).toContainText('参考影像暂不可播放，创作和教程仍可使用。')
    const failedRequests = videoRequests.length
    await page.getByRole('button', { name: '重试首页参考影像', exact: true }).click()
    await expect.poll(() => videoRequests.length).toBeGreaterThan(failedRequests)
    await expect(page.getByRole('status')).toContainText('参考影像暂不可播放，创作和教程仍可使用。')
    await expect(page.locator('.s2v-home-film').getByRole('img', { name: '冰岛黑沙海岸实拍参考，浪潮沿着山脚涌向岸边', exact: true })).toBeVisible()
    await expect(page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true })).toBeVisible()
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('retains usable public actions and FAQ at the minimum supported width', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.setViewportSize({ width: 320, height: 844 })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page.locator('main h1')).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1)
    const controls = [
      page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true }),
      page.getByRole('button', { name: '播放首页参考影像', exact: true }),
      ...modes.map((mode) => page.getByRole('link', { name: `尝试${mode.name}`, exact: true })),
    ]
    for (const control of controls) {
      await expect(control).toBeVisible()
      const box = await control.boundingBox()
      expect(box).not.toBeNull()
      expect(box!.x).toBeGreaterThanOrEqual(0)
      expect(box!.x + box!.width).toBeLessThanOrEqual(321)
    }
    const question = page.locator('main details').first()
    await question.locator('summary').click()
    await expect(question).toHaveJSProperty('open', true)
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1)
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })
})
