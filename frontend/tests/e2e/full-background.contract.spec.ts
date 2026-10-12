import { expect, test, type Locator, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Frontend layout and interaction contracts only. Public settings and failed
// authentication are fixtures; no account, task, wallet or supplier is changed.
const widths = [320, 390, 768, 1024, 1440, 1920, 2560]
const backgroundFilmRoute = /\/brand\/auth\/[^/?]+\.(?:webm|mp4)(?:\?.*)?$/
const entrances = [
  { path: '/', name: 'homepage', background: '.s2v-home-film', poster: '.s2v-home-film-poster', posterUrl: '/brand/auth/creator-studio-poster.jpg', filmUrl: '/brand/auth/creator-studio-loop.mp4', header: '.s2v-public-header' },
  { path: '/auth/login', name: 'login', background: '.s2v-auth-backdrop', poster: '.s2v-auth-poster', posterUrl: '/brand/auth/creator-studio-poster.jpg', filmUrl: '/brand/auth/creator-studio-loop.mp4', header: '.s2v-auth-header' },
  { path: '/auth/register', name: 'register', background: '.s2v-auth-backdrop', poster: '.s2v-auth-poster', posterUrl: '/brand/auth/creator-studio-poster.jpg', filmUrl: '/brand/auth/creator-studio-loop.mp4', header: '.s2v-auth-header' },
] as const
type Entrance = typeof entrances[number]

async function seedBackground(page: Page, motion: 'reduce' | 'no-preference' = 'reduce') {
  const fixture = await seedSurfaceData(page)
  await page.emulateMedia({ reducedMotion: motion })
  const unmatched: string[] = []
  const videoRequests: string[] = []
  const submissions: string[] = []
  page.on('request', (request) => {
    if (/\.(webm|mp4)(\?|$)/.test(request.url())) videoRequests.push(request.url())
  })
  await page.route(backgroundFilmRoute, (route) => route.abort())
  // Catch every API call so even an accidental new request cannot reach UAT.
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (request.method() === 'GET' && path === '/api/v1/settings/public') {
      return route.fulfill({ json: { registrationEnabled: true, emailVerifyEnabled: true, passwordResetEnabled: true } })
    }
    if (request.method() === 'POST' && ['/api/v1/auth/login', '/api/v1/auth/register'].includes(path)) {
      submissions.push(path)
      return route.fulfill({ status: 503, json: {} })
    }
    unmatched.push(`${request.method()} ${path}`)
    await route.fulfill({ status: 503, json: { message: 'Unexpected frontend fixture request' } })
  })
  return { fixture, unmatched, videoRequests, submissions }
}

async function box(locator: Locator) {
  await expect(locator).toBeVisible()
  return locator.evaluate((element) => {
    const rect = element.getBoundingClientRect()
    const style = getComputedStyle(element)
    return {
      left: rect.left, right: rect.right, top: rect.top, bottom: rect.bottom,
      width: rect.width, height: rect.height, objectFit: style.objectFit,
      radii: [style.borderTopLeftRadius, style.borderTopRightRadius, style.borderBottomLeftRadius, style.borderBottomRightRadius].map(Number.parseFloat),
      borders: [style.borderTopWidth, style.borderRightWidth, style.borderBottomWidth, style.borderLeftWidth].map(Number.parseFloat),
      margins: [style.marginTop, style.marginRight, style.marginBottom, style.marginLeft].map(Number.parseFloat),
      maxWidth: style.maxWidth,
    }
  })
}

