import { expect, test } from '@playwright/test'

test.describe('admin supplier tabs contracts', () => {
  test('keeps supplier, API, and Workflow surfaces available', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('s2v_access_token', 'supplier-tabs-admin'))
    await page.route('**/api/v1/auth/me', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 'u_admin', email: 'admin@test', displayName: '管理员', role: 'admin', balance: 0 }) }))
    await page.route('**/api/v1/admin/suppliers**', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'sup_a', code: 'supplier_a', name: '供应商 A', enabled: true, hasToken: true, status: 'ready' }] }) }))
    await page.route('**/api/v1/admin/carriers**', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ carriers: [{ id: 'car_a', code: 'carrier_a', name: '承运商 A', enabled: true }], models: [{ id: 'mdl_a', carrierId: 'car_a', carrierCode: 'carrier_a', code: 'model_a', name: '模型 A', enabled: true }] }) }))
    await page.route('**/api/v1/admin/model-configurations**', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [{ modelId: 'mdl_a', modelName: '模型 A', carrierId: 'car_a', carrierName: '承运商 A', workflowId: 'mode_text_to_video', workflowName: '文生视频', enabled: true }] }) }))
    await page.route('**/api/v1/admin/offerings**', async (route) => {
      const url = new URL(route.request().url())
      const workflow = url.searchParams.get('executionType') === 'workflow'
      const items = workflow ? [] : [{ id: 'exec_a', supplierId: 'sup_a', supplierName: '供应商 A', modelId: 'mdl_a', modelName: '模型 A', carrierName: '承运商 A', modeId: 'mode_text_to_video', modeName: '文生视频', name: '模型 A API', executionType: 'api', apiModelCode: 'model-a-v1', enabled: true, healthStatus: 'healthy', priority: 1, minDurationSec: 1, maxDurationSec: 10, resolutions: ['768P'], aspectRatios: ['16:9'], minImages: 0, maxImages: 0, supportsGeneratedAudio: false }]
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items, total: items.length, page: 1, pageSize: 10 }) })
    })
    await page.route('**/api/v1/admin/supplier-costs**', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [] }) }))

    await page.goto('/admin/supplier-models', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: '供应商', exact: true })).toBeVisible()
    await expect(page.getByText('供应商 A', { exact: true })).toBeVisible()
    const apiTab = page.getByRole('button', { name: 'API 1', exact: true })
    const workflowTab = page.getByRole('button', { name: 'Workflow 0', exact: true })
    await expect(apiTab).toBeVisible()
    await expect(workflowTab).toBeVisible()
    await apiTab.click()
    await expect(page.getByText('模型 A API', { exact: true })).toBeVisible()
    await workflowTab.click()
    await expect(page.getByText('当前筛选下没有Workflow执行配置。', { exact: true })).toBeVisible()
  })
})
