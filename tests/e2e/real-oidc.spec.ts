import { test, expect } from '@playwright/test'

const RUN_REAL_OIDC = process.env.E2E_REAL_OIDC === '1'
const APP_URL = process.env.BASE_URL || 'https://chat-stage.company.com'
const KEYCLOAK_URL = process.env.KEYCLOAK_URL || 'https://keycloak-stage.company.com'

test.describe('Real OIDC Auth Flow', () => {
  test.skip(!RUN_REAL_OIDC, 'Set E2E_REAL_OIDC=1 to enable real auth tests')

  test('redirects to Keycloak login from application', async ({ page }) => {
    await page.goto(APP_URL)

    // The app should trigger auth flow and redirect to Keycloak.
    await expect(page).toHaveURL(new RegExp(`^${KEYCLOAK_URL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`))
  })
})
