import { expect, test, type Page } from '@playwright/test'

// These tests exercise the browser against deterministic API contracts. They do
// not establish supplier execution, wallet settlement, or storage acceptance.
const modes = [
  { id: 'mode_text', inputType: 'text', name: '文生视频', description: '文字生成视频', enabled: true },
  { id: 'mode_multi', inputType: 'multi_image', name: '多图生成视频', description: '参考图片生成视频', enabled: true },
  { id: 'mode_frames', inputType: 'first_last', name: '首尾帧生成视频', description: '首尾画面生成视频', enabled: true },
]

const completedTask = {
  id: 'task_studio_ready', title: '已交付的海边样片', mode: '文生视频', status: '已完成',
  progress: 100, duration: '5 秒', resolution: '1080P', cost: 2.5, time: '刚刚', thumbUrl: '',
}
const pendingTask = { ...completedTask, id: 'task_studio_pending', title: '等待交付的海边样片' }
const videoUrl = 'https://media.sub2video.test/studio-delivery.mp4'

function estimate(durationSec: number) {
  return {
    available: true,
    selection: { carrier: { id: 'carrier_minimax', name: 'MINMAX' }, model: { id: 'model_h3', name: 'H3' } },
    billing: { currency: 'CNY', pricePerSecond: 0.5, estimatedTotal: durationSec * 0.5, durationSec, resolution: '1080P', generateAudio: false },
  }
}

function deferred() {
  let resolve!: () => void
  const promise = new Promise<void>((done) => { resolve = done })
  return { promise, resolve }
}

async function seedStudio(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'studio-contract-access-token')
    localStorage.setItem('s2v_refresh_token', 'studio-contract-refresh-token')
  })
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname.replace('/api/v1', '')
    const responses: Record<string, unknown> = {
      '/auth/me': { id: 'u_studio', email: 'studio@sub2video.test', displayName: '创作测试', workspaceName: '创作测试', role: 'user', balance: 100 },
      '/client/create/catalog': {
        modes,
        carriers: [{ id: 'carrier_minimax', name: 'MINMAX' }],
        models: [{ id: 'model_h3', carrierId: 'carrier_minimax', name: 'H3', description: '视频模型', inputTypes: ['text', 'multi_image', 'first_last'] }],
      },
      '/client/create/options': {
        durations: [5, 10], resolutions: ['1080P'], aspectRatios: ['16:9', '9:16'],
        supportsGeneratedAudio: true, minImages: 1, maxImages: 9, defaultDuration: 5, defaultResolution: '1080P',
      },
      '/client/projects': { items: [], currentProjectId: '' },
      '/client/storage/usage': { usedGb: 0, quotaGb: 5, fileCount: 0, projectCount: 0, trashCount: 0 },
      '/client/wallet': { balance: 100, products: [] },
      '/client/tasks': { items: [], total: 0, page: 1, pageSize: 20, counts: { 全部: 0, 排队中: 0, 生成中: 0, 已完成: 0, 失败: 0 } },
      '/client/overview': {
        workspace: { displayName: '创作测试' }, wallet: { balance: 100 },
        billing: { totalSpent: 0, monthSpent: 0 }, pricing: { available: true, referencePricePerSecond: 0.5, referenceNote: '按秒计费' },
        storage: { usedGb: 0, quotaGb: 5, fileCount: 0 }, tasks: { activeCount: 0, activeTask: null, recentCompleted: [] },
      },
    }
    if (path === '/client/tasks/estimate' && route.request().method() === 'POST') {
      await route.fulfill({ json: estimate((route.request().postDataJSON() as { durationSec: number }).durationSec) })
      return
    }
    // No unmatched API request can reach a real backend or create a paid task.
    if (route.request().method() !== 'GET' || !(path in responses)) {
      await route.fulfill({ status: 503, json: { message: 'This operation is outside the mocked studio contract.' } })
      return
    }
    await route.fulfill({ json: responses[path] })
  })
}

