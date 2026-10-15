import { expect, test, type Locator, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Deterministic layout fixtures only: these tests do not accept supplier
// execution, billing, storage, or generated-video delivery as verified.
const widths = [320, 390, 768, 1024, 1440, 1920]
const modes = [
  { id: 'mode_text_to_video', inputType: 'text', name: '文生视频', enabled: true },
  { id: 'mode_first_last', inputType: 'first_last', name: '首尾帧', enabled: true },
  { id: 'mode_multi_image', inputType: 'multi_image', name: '多图生成', enabled: true },
]

async function seedWorkspace(page: Page) {
  const fixture = await seedSurfaceData(page)
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'layout-fixture-access')
    localStorage.setItem('s2v_refresh_token', 'layout-fixture-refresh')
  })
  await page.route('**/api/v1/auth/me', (route) => route.fulfill({ json: {
    id: 'u_visual', email: 'creator@example.test', displayName: '林默',
    workspaceName: '李维工作室', role: 'user', balance: 256,
  } }))
  await page.route('**/api/v1/client/create/catalog', (route) => route.fulfill({ json: {
    modes,
    carriers: [{ id: 'car_minimax', name: 'MINMAX' }],
    models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', inputTypes: ['text', 'first_last', 'multi_image'] }],
  } }))
  await page.route('**/api/v1/client/create/options**', (route) => route.fulfill({ json: {
    durations: [5, 10, 15], resolutions: ['768P'], aspectRatios: ['16:9', '9:16'],
    supportsGeneratedAudio: true, minImages: 0, maxImages: 9,
    defaultDuration: 5, defaultResolution: '768P',
  } }))
  await page.route('**/api/v1/client/projects/prj_visual_coast/tree', (route) => route.fulfill({ json: {
    id: 'prj_visual_coast', name: '海岸清晨', desc: '布局测试作品', createdAt: '2026-09-20',
    trashed: false, isDefault: true, fileCount: 3,
    folders: {
      资产: { isSystem: true, desc: '参考素材', items: [], folders: {} },
      视频: {
        isSystem: true, desc: '生成镜头', folders: {},
        items: [1, 2, 3].map((index) => ({
          id: `file_layout_${index}`, name: `海岸清晨-镜头${index}.mp4`, type: 'video',
          size: '2 MB', time: '2026-09-25', sourceTask: 'tsk_visual_coast',
        })),
      },
    },
  } }))
  return fixture
}

async function rect(locator: Locator) {
  await expect(locator).toBeVisible()
  return locator.evaluate((element) => {
    const box = element.getBoundingClientRect()
    return { left: box.left, right: box.right, top: box.top, bottom: box.bottom, width: box.width, height: box.height }
  })
}

async function expectNoHorizontalOverflow(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
    'The document must fit the viewport without hiding a horizontal overflow.').toBeLessThanOrEqual(1)
  for (const container of await page.locator('main').all()) {
    expect(await container.evaluate((element) => element.scrollWidth - element.clientWidth),
      'The main content must fit; only an explicitly scrollable table may scroll sideways.').toBeLessThanOrEqual(1)
  }
  const balance = page.locator('.s2v-header-wallet .s2v-mono')
  if (await balance.count()) {
    expect(await balance.evaluate((element) => element.getBoundingClientRect().height), 'Currency and balance stay together on one line.').toBeLessThanOrEqual(22)
  }
}

async function expectWorkspaceAxes(page: Page, content: Locator, options?: { centeredHeading?: boolean }) {
  const [main, frame, heading, body] = await Promise.all([
    rect(page.locator('main.s2v-content')),
    rect(page.locator('main > .s2v-page')),
    rect(page.locator('main h1')),
    rect(content),
  ])
  const leftSpace = frame.left - main.left
  const rightSpace = main.right - frame.right
  expect(leftSpace, 'The page stays inside its workspace.').toBeGreaterThanOrEqual(0)
  expect(rightSpace, 'The page stays inside its workspace.').toBeGreaterThanOrEqual(0)
  expect(Math.abs(leftSpace - rightSpace), 'The page is centered in the area after the sidebar.').toBeLessThanOrEqual(2)
  if (options?.centeredHeading) {
    const headingCenter = (heading.left + heading.right) / 2
    const frameCenter = (frame.left + frame.right) / 2
    expect(Math.abs(headingCenter - frameCenter), 'The create hero stays centered in the page frame.').toBeLessThanOrEqual(4)
  } else {
    expect(Math.abs(heading.left - body.left), 'The heading and main content share a left axis.').toBeLessThanOrEqual(2)
  }
  expect(Math.abs(frame.right - body.right), 'The main content fills the page frame.').toBeLessThanOrEqual(2)
  if (page.viewportSize()!.width === 1920) {
    expect(frame.width, 'A wide workspace must use the available screen instead of leaving large empty rails.').toBeGreaterThanOrEqual(1500)
  }
  await expectNoHorizontalOverflow(page)
}