async function expectFullBackground(page: Page, entrance: Entrance, withPoster = true) {
  // Font-display swap must settle before we judge the final control geometry.
  // This does not relax overflow tolerance or conceal a stable narrow-screen bug.
  await page.evaluate(async () => { await document.fonts.ready })
  const viewport = await page.evaluate(() => ({
    width: document.documentElement.clientWidth, height: innerHeight,
    scrollWidth: document.documentElement.scrollWidth, dpr: devicePixelRatio,
  }))
  const background = await box(page.locator(entrance.background))
  const overflowingElements = await page.evaluate(() => Array.from(document.querySelectorAll('body *')).flatMap((element) => {
    const rect = element.getBoundingClientRect()
    if (rect.right <= document.documentElement.clientWidth + 1 && rect.left >= -1 && element.scrollWidth <= element.clientWidth + 1) return []
    const style = getComputedStyle(element)
    return [{
      tag: element.tagName, className: typeof element.className === 'string' ? element.className : '',
      left: rect.left, right: rect.right, width: rect.width, clientWidth: element.clientWidth, scrollWidth: element.scrollWidth,
      minWidth: style.minWidth, gridTemplateColumns: style.gridTemplateColumns,
      flex: style.flex, padding: style.padding, fontSize: style.fontSize,
    }]
  }))
  // Attach before any assertion so a failed layout still identifies which
  // control exceeded its container, without recording entered field values.
  await test.info().attach(`${entrance.name}-layout-readiness`, {
    body: JSON.stringify({ viewport, background, overflowingElements }, null, 2), contentType: 'application/json',
  })
  expect(Math.abs(background.left), 'Background begins at the viewport edge, outside foreground gutters.').toBeLessThanOrEqual(1)
  expect(background.right, 'Background reaches the opposite edge even at 1920/2560px.').toBeGreaterThanOrEqual(viewport.width - 1)
  expect(background.top, 'Navigation overlays the film rather than occupying a separate top strip.').toBeLessThanOrEqual(1)
  expect(background.bottom, 'The first screen is covered, without a compact mobile media header.').toBeGreaterThanOrEqual(viewport.height - 1)
  expect(background.radii, 'A viewport background must not be a rounded media card.').toEqual([0, 0, 0, 0])
  expect(background.borders).toEqual([0, 0, 0, 0])
  expect(background.margins).toEqual([0, 0, 0, 0])
  expect(background.maxWidth, 'A foreground content ceiling must not constrain background media.').toBe('none')
  expect(viewport.scrollWidth - viewport.width, 'Controls and content fit without document-wide horizontal scrolling.').toBeLessThanOrEqual(1)

  const header = await box(page.locator(entrance.header))
  if (header.bottom > 0 && header.top < viewport.height) {
    expect(Math.max(0, header.top)).toBeGreaterThanOrEqual(background.top - 1)
    expect(Math.min(viewport.height, header.bottom)).toBeLessThanOrEqual(background.bottom + 1)
  }
  // Local translucent scrims are allowed. An opaque header/form-side panel
  // would hide the continuous background despite correct media geometry.
  const foregrounds = entrance.name === 'homepage' ? page.locator(entrance.header) : page.locator(`${entrance.header}, .s2v-auth-panel`)
  const opaquePanels = await foregrounds.evaluateAll((elements) => elements.filter((element) => {
    const color = getComputedStyle(element).backgroundColor
    if (color === 'transparent') return false
    if (color.startsWith('rgba(')) return Number.parseFloat(color.split(',').at(-1) || '') >= 1
    return color.startsWith('rgb(')
  }).map((element) => element.className))
  expect(opaquePanels, 'The shared film remains beneath navigation and the form.').toEqual([])

  let poster: Awaited<ReturnType<typeof box>> | undefined
  if (withPoster) {
    const image = page.locator(entrance.poster)
    await expect(image).toHaveAttribute('src', entrance.posterUrl)
    await expect.poll(() => image.evaluate((element: HTMLImageElement) => element.complete && element.naturalWidth > 0)).toBe(true)
    poster = await box(image)
    expect(Math.abs(poster.left - background.left)).toBeLessThanOrEqual(1)
    expect(Math.abs(poster.right - background.right)).toBeLessThanOrEqual(1)
    expect(Math.abs(poster.top - background.top)).toBeLessThanOrEqual(1)
    expect(Math.abs(poster.bottom - background.bottom)).toBeLessThanOrEqual(1)
    expect(poster.objectFit, 'Cover crops proportionally instead of stretching or introducing side bands.').toBe('cover')
    expect(poster.radii).toEqual([0, 0, 0, 0])
    expect(poster.borders).toEqual([0, 0, 0, 0])
  }
  if (entrance.name !== 'homepage') {
    const form = await box(page.locator('.s2v-auth-form'))
    expect(form.left).toBeGreaterThanOrEqual(background.left - 1)
    expect(form.right).toBeLessThanOrEqual(background.right + 1)
    const scrolling = await page.locator('.s2v-auth-form').evaluate((element) => ({
      overflowY: getComputedStyle(element).overflowY, clientHeight: element.clientHeight, scrollHeight: element.scrollHeight,
    }))
    expect(scrolling.scrollHeight > scrolling.clientHeight + 1 && ['auto', 'scroll', 'hidden'].includes(scrolling.overflowY),
      'Tall registration uses page scrolling rather than trapping fields in an inner form scroller.').toBe(false)
  }
  await test.info().attach(`${entrance.name}-background-geometry`, {
    body: JSON.stringify({ viewport, background, poster, header }, null, 2), contentType: 'application/json',
  })
}

