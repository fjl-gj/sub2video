import { expect, test, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Browser contracts with mocked APIs. No supplier, payment, or real storage
// acceptance is implied by these tests.
const createPath = '/app/create?guide=coast-letter&shot=arrival'
const user = { id: 'u_journey', email: 'journey@example.test', displayName: '旅程测试', workspaceName: '旅程工作室', role: 'user', balance: 256 }
const catalog = {
  modes: [
    { id: 'mode_text', inputType: 'text', name: '文生视频', enabled: true, description: '文字生成' },
    { id: 'mode_multi', inputType: 'multi_image', name: '多图生成视频', enabled: true, description: '参考图片' },
    { id: 'mode_frames', inputType: 'first_last', name: '首尾帧生成视频', enabled: true, description: '起止画面' },
  ],
  carriers: [{ id: 'car_minimax', name: 'MINMAX' }],
  models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', description: '视频模型', inputTypes: ['text', 'multi_image', 'first_last'] }],
}
const options = { durations: [5, 10, 15], resolutions: ['768P', '1080P'], aspectRatios: ['16:9', '9:16'], supportsGeneratedAudio: true, minImages: 1, maxImages: 9, defaultDuration: 5, defaultResolution: '768P' }
const resultURL = 'https://journey-media.example.test/result.mp4'
const result = {
  id: 'task_journey', title: '海岸来信 · 远景', mode: '文生视频', status: '已完成', progress: 100,
  duration: '5 秒', durationSeconds: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false,
  cost: 2.5, thumbUrl: '', savedToProject: false, failureReason: '', executionWaiting: false,
  deliveryStatus: 'ready', downloadUrl: resultURL, createdAt: '2026-09-25T04:00:00Z', updatedAt: '2026-09-25T04:00:05Z',
}

async function seedJourney(page: Page, authenticated = true) {
  const fixture = await seedSurfaceData(page)
  if (authenticated) await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'journey-test-access')
    localStorage.setItem('s2v_refresh_token', 'journey-test-refresh')
  })
  await page.route('**/api/v1/auth/me', async (route) => { await route.fulfill({ json: user }) })
  for (const action of ['login', 'register']) {
    await page.route(`**/api/v1/auth/${action}`, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      await route.fulfill({ json: { user, token: { accessToken: 'journey-test-access', refreshToken: 'journey-test-refresh', expiresIn: 3600, tokenType: 'Bearer' } } })
    })
  }
  await page.route('**/api/v1/settings/public', async (route) => { await route.fulfill({ json: { emailVerifyEnabled: false, passwordResetEnabled: true, registrationEnabled: true } }) })
  await page.route('**/api/v1/client/create/catalog', async (route) => { await route.fulfill({ json: catalog }) })
  await page.route('**/api/v1/client/create/options**', async (route) => { await route.fulfill({ json: options }) })
  await page.route('**/api/v1/client/session/current-project', async (route) => {
    if (route.request().method() !== 'PUT') return route.fallback()
    await route.fulfill({ json: { currentProjectId: (route.request().postDataJSON() as { projectId: string }).projectId } })
  })
  await page.route(resultURL, async (route) => { await route.fulfill({ status: 403, body: 'Expired fixture media URL' }) })
  return fixture
}

async function login(page: Page) {
  await page.getByLabel('邮箱', { exact: true }).fill(user.email)
  await page.getByLabel('密码', { exact: true }).fill('test-password-only')
  await page.getByRole('button', { name: '登录', exact: true }).click()
}

