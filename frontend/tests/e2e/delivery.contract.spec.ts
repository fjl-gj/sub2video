import { expect, test, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Frontend error/recovery contracts only. Every API and media request below is
// intercepted; this suite does not establish real storage or supplier delivery.
const accessToken = 'delivery-contract-access'
const projectID = 'prj_delivery_contract'
const assetID = 'asset_delivery_contract'
const taskID = 'task_delivery_contract'
const filePath = `/api/v1/client/projects/${projectID}/files/${assetID}`
const sourceURL = 'https://private-supplier.example.test/internal-result.mp4'
const project = { id: projectID, name: '交付测试作品', desc: '确定性测试作品', createdAt: '2026-09-25', trashed: false, isDefault: false, fileCount: 1 }
const asset = { id: assetID, name: '交付测试成片.mp4', type: 'video', size: '2 MB', time: '2026-09-25', sourceTask: taskID, url: filePath, remoteUrl: sourceURL }
const task = {
  id: taskID, title: '交付合同测试任务', mode: '文生视频', status: '生成中', progress: 30,
  duration: '5 秒', durationSeconds: 5, resolution: '768P', aspectRatio: '16:9', generateAudio: false,
  cost: 2.5, thumbUrl: '', savedToProject: false, failureReason: '', executionWaiting: false,
  deliveryStatus: 'not_ready', downloadUrl: '', createdAt: '2026-09-25T04:00:00Z', updatedAt: '2026-09-25T04:00:00Z',
}

function projectTree(items: typeof asset[] = []) {
  return {
    ...project, fileCount: items.length,
    folders: {
      资产: { isSystem: true, desc: '参考素材', items: [], folders: {} },
      视频: { isSystem: true, desc: '生成成片', items: [], folders: {
        正片镜头: { isSystem: false, desc: '选中的镜头', items, folders: {} },
      } },
    },
  }
}

async function seedDelivery(page: Page) {
  const fixture = await seedSurfaceData(page)
  await page.addInitScript((token) => {
    localStorage.setItem('s2v_access_token', token)
    localStorage.setItem('s2v_refresh_token', 'delivery-contract-refresh')
  }, accessToken)
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({ json: { id: 'u_delivery_contract', email: 'delivery@example.test', displayName: '交付测试', workspaceName: '测试工作室', role: 'user', balance: 256 } })
  })
  return fixture
}

