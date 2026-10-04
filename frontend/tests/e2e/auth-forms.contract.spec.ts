import { expect, test, type Page } from '@playwright/test'

// UI contracts only: these fixtures do not send mail or create real accounts.
const settings = { registrationEnabled: true, emailVerifyEnabled: true, passwordResetEnabled: true }
async function seedAuth(page: Page, config = settings) {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  const unmatched: string[] = []
  await page.route('**/api/v1/**', async (route) => {
    if (new URL(route.request().url()).pathname === '/api/v1/settings/public') return route.fulfill({ json: config })
    unmatched.push(`${route.request().method()} ${route.request().url()}`)
    await route.fulfill({ status: 500, json: { message: 'Missing test fixture' } })
  })
  return unmatched
}

test('login uses real form semantics, reveals password and retries without clearing fields', async ({ page }) => {
  const unmatched = await seedAuth(page)
  const submissions: unknown[] = []
  await page.route('**/api/v1/auth/login', async (route) => {
    submissions.push(route.request().postDataJSON())
    await route.fulfill({ status: submissions.length === 1 ? 401 : 503, json: {} })
  })
  await page.goto('/auth/login')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  expect(submissions).toHaveLength(0)
  const email = page.getByLabel('邮箱', { exact: true })
  const password = page.getByLabel('密码', { exact: true })
  await expect(email).toHaveAttribute('autocomplete', 'username')
  await expect(password).toHaveAttribute('autocomplete', 'current-password')
  await email.fill('creator@example.test')
  await password.fill('sample-password')
  await page.getByRole('button', { name: '显示密码', exact: true }).click()
  await expect(password).toHaveAttribute('type', 'text')
  await page.getByRole('button', { name: '隐藏密码', exact: true }).click()
  await expect(password).toHaveAttribute('type', 'password')
  await password.press('Enter')
  await expect(page.getByRole('alert')).toHaveText('邮箱或密码不正确，请重新输入。')
  await expect(email).toHaveValue('creator@example.test')
  await expect(password).toHaveValue('sample-password')
  await password.press('Enter')
  await expect(page.getByRole('alert')).toHaveText('暂时无法连接服务，请稍后重试。')
  expect(submissions).toEqual([{ email: 'creator@example.test', password: 'sample-password' }, { email: 'creator@example.test', password: 'sample-password' }])
  expect(unmatched).toEqual([])
})

test('registration is gated by settings and can recover from an outage', async ({ page }) => {
  await seedAuth(page)
  let available = false
  await page.route('**/api/v1/settings/public', async (route) => {
    await route.fulfill({ status: available ? 200 : 503, json: available ? settings : {} })
  })
  await page.goto('/auth/register')
  await expect(page.getByRole('alert')).toContainText('无法获取注册设置')
  await expect(page.getByRole('button', { name: '注册并进入', exact: true })).toHaveCount(0)
  available = true
  await page.getByRole('button', { name: '重新加载注册设置' }).click()
  await expect(page.getByLabel('邮箱验证码', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '注册并进入', exact: true })).toBeDisabled()
})

test('closed registration hides the form and login advertises no fake registration or reset action', async ({ page }) => {
  const unmatched = await seedAuth(page, { ...settings, registrationEnabled: false, passwordResetEnabled: false })
  await page.goto('/auth/register?redirect=%2Fapp%2Fcreate')
  await expect(page.getByRole('status').filter({ hasText: '新账号注册暂未开放' })).toHaveText('新账号注册暂未开放，已有账号可以继续登录。')
  await expect(page.getByLabel('邮箱', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: '返回登录', exact: true }).click()
  await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === '/app/create')
  await expect(page.getByRole('button', { name: '创建新账号', exact: true })).toHaveCount(0)
  await expect(page.getByRole('link', { name: '忘记密码？', exact: true })).toHaveCount(0)
  expect(unmatched).toEqual([])
})