test.describe('five-page creator journey (mocked frontend only)', () => {
  test('continues a public tutorial through login, submission, result and a chosen project', async ({ page }) => {
    const { unmatchedRequests } = await seedJourney(page, false)
    const privateReads: string[] = []
    page.on('request', (request) => {
      if (request.url().includes('/api/v1/client/')) privateReads.push(request.url())
    })
    let submitted: Record<string, unknown> | undefined
    let submissionKey = ''
    let saveAttempts = 0
    let savedProject = ''
    await page.route(/\/api\/v1\/client\/tasks$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      submitted = route.request().postDataJSON() as Record<string, unknown>
      submissionKey = route.request().headers()['idempotency-key'] || ''
      await route.fulfill({ status: 201, json: { task: result, order: { id: 'order_journey', amount: 2.5, currency: 'CNY', status: 'paid' }, wallet: { balance: 253.5 } } })
    })
    await page.route('**/api/v1/client/tasks/task_journey', async (route) => { await route.fulfill({ json: { ...result, savedToProject: !!savedProject } }) })
    await page.route('**/api/v1/client/tasks/task_journey/save-to-project', async (route) => {
      expect(route.request().method()).toBe('POST')
      saveAttempts += 1
      const { projectId } = route.request().postDataJSON() as { projectId: string }
      expect(projectId).toBe('prj_visual_product')
      if (saveAttempts === 1) {
        await route.fulfill({ status: 503, json: { message: '保存暂时不可用，请重试' } })
        return
      }
      savedProject = projectId
      await route.fulfill({ json: { asset: { id: 'asset_journey', name: '海岸来信-远景.mp4', type: 'video', sourceTask: result.id, url: resultURL }, path: ['视频', '正片镜头'] } })
    })
    await page.route('**/api/v1/client/projects/prj_visual_product/tree', async (route) => {
      await route.fulfill({ json: {
        id: 'prj_visual_product', name: '秋日香氛', desc: '旅程测试目标作品', createdAt: '2026-09-25', trashed: false, fileCount: savedProject ? 1 : 0,
        folders: { 视频: { isSystem: true, desc: '生成成片', items: [], folders: { 正片镜头: { isSystem: false, desc: '默认镜头', folders: {}, items: savedProject ? [{ id: 'asset_journey', name: '海岸来信-远景.mp4', type: 'video', size: '2 MB', time: '2026-09-25', sourceTask: result.id, url: resultURL }] : [] } } } },
      } })
    })

    await page.goto('/', { waitUntil: 'networkidle' })
    await expect(page).toHaveURL(/\/$/)
    await page.locator('.s2v-public-footer').getByRole('link', { name: '镜头教程', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/guides/coast-letter')
    await expect(page.getByRole('heading', { name: '海岸来信', level: 1, exact: true })).toBeVisible()
    await expect(page.locator('video')).toHaveCount(0)
    expect(privateReads).toEqual([])
    await page.getByRole('link', { name: '制作这个镜头', exact: true }).or(page.getByRole('button', { name: '制作这个镜头', exact: true })).first().click()
    await expect(page).toHaveURL(/\/auth\/login\?redirect=/)
    const redirect = new URL(page.url()).searchParams.get('redirect')!
    expect(new URL(redirect, 'https://local.example.test').searchParams.get('guide')).toBe('coast-letter')
    expect(new URL(redirect, 'https://local.example.test').searchParams.get('shot')).toBe('arrival')
    await login(page)
    await expect(page).toHaveURL((url) => url.pathname === '/app/create')
    const prompt = page.getByRole('textbox', { name: '画面描述', exact: true })
    await expect(prompt).not.toHaveValue('')
    const tutorialPrompt = await prompt.inputValue()
    await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue('5')
    await expect(page.getByLabel('输出分辨率', { exact: true })).toHaveValue('768P')
    await expect(page.getByLabel('画幅比例', { exact: true })).toHaveValue('16:9')
    const submittedPrompt = `${tutorialPrompt}，镜头平稳。`
    await prompt.fill(submittedPrompt)
    await page.getByRole('button', { name: '提交生成任务', exact: true }).click()
    await expect(page).toHaveURL(/\/app\/works\/task_journey$/)
    expect(await page.evaluate(() => localStorage.getItem('sub2video.create-draft.u_journey'))).toBeNull()
    expect(submissionKey).not.toBe('')
    expect(submitted).toMatchObject({ modeId: 'mode_text', durationSec: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false, prompt: submittedPrompt })
    expect(submitted).not.toHaveProperty('durationSeconds')
    expect(JSON.stringify(submitted)).not.toMatch(/quoteId|supplier|cost|token|workflow|executionType|apiModelCode|workflowCode|upstream/i)

    await page.getByRole('button', { name: '保存到作品', exact: true }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByRole('combobox', { name: '目标作品', exact: true }).selectOption('prj_visual_product')
    await dialog.getByRole('button', { name: '保存成片', exact: true }).click()
    await expect(page.getByText('保存失败，请稍后重试；如已在其他页面保存，可重新加载任务状态。', { exact: true })).toBeVisible()
    await expect(dialog).toBeVisible()
    await expect(page.getByRole('link', { name: '查看作品', exact: true }).or(page.getByRole('button', { name: '查看作品', exact: true }))).toHaveCount(0)
    await dialog.getByRole('button', { name: '保存成片', exact: true }).click()
    await expect(dialog).toHaveCount(0)
    expect(saveAttempts).toBe(2)
    await page.getByRole('link', { name: '查看作品', exact: true }).or(page.getByRole('button', { name: '查看作品', exact: true })).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/assets')
    expect(new URL(page.url()).searchParams.get('project')).toBe('prj_visual_product')
    expect(new URL(page.url()).searchParams.get('folder')).toBe('视频/正片镜头')
    await expect(page.getByText('海岸来信-远景.mp4', { exact: true })).toBeVisible()
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.getByText('海岸来信-远景.mp4', { exact: true })).toBeVisible()
    expect(await page.evaluate(() => localStorage.getItem('sub2video.create-draft.u_journey'))).toBeNull()
    expect(unmatchedRequests).toEqual([])
  })

  test('preserves the tutorial destination through registration and rejects an external login destination', async ({ page }) => {
    await seedJourney(page, false)
    await page.goto(`/auth/login?redirect=${encodeURIComponent(createPath)}`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '创建新账号', exact: true }).click()
    await expect(page).toHaveURL(/\/auth\/register\?/)
    expect(new URL(page.url()).searchParams.get('redirect')).toBe(createPath)
    await page.getByLabel('昵称', { exact: true }).fill('旅程测试')
    await page.getByLabel('邮箱', { exact: true }).fill(user.email)
    await page.getByLabel('密码', { exact: true }).fill('test-password-only')
    await page.getByRole('checkbox', { name: '同意服务条款与隐私政策', exact: true }).check()
    await page.getByRole('button', { name: '注册并进入', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/create')
    expect(new URL(page.url()).searchParams.get('guide')).toBe('coast-letter')
    await page.evaluate(() => localStorage.clear())
    await page.goto(`/auth/login?redirect=${encodeURIComponent('https://external.example.test/account')}`, { waitUntil: 'networkidle' })
    await login(page)
    await expect(page).toHaveURL(/\/app\/overview$/)
  })

  test('requires a new login after an expired token before exposing private content', async ({ page }) => {
    await seedJourney(page)
    await page.route('**/api/v1/auth/me', async (route) => { await route.fulfill({ status: 401, json: { message: '登录已过期' } }) })
    await page.route('**/api/v1/auth/refresh', async (route) => { await route.fulfill({ status: 401, json: { message: '登录已过期' } }) })
    const privateCalls: string[] = []
    page.on('request', (request) => { if (request.url().includes('/api/v1/client/')) privateCalls.push(request.url()) })
    await page.goto(createPath, { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: '欢迎回来', exact: true })).toBeVisible()
    expect(new URL(page.url()).searchParams.get('redirect')).toBe(createPath)
    expect(privateCalls).toEqual([])
  })

  test('keeps an existing draft until the creator explicitly applies tutorial settings', async ({ page }) => {
    await seedJourney(page)
    await page.addInitScript(() => {
      const slot = { uploaded: false, assetId: '', picking: false }
      localStorage.setItem('sub2video.create-draft.u_journey', JSON.stringify({ version: 3, workflowInputType: 'multi_image', prompt: '这是尚未完成的商品镜头，请保留', ratio: '9:16', durationSec: '10', resolution: '1080P', firstFrame: slot, lastFrame: slot, extraFrames: [], multiFrames: [slot] }))
    })
    await page.goto(createPath, { waitUntil: 'networkidle' })
    const prompt = page.getByRole('textbox', { name: '画面描述', exact: true })
    await expect(prompt).toHaveValue('这是尚未完成的商品镜头，请保留')
    await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue('10')
    await page.getByRole('button', { name: '应用教程设置', exact: true }).click()
    await expect(prompt).not.toHaveValue('这是尚未完成的商品镜头，请保留')
    await expect(prompt).not.toHaveValue('')
    await expect(page.getByLabel('创作模式', { exact: true }).getByRole('button', { name: '文生视频', exact: true })).toHaveAttribute('aria-pressed', 'true')
    await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue('5')
  })

  test('adapts unsupported recommended parameters while retaining the tutorial mode', async ({ page }) => {
    await seedJourney(page)
    await page.route('**/api/v1/client/create/options**', async (route) => {
      await route.fulfill({ json: { ...options, durations: [10], defaultDuration: 10, resolutions: ['1080P'], defaultResolution: '1080P', aspectRatios: ['9:16'] } })
    })
    await page.goto(createPath, { waitUntil: 'networkidle' })
    await expect(page.getByText('教程推荐参数已按当前可用能力调整，请核对后提交。', { exact: true })).toBeVisible()
    await expect(page.getByLabel('创作模式', { exact: true }).getByRole('button', { name: '文生视频', exact: true })).toHaveAttribute('aria-pressed', 'true')
    await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue('10')
    await expect(page.getByLabel('输出分辨率', { exact: true })).toHaveValue('1080P')
    await expect(page.getByLabel('画幅比例', { exact: true })).toHaveValue('9:16')
  })

  test('reuses the unconfirmed submission key when the same tutorial is applied again', async ({ page }) => {
    const { unmatchedRequests } = await seedJourney(page)
    const keys: string[] = []
    const payloads: unknown[] = []
    await page.route(/\/api\/v1\/client\/tasks$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      keys.push(route.request().headers()['idempotency-key'] || '')
      payloads.push(route.request().postDataJSON())
      if (keys.length === 1) {
        await route.fulfill({ status: 503, json: { message: '提交结果暂未确认，请重试' } })
        return
      }
      await route.fulfill({ status: 201, json: { task: result, order: { id: 'order_journey', amount: 2.5, currency: 'CNY', status: 'paid' }, wallet: { balance: 253.5 } } })
    })
    await page.route('**/api/v1/client/tasks/task_journey', async (route) => { await route.fulfill({ json: result }) })
    await page.goto(createPath, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '提交生成任务', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('尚未确认提交结果')
    await page.getByRole('button', { name: '应用教程设置', exact: true }).click()
    await page.getByRole('button', { name: '确认原提交结果', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/works/task_journey')
    expect(keys).toHaveLength(2)
    expect(keys[0]).not.toBe('')
    expect(keys[1]).toBe(keys[0])
    expect(payloads[1]).toEqual(payloads[0])
    expect(unmatchedRequests).toEqual([])
  })

  test('preserves an immediate edit when opening the tutorial and returning to creation', async ({ page }) => {
    const { unmatchedRequests } = await seedJourney(page)
    await page.goto(createPath, { waitUntil: 'networkidle' })
    const editedPrompt = '镜头沿着海岸缓慢移动，保留我刚刚输入的这句话。'
    await page.getByRole('textbox', { name: '画面描述', exact: true }).fill(editedPrompt)
    await page.getByRole('link', { name: '查看教程', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/guides/coast-letter')
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_journey') || '{}').prompt)).toBe(editedPrompt)
    await page.getByRole('link', { name: '制作这个镜头', exact: true }).first().click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/create')
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue(editedPrompt)
    await expect(page.getByText('已保留你的草稿。应用教程设置会替换画面描述与生成设置，已选素材会保留。', { exact: true })).toBeVisible()
    expect(unmatchedRequests).toEqual([])
  })

  test('blocks creation after an options failure and recovers through the capability retry', async ({ page }) => {
    const { unmatchedRequests } = await seedJourney(page)
    let recover = false
    let recoveredReads = 0
    await page.route('**/api/v1/client/create/options**', async (route) => {
      if (!recover) return route.fulfill({ status: 503, json: { message: '能力暂不可用' } })
      recoveredReads += 1
      await route.fulfill({ json: options })
    })
    await page.goto(createPath, { waitUntil: 'networkidle' })
    const submit = page.getByRole('button', { name: '提交生成任务', exact: true })
    await expect(page.getByRole('alert')).toContainText('生成服务暂时异常，请稍后重新加载。')
    await expect(submit).toBeDisabled()
    recover = true
    await page.getByRole('button', { name: '重新加载生成能力', exact: true }).click()
    await expect(submit).toBeEnabled()
    await expect(page.getByRole('alert')).toHaveCount(0)
    await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue('5')
    expect(recoveredReads).toBeGreaterThan(0)
    expect(unmatchedRequests).toEqual([])
  })

  for (const guide of [
    { id: 'product-light', mode: '多图生成视频' },
    { id: 'scene-transition', mode: '首尾帧生成视频' },
  ]) {
    test(`${guide.id} requires the creator's images instead of treating public artwork as uploaded material`, async ({ page }) => {
      await seedJourney(page)
      await page.goto(`/guides/${guide.id}`, { waitUntil: 'networkidle' })
      await page.getByRole('link', { name: '制作这个镜头', exact: true }).or(page.getByRole('button', { name: '制作这个镜头', exact: true })).first().click()
      await expect(page).toHaveURL((url) => url.pathname === '/app/create')
      await expect(page.getByRole('button', { name: guide.mode, exact: true })).toHaveAttribute('aria-pressed', 'true')
      await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).not.toHaveValue('')
      await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
      await expect(page.getByRole('button', { name: '清除图片', exact: true })).toHaveCount(0)
    })
  }

  test('does not switch an unavailable tutorial mode or accept an unknown tutorial', async ({ page }) => {
    await seedJourney(page)
    await page.route('**/api/v1/client/create/catalog', async (route) => {
      await route.fulfill({ json: { ...catalog, modes: catalog.modes.map((mode) => ({ ...mode, enabled: mode.inputType !== 'text' })) } })
    })
    await page.goto(createPath, { waitUntil: 'networkidle' })
    await expect(page.getByText('当前没有可用生成能力，暂不可提交', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    await expect(page.getByLabel('创作模式', { exact: true }).getByRole('button', { pressed: true })).toHaveCount(0)
    await page.goto('/app/create?guide=not-a-guide&shot=arrival', { waitUntil: 'networkidle' })
    await expect(page.getByText('未找到这个镜头教程', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
  })

  test('refreshes delivery state and preserves the download action after a media error', async ({ page }) => {
    await seedJourney(page)
    let ready = false
    await page.route('**/api/v1/client/tasks/task_journey', async (route) => {
      await route.fulfill({ json: { ...result, deliveryStatus: ready ? 'ready' : 'pending_storage', downloadUrl: ready ? resultURL : '' } })
    })
    await page.goto('/app/works/task_journey', { waitUntil: 'networkidle' })
    const download = page.getByRole('link', { name: '下载成片', exact: true }).or(page.getByRole('button', { name: '下载成片', exact: true }))
    await expect(download).toHaveCount(0)
    await expect(page.getByRole('button', { name: '保存到作品', exact: true })).toHaveCount(0)
    ready = true
    await page.getByRole('button', { name: /刷新|重新加载/ }).first().click()
    await expect(page.getByText('视频预览暂不可用', { exact: true })).toBeVisible()
    await expect(download).toBeVisible()
    await expect(page.getByRole('button', { name: '保存到作品', exact: true })).toBeEnabled()
  })

  test('keeps all five journey pages within the minimum supported mobile width', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'Minimum-width coverage runs in the mobile project.')
    const { unmatchedRequests } = await seedJourney(page)
    await page.route('**/api/v1/client/tasks/task_journey', async (route) => { await route.fulfill({ json: result }) })
    await page.setViewportSize({ width: 320, height: 844 })
    for (const path of ['/', '/guides/coast-letter', createPath, '/app/works/task_journey', '/app/assets?project=prj_visual_coast&folder=视频']) {
      await page.goto(path, { waitUntil: 'networkidle' })
      await expect(page.locator('main h1')).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth), path).toBeLessThanOrEqual(1)
    }
    expect(unmatchedRequests).toEqual([])
  })
})