test.describe('delivery and asset recovery (mocked frontend only)', () => {
  test('stops polling after a task becomes inaccessible and allows an explicit retry', async ({ page }) => {
    await page.clock.install({ time: new Date('2026-09-25T04:00:00Z') })
    const { unmatchedRequests } = await seedDelivery(page)
    let state: 'running' | 'missing' | 'restored' = 'running'
    let reads = 0
    await page.route(`**/api/v1/client/tasks/${taskID}`, async (route) => {
      reads++
      if (state === 'missing') await route.fulfill({ status: 404, json: { message: '任务不存在' } })
      else await route.fulfill({ json: { ...task, status: state === 'restored' ? '失败' : '生成中' } })
    })
    await page.goto(`/app/works/${taskID}`, { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: task.title, exact: true })).toBeVisible()
    expect(reads).toBe(1)

    state = 'missing'
    await page.clock.fastForward(6_000)
    await expect(page.getByText('未找到这个任务，或你没有查看权限。', { exact: true })).toBeVisible()
    expect(reads).toBe(2)
    await page.clock.fastForward(120_000)
    expect(reads).toBe(2)

    state = 'restored'
    await page.getByRole('button', { name: '重新加载', exact: true }).click()
    await expect(page.getByRole('heading', { name: task.title, exact: true })).toBeVisible()
    await expect(page.getByText('未找到这个任务，或你没有查看权限。', { exact: true })).toHaveCount(0)
    expect(reads).toBe(3)
    expect(unmatchedRequests).toEqual([])
  })

  test('keeps a new project name after failure and opens the successful project by a durable URL', async ({ page }) => {
    const { unmatchedRequests } = await seedDelivery(page)
    const name = '我的第一段商业短片'
    let attempts = 0
    await page.route(/\/api\/v1\/client\/projects$/, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      attempts++
      expect(route.request().postDataJSON()).toEqual({ name })
      if (attempts === 1) await route.fulfill({ status: 503, json: { message: '创建暂不可用' } })
      else await route.fulfill({ status: 201, json: { project: { ...project, name, fileCount: 0 } } })
    })
    await page.route(`**/api/v1/client/projects/${projectID}/tree`, async (route) => {
      await route.fulfill({ json: { ...projectTree(), name } })
    })
    await page.goto('/app/assets', { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: '新建作品', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '新建作品', exact: true })
    await dialog.getByLabel('作品名称', { exact: true }).fill(name)
    await dialog.getByRole('button', { name: '创建作品', exact: true }).click()
    await expect(dialog.getByText('创建作品失败，请稍后重试。', { exact: true })).toBeVisible()
    await expect(dialog.getByLabel('作品名称', { exact: true })).toHaveValue(name)
    await expect(dialog).toBeVisible()
    expect(attempts).toBe(1)

    await dialog.getByRole('button', { name: '创建作品', exact: true }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page).toHaveURL((url) => url.pathname === '/app/assets' && url.searchParams.get('project') === projectID)
    await expect(page.getByRole('heading', { name, exact: true })).toBeVisible()
    await page.reload({ waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name, exact: true })).toBeVisible()
    expect(attempts).toBe(2)
    expect(unmatchedRequests).toEqual([])
  })

  test('keeps the file and deletion confirmation after failure, then refreshes the directory on success', async ({ page }) => {
    const { unmatchedRequests } = await seedDelivery(page)
    let deleteAttempts = 0
    let deleted = false
    await page.route(`**/api/v1/client/projects/${projectID}/tree`, async (route) => {
      await route.fulfill({ json: projectTree(deleted ? [] : [asset]) })
    })
    await page.route(`**${filePath}`, async (route) => {
      if (route.request().method() !== 'DELETE') return route.fallback()
      deleteAttempts++
      if (deleteAttempts === 1) await route.fulfill({ status: 503, json: { message: '对象存储删除失败' } })
      else { deleted = true; await route.fulfill({ json: { ok: true } }) }
    })
    await page.goto(`/app/assets?project=${projectID}&folder=${encodeURIComponent('视频/正片镜头')}`, { waitUntil: 'networkidle' })
    await expect(page.getByText(asset.name, { exact: true })).toBeVisible()
    await page.getByRole('button', { name: '移除文件', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '删除文件', exact: true })
    await dialog.getByRole('button', { name: '删除文件', exact: true }).click()
    await expect(dialog.getByText('文件删除失败，请稍后重试。', { exact: true })).toBeVisible()
    await expect(dialog).toBeVisible()
    await expect(page.getByText(asset.name, { exact: true })).toBeVisible()
    expect(deleted).toBe(false)

    await dialog.getByRole('button', { name: '删除文件', exact: true }).click()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByText(asset.name, { exact: true })).toHaveCount(0)
    await expect(page.getByRole('heading', { name: '这个目录还是空的', exact: true })).toBeVisible()
    expect(deleteAttempts).toBe(2)
    expect(unmatchedRequests).toEqual([])
  })

  test('preserves a protected download and source-task navigation when asset media cannot be played', async ({ page }) => {
    const { unmatchedRequests } = await seedDelivery(page)
    const mediaReads: URL[] = []
    const supplierReads: string[] = []
    await page.route(`**/api/v1/client/projects/${projectID}/tree`, async (route) => {
      await route.fulfill({ json: projectTree([asset]) })
    })
    await page.route(`**${filePath}?*`, async (route) => {
      mediaReads.push(new URL(route.request().url()))
      await route.fulfill({ status: 503, contentType: 'text/plain', body: 'Deterministic media failure' })
    })
    await page.route('https://private-supplier.example.test/**', async (route) => {
      supplierReads.push(route.request().url())
      await route.abort('blockedbyclient')
    })
    await page.route(`**/api/v1/client/tasks/${taskID}`, async (route) => {
      await route.fulfill({ json: { ...task, status: '失败' } })
    })
    await page.goto(`/app/assets?project=${projectID}&folder=${encodeURIComponent('视频/正片镜头')}`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: `预览 ${asset.name}`, exact: true }).click()
    const dialog = page.getByRole('dialog', { name: asset.name, exact: true })
    await expect(dialog.getByText('文件预览暂不可用，可以下载后查看。', { exact: true })).toBeVisible()
    const download = dialog.getByRole('link', { name: '下载文件', exact: true })
    await expect(download).toBeVisible()
    const href = await download.getAttribute('href')
    expect(href).toBeTruthy()
    const downloadURL = new URL(href!, page.url())
    expect(downloadURL.origin).toBe(new URL(page.url()).origin)
    expect(downloadURL.pathname).toBe(filePath)
    expect(downloadURL.searchParams.get('access_token')).toBe(accessToken)
    expect(mediaReads.length).toBeGreaterThan(0)
    for (const url of mediaReads) expect(url.searchParams.get('access_token')).toBe(accessToken)
    expect(supplierReads).toEqual([])

    await dialog.getByRole('link', { name: '查看来源任务', exact: true }).click()
    await expect(page).toHaveURL((url) => url.pathname === `/app/works/${taskID}`)
    await expect(page.getByRole('heading', { name: task.title, exact: true })).toBeVisible()
    expect(unmatchedRequests).toEqual([])
  })
})
