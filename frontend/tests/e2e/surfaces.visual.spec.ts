import { expect, test } from '@playwright/test'

const cases = [
  { name: 'client-overview', path: '/app/overview', heading: '欢迎回来，李维工作室' },
  { name: 'client-create', path: '/app/create', heading: '创建视频生成任务' },
  { name: 'admin-overview', path: '/admin/overview', heading: '运行大盘' },
  { name: 'admin-tasks', path: '/admin/tasks', heading: '任务中心' },
]

for (const item of cases) {
  test(`${item.name} matches the surface baseline`, async ({ page }) => {
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