async function expectReadableCards(grid: Locator, cards: Locator) {
  const frame = await rect(grid)
  const boxes = await cards.evaluateAll((elements) => elements.map((element) => {
    const box = element.getBoundingClientRect()
    return { left: box.left, right: box.right, top: box.top, bottom: box.bottom, width: box.width }
  }))
  expect(boxes.length, 'The fixture must contain cards; an empty page is not a layout pass.').toBeGreaterThan(0)
  for (const [index, box] of boxes.entries()) {
    expect(box.width, 'Cards need room for a readable title and actions, including with the sidebar open.').toBeGreaterThanOrEqual(Math.min(220, frame.width) - 1)
    expect(box.left).toBeGreaterThanOrEqual(frame.left - 1)
    expect(box.right).toBeLessThanOrEqual(frame.right + 1)
    for (const other of boxes.slice(index + 1)) {
      const overlapX = Math.min(box.right, other.right) - Math.max(box.left, other.left)
      const overlapY = Math.min(box.bottom, other.bottom) - Math.max(box.top, other.top)
      expect(overlapX > 1 && overlapY > 1, 'Sibling cards must not overlap.').toBe(false)
    }
  }
}

async function expectReachable(page: Page, control: Locator) {
  await control.scrollIntoViewIfNeeded()
  await expect(control).toBeInViewport()
  const box = await rect(control)
  expect(box.left).toBeGreaterThanOrEqual(0)
  expect(box.right).toBeLessThanOrEqual(page.viewportSize()!.width)
  // Trial mode also detects another element covering the control. It never
  // submits a task, navigates, or mutates even the deterministic fixture.
  await control.click({ trial: true })
}

async function expectHomeModeRows(page: Page, width: number) {
  const rows = page.locator('.s2v-home-mode-row')
  await expect(rows).toHaveCount(3)
  for (const mode of modes) {
    const entry = page.getByRole('link', { name: `用${mode.name}开始创作`, exact: true })
    await expect(entry).toHaveAttribute('href', `/app/create?mode=${mode.inputType}`)
    await expectReachable(page, entry)
    expect((await rect(entry)).height, 'Every mode entry has a usable touch target.').toBeGreaterThanOrEqual(44)
  }
  if (width >= 768) {
    const boxes = await Promise.all([0, 1, 2].map((index) => rect(rows.nth(index))))
    expect(Math.abs(boxes[0]!.top - boxes[1]!.top), 'Desktop mode rows share one horizontal band.').toBeLessThanOrEqual(2)
    expect(Math.abs(boxes[1]!.top - boxes[2]!.top)).toBeLessThanOrEqual(2)
  } else {
    const boxes = await Promise.all([0, 1, 2].map((index) => rect(rows.nth(index))))
    expect(boxes[0]!.bottom, 'Phone mode rows stack vertically.').toBeLessThanOrEqual(boxes[1]!.top + 1)
    expect(boxes[1]!.bottom).toBeLessThanOrEqual(boxes[2]!.top + 1)
  }
  await expectNoHorizontalOverflow(page)
}

