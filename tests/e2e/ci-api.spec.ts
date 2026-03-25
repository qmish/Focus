import { test, expect } from '@playwright/test'
import { createHmac, randomUUID } from 'crypto'

const API_URL = process.env.API_URL || 'http://localhost:8080'
const SESSION_SECRET = process.env.SESSION_SECRET || 'ci-session-secret'

const toBase64Url = (value: string) => Buffer.from(value).toString('base64url')

const createSessionToken = (roles: string[]) => {
  const now = Math.floor(Date.now() / 1000)
  const payload = {
    user_id: randomUUID(),
    email: 'e2e@example.com',
    name: 'E2E User',
    roles,
    keycloak_id: randomUUID(),
    session_id: randomUUID(),
    iss: 'focus-api',
    aud: ['focus-frontend'],
    iat: now,
    nbf: now,
    exp: now + 3600,
  }

  const header = { alg: 'HS256', typ: 'JWT' }
  const encodedHeader = toBase64Url(JSON.stringify(header))
  const encodedPayload = toBase64Url(JSON.stringify(payload))
  const unsigned = `${encodedHeader}.${encodedPayload}`
  const signature = createHmac('sha256', SESSION_SECRET).update(unsigned).digest('base64url')

  return `${unsigned}.${signature}`
}

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

test.describe('API Happy Paths', () => {
  test('auth me returns claims for valid user token', async ({ request }) => {
    const token = createSessionToken(['user'])
    const response = await request.get(`${API_URL}/api/v1/auth/me`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    expect(response.status()).toBe(200)
    const body = await response.json()
    expect(body).toHaveProperty('id')
    expect(body).toHaveProperty('roles')
  })

  test('rooms list works for authenticated user', async ({ request }) => {
    const token = createSessionToken(['user'])
    const response = await request.get(`${API_URL}/api/v1/rooms`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    expect(response.status()).toBe(200)
    const body = await response.json()
    expect(body).toHaveProperty('data')
    expect(Array.isArray(body.data)).toBeTruthy()
  })

  test('admin stats works for admin role token', async ({ request }) => {
    const token = createSessionToken(['admin'])
    const response = await request.get(`${API_URL}/api/v1/admin/stats`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    expect(response.status()).toBe(200)
    const body = await response.json()
    expect(body).toHaveProperty('users')
    expect(body).toHaveProperty('rooms')
  })
})
