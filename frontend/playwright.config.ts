import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  // A single Vite server serves deterministic screenshot fixtures. Four
  // workers keep browser coverage concurrent without starving route loads.
  workers: 4,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: [['list'], ['html', { outputFolder: '.tmp_playwright_report', open: 'never' }]],
  use: {
    baseURL: 'http://127.0.0.1:4185',
    locale: 'zh-CN',
    colorScheme: 'light',
    reducedMotion: 'reduce',
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'pnpm dev --host 127.0.0.1 --port 4185',
    // Unhandled fixture requests must never reach a developer's live backend.
    env: { VITE_API_PROXY: 'http://127.0.0.1:9' },
    url: 'http://127.0.0.1:4185',
    reuseExistingServer: process.env.PLAYWRIGHT_REUSE_EXISTING_SERVER === '1',
    timeout: 120_000,
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } },
    { name: 'mobile', use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 }, isMobile: false, hasTouch: true } },
  ],
})