async function expectReachable(control: Locator) {
  await control.scrollIntoViewIfNeeded()
  await expect(control).toBeInViewport()
  const hit = await control.evaluate((element) => {
    const rect = element.getBoundingClientRect()
    const target = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2)
    return { clickable: target === element || element.contains(target), top: rect.top, bottom: rect.bottom, viewportHeight: innerHeight }
  })
  expect(hit.clickable, 'The background or another foreground element must not intercept the operation.').toBe(true)
  expect(hit.top).toBeGreaterThanOrEqual(-1)
  expect(hit.bottom).toBeLessThanOrEqual(hit.viewportHeight + 1)
}

async function fillAuthentication(page: Page, register: boolean) {
  if (register) await page.getByLabel('昵称', { exact: true }).fill('布局测试创作者')
  await page.getByLabel('邮箱', { exact: true }).fill('layout@example.test')
  await page.getByLabel('密码', { exact: true }).fill('layout-fixture-password')
  if (register) {
    await page.getByLabel('邮箱验证码', { exact: true }).fill('123456')
    await page.getByRole('checkbox', { name: '同意服务条款与隐私政策' }).check()
  }
}

for (const width of widths) {
  for (const entrance of entrances) {
    test(`${entrance.name} keeps a continuous full background at ${width}px`, async ({ page }) => {
      const fixture = await seedBackground(page)
      await page.setViewportSize({ width, height: width <= 390 ? 844 : 900 })
      await page.goto(entrance.path, { waitUntil: 'networkidle' })
      await expectFullBackground(page, entrance)
      if (entrance.name === 'homepage') {
        await expectReachable(page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true }))
      } else {
        await expect(page.getByLabel('邮箱', { exact: true })).toBeEditable()
        await expectReachable(page.getByRole('button', { name: '显示密码', exact: true }))
      }
      expect(fixture.unmatched).toEqual([])
      expect(fixture.fixture.unmatchedRequests).toEqual([])
      expect(fixture.submissions).toEqual([])
    })
  }
}

for (const entrance of entrances) {
  test(`${entrance.name} fits the 305px usable width of a 320px native-scrollbar viewport`, async ({ page }) => {
    const fixture = await seedBackground(page)
    // Headless Chromium uses overlay scrollbars. Exercise the 305px usable
    // width measured in the real 320px IAB viewport; native scrolling itself
    // is verified separately in that browser, not simulated by this fixture.
    await page.setViewportSize({ width: 305, height: 480 })
    await page.goto(entrance.path, { waitUntil: 'networkidle' })
    await expectFullBackground(page, entrance)
    const control = entrance.name === 'homepage'
      ? page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true })
      : page.getByRole('button', { name: '显示密码', exact: true })
    await expectReachable(control)
    expect(fixture.unmatched).toEqual([])
    expect(fixture.submissions).toEqual([])
  })
}

