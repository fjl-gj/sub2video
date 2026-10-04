import { expect, test, type Page, type Route } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Browser contracts only: every API call is intercepted. Token values are
// fixtures, and no supplier execution, payment, or real account is exercised.
const initial = { accessToken: 'auth-session-old-access', refreshToken: 'auth-session-old-refresh' }
const renewed = { accessToken: 'auth-session-renewed-access', refreshToken: 'auth-session-renewed-refresh' }
const replacement = { accessToken: 'auth-session-replacement-access', refreshToken: 'auth-session-replacement-refresh' }
const user = { id: 'u_session', email: 'session@example.test', displayName: '会话测试', workspaceName: '会话测试工作室', role: 'user', balance: 256 }
const otherUser = { ...user, id: 'u_other_session', email: 'other-session@example.test', displayName: '第二用户', workspaceName: '第二工作室' }
const catalog = {
  modes: [{ id: 'mode_text', inputType: 'text', name: '文生视频', description: '通过文字生成视频', enabled: true }],
  carriers: [{ id: 'car_minimax', name: 'MINMAX' }],
  models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', description: '视频模型', inputTypes: ['text'] }],
}
const options = { durations: [5, 10], resolutions: ['768P'], aspectRatios: ['16:9'], supportsGeneratedAudio: false, minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P' }
const overview = {
  workspace: { displayName: user.workspaceName }, wallet: { balance: 256 },
  billing: { totalSpent: 0, monthSpent: 0 }, pricing: { available: true, referencePricePerSecond: 0.5, referenceNote: '按秒计费' },
  storage: { usedGb: 0, quotaGb: 5, fileCount: 0 }, tasks: { activeCount: 0, activeTask: null, recentCompleted: [] },
}
const homePaths = ['/client/wallet', '/client/overview', '/client/create/catalog']

function barrier() {
  let release!: () => void
  const promise = new Promise<void>((resolve) => { release = resolve })
  return { promise, release }
}

function bearer(route: Route) { return route.request().headers().authorization || '' }
async function expired(route: Route) { await route.fulfill({ status: 401, json: { code: 'TOKEN_EXPIRED', message: '登录已过期' } }) }
async function refreshSuccess(route: Route) { await route.fulfill({ json: { token: { ...renewed, expiresIn: 3600, tokenType: 'Bearer' } } }) }
async function storedTokens(page: Page) {
  return page.evaluate(() => ({ accessToken: localStorage.getItem('s2v_access_token'), refreshToken: localStorage.getItem('s2v_refresh_token') }))
}

async function seedSession(page: Page) {
  const fixture = await seedSurfaceData(page)
  await page.addInitScript((tokens) => {
    // Shared marker prevents a reload or a second tab from reseeding a session
    // that the application has deliberately renewed, replaced, or cleared.
    if (!localStorage.getItem('auth-session-fixture-seeded')) {
      localStorage.setItem('s2v_access_token', tokens.accessToken)
      localStorage.setItem('s2v_refresh_token', tokens.refreshToken)
      localStorage.setItem('auth-session-fixture-seeded', '1')
    }
  }, initial)
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({ json: bearer(route) === `Bearer ${replacement.accessToken}` ? otherUser : user })
  })
  await page.route('**/api/v1/auth/login', async (route) => {
    const secondUser = (route.request().postDataJSON() as { email: string }).email === otherUser.email
    await route.fulfill({ json: { user: secondUser ? otherUser : user, token: { ...(secondUser ? replacement : renewed), expiresIn: 3600, tokenType: 'Bearer' } } })
  })
  await page.route('**/api/v1/auth/logout', async (route) => { await route.fulfill({ json: { ok: true } }) })
  await page.route('**/api/v1/client/create/catalog', async (route) => { await route.fulfill({ json: catalog }) })
  await page.route('**/api/v1/client/create/options**', async (route) => { await route.fulfill({ json: options }) })
  await page.route('**/api/v1/client/overview', async (route) => { await route.fulfill({ json: overview }) })
  return fixture
}

async function login(page: Page, email = user.email) {
  await page.getByLabel('邮箱', { exact: true }).fill(email)
  await page.getByLabel('密码', { exact: true }).fill('fixture-password-only')
  await page.getByRole('button', { name: '登录', exact: true }).click()
}