test.describe('commercial studio interactions (mocked frontend only)', () => {
  test.beforeEach(async ({ page }) => {
    await seedStudio(page)
  })

  test('opens each overview entry and waits for required images before estimating', async ({ page }) => {
    for (const mode of modes) {
      await page.goto('/app/overview', { waitUntil: 'networkidle' })
      const entry = page.getByRole('button', { name: new RegExp(mode.name) })
      await expect(entry).toBeVisible()
      await entry.click()
      await expect(page).toHaveURL(new RegExp(`/app/create\\?mode=${mode.inputType}$`))
      await expect(page.getByRole('button', { name: mode.name, exact: true })).toHaveAttribute('aria-pressed', 'true')
      await expect(page.getByLabel('费用预估', { exact: true })).toContainText(mode.inputType === 'text' ? '2.50' : '等待素材')
    }
  })

  test('distinguishes an estimate in progress or failure from a free price, then retries', async ({ page }) => {
    const responseGate = deferred()
    let fail = true
    await page.route('**/api/v1/client/tasks/estimate', async (route) => {
      await responseGate.promise
      if (fail) await route.fulfill({ status: 503, json: { message: '估价服务暂时不可用' } })
      else await route.fulfill({ json: estimate(5) })
    })

    try {
      await page.goto('/app/create?mode=text', { waitUntil: 'domcontentloaded' })
      const cost = page.getByLabel('费用预估', { exact: true })
      await expect(cost).toContainText('预估中')
      await expect(cost).not.toContainText(/¥\s*0\.00/)
      responseGate.resolve()
      await expect(cost).toContainText('暂无法估价')
      await expect(cost).not.toContainText(/¥\s*0\.00/)
      await expect(page.getByText('费用预估失败，请检查网络后重试。确认费用后才能提交。', { exact: true })).toBeVisible()

      fail = false
      await page.getByRole('button', { name: /重新预估|重试估价/ }).click()
      await expect(cost).toContainText('2.50')
      await expect(cost).not.toContainText('暂无法估价')
    } finally {
      responseGate.resolve()
    }
  })

  test('keeps the new duration price after a slower previous estimate arrives', async ({ page }) => {
    const olderResponseGate = deferred()
    let olderRequests = 0
    await page.route('**/api/v1/client/tasks/estimate', async (route) => {
      const { durationSec } = route.request().postDataJSON() as { durationSec: number }
      if (durationSec === 5) {
        olderRequests += 1
        await olderResponseGate.promise
      }
      await route.fulfill({ json: estimate(durationSec) })
    })

    try {
      await page.goto('/app/create?mode=text', { waitUntil: 'domcontentloaded' })
      await expect.poll(() => olderRequests).toBeGreaterThan(0)
      await page.getByRole('combobox', { name: '视频时长', exact: true }).selectOption('10')
      const cost = page.getByLabel('费用预估', { exact: true })
      await expect(cost).toContainText('5.00')
      olderResponseGate.resolve()
      await page.waitForLoadState('networkidle')
      await expect(page.getByRole('combobox', { name: '视频时长', exact: true })).toHaveValue('10')
      await expect(cost).toContainText('5.00')
      await expect(cost).not.toContainText('2.50')
    } finally {
      olderResponseGate.resolve()
    }
  })

  test('uses only a delivered result for video preview and reports media errors', async ({ page }) => {
    await page.route(/\/api\/v1\/client\/tasks(?:\?.*)?$/, async (route) => {
      await route.fulfill({ json: { items: [pendingTask, completedTask], total: 2, page: 1, pageSize: 3, counts: { 全部: 2, 已完成: 2 } } })
    })
    for (const task of [pendingTask, completedTask]) {
      await page.route(`**/api/v1/client/tasks/${task.id}`, async (route) => {
        const ready = task.id === completedTask.id
        await route.fulfill({ json: {
          ...task, durationSeconds: 5, aspectRatio: '16:9', generateAudio: false, savedToProject: false,
          failureReason: '', executionWaiting: false, deliveryStatus: ready ? 'ready' : 'pending_storage',
          downloadUrl: ready ? videoUrl : '', createdAt: '', updatedAt: '',
        } })
      })
    }
    const mediaResponseGate = deferred()
    await page.route(videoUrl, async (route) => {
      await mediaResponseGate.promise
      await route.fulfill({ status: 403, body: 'The result is no longer readable.' })
    })

    try {
      await page.goto('/app/create?mode=text', { waitUntil: 'networkidle' })
      const preview = page.getByRole('region', { name: '预览与结果', exact: true })
      await expect(preview).toBeVisible()
      await expect(preview.locator('video, canvas')).toHaveCount(0)
      await expect(preview.getByRole('button', { name: /播放/ })).toHaveCount(0)

      await page.getByRole('button', { name: new RegExp(pendingTask.title) }).click()
      await expect(preview).toContainText(pendingTask.title)
      await expect(preview.locator('video')).toHaveCount(0)

      await page.getByRole('button', { name: new RegExp(completedTask.title) }).click()
      await expect(preview.locator('video')).toHaveAttribute('src', videoUrl)
      await expect(preview.locator('video')).toHaveAttribute('controls', '')
      mediaResponseGate.resolve()
      await expect(preview.getByText('暂时无法预览此素材', { exact: true })).toBeVisible()
      await expect(preview.locator('video')).toHaveCount(0)
    } finally {
      mediaResponseGate.resolve()
    }
  })

  test('clears a reference image from the keyboard without reopening the picker', async ({ page }) => {
    await page.goto('/app/create?mode=first_last', { waitUntil: 'networkidle' })
    for (const key of ['Enter', 'Space']) {
      await page.getByRole('button', { name: '选择首帧', exact: true }).click()
      const picker = page.getByRole('dialog', { name: '添加或选取素材', exact: true })
      await expect(picker).toBeVisible()
      await picker.getByLabel('参考图片文件', { exact: true }).setInputFiles({
        name: 'frame.png', mimeType: 'image/png',
        buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lN8AAAAASUVORK5CYII=', 'base64'),
      })
      await expect(picker).toHaveCount(0)
      const clear = page.getByRole('button', { name: '清除图片', exact: true })
      await clear.focus()
      await clear.press(key)
      await expect(clear).toHaveCount(0)
      await expect(picker).toHaveCount(0)
      await expect(page.getByRole('button', { name: '选择首帧', exact: true }).getByRole('img', { name: 'frame.png', exact: true })).toHaveCount(0)
    }
  })

  test('includes selected visual references in estimate capability routing', async ({ page }) => {
    const estimates: Array<Record<string, unknown>> = []
    page.on('request', (request) => {
      if (request.method() === 'POST' && request.url().endsWith('/client/tasks/estimate')) {
        estimates.push(request.postDataJSON() as Record<string, unknown>)
      }
    })

    await page.goto('/app/create?mode=multi_image', { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '选择参考图 1', exact: true }).click()
    const picker = page.getByRole('dialog', { name: '添加或选取素材', exact: true })
    await picker.getByLabel('参考图片文件', { exact: true }).setInputFiles({
      name: 'reference.png', mimeType: 'image/png',
      buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lN8AAAAASUVORK5CYII=', 'base64'),
    })
    await expect.poll(() => estimates.some((payload) => {
      const sourceMeta = payload.sourceMeta as { refs?: unknown[] } | undefined
      return Array.isArray(sourceMeta?.refs) && sourceMeta.refs.length === 1
    })).toBe(true)

    await page.goto('/app/create?mode=first_last', { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '选择首帧', exact: true }).click()
    const firstPicker = page.getByRole('dialog', { name: '添加或选取素材', exact: true })
    await firstPicker.getByLabel('参考图片文件', { exact: true }).setInputFiles({
      name: 'first.png', mimeType: 'image/png',
      buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lN8AAAAASUVORK5CYII=', 'base64'),
    })
    await page.getByRole('button', { name: '选择尾帧', exact: true }).click()
    const lastPicker = page.getByRole('dialog', { name: '添加或选取素材', exact: true })
    await lastPicker.getByLabel('参考图片文件', { exact: true }).setInputFiles({
      name: 'last.png', mimeType: 'image/png',
      buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lN8AAAAASUVORK5CYII=', 'base64'),
    })
    await expect.poll(() => estimates.some((payload) => {
      const sourceMeta = payload.sourceMeta as { first?: unknown; last?: unknown } | undefined
      return Boolean(sourceMeta?.first && sourceMeta?.last)
    })).toBe(true)
  })

  test('puts creation inputs before the empty preview without horizontal overflow on small screens', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'This contract covers 390px and the supported minimum width of 320px.')
    for (const width of [390, 320]) {
      await page.setViewportSize({ width, height: 844 })
      for (const mode of modes) {
        await page.goto(`/app/create?mode=${mode.inputType}`, { waitUntil: 'networkidle' })
        const prompt = page.getByRole('textbox', { name: '画面描述', exact: true })
        const preview = page.getByRole('region', { name: '预览与结果', exact: true })
        await expect(prompt).toBeVisible()
        await expect(preview).toBeVisible()
        const inputBounds = await prompt.boundingBox()
        const previewBounds = await preview.boundingBox()
        const submitBounds = await page.getByRole('button', { name: '提交生成任务', exact: true }).boundingBox()
        expect(inputBounds).not.toBeNull()
        expect(previewBounds).not.toBeNull()
        expect(submitBounds).not.toBeNull()
        expect(submitBounds!.height, 'The main creation action must meet the 44px touch target.').toBeGreaterThanOrEqual(44)
        expect(inputBounds!.y + inputBounds!.height).toBeLessThan(previewBounds!.y)
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
        expect(overflow, `${mode.name} at ${width}px must fit the viewport`).toBeLessThanOrEqual(1)
      }
    }
  })
})
