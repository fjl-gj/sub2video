import { expect, test } from '@playwright/test'

const liveEnabled = process.env.SUB2VIDEO_LIVE === '1'
const email = process.env.SUB2VIDEO_LIVE_EMAIL || ''
const password = process.env.SUB2VIDEO_LIVE_PASSWORD || ''

// Authentication exchanges must not be saved in failure traces or videos.
// Only explicitly captured anonymous homepage screenshots are retained.
test.use({ trace: 'off', video: 'off', screenshot: 'off' })

test.describe('Sub2Video live public entrance', () => {
  test.skip(!liveEnabled, 'set SUB2VIDEO_LIVE=1 to run against a real frontend and backend')
  test.skip(!email || !password, 'set SUB2VIDEO_LIVE_EMAIL and SUB2VIDEO_LIVE_PASSWORD')

  test('continues a public shot through real login and logout without creating a task', async ({ page }, testInfo) => {
    test.setTimeout(90_000)
    const anonymousPrivateRequests: string[] = []
    const blockedWrites: string[] = []
    const videoRequests: string[] = []
    let anonymous = true
    page.on('request', (request) => {
      const path = new URL(request.url()).pathname
      if (anonymous && /^\/api\/v1\/(client|admin)\//.test(path)) anonymousPrivateRequests.push(`${request.method()} ${path}`)
      // Authenticated creation can load the account's existing delivered clips.
      // Only the entrance's decorative film is governed by this assertion.
      if (path === '/brand/auth/iceland-coast.webm') videoRequests.push(path)
    })
    // All allowed requests reach the real service. Block unexpected writes
    // before transmission so a navigation regression cannot charge or upload.
    await page.route('**/api/v1/**', async (route) => {
      const request = route.request()
      const path = new URL(request.url()).pathname
      const allowed = ['GET', 'HEAD'].includes(request.method())
        || (request.method() === 'POST' && ['/api/v1/auth/login', '/api/v1/auth/logout', '/api/v1/auth/refresh', '/api/v1/client/tasks/estimate'].includes(path))
      if (allowed) return route.continue()
      blockedWrites.push(`${request.method()} ${path}`)
      await route.abort('blockedbyclient')
    })

    await page.setViewportSize({ width: 1440, height: 900 })
    await page.emulateMedia({ reducedMotion: 'reduce' })
    const posterResponsePromise = page.waitForResponse((response) => new URL(response.url()).pathname === '/brand/auth/iceland-coast.jpg')
    await page.goto('/', { waitUntil: 'networkidle' })
    expect((await posterResponsePromise).status(), 'The self-hosted homepage poster must load.').toBe(200)
    await expect(page.getByRole('heading', { name: /把一个想法，\s*拍成你的下一幕。/, level: 1 })).toBeVisible()
    const poster = page.locator('.s2v-home-film-poster')
    await expect(poster).toBeVisible()
    expect(await poster.evaluate((element) => {
      const image = element as HTMLImageElement
      return image.complete && image.naturalWidth > 0 && new URL(image.currentSrc).origin === location.origin
    }), 'The visible poster must be a decoded image from this deployment.').toBe(true)
    expect(anonymousPrivateRequests).toEqual([])
    expect(videoRequests).toEqual([])

    await page.locator('a[href="/guides/coast-letter"]').first().click()
    await expect(page.getByRole('heading', { name: '海岸来信', level: 1, exact: true })).toBeVisible()
    const shot = page.locator('article.s2v-guide-shot').filter({ has: page.getByRole('heading', { name: '观察动作', level: 3, exact: true }) })
    await shot.getByRole('link', { name: '制作这个镜头', exact: true }).click()
    const destination = '/app/create?guide=coast-letter&shot=detail'
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === destination)
    expect(anonymousPrivateRequests).toEqual([])

    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByLabel('密码', { exact: true }).fill(password)
    const loginResponsePromise = page.waitForResponse((response) => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/auth/login')
    const catalogResponsePromise = page.waitForResponse((response) => response.request().method() === 'GET' && new URL(response.url()).pathname === '/api/v1/client/create/catalog')
    anonymous = false
    await page.getByRole('button', { name: '登录', exact: true }).click()
    expect((await loginResponsePromise).status(), 'Real login must succeed.').toBe(200)
    await expect(page).toHaveURL((url) => `${url.pathname}${url.search}` === destination, { timeout: 30_000 })
    const catalogResponse = await catalogResponsePromise
    expect(catalogResponse.status(), 'The authenticated creation catalog must load.').toBe(200)
    const catalog = await catalogResponse.json() as { modes?: Array<{ inputType?: string; enabled?: boolean }>; models?: unknown[] }
    expect(catalog.modes?.some((mode) => mode.inputType === 'text' && mode.enabled === true), 'The live text mode must be available.').toBe(true)
    expect(catalog.models?.length ?? 0, 'The live catalog must offer at least one model.').toBeGreaterThan(0)
    await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '文生视频', exact: true })).toHaveAttribute('aria-pressed', 'true')
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).not.toHaveValue('')

    // A reload verifies the stored session against the real identity endpoint.
    const identityResponsePromise = page.waitForResponse((response) => response.request().method() === 'GET' && new URL(response.url()).pathname === '/api/v1/auth/me')
    await page.reload({ waitUntil: 'networkidle' })
    expect((await identityResponsePromise).status(), 'The server must recognize the saved session.').toBe(200)
    await expect(page).toHaveURL((url) => `${url.pathname}${url.search}` === destination)
    await page.getByRole('button', { name: '用户菜单', exact: true }).click()
    const logoutResponsePromise = page.waitForResponse((response) => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/auth/logout')
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    expect((await logoutResponsePromise).ok(), 'Real server-side logout must succeed.').toBe(true)
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login')
    await expect(page.getByRole('heading', { name: '欢迎回来', exact: true })).toBeVisible()
    expect(await page.evaluate(() => !localStorage.getItem('s2v_access_token') && !localStorage.getItem('s2v_refresh_token')), 'Logout must clear local credentials.').toBe(true)

    anonymous = true
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page.getByRole('link', { name: '登录 / 开始创作', exact: true })).toBeVisible()
    await page.evaluate(() => document.fonts.ready)
    await page.screenshot({ path: testInfo.outputPath('live-public-home-desktop.png'), fullPage: true, animations: 'disabled' })
    await page.setViewportSize({ width: 390, height: 844 })
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.locator('.s2v-home-film-poster')).toBeVisible()
    await expect(page.getByRole('button', { name: '播放首页参考影像', exact: true })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1)
    expect(videoRequests, 'Mobile and reduced-motion visits must not automatically fetch video.').toEqual([])
    expect(anonymousPrivateRequests).toEqual([])
    expect(blockedWrites, 'This entrance flow must not attempt registration, task creation, uploads, or other writes.').toEqual([])
    await page.screenshot({ path: testInfo.outputPath('live-public-home-mobile.png'), fullPage: true, animations: 'disabled' })
  })
})
