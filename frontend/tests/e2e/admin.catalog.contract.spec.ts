import { expect, test } from '@playwright/test'

test.describe('admin carrier catalog contracts', () => {
  test('keeps the carrier catalog visible when pricing data is temporarily unavailable', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('s2v_access_token', 'admin-catalog-access-token')
      localStorage.setItem('s2v_refresh_token', 'admin-catalog-refresh-token')
    })
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'u_admin_catalog',
          email: 'admin@sub2video.test',
          displayName: '目录管理员',
          role: 'admin',
          workspaceName: '管理空间',
          balance: 0,
        }),
      })
    })
    await page.route('**/api/v1/admin/carriers', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          carriers: [{ id: 'car_new', code: 'new-carrier', name: '新增承运商', enabled: true }],
          models: [{ id: 'mdl_new', carrierId: 'car_new', code: 'new-model', name: '新增模型', enabled: true }],
        }),
      })
    })
    await page.route('**/api/v1/admin/creation-modes', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [] }) })
    })
    await page.route('**/api/v1/admin/model-configurations', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: [] }) })
    })
    await page.route('**/api/v1/admin/retail-pricing', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ code: 'INTERNAL', message: 'temporary pricing failure' }),
      })
    })

    await page.goto('/admin/carriers-models', { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: '承运商', exact: true })).toBeVisible()
    await expect(page.getByRole('heading', { name: '新增承运商', exact: true })).toBeVisible()
    await expect(page.getByText('新增模型', { exact: true })).toBeVisible()
  })
})
