import { expect, test, type Page } from '@playwright/test'

const cases = [
  { name: 'client-overview', path: '/app/overview', heading: '欢迎回来，李维工作室', auth: true },
  { name: 'client-create', path: '/app/create', heading: '创作', auth: true },
  { name: 'client-garden', path: '/app/garden', heading: '广场', auth: true },
  { name: 'client-works', path: '/app/works', heading: '任务', auth: true },
  { name: 'client-assets', path: '/app/assets', heading: '资产', auth: true },
  { name: 'client-trash', path: '/app/trash', heading: '回收站', auth: true },
  { name: 'client-wallet', path: '/app/wallet', heading: '钱包', auth: true },
  { name: 'client-tickets', path: '/app/tickets', heading: '帮助与工单', auth: true },
  { name: 'client-profile', path: '/app/profile', heading: '个人资料', auth: true },
  { name: 'admin-overview', path: '/admin/overview', heading: '运行大盘', auth: false },
  { name: 'admin-tasks', path: '/admin/tasks', heading: '任务中心', auth: false },
  { name: 'admin-tickets', path: '/admin/tickets', heading: '工单中心', auth: false },
  { name: 'admin-orders', path: '/admin/orders', heading: '订单管理', auth: false },
]

async function seedClientAuth(page: Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'u_visual',
        email: 'demo@sub2video.local',
        displayName: '林默',
        role: 'user',
        workspaceName: '李维工作室',
        balance: 256,
      }),
    })
  })
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'visual-test-access-token')
    localStorage.setItem('s2v_refresh_token', 'visual-test-refresh-token')
  })
}

for (const item of cases) {
  test(`${item.name} matches the surface baseline`, async ({ page }) => {
    if (item.auth) await seedClientAuth(page)
    await page.goto(item.path, { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: item.heading })).toBeVisible({ timeout: 15_000 })
    await expect(page).toHaveScreenshot(`${item.name}.png`, {
      fullPage: true,
      animations: 'disabled',
      caret: 'hide',
      scale: 'css',
      maxDiffPixelRatio: 0.02,
    })
  })
}
