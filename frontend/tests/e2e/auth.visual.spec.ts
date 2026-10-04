import { expect, test } from '@playwright/test'

for (const surface of [
  { name: 'auth-login', path: '/auth/login', title: '登录 Sub2Video' },
  { name: 'auth-register', path: '/auth/register', title: '创建账号' },
]) {
  test(`${surface.name} uses the commercial film entrance`, async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    const unexpected: string[] = []
    await page.route('**/api/v1/**', async (route) => {
      if (new URL(route.request().url()).pathname === '/api/v1/settings/public') return route.fulfill({ json: { registrationEnabled: true, emailVerifyEnabled: true, passwordResetEnabled: true } })
      unexpected.push(route.request().url())
      await route.fulfill({ status: 500, json: {} })
    })
    await page.goto(surface.path, { waitUntil: 'networkidle' })
    await expect(page.getByRole('heading', { name: surface.title, exact: true })).toBeVisible()
    await expect(page.getByLabel('邮箱', { exact: true })).toBeVisible()
    await page.evaluate(() => document.fonts.ready)
    expect(await page.locator('.s2v-auth-poster').evaluate((image: HTMLImageElement) => image.complete && image.naturalWidth > 0)).toBe(true)
    // Reduced motion keeps the local poster deterministic without pretending a
    // static screenshot verifies playback or a real authenticated backend.
    await expect(page.locator('video')).toHaveCount(0)
    expect(unexpected).toEqual([])
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1)
    await expect(page).toHaveScreenshot(`${surface.name}.png`, { fullPage: true, animations: 'disabled', caret: 'hide', scale: 'css', maxDiffPixelRatio: 0.01 })
  })
}