test.describe('authentication session contracts (mocked frontend only)', () => {
  test('isolates drafts across A to B to A without discarding the original draft', async ({ page }) => {
    await seedSession(page)
    const promptA = '账号 A 的未完成镜头，仅应由 A 恢复'
    const promptB = '账号 B 的独立镜头'
    await page.goto('/app/create?mode=text', { waitUntil: 'networkidle' })
    await page.getByLabel('画面描述', { exact: true }).fill(promptA)
    await page.getByRole('button', { name: '用户菜单', exact: true }).click()
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    await expect(page).toHaveURL(/\/auth\/login/)
    await login(page, otherUser.email)
    await page.locator('a[href="/app/create"]:visible').first().click()
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue('')
    await page.getByLabel('画面描述', { exact: true }).fill(promptB)
    await page.getByRole('button', { name: '用户菜单', exact: true }).click()
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    await expect(page).toHaveURL(/\/auth\/login/)
    await login(page)
    await page.locator('a[href="/app/create"]:visible').first().click()
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue(promptA)
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_other_session') || '{}').prompt)).toBe(promptB)
  })

  test('never assigns an unowned legacy draft to the current account', async ({ page }) => {
    await seedSession(page)
    await page.addInitScript(() => localStorage.setItem('sub2video.create-draft', JSON.stringify({ version: 4, workflowInputType: 'text', prompt: '无法确认归属的旧草稿' })))
    await page.goto('/app/create?mode=text', { waitUntil: 'networkidle' })
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue('')
    expect(await page.evaluate(() => localStorage.getItem('sub2video.create-draft'))).toContain('无法确认归属的旧草稿')
  })

  test('clears private project cache on account change and ignores a late previous-account response', async ({ page }) => {
    await seedSession(page)
    const lateRead = barrier()
    let holdOldReads = false
    let delayed = 0
    const projectsPath = /\/api\/v1\/client\/projects(?:\?.*)?$/
    await page.route(projectsPath, async (route) => {
      if (bearer(route) === `Bearer ${replacement.accessToken}`) return route.fulfill({ status: 503, json: { message: '新账号资产暂不可读取' } })
      if (holdOldReads) { delayed++; await lateRead.promise }
      return route.fallback()
    })
    try {
      await page.goto('/app/create?mode=text', { waitUntil: 'networkidle' })
      holdOldReads = true
      if (await page.getByRole('button', { name: '打开菜单', exact: true }).isVisible()) await page.getByRole('button', { name: '打开菜单', exact: true }).click()
      await page.locator('a[href="/app/assets"]:visible').first().click()
      await expect.poll(() => delayed).toBe(1)
      await page.getByRole('button', { name: '用户菜单', exact: true }).click()
      await page.getByRole('button', { name: '退出登录', exact: true }).click()
      await expect(page).toHaveURL(/\/auth\/login/)
      await login(page, otherUser.email)
      await expect(page).toHaveURL(/\/app\/overview/)
      if (await page.getByRole('button', { name: '打开菜单', exact: true }).isVisible()) await page.getByRole('button', { name: '打开菜单', exact: true }).click()
      await page.locator('a[href="/app/assets"]:visible').first().click()
      await expect(page.getByText('资产暂时无法读取，请重新加载。', { exact: true })).toBeVisible()
      const response = page.waitForResponse((res) => /\/client\/projects/.test(res.url()) && res.request().headers().authorization === `Bearer ${initial.accessToken}`)
      lateRead.release()
      await response
      await expect(page.getByText('海岸清晨', { exact: true })).toHaveCount(0)
      await expect(page.getByText('秋日香氛', { exact: true })).toHaveCount(0)
      await expect(page.getByText('资产暂时无法读取，请重新加载。', { exact: true })).toBeVisible()
      await page.locator('a[href="/app/create"]:visible').first().click()
      await expect(page.getByLabel('画面描述', { exact: true })).toBeVisible()
      // This toolbar reads the shared cache even when the new user's API
      // failed; hiding asset-page errors alone would not satisfy isolation.
      await expect(page.getByLabel('当前作品', { exact: true })).toHaveCount(0)
    } finally { lateRead.release() }
  })

  test('an earlier wallet read cannot overwrite a confirmed redemption in any balance surface', async ({ page }) => {
    await seedSession(page)
    const staleRead = barrier()
    let reads = 0
    await page.route('**/api/v1/client/wallet', async (route) => {
      reads++
      if (reads > 2) return route.fulfill({ json: { balance: 300, products: [] } })
      await staleRead.promise
      await route.fulfill({ json: { balance: 1, products: [] } })
    })
    await page.route('**/api/v1/client/wallet/redeem', (route) => route.fulfill({ json: { credited: 44, wallet: { balance: 300 } } }))
    try {
      await page.goto('/app/wallet', { waitUntil: 'domcontentloaded' })
      await expect.poll(() => reads).toBe(2)
      await page.getByLabel('兑换码', { exact: true }).fill('FIXTURE-CODE')
      await page.getByRole('button', { name: '确认兑换', exact: true }).click()
      await expect(page.locator('.s2v-header').getByText('¥ 300.00', { exact: true })).toBeVisible()
      staleRead.release()
      await page.waitForLoadState('networkidle')
      await expect(page.locator('.s2v-header').getByText('¥ 300.00', { exact: true })).toBeVisible()
      await expect(page.getByText('¥ 1.00', { exact: true })).toHaveCount(0)
      await page.locator('a[href="/app/create"]:visible').first().click()
      await expect(page.getByText('钱包余额: ¥ 300.00', { exact: true })).toBeVisible()
    } finally { staleRead.release() }
  })

  test('profile is truthful about readonly and unavailable account features', async ({ page }) => {
    await seedSession(page)
    await page.goto('/app/profile', { waitUntil: 'networkidle' })
    await expect(page.getByLabel('昵称', { exact: true })).toHaveValue(user.displayName)
    await expect(page.getByLabel('昵称', { exact: true })).toHaveAttribute('readonly', '')
    await expect(page.getByText('专业创作者', { exact: true })).toHaveCount(0)
    await expect(page.getByText('邮箱已验证', { exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '保存资料', exact: true })).toHaveCount(0)
    await expect(page.getByText('通知偏好设置暂未开放。', { exact: true })).toBeVisible()
    await page.getByRole('button', { name: '重置密码', exact: true }).click()
    await expect(page).toHaveURL(/\/auth\/forgot-password$/)
  })

  for (const scenario of ['reordered replies', 'rejected third mutation', 'wallet failure and retry'] as const) {
  test(`reconciles overlapping refund and redemption: ${scenario}`, async ({ page }) => {
    await seedSession(page)
    const refundReply = barrier()
    const staleReconciliation = barrier()
    let refunds = 0
    let redemptions = 0
    let walletBalance = 256
    let walletReadsAfterRedeem = 0
    let walletUnavailable = scenario === 'wallet failure and retry'
    await page.route('**/api/v1/client/wallet', async (route) => {
      if (walletBalance === 300) {
        walletReadsAfterRedeem++
        if (scenario === 'rejected third mutation' && walletReadsAfterRedeem === 1) {
          await staleReconciliation.promise
          return route.fulfill({ json: { balance: 280, products: [] } })
        }
        if (walletUnavailable) return route.fulfill({ status: 503, json: { message: '测试钱包暂不可用' } })
      }
      await route.fulfill({ json: { balance: walletBalance, products: [] } })
    })
    await page.route(/\/api\/v1\/client\/tasks(?:\?.*)?$/, async (route) => {
      if (route.request().method() !== 'GET') return route.fallback()
      await route.fulfill({ json: { items: [{ id: 'task_refund_order', title: '退款顺序测试', mode: '文生视频', status: '排队中', canCancel: true, progress: 0, duration: '5 秒', resolution: '768P', cost: 2.5, time: '刚刚', thumbUrl: '' }], total: 1, page: 1, pageSize: 20, counts: { 全部: 1, 排队中: 1 } } })
    })
    await page.route('**/api/v1/client/tasks/task_refund_order/cancel', async (route) => {
      refunds++
      await refundReply.promise
      await route.fulfill({ json: { task: { id: 'task_refund_order', status: '已取消' }, refundAmount: 2.5, wallet: { balance: 258.5 } } })
    })
    await page.route('**/api/v1/client/wallet/redeem', async (route) => {
      redemptions++
      if (redemptions > 1) return route.fulfill({ status: 400, json: { code: 'BAD_REQUEST', message: '测试兑换码无效' } })
      walletBalance = 300
      await route.fulfill({ json: { credited: 41.5, wallet: { balance: 300 } } })
    })
    try {
      await page.goto('/app/overview', { waitUntil: 'networkidle' })
      await page.locator('a[href="/app/works"]:visible').first().click()
      await page.getByRole('button', { name: '终止任务并退款', exact: true }).click()
      await page.getByRole('button', { name: '终止并退款', exact: true }).click()
      await expect.poll(() => refunds).toBe(1)
      await page.goBack()
      await expect(page).toHaveURL(/\/app\/overview$/)
      await page.locator('a[href="/app/wallet"]:visible').first().click()
      await page.getByLabel('兑换码', { exact: true }).fill('OVERLAP-FIXTURE')
      await page.getByRole('button', { name: '确认兑换', exact: true }).click()
      await expect(page.getByText('兑换成功，余额增加 ¥41.50', { exact: true })).toBeVisible()
      refundReply.release()
      await expect.poll(() => walletReadsAfterRedeem).toBe(1)
      if (scenario === 'rejected third mutation') {
        // A and B are finished, but their authoritative wallet read is still
        // pending. C fails explicitly and must still trigger a fresh read.
        await page.getByLabel('兑换码', { exact: true }).fill('REJECTED-FIXTURE')
        await page.getByRole('button', { name: '确认兑换', exact: true }).click()
        await expect(page.getByText('测试兑换码无效', { exact: true })).toBeVisible()
        await expect.poll(() => walletReadsAfterRedeem).toBe(2)
        await expect(page.locator('.s2v-header').getByText('¥ 300.00', { exact: true })).toBeVisible()
        const lateResponse = page.waitForResponse((response) => response.url().endsWith('/client/wallet'))
        staleReconciliation.release()
        await lateResponse
        await expect(page.getByText('¥ 280.00', { exact: true })).toHaveCount(0)
      } else if (scenario === 'wallet failure and retry') {
        await expect(page.getByText('余额同步失败，当前显示上次确认的余额，请重新加载。', { exact: true })).toBeVisible()
        await expect(page.getByText('上次读取余额:', { exact: true })).toBeVisible()
        walletUnavailable = false
        await page.getByRole('button', { name: '重新加载余额', exact: true }).click()
        await expect(page.getByText('余额同步失败，当前显示上次确认的余额，请重新加载。', { exact: true })).toHaveCount(0)
      }
      await expect(page.locator('.s2v-header').getByText('¥ 300.00', { exact: true })).toBeVisible()
      await expect(page.getByText('¥ 258.50', { exact: true })).toHaveCount(0)
    } finally { refundReply.release(); staleReconciliation.release() }
  })
  }

  test('renews an expired token during bootstrap before loading private page data', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    const meTokens: string[] = []
    const earlyPrivateReads: string[] = []
    let authenticated = false
    let refreshCalls = 0
    page.on('request', (request) => {
      if (!authenticated && new URL(request.url()).pathname.startsWith('/api/v1/client/')) earlyPrivateReads.push(request.url())
    })
    await page.route('**/api/v1/auth/me', async (route) => {
      meTokens.push(bearer(route))
      if (bearer(route) === `Bearer ${initial.accessToken}`) return expired(route)
      expect(bearer(route)).toBe(`Bearer ${renewed.accessToken}`)
      authenticated = true
      await route.fulfill({ json: user })
    })
    await page.route('**/api/v1/auth/refresh', async (route) => { refreshCalls += 1; await refreshSuccess(route) })
    await page.goto('/app/overview', { waitUntil: 'networkidle' })
    await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toBeVisible()
    expect(meTokens).toEqual([`Bearer ${initial.accessToken}`, `Bearer ${renewed.accessToken}`])
    expect(refreshCalls).toBe(1)
    expect(earlyPrivateReads).toEqual([])
    expect(await storedTokens(page)).toEqual(renewed)
    expect(unmatchedRequests).toEqual([])
  })

  for (const { lateCatalog, withoutLocks } of [
    { lateCatalog: false, withoutLocks: false },
    { lateCatalog: true, withoutLocks: false },
    { lateCatalog: false, withoutLocks: true },
  ]) {
    test(`shares one refresh across homepage requests${lateCatalog ? ' and reuses it for a late old-token 401' : ' receiving 401 together'}${withoutLocks ? ' without Web Locks' : ''}`, async ({ page }) => {
      const { unmatchedRequests } = await seedSession(page)
      if (withoutLocks) await page.addInitScript(() => { Object.defineProperty(navigator, 'locks', { value: undefined, configurable: true }) })
      const allowRefresh = barrier()
      const allowLate401 = barrier()
      const initialReads: string[] = []
      const replayReads: string[] = []
      let refreshCalls = 0
      await page.route('**/api/v1/auth/refresh', async (route) => {
        refreshCalls += 1
        expect(route.request().postDataJSON()).toEqual({ refreshToken: initial.refreshToken })
        await allowRefresh.promise
        await refreshSuccess(route)
      })
      for (const path of homePaths) {
        await page.route(`**/api/v1${path}`, async (route) => {
          if (bearer(route) === `Bearer ${initial.accessToken}`) {
            initialReads.push(path)
            if (lateCatalog && path === '/client/create/catalog') await allowLate401.promise
            await expired(route)
            return
          }
          expect(bearer(route)).toBe(`Bearer ${renewed.accessToken}`)
          replayReads.push(path)
          await route.fallback()
        })
      }
      try {
        await page.goto('/app/overview', { waitUntil: 'domcontentloaded' })
        await expect.poll(() => initialReads.length).toBe(3)
        await expect.poll(() => refreshCalls).toBe(1)
        allowRefresh.release()
        if (lateCatalog) {
          await expect.poll(() => replayReads.length).toBe(2)
          expect(await storedTokens(page)).toEqual(renewed)
        }
        allowLate401.release()
        await expect.poll(() => replayReads.length).toBe(3)
        await expect(page.getByRole('button', { name: /文生视频/ }).first()).toBeEnabled()
        expect(initialReads.sort()).toEqual([...homePaths].sort())
        expect(replayReads.sort()).toEqual([...homePaths].sort())
        expect(refreshCalls).toBe(1)
        expect(await storedTokens(page)).toEqual(renewed)
        if (withoutLocks) expect(await page.evaluate(() => typeof navigator.locks)).toBe('undefined')
        expect(unmatchedRequests).toEqual([])
      } finally {
        allowRefresh.release()
        allowLate401.release()
      }
    })
  }

  test('redirects an expired mounted session and preserves the creation destination and text draft', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    const destination = '/app/create?mode=text&source=session-test'
    await page.goto(destination, { waitUntil: 'networkidle' })
    const prompt = '保留这段刚输入的海边镜头描述，重新登录后继续。'
    await page.getByRole('textbox', { name: '画面描述', exact: true }).fill(prompt)
    let refreshCalls = 0
    let taskSubmissions = 0
    page.on('request', (request) => { if (request.method() === 'POST' && new URL(request.url()).pathname === '/api/v1/client/tasks') taskSubmissions += 1 })
    await page.route('**/api/v1/client/tasks/estimate', expired)
    await page.route('**/api/v1/auth/refresh', async (route) => {
      refreshCalls += 1
      await route.fulfill({ status: 401, json: { code: 'INVALID_REFRESH_TOKEN', message: '请重新登录' } })
    })
    await page.getByLabel('视频时长', { exact: true }).selectOption('10')
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === destination)
    await expect(page.getByRole('heading', { name: '登录 Sub2Video', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toHaveCount(0)
    expect(await storedTokens(page)).toEqual({ accessToken: null, refreshToken: null })
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_session') || '{}').prompt)).toBe(prompt)
    await page.unroute('**/api/v1/client/tasks/estimate', expired)
    await login(page)
    await expect(page).toHaveURL((url) => `${url.pathname}${url.search}` === destination)
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue(prompt)
    expect(refreshCalls).toBe(1)
    expect(taskSubmissions).toBe(0)
    expect(unmatchedRequests).toEqual([])
  })

  test('terminates a replayed 401 without a second refresh or stale authenticated state', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    const requestTokens: string[] = []
    let refreshCalls = 0
    await page.route('**/api/v1/client/wallet', async (route) => {
      requestTokens.push(bearer(route))
      await expired(route)
    })
    await page.route('**/api/v1/auth/refresh', async (route) => {
      refreshCalls += 1
      await refreshSuccess(route)
    })
    await page.goto('/app/overview', { waitUntil: 'domcontentloaded' })
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === '/app/overview')
    expect(requestTokens).toEqual([`Bearer ${initial.accessToken}`, `Bearer ${renewed.accessToken}`])
    expect(refreshCalls).toBe(1)
    expect(await storedTokens(page)).toEqual({ accessToken: null, refreshToken: null })
    await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toHaveCount(0)
    expect(unmatchedRequests).toEqual([])
  })

  test('keeps the session after a temporary refresh failure and allows an explicit page retry', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    let refreshCalls = 0
    let firstOverview = true
    await page.route('**/api/v1/client/overview', async (route) => {
      if (firstOverview) { firstOverview = false; return expired(route) }
      await route.fallback()
    })
    await page.route('**/api/v1/auth/refresh', async (route) => {
      refreshCalls += 1
      await route.fulfill({ status: 503, json: { message: '登录服务暂时不可用' } })
    })
    await page.goto('/app/overview', { waitUntil: 'networkidle' })
    await expect(page.getByText('暂时无法读取工作空间，请重试。', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toBeVisible()
    expect(await storedTokens(page)).toEqual(initial)
    await expect(page).toHaveURL((url) => url.pathname === '/app/overview')
    await page.getByRole('button', { name: '重新加载', exact: true }).click()
    await expect(page.getByText('暂时无法读取工作空间，请重试。', { exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: /文生视频/ }).first()).toBeEnabled()
    expect(refreshCalls).toBe(1)
    expect(await storedTokens(page)).toEqual(initial)
    expect(unmatchedRequests).toEqual([])
  })

  test('preserves a cold session during refresh unavailability and reconnects without another login', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    const destination = '/app/create?mode=text&source=reconnect-test'
    const privateReads: string[] = []
    let refreshCalls = 0
    let available = false
    page.on('request', (request) => {
      if (new URL(request.url()).pathname.startsWith('/api/v1/client/')) privateReads.push(request.url())
    })
    await page.route('**/api/v1/auth/me', async (route) => {
      if (bearer(route) === `Bearer ${initial.accessToken}`) return expired(route)
      expect(bearer(route)).toBe(`Bearer ${renewed.accessToken}`)
      await route.fulfill({ json: user })
    })
    await page.route('**/api/v1/auth/refresh', async (route) => {
      refreshCalls += 1
      if (available) return refreshSuccess(route)
      await route.fulfill({ status: 503, json: { message: '登录服务暂时不可用' } })
    })
    await page.goto(destination, { waitUntil: 'networkidle' })
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('reason') === 'session-unavailable' && url.searchParams.get('redirect') === destination)
    await expect(page.getByRole('region', { name: '账号入口', exact: true }).getByRole('status')).toContainText('暂时无法确认登录状态，请重试。')
    expect(await storedTokens(page)).toEqual(initial)
    expect(privateReads).toEqual([])
    expect(refreshCalls).toBe(1)
    available = true
    await page.getByRole('button', { name: '重新连接', exact: true }).click()
    await expect(page).toHaveURL((url) => `${url.pathname}${url.search}` === destination)
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toBeVisible()
    expect(await storedTokens(page)).toEqual(renewed)
    expect(refreshCalls).toBe(2)
    expect(unmatchedRequests).toEqual([])
  })

  test('leaves protected pages and clears local credentials when logout revocation returns 503', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    let logoutCalls = 0
    await page.route('**/api/v1/auth/logout', async (route) => {
      logoutCalls += 1
      await route.fulfill({ status: 503, json: { message: '注销服务暂时不可用' } })
    })
    await page.goto('/app/overview', { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '用户菜单', exact: true }).click()
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login')
    await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toHaveCount(0)
    expect(await storedTokens(page)).toEqual({ accessToken: null, refreshToken: null })
    expect(logoutCalls).toBe(1)
    await page.goto('/app/overview', { waitUntil: 'networkidle' })
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === '/app/overview')
    expect(await storedTokens(page)).toEqual({ accessToken: null, refreshToken: null })
    expect(unmatchedRequests).toEqual([])
  })

  test('allows another login while logout is pending and ignores its late response', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    const allowLogout = barrier()
    let logoutCalls = 0
    await page.route('**/api/v1/auth/logout', async (route) => {
      logoutCalls += 1
      expect(route.request().postDataJSON()).toEqual({ refreshToken: initial.refreshToken })
      await allowLogout.promise
      await route.fulfill({ json: { ok: true } })
    })
    try {
      await page.goto('/app/overview', { waitUntil: 'networkidle' })
      await page.getByRole('button', { name: '用户菜单', exact: true }).click()
      await page.getByRole('button', { name: '退出登录', exact: true }).click()
      await expect.poll(() => logoutCalls).toBe(1)
      await expect(page).toHaveURL((url) => url.pathname === '/auth/login')
      expect(await storedTokens(page)).toEqual({ accessToken: null, refreshToken: null })
      await login(page, otherUser.email)
      await expect(page).toHaveURL((url) => url.pathname === '/app/overview')
      expect(await storedTokens(page)).toEqual(replacement)
      const logoutResponse = page.waitForResponse('**/api/v1/auth/logout')
      allowLogout.release()
      await logoutResponse
      await page.waitForLoadState('networkidle')
      await expect(page).toHaveURL((url) => url.pathname === '/app/overview')
      await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toBeVisible()
      expect(await storedTokens(page)).toEqual(replacement)
      expect(logoutCalls).toBe(1)
      expect(unmatchedRequests).toEqual([])
    } finally { allowLogout.release() }
  })

  for (const { replaceSession, lateFailure } of [
    { replaceSession: false, lateFailure: false },
    { replaceSession: true, lateFailure: false },
    { replaceSession: true, lateFailure: true },
  ]) {
    test(`ignores a late refresh ${lateFailure ? 'rejection' : 'success'} after ${replaceSession ? 'logout and another login' : 'logout'}`, async ({ page }) => {
      const { unmatchedRequests } = await seedSession(page)
      const allowRefresh = barrier()
      let refreshCalls = 0
      let logoutCalls = 0
      let oldReplays = 0
      await page.route('**/api/v1/client/wallet', async (route) => {
        if (bearer(route) === `Bearer ${initial.accessToken}`) return expired(route)
        if (bearer(route) === `Bearer ${renewed.accessToken}`) oldReplays += 1
        await route.fallback()
      })
      await page.route('**/api/v1/auth/refresh', async (route) => {
        refreshCalls += 1
        await allowRefresh.promise
        if (lateFailure) await route.fulfill({ status: 401, json: { code: 'INVALID_REFRESH_TOKEN', message: '旧会话已失效' } })
        else await refreshSuccess(route)
      })
      await page.route('**/api/v1/auth/logout', async (route) => {
        logoutCalls += 1
        await route.fulfill({ json: { ok: true } })
      })
      try {
        await page.goto('/app/overview', { waitUntil: 'domcontentloaded' })
        await expect.poll(() => refreshCalls).toBe(1)
        await page.getByRole('button', { name: '用户菜单', exact: true }).click()
        await page.getByRole('button', { name: '退出登录', exact: true }).click()
        await expect(page).toHaveURL((url) => url.pathname === '/auth/login')
        expect(await storedTokens(page)).toEqual({ accessToken: null, refreshToken: null })
        if (replaceSession) {
          await login(page, otherUser.email)
          await expect(page).toHaveURL((url) => url.pathname === '/app/overview')
          expect(await storedTokens(page)).toEqual(replacement)
        }
        const refreshResponse = page.waitForResponse('**/api/v1/auth/refresh')
        allowRefresh.release()
        await refreshResponse
        await page.waitForLoadState('networkidle')
        expect(await storedTokens(page)).toEqual(replaceSession ? replacement : { accessToken: null, refreshToken: null })
        expect(oldReplays).toBe(0)
        expect(logoutCalls).toBe(1)
        expect(refreshCalls).toBe(1)
        await expect(page).toHaveURL((url) => url.pathname === (replaceSession ? '/app/overview' : '/auth/login'))
        expect(unmatchedRequests).toEqual([])
      } finally { allowRefresh.release() }
    })
  }

  test('replays a task submission with its original payload and Idempotency-Key', async ({ page }) => {
    const { unmatchedRequests } = await seedSession(page)
    const submissions: Array<{ token: string; key: string; payload: unknown }> = []
    let refreshCalls = 0
    const task = { id: 'task_session_replay', title: '会话重放任务', mode: '文生视频', status: '排队中', progress: 0, duration: '5 秒', durationSeconds: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false, cost: 2.5, thumbUrl: '', savedToProject: false, failureReason: '', executionWaiting: false, deliveryStatus: 'not_ready', downloadUrl: '', createdAt: '', updatedAt: '' }
    await page.route('**/api/v1/auth/refresh', async (route) => { refreshCalls += 1; await refreshSuccess(route) })
    await page.route(/\/api\/v1\/client\/tasks$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      submissions.push({ token: bearer(route), key: route.request().headers()['idempotency-key'] || '', payload: route.request().postDataJSON() })
      if (submissions.length === 1) return expired(route)
      await route.fulfill({ status: 201, json: { task, order: { id: 'order_session_replay', amount: 2.5, currency: 'CNY', status: 'paid' }, wallet: { balance: 253.5 } } })
    })
    await page.route('**/api/v1/client/tasks/task_session_replay', async (route) => { await route.fulfill({ json: task }) })
    await page.goto('/app/create?mode=text', { waitUntil: 'networkidle' })
    await page.getByRole('textbox', { name: '画面描述', exact: true }).fill('镜头沿海岸平稳推进。')
    await page.getByRole('button', { name: '提交生成任务', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/works/task_session_replay')
    expect(submissions).toHaveLength(2)
    expect(submissions[0].token).toBe(`Bearer ${initial.accessToken}`)
    expect(submissions[1].token).toBe(`Bearer ${renewed.accessToken}`)
    expect(submissions[0].key).not.toBe('')
    expect(submissions[1].key).toBe(submissions[0].key)
    expect(submissions[1].payload).toEqual(submissions[0].payload)
    expect(refreshCalls).toBe(1)
    expect(unmatchedRequests).toEqual([])
  })

  test('shares a rotating refresh across two same-origin tabs', async ({ page, context }) => {
    const second = await context.newPage()
    const firstFixture = await seedSession(page)
    const secondFixture = await seedSession(second)
    const allowRefresh = barrier()
    const initialTabs = new Set<string>()
    const replayTabs = new Set<string>()
    let refreshCalls = 0
    for (const [name, tab] of [['first', page], ['second', second]] as const) {
      await tab.route('**/api/v1/client/wallet', async (route) => {
        if (bearer(route) === `Bearer ${initial.accessToken}`) {
          initialTabs.add(name)
          return expired(route)
        }
        expect(bearer(route)).toBe(`Bearer ${renewed.accessToken}`)
        replayTabs.add(name)
        await route.fallback()
      })
      await tab.route('**/api/v1/auth/refresh', async (route) => {
        refreshCalls += 1
        await allowRefresh.promise
        await refreshSuccess(route)
      })
    }
    try {
      await page.goto('/app/overview', { waitUntil: 'domcontentloaded' })
      await expect.poll(() => refreshCalls).toBe(1)
      await second.goto('/app/overview', { waitUntil: 'domcontentloaded' })
      await expect.poll(() => initialTabs.size).toBe(2)
      allowRefresh.release()
      await expect.poll(() => replayTabs.size).toBe(2)
      await Promise.all([page.waitForLoadState('networkidle'), second.waitForLoadState('networkidle')])
      expect(refreshCalls).toBe(1)
      expect(await storedTokens(page)).toEqual(renewed)
      expect(await storedTokens(second)).toEqual(renewed)
      await expect(page.getByRole('button', { name: '用户菜单', exact: true })).toBeVisible()
      await expect(second.getByRole('button', { name: '用户菜单', exact: true })).toBeVisible()
      expect(firstFixture.unmatchedRequests).toEqual([])
      expect(secondFixture.unmatchedRequests).toEqual([])
    } finally { allowRefresh.release() }
  })

  test('propagates logout to another tab without losing its unfinished text draft', async ({ page, context }) => {
    const second = await context.newPage()
    const firstFixture = await seedSession(page)
    const secondFixture = await seedSession(second)
    const destination = '/app/create?mode=text&source=second-tab'
    await page.goto('/app/overview', { waitUntil: 'networkidle' })
    await second.goto(destination, { waitUntil: 'networkidle' })
    const prompt = '另一个标签页正在编辑的镜头内容。'
    await second.getByRole('textbox', { name: '画面描述', exact: true }).fill(prompt)
    await page.getByRole('button', { name: '用户菜单', exact: true }).click()
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login')
    await expect(second).toHaveURL((url) => url.pathname === '/auth/login' && url.searchParams.get('redirect') === destination)
    await expect(second.getByRole('button', { name: '用户菜单', exact: true })).toHaveCount(0)
    expect(await storedTokens(second)).toEqual({ accessToken: null, refreshToken: null })
    expect(await second.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_session') || '{}').prompt)).toBe(prompt)
    expect(firstFixture.unmatchedRequests).toEqual([])
    expect(secondFixture.unmatchedRequests).toEqual([])
  })
})
