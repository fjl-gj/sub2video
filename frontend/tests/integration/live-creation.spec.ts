import { expect, test } from '@playwright/test'

const liveEnabled = process.env.SUB2VIDEO_LIVE === '1'
const email = process.env.SUB2VIDEO_LIVE_EMAIL || ''
const password = process.env.SUB2VIDEO_LIVE_PASSWORD || ''
const createTasks = process.env.SUB2VIDEO_LIVE_CREATE_TASKS === '1'

test.describe('Sub2Video live creation flow', () => {
  test.skip(!liveEnabled, 'set SUB2VIDEO_LIVE=1 to run against a real frontend and backend')
  test.skip(!email || !password, 'set SUB2VIDEO_LIVE_EMAIL and SUB2VIDEO_LIVE_PASSWORD')

  test('logs in, loads the live catalog, and estimates without charging', async ({ page }) => {
    await page.goto('/auth/login', { waitUntil: 'domcontentloaded' })
    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByLabel('密码', { exact: true }).fill(password)

    const loginButton = page.getByRole('button', { name: '登录', exact: true })
    await expect(loginButton).toHaveCount(1)
    await loginButton.click()
    await expect(page).toHaveURL(/\/app\/overview(?:$|\?)/, { timeout: 30_000 })

    const estimateResponsePromise = page.waitForResponse((response) => (
      response.request().method() === 'POST'
      && response.url().includes('/api/v1/client/tasks/estimate')
    ), { timeout: 30_000 })
    await page.goto('/app/create', { waitUntil: 'domcontentloaded' })
    await expect(page.getByRole('heading', { name: '创作', exact: true })).toBeVisible()

    for (const mode of ['文生视频', '首尾帧生成视频', '多图生成视频']) {
      await expect(page.getByRole('button', { name: mode, exact: true })).toHaveCount(1)
    }

    const unavailable = page.getByText('当前没有可用生成能力，暂不可提交', { exact: true })
    await expect(unavailable).toHaveCount(0)

    const carrierSelect = page.getByLabel('承运商', { exact: true })
    const modelSelect = page.getByLabel('模型', { exact: true })
    await expect(carrierSelect).toHaveCount(1)
    await expect(modelSelect).toHaveCount(1)

    const carrierValues = await carrierSelect.locator('option').evaluateAll((options) => options
      .map((option) => (option as HTMLOptionElement).value)
      .filter(Boolean))
    expect(carrierValues.length).toBeGreaterThan(0)
    await carrierSelect.selectOption(carrierValues[0])

    await expect.poll(async () => {
      return await modelSelect.locator('option').evaluateAll((options) => options
        .map((option) => (option as HTMLOptionElement).value)
        .filter(Boolean))
    }, { timeout: 15_000 }).not.toEqual([])
    const modelValues = await modelSelect.locator('option').evaluateAll((options) => options
      .map((option) => (option as HTMLOptionElement).value)
      .filter(Boolean))
    await modelSelect.selectOption(modelValues[0])

    const prompt = page.locator('textarea[placeholder^="描述主体动作"]')
    await expect(prompt).toHaveCount(1)
    await prompt.fill('一辆白色电动车沿海边公路缓慢前进，镜头平稳跟随，阳光自然。')

    const estimateResponse = await estimateResponsePromise
    expect(estimateResponse.ok()).toBeTruthy()
    const estimate = await estimateResponse.json() as {
      available?: boolean
      quoteId?: string
      billing?: { estimatedTotal?: number; pricePerSecond?: number }
      selection?: { carrier?: unknown; model?: unknown }
    }

    const assertEstimate = (value: typeof estimate) => {
      expect(value.available).toBe(true)
      expect(value.billing?.estimatedTotal).toBeGreaterThan(0)
      expect(value.billing?.pricePerSecond).toBeGreaterThan(0)
      expect(value.quoteId).toBeUndefined()
      expect(value.selection?.carrier).toBeTruthy()
      expect(value.selection?.model).toBeTruthy()
    }
    assertEstimate(estimate)

    for (const mode of ['首尾帧生成视频', '多图生成视频']) {
      const modeEstimatePromise = page.waitForResponse((response) => (
        response.request().method() === 'POST'
        && response.url().includes('/api/v1/client/tasks/estimate')
        && response.ok()
      ), { timeout: 30_000 })
      const modeButton = page.getByRole('button', { name: mode, exact: true })
      await expect(modeButton).toHaveCount(1)
      await modeButton.click()
      const modeEstimateResponse = await modeEstimatePromise
      expect(modeEstimateResponse.ok()).toBeTruthy()
      assertEstimate(await modeEstimateResponse.json() as typeof estimate)
    }

    const visibleText = await page.locator('body').innerText()
    expect(visibleText.toLowerCase()).not.toContain('autodl')
    expect(visibleText.toLowerCase()).not.toContain('workflow')
    expect(visibleText.toLowerCase()).not.toContain('supplier')

    if (!createTasks) {
      return
    }

    const submitResponsePromise = page.waitForResponse((response) => (
      response.request().method() === 'POST'
      && response.url().endsWith('/api/v1/client/tasks')
    ), { timeout: 30_000 })
    const submitButton = page.getByRole('button', { name: '提交生成任务', exact: true })
    await expect(submitButton).toBeEnabled()
    await submitButton.click()
    const submitResponse = await submitResponsePromise
    expect(submitResponse.status()).toBeGreaterThanOrEqual(200)
    expect(submitResponse.status()).toBeLessThan(300)
    await expect(page).toHaveURL(/\/app\/works(?:$|\?)/, { timeout: 30_000 })
  })
})
