import { defineConfig } from '@playwright/test'
import integration from './playwright.integration.config'

// Unlike opt-in integration suites, this explicit command must not succeed
// with zero executed tests when its credentials or target are missing.
for (const name of ['SUB2VIDEO_LIVE_BASE_URL', 'SUB2VIDEO_LIVE_EMAIL', 'SUB2VIDEO_LIVE_PASSWORD']) {
  if (!process.env[name]?.trim()) throw new Error(`Read-only UAT requires ${name}; no requests were sent`)
}
let target: URL
try { target = new URL(process.env.SUB2VIDEO_LIVE_BASE_URL!) }
catch { throw new Error('Read-only UAT requires a valid HTTP(S) origin') }
if (!['http:', 'https:'].includes(target.protocol) || target.username || target.password || target.search || target.hash || target.pathname !== '/') {
  throw new Error('Read-only UAT requires an HTTP(S) origin without embedded credentials, path or query')
}
if (!['ready', 'unconfigured'].includes(process.env.SUB2VIDEO_UAT_PROFILE || '')) {
  throw new Error('Set SUB2VIDEO_UAT_PROFILE=ready or unconfigured; unavailable generation is never accepted as ready')
}

// Playwright 1.62 also writes a DOM snapshot into error-context.md even when
// screenshots and traces are off. Do not capture authenticated page contents.
process.env.PLAYWRIGHT_NO_COPY_PROMPT = '1'

export default defineConfig({
  ...integration,
  testMatch: 'uat-readonly.spec.ts',
  retries: 0,
  webServer: undefined,
  use: { ...integration.use, baseURL: target.origin, serviceWorkers: 'block', trace: 'off', screenshot: 'off', video: 'off' },
})
