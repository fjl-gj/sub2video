import { expect, test } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

const surfaces = [
  { name: 'journey-product-home', path: '/', auth: false, content: '一句话，一个镜头。' },
  { name: 'journey-guide-detail', path: '/guides/coast-letter', auth: false, content: '海岸来信' },
  { name: 'journey-guided-create', path: '/app/create?guide=coast-letter&shot=arrival', auth: true, content: '海岸来信' },
  { name: 'journey-result-detail', path: '/app/works/tsk_visual_coast', auth: true, content: '海岸清晨 · 镜头 03' },
  { name: 'journey-project-files', path: '/app/assets?project=prj_visual_coast&folder=视频', auth: true, content: '海岸清晨-镜头03.mp4' },
]

for (const surface of surfaces) {
  test(`${surface.name} retains the commercial journey baseline`, async ({ page }, testInfo) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    const { unmatchedRequests } = await seedSurfaceData(page)
    if (surface.auth) await page.addInitScript(() => {
      localStorage.setItem('s2v_access_token', 'journey-visual-access')
      localStorage.setItem('s2v_refresh_token', 'journey-visual-refresh')
    })
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({ json: { id: 'u_visual', email: 'creator@example.test', displayName: '林默', workspaceName: '李维工作室', role: 'user', balance: 256 } })
    })
    await page.route('**/api/v1/client/create/catalog', async (route) => {
      await route.fulfill({ json: {
        modes: [{ id: 'mode_text', inputType: 'text', name: '文生视频', description: '文字生成视频', enabled: true }, { id: 'mode_multi', inputType: 'multi_image', name: '多图生成视频', description: '参考图片生成', enabled: true }, { id: 'mode_frames', inputType: 'first_last', name: '首尾帧生成视频', description: '起止画面生成', enabled: true }],
        carriers: [{ id: 'car_minimax', name: 'MINMAX' }], models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', description: '视频模型', inputTypes: ['text', 'multi_image', 'first_last'] }],
      } })
    })
    await page.route('**/api/v1/client/create/options**', async (route) => {
      await route.fulfill({ json: { durations: [5, 10, 15], resolutions: ['768P'], aspectRatios: ['16:9', '9:16'], supportsGeneratedAudio: true, minImages: 1, maxImages: 9, defaultDuration: 5, defaultResolution: '768P' } })
    })
    await page.route('**/api/v1/client/session/current-project', async (route) => {
      if (route.request().method() !== 'PUT') return route.fallback()
      await route.fulfill({ json: { currentProjectId: 'prj_visual_coast' } })
    })
    await page.route('**/api/v1/client/projects/prj_visual_coast/tree', async (route) => {
      await route.fulfill({ json: {
        id: 'prj_visual_coast', name: '海岸清晨', desc: '人物、场景与生成成片', createdAt: '2026-09-20', trashed: false, isDefault: true, fileCount: 1,
        folders: {
          资产: { isSystem: true, desc: '参考素材', items: [], folders: {} },
          视频: { isSystem: true, desc: '生成成片', folders: {}, items: [{ id: 'file_visual_coast', name: '海岸清晨-镜头03.mp4', type: 'video', size: '2 MB', time: '2026-09-25', sourceTask: 'tsk_visual_coast' }] },
        },
      } })
    })
    await page.goto(surface.path, { waitUntil: 'networkidle' })
    await expect(page.locator('main')).toContainText(surface.content)
    await expect(page.locator('main h1')).toBeVisible()
    await page.evaluate(() => document.fonts.ready)
    expect(unmatchedRequests, 'Screenshots may not silently use a live backend or a missing fixture.').toEqual([])
    // Public visual fixtures use reduced motion: the homepage shows its authored
    // poster, while tutorial pages still contain no simulated video results.
    if (!surface.auth) await expect(page.locator('video')).toHaveCount(0)
    if (testInfo.project.name === 'mobile') {
      expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1)
    }
    const projectFiles = surface.name === 'journey-project-files'
    if (projectFiles) await expect(page.getByRole('button', { name: '导演台', exact: true })).toBeVisible()
    // Typography/entrance changes occupy few pixels over large brand media.
    // Compare public surfaces and the project entry exactly, so the old 2%
    // allowance cannot silently accept a stale headline or action position.
    await expect(page).toHaveScreenshot(`${surface.name}.png`, { fullPage: true, animations: 'disabled', caret: 'hide', scale: 'css', maxDiffPixelRatio: projectFiles || !surface.auth ? 0 : 0.02 })
  })
}
