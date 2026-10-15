import { expect, test, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Frontend contracts only. All APIs and background-video requests are mocked;
// these tests do not exercise suppliers, wallet accounting, or real delivery.
const homePosterAlt = '动漫创作者在影像工作室调节镜头，镜头中映出山水风景'
const homePosterUrl = '/brand/auth/creator-studio-poster.jpg'
const homeFilmUrl = '/brand/auth/creator-studio-loop.mp4'
const user = { id: 'u_public_home', email: 'home@example.test', displayName: '首页测试', workspaceName: '首页测试工作室', role: 'user', balance: 256 }
const modes = [
  { name: '文生视频', inputType: 'text', id: 'mode_text', hint: '只需提示词' },
  { name: '首尾帧', inputType: 'first_last', id: 'mode_frames', hint: '提示词 + 首帧 + 尾帧' },
  { name: '多图生成', inputType: 'multi_image', id: 'mode_multi', hint: '提示词 + 至少一张参考图' },
] as const
type HomeMode = typeof modes[number]
const homeHeadline = '一句话，一个镜头。'
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
    await route.fulfill({ status: 503, contentType: 'text/plain', body: 'Background video unavailable in deterministic browser fixture.' })
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
      modes: modes.map(({ name, inputType, id }) => ({ name, inputType, id, enabled: true, description: name })),
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

async function expectModeRows(page: Page) {
  const section = page.locator('#modes')
  await expect(section.getByRole('heading', { name: '三种创作方式', level: 2 })).toBeVisible()
  for (const mode of modes) {
    const enter = section.getByRole('link', { name: `用${mode.name}开始创作`, exact: true })
    await expect(enter).toBeVisible()
    await expect(enter).toContainText(mode.name)
    await expect(enter).toContainText(mode.hint)
    await expect(enter).toHaveAttribute('href', `/app/create?mode=${mode.inputType}`)
  }
  await expect(section.getByRole('textbox')).toHaveCount(0)
  await expect(section.locator('input[type="file"], video')).toHaveCount(0)
  await expect(page.getByRole('tablist')).toHaveCount(0)
}

async function expectPosterOnly(page: Page, videoRequests: string[]) {
  const film = page.locator('.s2v-home-film')
  await expect(film).toBeVisible()
  const poster = film.getByRole('img', { name: homePosterAlt, exact: true })
  await expect(poster).toBeVisible()
  await expect(poster).toHaveAttribute('src', homePosterUrl)
  await expect(film.locator('video[src], source[src]')).toHaveCount(0)
  await expect(page.getByRole('button', { name: /(?:播放|暂停|重试)首页背景影像/ })).toHaveCount(0)
  expect(videoRequests).toEqual([])
}

test.describe('public cinematic homepage contracts (mocked frontend only)', () => {
  for (const width of [320, 390, 768, 1024, 1440, 1920, 2560]) {
    test(`fills the first-screen background and preserves content gutters at ${width}px`, async ({ page }, testInfo) => {
      test.skip(testInfo.project.name !== 'desktop', 'The seven-width matrix runs once; mobile interaction contracts still run separately.')
      const height = width <= 390 ? 844 : 900
      await page.setViewportSize({ width, height })
      const { unmatchedRequests, privateRequests, mutations, videoRequests } = await seedPublicHome(page)
      await page.goto('/', { waitUntil: 'networkidle' })
      await page.evaluate(() => document.fonts.ready)
      const geometry = await page.evaluate(() => {
        const hero = document.querySelector<HTMLElement>('.s2v-home-hero')!
        const poster = document.querySelector<HTMLImageElement>('.s2v-home-film-poster')!
        const header = document.querySelector<HTMLElement>('.s2v-public-header')!
        const title = document.querySelector<HTMLElement>('.s2v-home-hero h1')!
        const action = document.querySelector<HTMLElement>('.s2v-home-primary-link')!
        const modes = document.querySelector<HTMLElement>('#modes')!
        const footer = document.querySelector<HTMLElement>('.s2v-public-footer')!
        const rect = (element: HTMLElement) => {
          const box = element.getBoundingClientRect()
          return { left: box.left, right: box.right, top: box.top, bottom: box.bottom, height: box.height }
        }
        const footerInner = () => {
          const box = footer.getBoundingClientRect()
          const style = getComputedStyle(footer)
          return { left: box.left + parseFloat(style.paddingLeft), right: box.right - parseFloat(style.paddingRight) }
        }
        return {
          viewport: document.documentElement.clientWidth,
          overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
          hero: rect(hero), poster: rect(poster), header: rect(header), modes: rect(modes), footer: footerInner(),
          title: rect(title), action: rect(action),
          titleAlign: getComputedStyle(title).textAlign,
          fit: getComputedStyle(poster).objectFit,
          radius: getComputedStyle(hero).borderRadius,
          headerBackground: getComputedStyle(header).backgroundColor,
        }
      })
      expect(geometry.hero.left).toBe(0)
      expect(geometry.hero.top).toBe(0)
      expect(geometry.hero.right).toBe(geometry.viewport)
      expect(geometry.hero.height).toBeGreaterThanOrEqual(height)
      expect(geometry.poster).toEqual(geometry.hero)
      expect(geometry.fit).toBe('cover')
      expect(geometry.radius).toBe('0px')
      expect(geometry.header.top).toBe(0)
      expect(geometry.header.bottom).toBeLessThan(geometry.hero.bottom)
      expect(geometry.headerBackground).toBe('rgba(0, 0, 0, 0)')
      expect(geometry.modes.left).toBeGreaterThanOrEqual(16)
      expect(Math.abs(geometry.modes.left - geometry.footer.left)).toBeLessThanOrEqual(1)
      expect(Math.abs(geometry.modes.right - geometry.footer.right)).toBeLessThanOrEqual(1)
      expect(geometry.overflow).toBeLessThanOrEqual(1)
      expect(geometry.titleAlign, 'The artistic headline is aligned to the right.').toBe('right')
      expect(Math.abs(geometry.title.right - geometry.action.right), 'Headline and primary action share the right axis.').toBeLessThanOrEqual(1)
      expect(geometry.action.top - geometry.title.bottom, 'The headline leaves room before the creation action.').toBeGreaterThanOrEqual(24)
      expect(geometry.modes.top - geometry.action.bottom, 'Mode rows sit below the creation action.').toBeGreaterThanOrEqual(12)
      expect(geometry.hero.bottom - geometry.modes.bottom).toBeGreaterThanOrEqual(width < 768 ? 20 : 24)
      await expectPosterOnly(page, videoRequests)
      await expect(page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true }).locator('svg')).toHaveCount(0)
      await page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true }).click({ trial: true })
      expect(privateRequests).toEqual([])
      expect(mutations).toEqual([])
      expect(unmatchedRequests).toEqual([])
    })
  }

  test('restores the tutorial shell after leaving the full-background homepage', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page.locator('.s2v-public-home')).toHaveCount(1)
    await page.locator('.s2v-public-footer').getByRole('link', { name: '镜头教程', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/guides/coast-letter')
    await expect(page.locator('.s2v-public-home')).toHaveCount(0)
    expect(await page.locator('.s2v-public-main').evaluate((element) => element.getBoundingClientRect().left)).toBeGreaterThanOrEqual(16)
    expect(await page.locator('.s2v-public-header').evaluate((element) => getComputedStyle(element).position)).toBe('static')
    await page.locator('.s2v-public-header').getByRole('link', { name: 'Sub2Video 产品首页', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/')
    await expect(page.locator('.s2v-public-home')).toHaveCount(1)
    expect(await page.locator('.s2v-home-hero').evaluate((element) => element.getBoundingClientRect().left)).toBe(0)
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('tutorial return and missing-tutorial recovery use live destinations instead of removed homepage sections', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.goto('/guides/coast-letter', { waitUntil: 'networkidle' })
    const back = page.getByRole('link', { name: '返回产品首页', exact: true })
    await expect(back).toHaveAttribute('href', '/')
    await back.click()
    await expect(page).toHaveURL((url) => url.pathname === '/' && url.hash === '')
    await expect(page.getByRole('heading', { name: homeHeadline, exact: true })).toBeVisible()
    await page.goto('/guides/unknown-tutorial', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: '未找到这个镜头教程', exact: true })).toBeVisible()
    await page.getByRole('link', { name: '查看入门教程', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/guides/coast-letter')
    await expect(page.getByRole('heading', { name: '海岸来信', exact: true })).toBeVisible()
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('is public and does not read private data, estimate, or create a task', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations, videoRequests } = await seedPublicHome(page)
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: homeHeadline, exact: true, level: 1 })).toBeVisible()
    const hero = page.locator('.s2v-home-hero')
    await expect(hero.getByRole('link', { name: '开始创作', exact: true })).toHaveAttribute('href', '/app/create')
    await expect(hero.getByRole('link', { name: '开始创作', exact: true }).locator('svg')).toHaveCount(0)
    await expectModeRows(page)
    await expect(page.locator('.s2v-public-footer').getByRole('link', { name: '镜头教程', exact: true })).toHaveAttribute('href', '/guides/coast-letter')
    await expect(page.locator('.s2v-public-footer').getByRole('link', { name: '帮助', exact: true })).toHaveAttribute('href', '/app/tickets')
    await expectPosterOnly(page, videoRequests)
    await expect(page.locator('.s2v-home-film')).not.toContainText('实拍影像参考 · 冰岛海岸 · 非生成成片')
    await expect(page.locator('.s2v-home-film').getByRole('link', { name: 'CC BY 3.0', exact: true })).toHaveCount(0)
    await expect(page.locator('.s2v-home-film').getByRole('link', { name: 'Alexander Grebenkov', exact: true })).toHaveCount(0)
    await expect(page.locator('main details, #guides, #process')).toHaveCount(0)
    await expect(page.getByRole('link', { name: '浏览镜头教程', exact: true })).toHaveCount(0)
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(await page.evaluate(() => localStorage.getItem('sub2video.create-draft.u_public_home'))).toBeNull()
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
      await page.addInitScript((saved) => { localStorage.setItem('sub2video.create-draft.u_public_home', JSON.stringify(saved)) }, draft)
      await page.goto('/', { waitUntil: 'networkidle' })
      expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_public_home') || '{}'))).toEqual(draft)
      await expectModeRows(page)
      await page.getByRole('link', { name: `用${mode.name}开始创作`, exact: true }).click()
      const destination = `/app/create?mode=${mode.inputType}`
      await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === destination)
      expect(privateRequests).toEqual([])
      expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_public_home') || '{}'))).toEqual(draft)
      await login(page)
      await expect(page).toHaveURL((url) => `${url.pathname}${url.search}` === destination)
      await expect(page.getByLabel('创作模式', { exact: true }).getByRole('button', { name: mode.name, exact: true })).toHaveAttribute('aria-pressed', 'true')
      await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue(draft.prompt)
      await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue(draft.durationSec)
      await expect(page.getByLabel('输出分辨率', { exact: true })).toHaveValue(draft.resolution)
      await expect(page.getByLabel('画幅比例', { exact: true })).toHaveValue(draft.ratio)
      expect(mutations).not.toContain('POST /api/v1/client/tasks')
      expect(mutations).not.toContain('POST /api/v1/client/uploads')
      expect(unmatchedRequests).toEqual([])
    })
  }

  test('mode entry links stay keyboard-focusable without reading private data or replacing drafts', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.addInitScript((saved) => { localStorage.setItem('sub2video.create-draft.u_public_home', JSON.stringify(saved)) }, draft)
    await page.goto('/', { waitUntil: 'networkidle' })
    await expectModeRows(page)
    const first = page.getByRole('link', { name: `用${modes[0].name}开始创作`, exact: true })
    await first.focus()
    await expect(first).toBeFocused()
    await first.press('Tab')
    await expect(page.getByRole('link', { name: `用${modes[1].name}开始创作`, exact: true })).toBeFocused()
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_public_home') || '{}'))).toEqual(draft)
    expect(new URL(page.url()).pathname).toBe('/')
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  for (const viewport of [
    { width: 320, height: 480 },
    { width: 360, height: 667 },
    { width: 390, height: 667 },
    { width: 1280, height: 480 },
    { width: 305, height: 480 },
  ]) {
    test(`headline and primary action fit the short ${viewport.width}x${viewport.height} first screen`, async ({ page }) => {
      const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
      // 305px is the usable content width measured in the real 320px browser
      // with native scrollbars; this checks width, not simulated scrollbars.
      await page.setViewportSize(viewport)
      await page.goto('/', { waitUntil: 'networkidle' })
      await page.evaluate(() => document.fonts.ready)
      const title = page.getByRole('heading', { name: homeHeadline, exact: true, level: 1 })
      const enter = page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true })
      for (const element of [title, enter]) {
        await expect(element).toBeVisible()
        await expect(element).toBeInViewport()
      }
      const geometry = await page.evaluate(() => {
        const rect = (selector: string) => {
          const box = document.querySelector<HTMLElement>(selector)!.getBoundingClientRect()
          return { left: box.left, right: box.right, top: box.top, bottom: box.bottom, height: box.height }
        }
        const titleEl = document.querySelector<HTMLElement>('.s2v-home-hero h1')!
        return {
          title: rect('.s2v-home-hero h1'), enter: rect('.s2v-home-primary-link'),
          header: rect('.s2v-public-header'),
          titleAlign: getComputedStyle(titleEl).textAlign,
          titleHeight: titleEl.getBoundingClientRect().height,
          titleLineHeight: Number.parseFloat(getComputedStyle(titleEl).lineHeight),
          width: document.documentElement.clientWidth, height: innerHeight,
          overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        }
      })
      await test.info().attach('short-homepage-control-geometry', {
        body: JSON.stringify(geometry, null, 2), contentType: 'application/json',
      })
      for (const box of [geometry.title, geometry.enter]) {
        expect(box.left).toBeGreaterThanOrEqual(-1)
        expect(box.right).toBeLessThanOrEqual(geometry.width + 1)
        expect(box.top).toBeGreaterThanOrEqual(0)
        expect(box.bottom).toBeLessThanOrEqual(geometry.height + 1)
      }
      const overlaps = (first: typeof geometry.title, second: typeof geometry.title) =>
        first.left < second.right && first.right > second.left && first.top < second.bottom && first.bottom > second.top
      for (const box of [geometry.title, geometry.enter]) {
        expect(overlaps(box, geometry.header), 'The entrance must remain below navigation.').toBe(false)
      }
      expect(overlaps(geometry.title, geometry.enter)).toBe(false)
      expect(geometry.titleAlign).toBe('right')
      expect(Math.abs(geometry.title.right - geometry.enter.right), 'Headline and primary action share their right edge.').toBeLessThanOrEqual(1)
      expect(geometry.enter.top - geometry.title.bottom).toBeGreaterThanOrEqual(24)
      expect(geometry.titleLineHeight).toBeGreaterThan(0)
      expect(geometry.titleHeight, 'The headline should stay on one line when space allows.').toBeLessThanOrEqual(geometry.titleLineHeight * 2 + 2)
      expect(geometry.enter.height).toBeGreaterThanOrEqual(viewport.width < 768 ? 52 : 56)
      expect(geometry.overflow).toBeLessThanOrEqual(1)
      // Trial verifies real pointer hit testing without navigating or mutating.
      await enter.click({ trial: true })
      await expect(enter.locator('svg')).toHaveCount(0)
      expect(privateRequests).toEqual([])
      expect(mutations).toEqual([])
      expect(unmatchedRequests).toEqual([])
    })
  }

  test('footer tutorial and help links follow their real destinations', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.goto('/', { waitUntil: 'networkidle' })
    const footer = page.locator('.s2v-public-footer')
    const tutorial = footer.getByRole('link', { name: '镜头教程', exact: true })
    await expect(tutorial).toHaveAttribute('href', '/guides/coast-letter')
    await expect(footer.getByRole('link', { name: '帮助', exact: true })).toHaveAttribute('href', '/app/tickets')
    await tutorial.click()
    await expect(page).toHaveURL((url) => url.pathname === '/guides/coast-letter')
    await expect(page.locator('main h1')).toContainText('海岸来信')
    await page.goto('/', { waitUntil: 'networkidle' })
    await page.locator('.s2v-public-footer').getByRole('link', { name: '帮助', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === '/app/tickets')
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('uses a loaded self-hosted artistic heading face and one reading face for public controls', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    const fontResponses: { url: string; status: number }[] = []
    page.on('response', (response) => {
      if (/\/fonts\/[^/?]+\.woff2(?:\?|$)/.test(response.url())) {
        fontResponses.push({ url: response.url(), status: response.status() })
      }
    })
    await page.goto('/', { waitUntil: 'networkidle' })
    await page.evaluate(() => document.fonts.ready)
    const typography = await page.evaluate(() => {
      const firstFamily = (element: Element) => getComputedStyle(element).fontFamily.split(',')[0]!.trim().replace(/^["']|["']$/g, '')
      const displayHeadings = Array.from(document.querySelectorAll('.s2v-home h1'))
      const reading = Array.from(document.querySelectorAll('.s2v-public-header nav a, .s2v-public-home-login, .s2v-home-primary-link, .s2v-home-modes-title, .s2v-public-footer nav a'))
      const displayFaces: { family: string; status: string; weight: string }[] = []
      document.fonts.forEach((face) => {
        if (face.family.replace(/^["']|["']$/g, '') === 'S2V Public Display') displayFaces.push({ family: face.family, status: face.status, weight: face.weight })
      })
      return {
        headings: displayHeadings.map((element) => ({ text: element.textContent, family: firstFamily(element) })),
        reading: reading.map((element) => ({ text: element.textContent, family: firstFamily(element) })),
        displayFaces,
        headlineLoaded: document.fonts.check('500 40px "S2V Public Display"', '一句话，一个镜头。'),
      }
    })
    expect(typography.headings).toHaveLength(1)
    for (const heading of typography.headings) expect(heading.family, `Artistic heading: ${heading.text}`).toBe('S2V Public Display')
    expect(typography.reading.length).toBeGreaterThanOrEqual(7)
    for (const item of typography.reading) expect(item.family, `Reading/control text: ${item.text}`).toBe('Source Han Sans SC')
    expect(typography.displayFaces.some((face) => face.status === 'loaded' && face.weight === '500')).toBe(true)
    expect(typography.headlineLoaded).toBe(true)
    const origin = new URL(page.url()).origin
    expect(fontResponses.length).toBeGreaterThan(0)
    for (const response of fontResponses) {
      expect(new URL(response.url).origin, 'Homepage fonts are delivered locally, without a third-party CDN.').toBe(origin)
      expect(response.status).toBe(200)
    }
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('does not fetch background video automatically with reduced motion', async ({ page }) => {
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

  test('keeps the poster, creation, and tutorials available after automatic desktop playback fails', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations, videoRequests } = await seedPublicHome(page)
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect.poll(() => videoRequests.length).toBeGreaterThan(0)
    expect(videoRequests.every((url) => new URL(url).pathname === homeFilmUrl)).toBe(true)
    await expect(page.locator('.s2v-home-film video')).toHaveCount(0)
    await expect(page.locator('.s2v-home-film').getByRole('img', { name: homePosterAlt, exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: /(?:播放|暂停|重试)首页背景影像/ })).toHaveCount(0)
    await expect(page.locator('.s2v-home-film')).toBeVisible()
    await expect(page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true })).toBeVisible()
    await page.locator('.s2v-public-footer').getByRole('link', { name: '镜头教程', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/guides/coast-letter')
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('failed desktop playback does not trigger additional requests when mode rows are used', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations, videoRequests } = await seedPublicHome(page)
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect.poll(() => videoRequests.length).toBeGreaterThan(0)
    await expect(page.locator('.s2v-home-film video')).toHaveCount(0)
    const failedRequests = videoRequests.length
    await expect(page.getByRole('button', { name: /(?:播放|暂停|重试)首页背景影像/ })).toHaveCount(0)
    await expectModeRows(page)
    for (const mode of modes) {
      await page.getByRole('link', { name: `用${mode.name}开始创作`, exact: true }).click({ trial: true })
    }
    expect(videoRequests).toHaveLength(failedRequests)
    expect(videoRequests.every((url) => new URL(url).pathname === homeFilmUrl)).toBe(true)
    await expect(page.locator('.s2v-home-film').getByRole('img', { name: homePosterAlt, exact: true })).toBeVisible()
    await expect(page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true })).toBeVisible()
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })

  test('retains usable public actions and all mode rows at the minimum supported width', async ({ page }) => {
    const { unmatchedRequests, privateRequests, mutations } = await seedPublicHome(page)
    await page.setViewportSize({ width: 320, height: 844 })
    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page.locator('main h1')).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1)
    const controls = [
      page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true }),
      ...modes.map((mode) => page.getByRole('link', { name: `用${mode.name}开始创作`, exact: true })),
    ]
    for (const control of controls) {
      await expect(control).toBeVisible()
      const box = await control.boundingBox()
      expect(box).not.toBeNull()
      expect(box!.x).toBeGreaterThanOrEqual(0)
      expect(box!.x + box!.width).toBeLessThanOrEqual(321)
      await control.click({ trial: true })
      expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1)
    }
    expect(privateRequests).toEqual([])
    expect(mutations).toEqual([])
    expect(unmatchedRequests).toEqual([])
  })
})
