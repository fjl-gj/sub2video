import { expect, test, type Page } from '@playwright/test'

// Frontend fault-injection contracts only. These responses do not establish
// provider availability, database correctness or a successful delivery.
const catalog = {
  modes: [{ id: 'mode_text', inputType: 'text', name: '文生视频', description: '描述一个镜头', enabled: true }],
  carriers: [{ id: 'carrier_h3', name: 'MINMAX' }],
  models: [
    { id: 'model_selected', carrierId: 'carrier_h3', name: '已选模型', description: '测试模型', inputTypes: ['text'] },
    { id: 'model_other', carrierId: 'carrier_h3', name: '另一模型', description: '不能自动替换已选模型', inputTypes: ['text'] },
  ],
}
const options = {
  durations: [5, 10], resolutions: ['768P'], aspectRatios: ['16:9'], supportsGeneratedAudio: false,
  minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P',
}
const privateDiagnostic = 'private-diagnostic-fixture: supplier=hidden; workflowCode=hidden; token=not-a-real-token'
type OptionsFailure = 'combination' | 'supply' | 'network' | 'server' | null

async function mockCreation(page: Page, optionsFailure: OptionsFailure = null) {
  const state = {
    optionsFailure,
    storageUnavailable: false,
    optionModelIds: [] as string[],
    estimates: [] as Record<string, unknown>[],
    taskPosts: 0,
    unexpectedRequests: [] as string[],
  }
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'availability-mock-access')
    localStorage.setItem('s2v_refresh_token', 'availability-mock-refresh')
    localStorage.setItem('sub2video.create-draft.u_availability', JSON.stringify({
      version: 4, workflowInputType: 'text', prompt: '保持这个镜头的描述', carrierId: 'carrier_h3', modelId: 'model_selected',
      durationSec: '5', resolution: '768P', ratio: '16:9', audioMode: 'none',
      firstFrame: { uploaded: false, assetId: '', picking: false },
      lastFrame: { uploaded: false, assetId: '', picking: false }, extraFrames: [], multiFrames: [],
    }))
  })
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname.replace('/api/v1', '')
    if (path === '/client/create/options') {
      state.optionModelIds.push(url.searchParams.get('modelId') || '')
      if (state.optionsFailure === 'network') return route.abort('connectionreset')
      if (state.optionsFailure) {
        const code = state.optionsFailure === 'combination' ? 'UNAVAILABLE_COMBINATION'
          : state.optionsFailure === 'supply' ? 'SUPPLY_UNAVAILABLE' : 'INTERNAL'
        const status = state.optionsFailure === 'combination' ? 400 : state.optionsFailure === 'supply' ? 503 : 500
        // UI must use stable public codes, never display arbitrary diagnostics.
        return route.fulfill({ status, json: { code, message: privateDiagnostic } })
      }
      return route.fulfill({ json: options })
    }
    if (path === '/client/tasks/estimate' && request.method() === 'POST') {
      const payload = request.postDataJSON() as Record<string, unknown>
      state.estimates.push(payload)
      if (state.storageUnavailable) return route.fulfill({ status: 503, json: { code: 'STORAGE_UNAVAILABLE', message: privateDiagnostic } })
      return route.fulfill({ json: {
        available: true,
        selection: { carrier: catalog.carriers[0], model: { id: 'model_selected', name: '已选模型' } },
        billing: { currency: 'CNY', pricePerSecond: 0.03, estimatedTotal: 0.15, durationSec: 5, resolution: '768P', generateAudio: false },
      } })
    }
    if (path === '/client/tasks' && request.method() === 'POST') {
      state.taskPosts += 1
      return route.abort('blockedbyclient')
    }
    const reads: Record<string, unknown> = {
      '/auth/me': { id: 'u_availability', email: 'availability@example.test', displayName: '可用性测试', role: 'user', balance: 25, workspaceName: '测试空间' },
      '/settings/public': { registrationEnabled: false, emailVerifyEnabled: false, passwordResetEnabled: false },
      '/client/create/catalog': catalog,
      '/client/projects': { items: [], currentProjectId: '' },
      '/client/storage/usage': { usedGb: 0, quotaGb: 5, fileCount: 0, projectCount: 0, trashCount: 0 },
      '/client/wallet': { balance: 25, products: [] },
      '/client/tasks': { items: [], total: 0, page: 1, pageSize: 3, counts: { 全部: 0, 排队中: 0, 生成中: 0, 已完成: 0, 失败: 0 } },
    }
    if (request.method() === 'GET' && path in reads) return route.fulfill({ json: reads[path] })
    state.unexpectedRequests.push(`${request.method()} ${path}`)
    return route.abort('blockedbyclient')
  })
  return state
}