for (const viewport of [{ width: 320, height: 480 }, { width: 390, height: 667 }, { width: 1280, height: 480 }]) {
  for (const entrance of entrances.slice(1)) {
    test(`${entrance.name} short screen ${viewport.width}x${viewport.height} reaches the final operation`, async ({ page }) => {
      const fixture = await seedBackground(page)
      await page.setViewportSize(viewport)
      await page.goto(entrance.path, { waitUntil: 'networkidle' })
      const register = entrance.name === 'register'
      await fillAuthentication(page, register)
      const submit = page.getByRole('button', { name: register ? '注册并进入' : '登录', exact: true })
      await expect(submit).toBeEnabled()
      await expectReachable(submit)
      await submit.click()
      await expect(page.getByRole('alert')).toContainText('暂时无法连接服务')
      await expect(page.getByLabel('邮箱', { exact: true })).toHaveValue('layout@example.test')
      await expectReachable(page.getByRole('button', { name: register ? '返回登录' : '创建新账号', exact: true }))
      await expectReachable(page.locator('.s2v-auth-footer').getByRole('link', { name: '隐私政策', exact: true }))
      await expectFullBackground(page, entrance)
      expect(fixture.submissions).toEqual([register ? '/api/v1/auth/register' : '/api/v1/auth/login'])
      expect(fixture.unmatched).toEqual([])
    })
  }
}

for (const entrance of entrances) {
  test(`${entrance.name} playing and paused film retain the poster coverage`, async ({ page }) => {
    const fixture = await seedBackground(page)
    // Decode the actual self-hosted media file, never a supplier endpoint.
    // Playback proves this background asset, not generation or delivery.
    await page.unroute(backgroundFilmRoute)
    await page.setViewportSize({ width: 1440, height: 900 })
    if (entrance.name === 'homepage') await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.goto(entrance.path, { waitUntil: 'networkidle' })
    if (entrance.name === 'homepage') {
      await expect(page.getByRole('button', { name: /(?:播放|暂停|重试)首页背景影像/ })).toHaveCount(0)
    } else await page.getByRole('button', { name: '播放背景影像', exact: true }).click()
    const video = page.locator('video')
    await expect(video).toHaveAttribute('src', entrance.filmUrl)
    await expect.poll(() => video.evaluate((element: HTMLVideoElement) => !element.paused && element.readyState >= 2)).toBe(true)
    await expect(video).toHaveCSS('opacity', '1')
    const [film, poster] = await Promise.all([box(video), box(page.locator(entrance.poster))])
    expect(film.objectFit).toBe('cover')
    for (const edge of ['left', 'right', 'top', 'bottom'] as const) {
      expect(Math.abs(film[edge] - poster[edge]), `Video and poster share the ${edge} edge.`).toBeLessThanOrEqual(1)
    }
    expect(film.radii).toEqual([0, 0, 0, 0])
    expect(film.borders).toEqual([0, 0, 0, 0])
    if (entrance.name === 'homepage') {
      // Decorative media has no user-facing playback UI. It still stops when
      // offscreen and resumes only when playback preferences allow it.
      await page.locator('.s2v-public-footer').scrollIntoViewIfNeeded()
      await expect(video).toHaveJSProperty('paused', true)
      await page.evaluate(() => scrollTo(0, 0))
      await expect(video).toHaveJSProperty('paused', false)
      await page.emulateMedia({ reducedMotion: 'reduce' })
      await expect(video).toHaveJSProperty('paused', true)
      await page.emulateMedia({ reducedMotion: 'no-preference' })
      await expect(video).toHaveJSProperty('paused', false)
      await page.emulateMedia({ reducedMotion: 'reduce' })
    } else await page.getByRole('button', { name: '暂停背景影像', exact: true }).click()
    await expect(video).toHaveJSProperty('paused', true)
    await expectFullBackground(page, entrance)
    expect(fixture.unmatched).toEqual([])
    expect(fixture.submissions).toEqual([])
  })

  test(`${entrance.name} mobile starts with a full poster when motion is allowed`, async ({ page }) => {
    const fixture = await seedBackground(page, 'no-preference')
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto(entrance.path, { waitUntil: 'networkidle' })
    await expectFullBackground(page, entrance)
    await expect(page.locator('video')).toHaveCount(0)
    if (entrance.name === 'homepage') {
      await expect(page.getByRole('button', { name: /(?:播放|暂停|重试)首页背景影像/ })).toHaveCount(0)
    }
    expect(fixture.videoRequests).toEqual([])
    expect(fixture.unmatched).toEqual([])
  })

  test(`${entrance.name} desktop reduced motion starts with its poster without loading film`, async ({ page }) => {
    const fixture = await seedBackground(page, 'reduce')
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(entrance.path, { waitUntil: 'networkidle' })
    await expectFullBackground(page, entrance)
    await expect(page.locator('video')).toHaveCount(0)
    if (entrance.name === 'homepage') {
      await expect(page.getByRole('button', { name: /(?:播放|暂停|重试)首页背景影像/ })).toHaveCount(0)
    }
    expect(fixture.videoRequests).toEqual([])
    expect(fixture.unmatched).toEqual([])
    expect(fixture.submissions).toEqual([])
  })
}

