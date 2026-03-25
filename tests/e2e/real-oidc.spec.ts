import { test, expect } from '@playwright/test'

const RUN_REAL_OIDC = process.env.E2E_REAL_OIDC === '1'
const APP_URL = process.env.BASE_URL || 'https://chat-stage.company.com'
const KEYCLOAK_URL = process.env.KEYCLOAK_URL || 'https://keycloak-stage.company.com'
const OIDC_USERNAME = process.env.E2E_OIDC_USERNAME || ''
const OIDC_PASSWORD = process.env.E2E_OIDC_PASSWORD || ''

test.describe('Real OIDC Auth Flow', () => {
  test.skip(!RUN_REAL_OIDC, 'Set E2E_REAL_OIDC=1 to enable real auth tests')
  test.skip(!OIDC_USERNAME || !OIDC_PASSWORD, 'Set E2E_OIDC_USERNAME/E2E_OIDC_PASSWORD')

  test('auth -> rooms happy-path via Keycloak login', async ({ page }) => {
    await page.goto(APP_URL)

    const keycloakPattern = new RegExp(`^${KEYCLOAK_URL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`)
    if (!keycloakPattern.test(page.url())) {
      const loginButton = page.getByRole('button', { name: /войти|login/i })
      if (await loginButton.isVisible()) {
        await loginButton.click()
      }
    }

    await expect(page).toHaveURL(keycloakPattern)
    await page.fill('#username', OIDC_USERNAME)
    await page.fill('#password', OIDC_PASSWORD)
    await page.click('#kc-login')

    await expect(page).toHaveURL(/\/rooms|\/$/)
    await expect(page.getByText(/WS:|Комнаты|Rooms/i)).toBeVisible()
  })

  test('authenticated user can create room from UI', async ({ page }) => {
    const keycloakPattern = new RegExp(`^${KEYCLOAK_URL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`)
    await page.goto(`${APP_URL}/rooms`)

    if (keycloakPattern.test(page.url())) {
      await page.fill('#username', OIDC_USERNAME)
      await page.fill('#password', OIDC_PASSWORD)
      await page.click('#kc-login')
      await expect(page).toHaveURL(/\/rooms/)
    }

    const roomName = `OIDC E2E Room ${Date.now()}`
    await page.getByRole('button', { name: /создать комнату|create room/i }).click()
    await page.getByPlaceholder(/название|name/i).fill(roomName)
    await page.getByRole('button', { name: /создать|create/i }).click()
    await expect(page.getByText(roomName)).toBeVisible()
  })
})