async function expectPreservedChoice(page: Page) {
  await expect(page.getByRole('combobox', { name: '模型', exact: true })).toHaveValue('model_selected')
  await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue('保持这个镜头的描述')
  await expect(page.getByText(privateDiagnostic, { exact: false })).toHaveCount(0)
}

test.describe('creation availability and recovery (mocked frontend only)', () => {
  for (const scenario of [
    { failure: 'combination' as const, label: 'a selected model without usable supply', message: '当前模型或参数组合暂不可用，请调整选择或稍后重新加载。' },
    { failure: 'supply' as const, label: 'temporarily unavailable generation capacity', message: '当前生成能力暂不可用，请稍后重新加载，或联系帮助与工单。' },
  ]) {
    test(`explains ${scenario.label} without blaming the network or switching models`, async ({ page }) => {
      const state = await mockCreation(page, scenario.failure)
      await page.goto('/app/create')
      await expect(page.getByRole('alert')).toContainText(scenario.message)
      await expect(page.getByRole('alert')).not.toContainText('检查网络')
      await expectPreservedChoice(page)
      await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
      await expect(page.getByLabel('费用预估', { exact: true })).not.toContainText('¥')
      await expect(page.getByText('当前没有可用生成能力，暂不可提交', { exact: true })).toHaveCount(0)
      expect(state.optionModelIds.length).toBeGreaterThan(0)
      expect(state.optionModelIds.every((id) => id === 'model_selected')).toBe(true)
      expect(state.estimates).toHaveLength(0)
      expect(state.taskPosts).toBe(0)
      expect(state.unexpectedRequests).toEqual([])
    })
  }

  test('recovers a network failure through explicit retry while retaining the draft and model', async ({ page }) => {
    const state = await mockCreation(page, 'network')
    await page.goto('/app/create')
    await expect(page.getByRole('alert')).toContainText('无法连接生成服务，请检查网络后重新加载。')
    await expect(page.getByText('当前没有可用生成能力，暂不可提交', { exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    expect(state.estimates).toHaveLength(0)
    state.optionsFailure = null
    await page.getByRole('button', { name: '重新加载生成能力', exact: true }).click()
    await expect(page.getByRole('alert')).toHaveCount(0)
    await expect(page.getByLabel('费用预估', { exact: true })).toContainText('¥ 0.15')
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeEnabled()
    await expectPreservedChoice(page)
    expect(state.optionModelIds.every((id) => id === 'model_selected')).toBe(true)
    expect(state.estimates.length).toBeGreaterThan(0)
    expect(state.estimates.every((payload) => payload.modelId === 'model_selected')).toBe(true)
    expect(state.taskPosts).toBe(0)
    expect(state.unexpectedRequests).toEqual([])
  })

  test('distinguishes a server error from a network or supply failure and hides diagnostics', async ({ page }) => {
    const state = await mockCreation(page, 'server')
    await page.goto('/app/create')
    await expect(page.getByRole('alert')).toContainText('生成服务暂时异常，请稍后重新加载。')
    await expect(page.getByRole('alert')).not.toContainText('检查网络')
    await expect(page.getByText('当前没有可用生成能力，暂不可提交', { exact: true })).toHaveCount(0)
    await expectPreservedChoice(page)
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    expect(state.estimates).toHaveLength(0)
    expect(state.taskPosts).toBe(0)
    expect(state.unexpectedRequests).toEqual([])
  })

  test('explains storage unavailability at estimate and restores only the estimate on retry', async ({ page }) => {
    const state = await mockCreation(page)
    // Storage readiness is checked by POST /client/tasks/estimate after the
    // route is resolved; GET options alone is not proof of usable storage.
    state.storageUnavailable = true
    await page.goto('/app/create')
    await expect(page.getByText('素材与成片存储暂不可用，请稍后重试。本次未提交任务。', { exact: true })).toBeVisible()
    await expect(page.getByLabel('费用预估', { exact: true })).toContainText('暂无法估价')
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    await expectPreservedChoice(page)
    state.storageUnavailable = false
    await page.getByRole('button', { name: '重新预估费用', exact: true }).click()
    await expect(page.getByLabel('费用预估', { exact: true })).toContainText('¥ 0.15')
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeEnabled()
    await expect(page.getByText('素材与成片存储暂不可用，请稍后重试。本次未提交任务。', { exact: true })).toHaveCount(0)
    await expectPreservedChoice(page)
    expect(state.estimates.every((payload) => payload.modelId === 'model_selected')).toBe(true)
    expect(state.taskPosts).toBe(0)
    expect(state.unexpectedRequests).toEqual([])
  })
})
