import { expect, test, type Locator, type Page } from '@playwright/test'
import { seedSurfaceData } from './fixtures/surface-data'

// Mock browser evidence for the documented UI contract only. No mutation is
// fulfilled, and these tests do not establish billing, storage or delivery.
const widths = [320, 390, 768, 1024, 1440, 1920]
const surfaces = [
  {
    name: 'client', path: '/app/assets', heading: '资产', trigger: '新建作品',
    dialog: '新建作品', confirm: '创建作品', canvas: 'rgb(20, 22, 22)',
    panel: 'rgb(27, 30, 28)', destination: '钱包', destinationPath: '/app/wallet',
  },
  {
    name: 'admin', path: '/admin/carriers-models', heading: '承运商', trigger: '新增承运商',
    dialog: '新增承运商', confirm: '保存', canvas: 'rgb(245, 246, 244)',
    panel: 'rgb(255, 255, 255)', destination: '任务中心', destinationPath: '/admin/tasks',
  },
] as const

async function seedWorkspace(page: Page) {
  const fixture = await seedSurfaceData(page)
  await page.addInitScript(() => {
    localStorage.setItem('s2v_access_token', 'ui-standards-fixture-access')
    localStorage.setItem('s2v_refresh_token', 'ui-standards-fixture-refresh')
  })
  const readOnlyFixtures: Record<string, unknown> = {
    '/auth/me': {
      id: 'u_visual', email: 'creator@example.test', displayName: '林默',
      workspaceName: '李维工作室', role: 'admin', balance: 256,
    },
    '/client/create/catalog': {
      modes: [
        { id: 'mode_text_to_video', inputType: 'text', name: '文生视频', enabled: true },
        { id: 'mode_first_last', inputType: 'first_last', name: '首尾帧生成视频', enabled: true },
        { id: 'mode_multi_image', inputType: 'multi_image', name: '多图生成视频', enabled: true },
      ],
      carriers: [{ id: 'car_minimax', name: 'MINMAX' }],
      models: [{ id: 'mdl_h3', carrierId: 'car_minimax', name: 'H3', inputTypes: ['text', 'first_last', 'multi_image'] }],
    },
    '/client/create/options': {
      durations: [5, 10, 15], resolutions: ['768P'], aspectRatios: ['16:9', '9:16'],
      supportsGeneratedAudio: true, minImages: 0, maxImages: 9,
      defaultDuration: 5, defaultResolution: '768P',
    },
    '/admin/carriers': {
      carriers: [{ id: 'car_minimax', code: 'minimax', name: 'MINMAX', enabled: true }],
      models: [{ id: 'mdl_h3', carrierId: 'car_minimax', code: 'h3', name: 'H3', carrierCode: 'minimax', enabled: true }],
      total: 1, page: 1, pageSize: 10, cosConfigured: false, autodlEnvToken: false,
    },
  }
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname.replace('/api/v1', '')
    if (request.method() === 'GET' && path in readOnlyFixtures) {
      await route.fulfill({ json: readOnlyFixtures[path] })
      return
    }
    // The shared fixture rejects unknown calls, including all writes other
    // than its explicit no-charge estimate fixture. Never fall through live.
    await route.fallback()
  })
  return fixture
}

async function rect(locator: Locator) {
  await expect(locator).toBeVisible()
  return locator.evaluate((element) => {
    const box = element.getBoundingClientRect()
    return { left: box.left, right: box.right, top: box.top, bottom: box.bottom, width: box.width, height: box.height }
  })
}

