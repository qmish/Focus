import { test, expect } from '@playwright/test'

const API_URL = process.env.API_URL || 'http://localhost:8080'

test.describe('API Smoke', () => {
  test('health endpoint', async ({ request }) => {
    const response = await request.get(`${API_URL}/health`)
    expect(response.ok()).toBeTruthy()
    const body = await response.json()
    expect(body.status).toBe('healthy')
  })

  test('ready endpoint', async ({ request }) => {
    const response = await request.get(`${API_URL}/ready`)
    expect(response.ok()).toBeTruthy()
  })

  test('protected rooms endpoint without token', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/rooms`)
    expect(response.status()).toBe(401)
  })

  test('protected admin endpoint without token', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/admin/stats`)
    expect(response.status()).toBe(401)
  })
})

test.describe('API Flows', () => {
  test('auth login starts OIDC redirect flow', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/auth/login`, {
      maxRedirects: 0,
    })
    expect(response.status()).toBe(302)
  })

  test('messages endpoint rejects unauthenticated bot command', async ({ request }) => {
    const response = await request.post(`${API_URL}/api/v1/messages`, {
      data: {
        room_id: '00000000-0000-0000-0000-000000000000',
        content: '/status',
        type: 'text',
      },
    })
    expect(response.status()).toBe(401)
  })

  test('admin users endpoint rejects unauthenticated request', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/admin/users`)
    expect(response.status()).toBe(401)
  })

  test('webhook endpoint rejects invalid signature', async ({ request }) => {
    const response = await request.post(`${API_URL}/api/v1/webhooks/jitsi`, {
      headers: {
        'X-Jitsi-Signature': 'sha256=invalid',
        'Content-Type': 'application/json',
      },
      data: {
        event: 'conference.created',
        conference: {
          room_name: 'api-e2e',
        },
      },
    })
    expect(response.status()).toBe(401)
  })
})
