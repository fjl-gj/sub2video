import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.SUB2VIDEO_LIVE_BASE_URL || 'http://127.0.0.1:4185'
const startFrontend = process.env.SUB2VIDEO_START_FRONTEND === '1'

export default defineConfig({
  testDir: './tests/integration',
  fullyParallel: false,
  workers: 1,
  forbidOnly: true,
  retries: process.env.CI ? 2 : 0,
  reporter: [['list'], ['html', { outputFolder: '.tmp_playwright_integration_report', open: 'never' }]],
  use: {
    baseURL,
    locale: 'zh-CN',
    colorScheme: 'light',
    reducedMotion: 'reduce',
    trace: 'retain-on-failure',
  },
  ...(startFrontend ? {
    webServer: {
      command: 'pnpm dev --host 127.0.0.1 --port 4185',
      url: 'http://127.0.0.1:4185',
      reuseExistingServer: process.env.PLAYWRIGHT_REUSE_EXISTING_SERVER === '1',
      timeout: 120_000,
    },
  } : {}),
  projects: [
    {
      name: 'live-desktop',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
    },
    {
      name: 'live-mobile',
      use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 }, isMobile: false, hasTouch: true },
    },
  ],
})