async function expectUnobscured(page: Page, control: Locator, minimumSize?: number) {
  await expect(control).toBeEnabled()
  await control.scrollIntoViewIfNeeded()
  // Trial click performs Playwright's hit testing without submitting a task
  // or creating an asset. Geometry additionally checks the whole hit area.
  await control.click({ trial: true })
  const box = await rect(control)
  const viewport = page.viewportSize()!
  // Browser scrolling rounds to device pixels while DOMRects retain fractions
  // (observed bottom: 900.34375 in a 900px viewport). This 1px allowance applies
  // only to viewport geometry, never touch size or fixed-navigation hit tests.
  const geometryRounding = 1
  expect.soft(box.left, 'The action stays inside the left viewport edge.').toBeGreaterThanOrEqual(-geometryRounding)
  expect.soft(box.right, 'The action stays inside the right viewport edge.').toBeLessThanOrEqual(viewport.width + geometryRounding)
  expect.soft(box.top, 'The action is reachable above the fold after scrolling.').toBeGreaterThanOrEqual(-geometryRounding)
  expect.soft(box.bottom, 'The action stays inside the viewport after scrolling.').toBeLessThanOrEqual(viewport.height + geometryRounding)
  if (minimumSize !== undefined) {
    expect.soft(box.width, 'Primary touch actions need at least 44 CSS pixels of width.').toBeGreaterThanOrEqual(minimumSize)
    expect.soft(box.height, 'Primary touch actions need at least 44 CSS pixels of height.').toBeGreaterThanOrEqual(minimumSize)
  }
  const bottomNav = page.getByRole('navigation', { name: '移动导航', exact: true })
  if (await bottomNav.isVisible() && !await control.evaluate((element) => Boolean(element.closest('[role="dialog"]')))) {
    const nav = await rect(bottomNav)
    expect.soft(box.bottom, 'The fixed mobile navigation must not cover the action.').toBeLessThanOrEqual(nav.top)
  }
  const hitPoints = await control.evaluate((element) => {
    const box = element.getBoundingClientRect()
    return [[0.5, 0.5], [0.25, 0.25], [0.75, 0.25], [0.25, 0.75], [0.75, 0.75]].map(([x, y]) => {
      const hit = document.elementFromPoint(box.left + box.width * x, box.top + box.height * y)
      return Boolean(hit && (hit === element || element.contains(hit)))
    })
  })
  expect.soft(hitPoints, 'The center and inner corners must hit the action, not a fixed bar or overlay.').toEqual([true, true, true, true, true])
}

async function expectNoPageOverflow(page: Page) {
  expect.soft(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
    'The document must not scroll horizontally.').toBeLessThanOrEqual(1)
  expect.soft(await page.getByRole('main').evaluate((element) => element.scrollWidth - element.clientWidth),
    'Wide tables scroll locally rather than widening the workspace.').toBeLessThanOrEqual(1)
}

async function expectAdminAxesAndTable(page: Page) {
  const table = page.getByRole('table')
  await expect(table.getByRole('cell', { name: 'H3', exact: true })).toBeVisible()
  // These are the documented layout/component owners, not incidental utility
  // classes. Behavior controls elsewhere use accessible names.
  const query = page.getByRole('textbox', { name: '查询目录', exact: true })
  const [main, frame, headingGroup, filterGroup, card] = await Promise.all([
    rect(page.getByRole('main')), rect(page.locator('main > .s2v-page')),
    rect(page.locator('.s2v-page-header')),
    // The S2VInput label owns its caption, border and prefix icon. The inner
    // textbox starts after 1px border + 12px padding + 16px icon + 8px gap;
    // that intentional 37px inset is not a page-axis misalignment.
    rect(page.locator('label').filter({ has: query })),
    rect(page.locator('.s2v-panel').filter({ has: table })),
  ])
  expect.soft(Math.abs((frame.left - main.left) - (main.right - frame.right)),
    'The admin frame is centered in the area remaining after the sidebar.').toBeLessThanOrEqual(2)
  for (const [name, box] of [['heading group', headingGroup], ['filter group', filterGroup], ['records', card]] as const) {
    expect.soft(Math.abs(box.left - frame.left), `Admin ${name} shares the page's left axis.`).toBeLessThanOrEqual(2)
  }
  expect.soft(Math.abs(card.right - frame.right), 'The records fill the admin page frame.').toBeLessThanOrEqual(2)
  if (page.viewportSize()!.width === 1920) {
    expect.soft(frame.width, 'The wide admin workspace uses the available content area.').toBeGreaterThanOrEqual(1500)
  }
  const scrolling = await table.evaluate((element) => {
    let parent = element.parentElement
    while (parent && parent.tagName !== 'MAIN') {
      const style = getComputedStyle(parent)
      if (style.overflowX === 'auto' || style.overflowX === 'scroll') {
        const overflows = parent.scrollWidth > parent.clientWidth + 1
        parent.scrollLeft = parent.scrollWidth
        const frame = parent.getBoundingClientRect()
        const action = element.querySelector('th:last-child')!.getBoundingClientRect()
        return { found: true, overflows, scrollLeft: parent.scrollLeft, actionRight: action.right, frameRight: frame.right }
      }
      parent = parent.parentElement
    }
    return { found: false, overflows: false, scrollLeft: 0, actionRight: 0, frameRight: 0 }
  })
  expect(scrolling.found, 'The seven-column table has its own horizontal scroll container.').toBe(true)
  if (page.viewportSize()!.width <= 768) {
    expect.soft(scrolling.overflows, 'Narrow screens preserve readable columns in a local scroll area.').toBe(true)
    expect.soft(scrolling.scrollLeft, 'The table can scroll to its rightmost operations.').toBeGreaterThan(0)
  }
  expect.soft(scrolling.actionRight, 'The operations column is reachable within its scroll container.').toBeLessThanOrEqual(scrolling.frameRight + 1)
  await expectNoPageOverflow(page)
}

