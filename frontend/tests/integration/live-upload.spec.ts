import { createHash } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import { expect, test, type APIRequestContext } from '@playwright/test'

const enabled = process.env.SUB2VIDEO_LIVE === '1'
const email = process.env.SUB2VIDEO_LIVE_EMAIL || ''
const password = process.env.SUB2VIDEO_LIVE_PASSWORD || ''

// Upload authentication and headers must not be saved as traces or screenshots.
// This suite uploads one source image; it never creates a video task.
test.use({ trace: 'off', video: 'off', screenshot: 'off' })

async function accounting(request: APIRequestContext, headers: Record<string, string>) {
  const wallet = await request.get('/api/v1/client/wallet', { headers })
  const tasks = await request.get('/api/v1/client/tasks?page=1&pageSize=100', { headers })
  expect(wallet.status()).toBe(200)
  expect(tasks.status()).toBe(200)
  const walletData = await wallet.json() as { balance: number }
  const taskData = await tasks.json() as { total: number; items: Array<{ id: string }> }
  return { balance: walletData.balance, total: taskData.total, ids: taskData.items.map((item) => item.id) }
}

test.describe('Sub2Video live source-image upload', () => {
  test.skip(!enabled || !email || !password, 'configure the isolated live test environment and account')

  test('uploads and reads identical source bytes with authentication and without charging', async ({ request }, testInfo) => {
    test.skip(testInfo.project.name !== 'live-desktop', 'This API upload runs once; browser layouts are covered by the three-mode preflight.')
    const login = await request.post('/api/v1/auth/login', { data: { email, password } })
    expect(login.status(), 'Real test-account login must succeed').toBe(200)
    const session = await login.json() as { token: { accessToken: string } }
    expect(Boolean(session.token?.accessToken)).toBe(true)
    const headers = { Authorization: `Bearer ${session.token.accessToken}` }
    const original = await readFile(new URL('../../public/brand/auth/iceland-coast.jpg', import.meta.url))
    const originalHash = createHash('sha256').update(original).digest('hex')
    const before = await accounting(request, headers)
    try {
      const upload = await request.post('/api/v1/client/uploads', {
        headers,
        multipart: { file: { name: 'beta-source-image.jpg', mimeType: 'image/jpeg', buffer: original } },
      })
      expect(upload.status(), 'The real image upload must be accepted').toBe(201)
      const source = await upload.json() as { id: string; url: string; mimeType: string; sizeBytes: number }
      expect(source.id).toMatch(/^upl_/)
      expect(source.url).toBe(`/api/v1/client/uploads/${source.id}`)
      expect(source.mimeType).toBe('image/jpeg')
      expect(source.sizeBytes).toBe(original.length)
      expect(Object.keys(source).join(' ')).not.toMatch(/supplier|cost|token|workflow|executionType|apiModelCode|upstream/i)
      const authenticated = await request.get(source.url, { headers })
      expect(authenticated.status(), 'The owning user must be able to read the uploaded source').toBe(200)
      expect(authenticated.headers()['content-type']).toContain('image/jpeg')
      const downloaded = await authenticated.body()
      const downloadedHash = createHash('sha256').update(downloaded).digest('hex')
      expect(downloaded.length).toBe(original.length)
      expect(downloadedHash).toBe(originalHash)
      const anonymous = await request.get(source.url)
      expect(anonymous.status(), 'Anonymous source reads must be rejected').toBe(401)
      await testInfo.attach('source-upload-verification.json', {
        body: JSON.stringify({ uploadId: source.id, sizeBytes: downloaded.length, sha256: downloadedHash, authenticatedStatus: authenticated.status(), anonymousStatus: anonymous.status() }, null, 2),
        contentType: 'application/json',
      })
    } finally {
      expect(await accounting(request, headers), 'Source upload must not create a video task or charge the wallet').toEqual(before)
    }
  })
})
