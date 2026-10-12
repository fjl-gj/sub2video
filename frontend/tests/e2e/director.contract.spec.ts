import { readFile } from 'node:fs/promises'
import { expect, test, type Page } from '@playwright/test'
import type { DirectorBoard, DirectorShot } from '../../src/api/director'
import { seedSurfaceData } from './fixtures/surface-data'

// Stateful API fixtures prove browser contracts only, not database persistence,
// wallet atomicity, supplier generation, or playable media delivery.
const projectId = 'prj_visual_product'
const directorPath = `/app/projects/${projectId}/director`
const user = { id: 'u_director', email: 'director@example.test', displayName: '导演台测试', workspaceName: '测试工作室', role: 'user', balance: 256 }
const catalog = {
  modes: [{ id: 'mode_text', inputType: 'text', name: '文生视频', enabled: true, description: '文字生成' }],
  carriers: [{ id: 'car_minimax', name: 'MINMAX' }],
  models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', description: '视频模型', inputTypes: ['text'] }],
}
const options = { durations: [5, 10], resolutions: ['768P'], aspectRatios: ['16:9', '9:16'], supportsGeneratedAudio: false, minImages: 0, maxImages: 0, defaultDuration: 5, defaultResolution: '768P' }
function shot(overrides: Partial<DirectorShot> = {}): DirectorShot {
  return { id: 'shot_coast', title: '海岸清晨', prompt: '清晨的海岸，平稳推进镜头，人物走向海边。', modeId: 'mode_text', carrierId: 'car_minimax', modelId: 'mdl_h3', durationSec: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false, sourceMeta: {}, candidateTaskIds: [], selectedTaskId: '', ...overrides }
}
const result = {
  id: 'task_director_ready', title: '清晨 · 第一版', mode: '文生视频', modeId: 'mode_text', carrierId: 'car_minimax', modelId: 'mdl_h3',
  status: '已完成', progress: 100, duration: '5 秒', durationSeconds: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false,
  cost: 2.5, refundAmount: 0, refundedAt: '', thumbUrl: '', savedToProject: false, reviewRequired: false, canCancel: false,
  failureReason: '', executionWaiting: false, deliveryStatus: 'ready', downloadUrl: 'https://director-media.example.test/expired.mp4',
  createdAt: '2026-10-05T04:00:00Z', updatedAt: '2026-10-05T04:00:05Z',
}

async function seedDirector(page: Page, shots: DirectorShot[] = [shot()]) {
  const fixture = await seedSurfaceData(page)
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'director-test-access')
    localStorage.setItem('s2v_refresh_token', 'director-test-refresh')
  })
  await page.route('**/api/v1/auth/me', async (route) => { await route.fulfill({ json: user }) })
  await page.route('**/api/v1/client/create/catalog', async (route) => { await route.fulfill({ json: catalog }) })
  await page.route('**/api/v1/client/create/options**', async (route) => { await route.fulfill({ json: options }) })
  await page.route('**/api/v1/client/session/current-project', async (route) => { await route.fulfill({ json: { currentProjectId: projectId } }) })
  await page.route('https://director-media.example.test/**', async (route) => { await route.fulfill({ status: 403, body: 'Expired test media; not a real delivered video' }) })
  const state = { board: { projectId, revision: 1, shots: structuredClone(shots), updatedAt: '2026-10-05T04:00:00Z' } as DirectorBoard, writes: 0, forceConflict: false, creates: [] as Record<string, unknown>[] }
  await page.route(`**/api/v1/client/projects/${projectId}/director-board`, async (route) => {
    if (route.request().method() === 'PUT') {
      const payload = route.request().postDataJSON() as { expectedRevision: number; shots: DirectorShot[] }
      if (state.forceConflict || payload.expectedRevision !== state.board.revision) {
        await route.fulfill({ status: 409, json: { code: 'DIRECTOR_BOARD_CONFLICT', message: '版本冲突' } }); return
      }
      state.writes++
      state.board = { ...state.board, revision: state.board.revision + 1, shots: structuredClone(payload.shots) }
    }
    await route.fulfill({ json: state.board })
  })
  await page.route(/\/api\/v1\/client\/tasks\/task_director_[^/]+$/, async (route) => {
    const id = new URL(route.request().url()).pathname.split('/').pop()!
    const changes = id.endsWith('failed') ? { title: '清晨 · 失败版本', status: '失败', deliveryStatus: 'not_ready', downloadUrl: '', failureReason: '生成失败', refundAmount: 2.5 } : id.endsWith('pending') ? { title: '清晨 · 待交付版本', deliveryStatus: 'pending_storage', downloadUrl: '' } : {}
    await route.fulfill({ json: { ...result, id, ...changes } })
  })
  await page.route(/\/api\/v1\/client\/tasks$/, async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    state.creates.push(route.request().postDataJSON() as Record<string, unknown>)
    await route.fulfill({ status: 201, json: { task: result, order: { id: 'order_director', amount: 2.5, currency: 'CNY', status: 'paid' }, wallet: { balance: 253.5 } } })
  })
  return { ...fixture, state }
}