for (const entrance of entrances) {
  test(`${entrance.name} missing media keeps the full background and real controls usable`, async ({ page }) => {
    // Homepage playback starts automatically only on desktop. Authentication
    // retains its separate opt-in media controls and narrow-screen exercise.
    const fixture = await seedBackground(page, entrance.name === 'homepage' ? 'no-preference' : 'reduce')
    await page.setViewportSize(entrance.name === 'homepage' ? { width: 1440, height: 900 } : { width: 390, height: 667 })
    await page.route('**/brand/auth/*.jpg', (route) => route.abort())
    await page.goto(entrance.path, { waitUntil: 'networkidle' })
    await expect(page.locator(entrance.poster)).toHaveCount(0)
    if (entrance.name === 'homepage') {
      await expect(page.getByRole('button', { name: /(?:播放|暂停|重试)首页背景影像/ })).toHaveCount(0)
    } else {
      const playback = page.getByRole('button', { name: '播放背景影像', exact: true })
      await expectReachable(playback)
      await playback.click()
    }
    await expect.poll(() => fixture.videoRequests.length).toBeGreaterThan(0)
    await expect(page.locator('video')).toHaveCount(0)
    if (entrance.name === 'homepage') {
      const start = page.locator('.s2v-home-hero').getByRole('link', { name: '开始创作', exact: true })
      await expectReachable(start)
      await expect(start).toHaveAttribute('href', '/app/create')
      const tutorial = page.locator('.s2v-public-footer').getByRole('link', { name: '镜头教程', exact: true })
      await expect(tutorial).toHaveAttribute('href', '/guides/coast-letter')
      await expectReachable(tutorial)
      await tutorial.click({ trial: true })
      await page.evaluate(() => scrollTo(0, 0))
    } else {
      await fillAuthentication(page, entrance.name === 'register')
      const submit = page.getByRole('button', { name: entrance.name === 'register' ? '注册并进入' : '登录', exact: true })
      await expectReachable(submit)
      await submit.click()
      await expect(page.getByRole('alert')).toContainText('暂时无法连接服务')
      expect(fixture.submissions).toHaveLength(1)
    }
    await expectFullBackground(page, entrance, false)
    expect(fixture.unmatched).toEqual([])
  })
}
