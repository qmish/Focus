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