test.describe('director workspace browser contracts (mocked API)', () => {
  test('enters from a project, saves ordered shots and restores them on refresh', async ({ page }) => {
    const { state, unmatchedRequests } = await seedDirector(page, [])
    await page.goto(`/app/assets?project=${projectId}`, { waitUntil: 'networkidle' })
    await page.getByRole('link', { name: '导演台', exact: true }).or(page.getByRole('button', { name: '导演台', exact: true })).click()
    await expect(page).toHaveURL(directorPath)
    await page.getByRole('button', { name: '新增镜头', exact: true }).click()
    await page.getByLabel('镜头名称', { exact: true }).fill('开场全景')
    await page.getByLabel('画面描述', { exact: true }).fill('清晨，山路延伸到海边。')
    await page.getByRole('button', { name: '新增镜头', exact: true }).click()
    await page.getByLabel('镜头名称', { exact: true }).fill('人物近景')
    await page.getByRole('button', { name: '上移人物近景', exact: true }).click()
    await page.getByRole('button', { name: '保存镜头板', exact: true }).click()
    await expect(page.getByText('镜头板已保存', { exact: true })).toBeVisible()
    expect(state.board.shots.map((item) => item.title)).toEqual(['人物近景', '开场全景'])
    expect(state.writes).toBe(1)
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.getByLabel('镜头名称', { exact: true })).toHaveValue('人物近景')
    expect(state.creates).toHaveLength(0)
    expect(unmatchedRequests).toEqual([])
  })

  test('keeps unsaved edits on conflict and reloads only after an explicit discard decision', async ({ page }) => {
    const { state } = await seedDirector(page)
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    await page.getByLabel('画面描述', { exact: true }).fill('这一段本地修改不能静默丢失。')
    state.forceConflict = true
    await page.getByRole('button', { name: '保存镜头板', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('你的编辑仍保留')
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue('这一段本地修改不能静默丢失。')
    page.once('dialog', (dialog) => dialog.dismiss())
    await page.getByRole('button', { name: '重新加载镜头板', exact: true }).click()
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue('这一段本地修改不能静默丢失。')
    state.forceConflict = false
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: '重新加载镜头板', exact: true }).click()
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveValue(shot().prompt)
    expect(state.writes).toBe(0)
    expect(state.creates).toHaveLength(0)
  })

  test('selects only a delivered candidate and preserves the choice after refresh', async ({ page }) => {
    const { state, unmatchedRequests } = await seedDirector(page, [shot({ candidateTaskIds: ['task_director_ready', 'task_director_pending', 'task_director_failed'] })])
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    await expect(page.getByRole('article', { name: '候选 清晨 · 待交付版本', exact: true }).getByText('正在准备交付，完成后可选用。')).toBeVisible()
    await expect(page.getByRole('article', { name: '候选 清晨 · 失败版本', exact: true })).toContainText('已退款 ¥2.50')
    await expect(page.getByRole('button', { name: '采用这一版', exact: true })).toHaveCount(1)
    await page.getByRole('button', { name: '采用这一版', exact: true }).click()
    await expect(page.getByRole('button', { name: '已选用这一版', exact: true })).toBeDisabled()
    expect(state.board.shots[0].selectedTaskId).toBe('task_director_ready')
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.getByText('当前选用：清晨 · 第一版', { exact: true })).toBeVisible()
    expect(state.creates).toHaveLength(0)
    expect(unmatchedRequests).toEqual([])
  })

  test('removes a shot without cancelling or deleting its tasks', async ({ page }) => {
    const { state, unmatchedRequests } = await seedDirector(page, [shot({ candidateTaskIds: ['task_director_ready'], selectedTaskId: 'task_director_ready' })])
    const mutations: string[] = []
    page.on('request', (request) => { if (request.url().includes('/api/v1/') && request.method() !== 'GET') mutations.push(`${request.method()} ${new URL(request.url()).pathname}`) })
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    page.once('dialog', async (dialog) => { expect(dialog.message()).toContain('关联任务和已交付视频会保留'); await dialog.accept() })
    await page.getByRole('button', { name: '移除镜头', exact: true }).click()
    await page.getByRole('button', { name: '保存镜头板', exact: true }).click()
    await expect(page.getByText('镜头板已保存', { exact: true })).toBeVisible()
    expect(state.board.shots).toEqual([])
    expect(mutations).toEqual([`PUT /api/v1/client/projects/${projectId}/director-board`])
    expect(unmatchedRequests).toEqual([])
  })

  test('requires explicit application before replacing the normal draft, then submits with director context', async ({ page }) => {
    const { state, unmatchedRequests } = await seedDirector(page)
    await page.addInitScript(() => {
      if (!localStorage.getItem('sub2video.create-draft.u_director')) localStorage.setItem('sub2video.create-draft.u_director', JSON.stringify({ version: 3, workflowInputType: 'text', prompt: '普通创作未保存的想法', ratio: '9:16', durationSec: '10', resolution: '768P', firstFrame: { uploaded: false, assetId: '', picking: false }, lastFrame: { uploaded: false, assetId: '', picking: false }, extraFrames: [], multiFrames: [] }))
    })
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '前往创作', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/create' && url.searchParams.get('directorShot') === 'shot_coast')
    const prompt = page.getByRole('textbox', { name: '画面描述', exact: true })
    await expect(prompt).toHaveValue('普通创作未保存的想法')
    expect(state.creates).toHaveLength(0)
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await expect(prompt).toHaveValue(shot().prompt)
    await expect(page.getByLabel('视频时长', { exact: true })).toHaveValue('5')
    await page.getByRole('button', { name: '提交生成任务', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/works/task_director_ready')
    expect(state.creates).toHaveLength(1)
    expect(state.creates[0]).toMatchObject({ directorContext: { projectId, shotId: 'shot_coast', expectedRevision: 1 }, prompt: shot().prompt, durationSec: 5 })
    expect(JSON.stringify(state.creates[0])).not.toMatch(/quoteId|supplier|workflowCode|apiModelCode|executionType|upstream/i)
    const draft = await page.evaluate(() => JSON.parse(localStorage.getItem('sub2video.create-draft.u_director') || 'null'))
    expect(draft?.prompt).toBe('普通创作未保存的想法')
    expect(unmatchedRequests).toEqual([])
  })

  test('exports a shot checklist without media URLs and does not trigger generation', async ({ page }) => {
    const { state } = await seedDirector(page, [shot({ candidateTaskIds: ['task_director_ready'], selectedTaskId: 'task_director_ready' })])
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    const downloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: '导出镜头清单', exact: true }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toBe(`director-${projectId}.json`)
    const document = JSON.parse(await readFile((await download.path())!, 'utf8'))
    expect(document).toMatchObject({ format: 'sub2video-director-board', version: 1, projectId, shots: [{ id: 'shot_coast', selectedTaskId: 'task_director_ready' }] })
    expect(JSON.stringify(document)).not.toMatch(/https?:|supplier|workflowCode|apiModelCode|executionType|downloadUrl/)
    expect(state.creates).toHaveLength(0)
  })

  test('saves shot preparation without balance or an available estimate and preserves unrelated shots and candidates', async ({ page }) => {
    const other = shot({ id: 'shot_other', title: '远景收尾', prompt: '保留另一个镜头', candidateTaskIds: ['task_director_ready'], selectedTaskId: 'task_director_ready' })
    const { state } = await seedDirector(page, [shot({ candidateTaskIds: ['task_director_pending'] }), other])
    await page.route('**/api/v1/auth/me', async (route) => { await route.fulfill({ json: { ...user, balance: 0 } }) })
    await page.route('**/api/v1/client/wallet', async (route) => { await route.fulfill({ json: { balance: 0, products: [] } }) })
    await page.route('**/api/v1/client/tasks/estimate', async (route) => {
      await route.fulfill({ status: 503, json: { code: 'STORAGE_UNAVAILABLE', message: '对象存储暂不可用' } })
    })
    await page.goto(`/app/create?directorProject=${projectId}&directorShot=shot_coast`, { waitUntil: 'networkidle' })
    await expect(page.getByRole('button', { name: '保存到镜头', exact: true })).toBeDisabled()
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await page.getByRole('textbox', { name: '画面描述', exact: true }).fill('准备阶段修改描述，暂不生成。')
    await page.getByLabel('视频时长', { exact: true }).selectOption('10')
    await page.getByLabel('画幅比例', { exact: true }).selectOption('9:16')
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    await expect(page.getByRole('button', { name: '保存到镜头', exact: true })).toBeEnabled()
    await page.getByRole('button', { name: '保存到镜头', exact: true }).click()
    await expect(page.getByText('镜头素材与参数已保存，未创建任务或扣费。', { exact: true })).toBeVisible()
    expect(state.board.shots[0]).toMatchObject({ prompt: '准备阶段修改描述，暂不生成。', durationSec: 10, aspectRatio: '9:16', candidateTaskIds: ['task_director_pending'] })
    expect(state.board.shots[1]).toEqual(other)
    expect(state.writes).toBe(1)
    expect(state.creates).toHaveLength(0)
    await page.evaluate(() => document.fonts.ready)
    await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }))
    await expect(page).toHaveScreenshot('director-save-inputs.png', { animations: 'disabled' })
  })

  test('keeps preparation on a save conflict until the latest board is explicitly read and reapplied', async ({ page }) => {
    const { state } = await seedDirector(page)
    await page.goto(`/app/create?directorProject=${projectId}&directorShot=shot_coast`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await page.getByRole('textbox', { name: '画面描述', exact: true }).fill('需要恢复的本地镜头编辑')
    const external = shot({ id: 'shot_external', title: '另一个页面新增', prompt: '不能被过期保存抹掉' })
    state.board.shots.push(external)
    state.board.revision++
    await page.getByRole('button', { name: '保存到镜头', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('本地编辑仍保留')
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue('需要恢复的本地镜头编辑')
    await expect(page.getByRole('button', { name: '保存到镜头', exact: true })).toBeDisabled()
    expect(state.writes).toBe(0)
    expect(state.board.shots[1]).toEqual(external)
    await page.getByRole('button', { name: '重新读取镜头', exact: true }).click()
    await expect(page.getByRole('button', { name: '重新应用镜头', exact: true })).toBeEnabled()
    await expect(page.getByRole('button', { name: '保存到镜头', exact: true })).toBeDisabled()
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: '重新应用镜头', exact: true }).click()
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue('需要恢复的本地镜头编辑')
    await page.getByRole('button', { name: '保存到镜头', exact: true }).click()
    await expect(page.getByText('镜头素材与参数已保存，未创建任务或扣费。', { exact: true })).toBeVisible()
    expect(state.board.revision).toBe(3)
    expect(state.board.shots[0]!.prompt).toBe('需要恢复的本地镜头编辑')
    expect(state.board.shots[1]).toEqual(external)
    expect(state.writes).toBe(1)
    expect(state.creates).toHaveLength(0)
  })

  test('replays the frozen director request after a lost response even if the shot was removed', async ({ page }) => {
    const { state } = await seedDirector(page)
    const requests: { key: string; payload: Record<string, unknown> }[] = []
    await page.route(/\/api\/v1\/client\/tasks$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      requests.push({ key: route.request().headers()['idempotency-key'], payload: route.request().postDataJSON() })
      if (requests.length === 1) { await route.fulfill({ status: 503, json: { message: 'Unknown result in test' } }); return }
      await route.fulfill({ status: 200, json: { task: result, order: { id: 'order_director', amount: 2.5, currency: 'CNY', status: 'paid' }, wallet: { balance: 253.5 } } })
    })
    await page.goto(`/app/create?directorProject=${projectId}&directorShot=shot_coast`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await page.getByRole('button', { name: '提交生成任务', exact: true }).click()
    await expect(page.getByRole('button', { name: '确认原提交结果', exact: true })).toBeVisible()
    state.board = { ...state.board, revision: 3, shots: [] }
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.getByRole('button', { name: '确认原提交结果', exact: true })).toBeEnabled()
    await page.getByRole('button', { name: '确认原提交结果', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/app/works/task_director_ready')
    expect(requests).toHaveLength(2)
    expect(requests[0].key).toBeTruthy()
    expect(requests[1]).toEqual(requests[0])
    expect(requests[1].payload.directorContext).toEqual({ projectId, shotId: 'shot_coast', expectedRevision: 1 })
    await expect(page.getByRole('button', { name: '返回导演台', exact: true })).toBeVisible()
    expect(await page.evaluate(() => localStorage.getItem('sub2video.pending-creation.u_director'))).toBeNull()
  })

  test('requires another explicit application after a rejected board revision and retains shot edits', async ({ page }) => {
    const { state } = await seedDirector(page)
    await page.route(/\/api\/v1\/client\/tasks$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      state.creates.push(route.request().postDataJSON())
      state.board.revision++
      await route.fulfill({ status: 409, json: { code: 'DIRECTOR_BOARD_CONFLICT', message: '版本冲突' } })
    })
    await page.goto(`/app/create?directorProject=${projectId}&directorShot=shot_coast`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await page.getByRole('textbox', { name: '画面描述', exact: true }).fill('导演创作尚未提交的修改')
    await page.getByRole('button', { name: '提交生成任务', exact: true }).click()
    await expect(page.getByRole('button', { name: '应用镜头到创作', exact: true })).toBeEnabled()
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    expect(await page.evaluate(() => localStorage.getItem('sub2video.pending-creation.u_director'))).toBeNull()
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await expect(page.getByRole('textbox', { name: '画面描述', exact: true })).toHaveValue('导演创作尚未提交的修改')
    expect(state.creates).toHaveLength(1)
  })

  test('does not silently switch a shot whose original mode is unavailable', async ({ page }) => {
    const { state } = await seedDirector(page, [shot({ modeId: 'mode_removed' })])
    await page.goto(`/app/create?directorProject=${projectId}&directorShot=shot_coast`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('这个镜头的创作模式暂不可用')
    await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeDisabled()
    expect(state.creates).toHaveLength(0)
  })

  test('clears a director request that is definitively rejected because its project was removed', async ({ page }) => {
    await seedDirector(page)
    await page.route(/\/api\/v1\/client\/tasks$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      await route.fulfill({ status: 404, json: { code: 'NOT_FOUND', message: '作品不存在或已移入回收站' } })
    })
    await page.goto(`/app/create?directorProject=${projectId}&directorShot=shot_coast`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
    await page.getByRole('button', { name: '提交生成任务', exact: true }).click()
    await expect(page.getByRole('button', { name: '应用镜头到创作', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '确认原提交结果', exact: true })).toHaveCount(0)
    expect(await page.evaluate(() => localStorage.getItem('sub2video.pending-creation.u_director'))).toBeNull()
  })

  test('lets logout leave a dirty board without exposing the former account contents', async ({ page }) => {
    const { state } = await seedDirector(page)
    await page.route('**/api/v1/auth/logout', async (route) => { await route.fulfill({ json: { ok: true } }) })
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    await page.getByLabel('画面描述', { exact: true }).fill('这个账号的私有未保存内容')
    let confirmations = 0
    page.on('dialog', async (dialog) => { confirmations++; await dialog.dismiss() })
    await page.getByRole('button', { name: '用户菜单', exact: true }).click()
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === '/auth/login')
    await expect(page.getByLabel('画面描述', { exact: true })).toHaveCount(0)
    expect(confirmations).toBe(0)
    expect(state.writes).toBe(0)
    expect(await page.evaluate(() => localStorage.getItem('s2v_access_token'))).toBeNull()
  })

  for (const mode of ['first_last', 'multi_image'] as const) {
    test(`restores ${mode} upload references from the server board with authenticated previews`, async ({ page }) => {
      const sourceMeta = mode === 'first_last'
        ? { first: { type: 'upload' as const, id: 'upl_first' }, last: { type: 'upload' as const, id: 'upl_last' } }
        : { refs: [{ type: 'upload' as const, id: 'upl_first' }, { type: 'upload' as const, id: 'upl_last' }] }
      const { state, unmatchedRequests } = await seedDirector(page, [shot({ modeId: `mode_${mode}`, sourceMeta })])
      await page.route('**/api/v1/client/create/catalog', async (route) => {
        await route.fulfill({ json: { ...catalog, modes: [{ id: `mode_${mode}`, inputType: mode, name: mode === 'first_last' ? '首尾帧生成视频' : '多图生成视频', enabled: true }], models: [{ ...catalog.models[0], inputTypes: [mode] }] } })
      })
      await page.route('**/api/v1/client/create/options**', async (route) => {
        await route.fulfill({ json: { ...options, supportsGeneratedAudio: true, minImages: mode === 'first_last' ? 2 : 1, maxImages: 9 } })
      })
      const previewRequests: string[] = []
      await page.route('**/api/v1/client/uploads/upl_*', async (route) => {
        const url = new URL(route.request().url())
        expect(url.searchParams.get('access_token')).toBe('director-test-access')
        previewRequests.push(url.pathname)
        await route.fulfill({ contentType: 'image/svg+xml', body: '<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" fill="#777"/></svg>' })
      })
      await page.goto(`/app/create?directorProject=${projectId}&directorShot=shot_coast`, { waitUntil: 'networkidle' })
      await page.getByRole('button', { name: '应用镜头到创作', exact: true }).click()
      const slot = page.getByRole('button', { name: mode === 'first_last' ? '选择首帧' : '选择参考图 1', exact: true })
      await expect(slot.locator('img')).toBeVisible()
      await expect.poll(() => slot.locator('img').evaluate((element: HTMLImageElement) => element.complete && element.naturalWidth > 0)).toBe(true)
      await expect(page.getByLabel('音频设置', { exact: true }).locator('option[value="reference"]')).toHaveCount(0)
      await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeEnabled()
      expect(previewRequests).toContain('/api/v1/client/uploads/upl_first')
      expect(state.creates).toHaveLength(0)
      expect(unmatchedRequests).toEqual([])
    })
  }

  test('links an existing task through the account task picker without creating a task', async ({ page }) => {
    const { state, unmatchedRequests } = await seedDirector(page)
    let links = 0
    await page.route(`**/api/v1/client/projects/${projectId}/director-board/shots/shot_coast/candidates`, async (route) => {
      expect(route.request().method()).toBe('POST')
      expect(route.request().postDataJSON()).toEqual({ taskId: 'tsk_visual_coast' })
      links++
      state.board.shots[0].candidateTaskIds = ['tsk_visual_coast']
      state.board.revision++
      await route.fulfill({ json: state.board })
    })
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '关联已有任务', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '关联已有任务', exact: true })
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: '加入候选', exact: true }).first().click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByRole('article', { name: '候选 海岸清晨 · 镜头 03', exact: true })).toBeVisible()
    expect(links).toBe(1)
    expect(state.creates).toHaveLength(0)
    expect(unmatchedRequests).toEqual([])
  })

  test('renders the shot workspace across phone, tablet and desktop widths', async ({ page }, testInfo) => {
    await seedDirector(page, [shot({ candidateTaskIds: ['task_director_ready', 'task_director_pending'] }), shot({ id: 'shot_person', title: '人物近景', prompt: '人物停下脚步，望向海面。' })])
    await page.goto(directorPath, { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: '秋日香氛 · 导演台', exact: true })).toBeVisible()
    for (const width of (testInfo.project.name === 'mobile' ? [320, 390] : [768, 1024, 1440, 1920])) {
      await page.setViewportSize({ width, height: 900 })
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
      const edit = page.getByLabel('镜头名称', { exact: true })
      await expect(edit).toBeVisible()
      const bounds = await edit.boundingBox()
      expect(bounds!.width).toBeGreaterThan(180)
      expect(bounds!.x).toBeGreaterThanOrEqual(0)
      expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(width)
    }
    await page.setViewportSize(testInfo.project.name === 'mobile' ? { width: 390, height: 844 } : { width: 1440, height: 900 })
    await expect(page).toHaveScreenshot('director-workspace.png', { fullPage: true, animations: 'disabled' })
    if (testInfo.project.name === 'mobile') {
      await page.getByRole('region', { name: '镜头编辑', exact: true }).scrollIntoViewIfNeeded()
      await expect(page).toHaveScreenshot('director-editor.png', { animations: 'disabled' })
      await page.getByRole('region', { name: '镜头候选', exact: true }).scrollIntoViewIfNeeded()
      await expect(page).toHaveScreenshot('director-candidates.png', { animations: 'disabled' })
    }
  })
})