test('verification, cooldown, consent and password requirements are real registration gates', async ({ page }) => {
  const unmatched = await seedAuth(page)
  let sent = 0
  let submitted: Record<string, unknown> | undefined
  await page.route('**/api/v1/auth/send-verify-code', async (route) => {
    sent += 1
    await route.fulfill({ status: sent === 1 ? 429 : 200, json: sent === 1 ? {} : { countdown: 60 } })
  })
  await page.route('**/api/v1/auth/register', async (route) => {
    submitted = route.request().postDataJSON()
    await route.fulfill({ status: 409, json: { message: '该邮箱已注册' } })
  })
  await page.goto('/auth/register')
  await page.getByLabel('昵称', { exact: true }).fill('影像创作者')
  await page.getByLabel('邮箱', { exact: true }).fill('creator@example.test')
  await page.getByLabel('密码', { exact: true }).fill('valid-password')
  const send = page.getByRole('button', { name: '发送验证码', exact: true })
  await send.click()
  await expect(page.getByRole('alert')).toHaveText('操作过于频繁，请稍后重试。')
  await send.click()
  await expect(page.getByRole('button', { name: /秒后重发/ })).toBeDisabled()
  const code = page.getByLabel('邮箱验证码', { exact: true })
  await expect(code).toHaveAttribute('autocomplete', 'one-time-code')
  await code.fill('123456')
  const submit = page.getByRole('button', { name: '注册并进入', exact: true })
  await expect(submit).toBeDisabled()
  await page.getByRole('checkbox', { name: '同意服务条款与隐私政策' }).check()
  await expect(submit).toBeEnabled()
  await page.getByLabel('邮箱', { exact: true }).fill('second@example.test')
  await expect(code).toHaveValue('')
  await expect(submit).toBeDisabled()
  await code.fill('123456')
  await code.press('Enter')
  await expect(page.getByRole('alert')).toHaveText('该邮箱已注册')
  expect(submitted).toMatchObject({ email: 'second@example.test', displayName: '影像创作者', password: 'valid-password', verifyCode: '123456' })
  expect(unmatched).toEqual([])
})

test('media failure leaves an informative poster and a usable login form', async ({ page }) => {
  await seedAuth(page)
  await page.route('**/brand/auth/*.webm', async (route) => { await route.abort() })
  await page.goto('/auth/login')
  await page.getByRole('button', { name: '播放背景影像', exact: true }).click()
  await expect(page.getByRole('status')).toHaveText('正在展示影像封面')
  await expect(page.locator('video')).toHaveCount(0)
  await expect(page.getByRole('img', { name: /冰岛黑沙海岸/ })).toBeVisible()
  await expect(page.getByRole('button', { name: '登录', exact: true })).toBeEnabled()
})

test('local background film really plays and pauses on request', async ({ page }) => {
  await seedAuth(page)
  await page.goto('/auth/login')
  const video = page.locator('video')
  await expect(video).toHaveCount(0)
  await page.getByRole('button', { name: '播放背景影像', exact: true }).click()
  await expect.poll(async () => video.evaluate((element: HTMLVideoElement) => !element.paused && element.readyState >= 2)).toBe(true)
  await expect.poll(async () => video.evaluate((element: HTMLVideoElement) => element.currentTime)).toBeGreaterThan(0)
  await expect(video).toHaveJSProperty('muted', true)
  await page.getByRole('button', { name: '暂停背景影像', exact: true }).click()
  await expect(video).toHaveJSProperty('paused', true)
  await expect(video).toHaveCSS('opacity', '1')
})

test('a missing poster and video do not hide the login or brand copy', async ({ page }) => {
  await seedAuth(page)
  await page.route('**/brand/auth/*', async (route) => { await route.abort() })
  await page.goto('/auth/login')
  await expect(page.getByRole('img', { name: /冰岛黑沙海岸/ })).toHaveCount(0)
  await page.getByRole('button', { name: '播放背景影像', exact: true }).click()
  await expect(page.getByRole('heading', { name: '登录 Sub2Video', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: /下一幕，\s*由你开场。/ })).toBeVisible()
  await expect(page.getByLabel('邮箱', { exact: true })).toBeEditable()
})

test('login and registration fit 320px without overflow and preserve destination', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile', 'Minimum viewport contract is exercised once.')
  await seedAuth(page)
  await page.setViewportSize({ width: 320, height: 844 })
  await page.goto('/auth/login?redirect=%2Fapp%2Fcreate%3Fmode%3Dtext')
  for (const destination of ['创建新账号', '返回登录']) {
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1)
    await page.getByRole('button', { name: destination, exact: true }).click()
    expect(new URL(page.url()).searchParams.get('redirect')).toBe('/app/create?mode=text')
  }
})