async function expectModalKeyboardLoop(page: Page, dialog: Locator) {
  await expect.poll(() => dialog.evaluate((element) => element.contains(document.activeElement)),
    { message: 'Opening a dialog moves focus into it.' }).toBe(true)
  const focusCount = await dialog.locator('button, input, select, textarea, a[href], [tabindex]').evaluateAll((elements) =>
    elements.filter((element) => element instanceof HTMLElement && element.tabIndex >= 0
      && !element.matches(':disabled') && element.getClientRects().length > 0).length)
  expect(focusCount, 'The dialog exposes meaningful keyboard controls.').toBeGreaterThanOrEqual(2)
  for (const key of ['Tab', 'Shift+Tab']) {
    for (let index = 0; index <= focusCount; index++) {
      await page.keyboard.press(key)
      expect(await dialog.evaluate((element) => element.contains(document.activeElement)),
        `${key} must remain inside the modal, including at the ends of its focus order.`).toBe(true)
    }
  }
}

test.describe('UI standards with deterministic Mock responses', () => {
  test.beforeEach(async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', 'Explicit six-width matrix runs once; existing mobile visual coverage remains separate.')
    await page.emulateMedia({ reducedMotion: 'reduce' })
  })

  for (const width of widths) {
    test(`primary actions and admin table stay usable at ${width}px`, async ({ page }) => {
      test.setTimeout(60_000)
      await page.setViewportSize({ width, height: 900 })
      const { unmatchedRequests } = await seedWorkspace(page)

      await test.step('Creation action clears fixed navigation without submitting', async () => {
        await page.goto('/app/create?mode=text')
        await page.getByRole('textbox', { name: '画面描述', exact: true }).fill('清晨海岸，镜头缓慢向前移动。')
        await page.evaluate(() => document.fonts.ready)
        await expectUnobscured(page, page.getByRole('button', { name: '提交生成任务', exact: true }), 44)
        await expectNoPageOverflow(page)
      })

      await test.step('Asset creation action remains reachable', async () => {
        await page.goto('/app/assets')
        await expect(page.getByRole('button', { name: '海岸清晨', exact: true })).toBeVisible()
        await expectUnobscured(page, page.getByRole('button', { name: '新建作品', exact: true }), 44)
        await expectNoPageOverflow(page)
      })

      await test.step('Admin records align and scroll inside the table', async () => {
        await page.goto('/admin/carriers-models')
        await expectAdminAxesAndTable(page)
        // Desktop administration may use compact buttons. On phone layouts
        // the principal action must meet the documented touch minimum.
        await expectUnobscured(page, page.getByRole('button', { name: '新增承运商', exact: true }), width < 768 ? 44 : undefined)
      })
      expect(unmatchedRequests, 'No mutation or unmocked business request is allowed.').toEqual([])
    })
  }

  for (const surface of surfaces) {
    for (const width of [320, 1024]) {
      test(`${surface.name} dialog fits, traps focus and restores its trigger at ${width}px`, async ({ page }) => {
        await page.setViewportSize({ width, height: width === 320 ? 640 : 900 })
        const { unmatchedRequests } = await seedWorkspace(page)
        await page.goto(surface.path)
        await expect(page.getByRole('heading', { name: surface.heading, exact: true })).toBeVisible()
        const trigger = page.getByRole('button', { name: surface.trigger, exact: true })
        await expect(trigger).toBeVisible()
        await trigger.focus()
        await trigger.press('Enter')
        const dialog = page.getByRole('dialog', { name: surface.dialog, exact: true })
        await expect(dialog).toBeVisible()
        await page.evaluate(() => document.fonts.ready)
        const box = await rect(dialog)
        expect.soft(box.left).toBeGreaterThanOrEqual(0)
        expect.soft(box.right).toBeLessThanOrEqual(width)
        expect.soft(box.top).toBeGreaterThanOrEqual(0)
        expect.soft(box.bottom).toBeLessThanOrEqual(page.viewportSize()!.height)
        expect.soft(await dialog.evaluate((element) => element.scrollWidth - element.clientWidth),
          'The dialog content must not overflow horizontally.').toBeLessThanOrEqual(1)

        // Literal expectations come from docs/ui/design-system.md, so a
        // portal inheriting the wrong surface cannot pass by reading itself.
        // Reka intentionally hides the background from the accessibility tree
        // while the modal is open; its rendered surface is still visible.
        await expect(page.getByRole('main', { includeHidden: true })).toHaveCSS('background-color', surface.canvas)
        await expect(dialog).toHaveCSS('background-color', surface.panel)
        await expect(dialog).toHaveCSS('border-top-left-radius', '12px')
        const confirm = dialog.getByRole('button', { name: surface.confirm, exact: true })
        await expect(confirm).toHaveCSS('border-top-left-radius', '8px')
        await expectUnobscured(page, confirm, width < 768 ? 44 : undefined)
        await expectModalKeyboardLoop(page, dialog)
        await page.keyboard.press('Escape')
        await expect(dialog).toBeHidden()
        await expect(trigger, 'Closing with Escape restores focus to the control that opened the dialog.').toBeFocused()
        expect(unmatchedRequests, 'Opening and cancelling a dialog must not create data.').toEqual([])
      })
    }

    for (const width of [320, 390]) {
      test(`${surface.name} mobile navigation closes on Escape and navigation at ${width}px`, async ({ page }) => {
        await page.setViewportSize({ width, height: 844 })
        const { unmatchedRequests } = await seedWorkspace(page)
        await page.goto(surface.path)
        await expect(page.getByRole('heading', { name: surface.heading, exact: true })).toBeVisible()
        const menu = page.getByRole('button', { name: '打开菜单', exact: true })
        await expectUnobscured(page, menu)
        await menu.click()
        const drawer = page.locator('.s2v-sidebar-overlay')
        const destination = drawer.getByRole('link', { name: surface.destination, exact: true })
        await expect(destination).toBeVisible()
        const box = await rect(drawer)
        expect.soft(box.left).toBeGreaterThanOrEqual(0)
        expect.soft(box.right).toBeLessThanOrEqual(width)
        await destination.focus()
        await page.keyboard.press('Escape')
        await expect(drawer).toHaveCount(0)
        await expect(page.getByRole('button', { name: '关闭菜单', exact: true })).toHaveCount(0)
        await menu.click()
        await destination.click()
        await expect(page).toHaveURL(new RegExp(`${surface.destinationPath}$`))
        await expect(drawer).toHaveCount(0)
        await expect(page.getByRole('button', { name: '关闭菜单', exact: true })).toHaveCount(0)
        await expect(page.getByRole('heading', { name: surface.destination, exact: true })).toBeVisible()
        await expectNoPageOverflow(page)
        await expectUnobscured(page, menu)
        expect(unmatchedRequests, 'Navigation must stay within the deterministic read-only fixtures.').toEqual([])
      })
    }
  }
})