test.describe('Responsive geometry with mocked display data', () => {
  test.beforeEach(async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', 'Custom viewport matrix runs once; the mobile project retains its separate visual tests.')
    await page.emulateMedia({ reducedMotion: 'reduce' })
  })

  for (const width of widths) {
    test(`public content shares balanced axes at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 })
      const { unmatchedRequests } = await seedSurfaceData(page)
      await page.goto('/', { waitUntil: 'networkidle' })
      await expect(page.getByRole('heading', { level: 1, name: '一句话，一个镜头。', exact: true })).toBeVisible()
      await expect(page.getByRole('heading', { name: '三种创作方式', exact: true })).toBeVisible()
      await page.evaluate(() => document.fonts.ready)

      const hero = await rect(page.locator('.s2v-home-hero'))
      const modesFrame = await rect(page.locator('#modes'))
      const footer = await page.locator('.s2v-public-footer').evaluate((element) => {
        const box = element.getBoundingClientRect()
        const style = getComputedStyle(element)
        return { left: box.left + parseFloat(style.paddingLeft), right: box.right - parseFloat(style.paddingRight) }
      })
      expect(Math.abs(modesFrame.left - footer.left), 'Creation modes and footer share the foreground left axis.').toBeLessThanOrEqual(2)
      expect(Math.abs(modesFrame.right - footer.right), 'Creation modes and footer share the foreground right axis.').toBeLessThanOrEqual(2)
      const documentWidth = await page.evaluate(() => document.documentElement.clientWidth)
      expect(Math.abs(modesFrame.left - (documentWidth - modesFrame.right)), 'Public foreground has equal outer gutters.').toBeLessThanOrEqual(2)
      expect(hero.left, 'The film background has no foreground gutter.').toBe(0)
      expect(hero.right).toBe(documentWidth)
      if (width === 1920) expect(hero.width, 'The desktop presentation uses the wide screen.').toBeGreaterThanOrEqual(1600)
      await expectReachable(page, page.getByRole('link', { name: '开始创作', exact: true }))
      await expectHomeModeRows(page, width)
      expect(unmatchedRequests, 'Layout checks cannot silently call a live backend.').toEqual([])
      if (width === 390 || width === 1920) {
        // Reviewable viewport baseline with the deterministic display fixtures
        // above, not a production screenshot or proof of generated delivery.
        await page.evaluate(() => window.scrollTo(0, 0))
        await expect(page.getByRole('heading', { level: 1 })).toBeInViewport()
        await expect(page).toHaveScreenshot(`responsive-home-${width}.png`, {
          fullPage: false, animations: 'disabled', caret: 'hide', scale: 'css', maxDiffPixelRatio: 0.02,
        })
      }
    })

    test(`workspace remains usable and balanced at ${width}px`, async ({ page }) => {
      test.setTimeout(60_000)
      await page.setViewportSize({ width, height: 900 })
      const { unmatchedRequests } = await seedWorkspace(page)

      await test.step('Creation keeps its heading, inputs and result frame aligned', async () => {
        await page.goto('/app/create?mode=text', { waitUntil: 'networkidle' })
        await expect(page.getByRole('combobox', { name: '视频时长', exact: true })).toBeVisible()
        await page.evaluate(() => document.fonts.ready)
        await expectWorkspaceAxes(page, page.locator('.s2v-studio-layout'), { centeredHeading: width > 600 })
        if (width <= 390) {
          const modeGroup = page.getByLabel('创作模式', { exact: true })
          for (const mode of modes) {
            const button = modeGroup.getByRole('button', { name: mode.name, exact: true })
            await expectReachable(page, button)
            await button.click()
            await expect(button).toHaveAttribute('aria-pressed', 'true')
          }
          await modeGroup.getByRole('button', { name: '文生视频', exact: true }).click()
          const prompt = page.getByRole('textbox', { name: '画面描述', exact: true })
          await expectReachable(page, prompt)
          await prompt.fill('清晨海岸，镜头缓慢向前移动。')
          await expectReachable(page, page.getByRole('combobox', { name: '视频时长', exact: true }))
          await expect(page.getByRole('button', { name: '提交生成任务', exact: true })).toBeEnabled()
          await expectReachable(page, page.getByRole('button', { name: '提交生成任务', exact: true }))
          await expectReachable(page, page.getByRole('button', { name: '上传素材', exact: true }))
          await expectNoHorizontalOverflow(page)
        }
      })

      if (width === 320 || width === 390 || width === 1920) {
        // Keep the first viewport comparable after the mobile reachability
        // checks have scrolled the mocked workspace to its submit controls.
        await page.locator('main.s2v-content').evaluate((element) => element.scrollTo(0, 0))
        await expect(page.getByRole('heading', { name: /和 Sub2Video 聊聊/ })).toBeInViewport()
        await expect(page).toHaveScreenshot(`responsive-create-${width}.png`, {
          fullPage: false, animations: 'disabled', caret: 'hide', scale: 'css', maxDiffPixelRatio: 0.02,
        })
      }

      await test.step('Overview cards fit the available space', async () => {
        await page.goto('/app/overview', { waitUntil: 'networkidle' })
        await expect(page.getByRole('heading', { name: '首页概览', exact: true })).toBeVisible()
        await expectWorkspaceAxes(page, page.locator('.s2v-studio-overview'))
        await expectReadableCards(page.locator('.s2v-studio-overview .s2v-studio-modes'), page.locator('.s2v-studio-overview .s2v-studio-mode'))
      })

      await test.step('Tutorial cards retain readable widths', async () => {
        await page.goto('/app/garden', { waitUntil: 'networkidle' })
        await expectWorkspaceAxes(page, page.getByRole('region', { name: '镜头教程', exact: true }))
        await expectReadableCards(page.locator('.s2v-guide-grid'), page.locator('.s2v-guide-card'))
      })

      await test.step('Assets and their nested folders do not create a cramped second sidebar', async () => {
        await page.goto('/app/assets', { waitUntil: 'networkidle' })
        await expect(page.getByRole('button', { name: '海岸清晨', exact: true })).toBeVisible()
        await expectWorkspaceAxes(page, page.locator('.s2v-library-projects'))
        await expectReadableCards(page.locator('.s2v-library-projects'), page.locator('.s2v-project-card'))
        await page.goto('/app/assets?project=prj_visual_coast&folder=%E8%A7%86%E9%A2%91', { waitUntil: 'networkidle' })
        await expect(page.getByRole('heading', { name: '海岸清晨', exact: true })).toBeVisible()
        await expectWorkspaceAxes(page, page.locator('.s2v-library-interior'))
        await expectReadableCards(page.locator('.s2v-library-file-grid'), page.locator('.s2v-library-file'))
        await expectReachable(page, page.getByRole('button', { name: '上传素材', exact: true }))
      })

      await test.step('A loaded reference video stays centered inside the delivery frame', async () => {
        // Reuse the licensed local reference film to exercise media geometry.
        // This ready response is a display fixture, not a supplier delivery.
        await page.route('**/api/v1/client/tasks/tsk_layout_media', (route) => route.fulfill({ json: {
          id: 'tsk_layout_media', title: '播放器布局测试 · 本地参考片', mode: '文生视频',
          status: '已完成', progress: 100, duration: '5 秒', durationSeconds: 5,
          resolution: '768P', aspectRatio: '16:9', generateAudio: false, cost: 0,
          savedToProject: false, deliveryStatus: 'ready',
          downloadUrl: '/brand/auth/iceland-coast.webm',
          createdAt: '2026-09-25T04:00:00Z', updatedAt: '2026-09-25T04:00:05Z',
        } }))
        await page.goto('/app/works/tsk_layout_media', { waitUntil: 'networkidle' })
        const video = page.locator('.s2v-delivery-screen video')
        await expect(video).toBeVisible()
        await expect.poll(() => video.evaluate((element: HTMLVideoElement) => element.videoWidth)).toBeGreaterThan(0)
        await expectWorkspaceAxes(page, page.locator('.s2v-delivery-layout'))
        const [screen, media] = await Promise.all([rect(page.locator('.s2v-delivery-screen')), rect(video)])
        expect(Math.abs((media.top - screen.top) - (screen.bottom - media.bottom)),
          'A height-limited video must have equal space above and below.').toBeLessThanOrEqual(2)
        expect(media.left).toBeGreaterThanOrEqual(screen.left)
        expect(media.right).toBeLessThanOrEqual(screen.right)
        if (width === 1920) {
          await expect(page).toHaveScreenshot('responsive-delivery-1920.png', {
            fullPage: false, animations: 'disabled', caret: 'hide', scale: 'css', maxDiffPixelRatio: 0.02,
          })
        }
        await expectReachable(page, page.getByRole('button', { name: '下载成片', exact: true }))
        await expectReachable(page, page.getByRole('button', { name: '保存到作品', exact: true }))
      })

      expect(unmatchedRequests, 'Layout checks cannot silently call a live backend or submit tasks.').toEqual([])
    })
  }
})
