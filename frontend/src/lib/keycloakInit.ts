import Keycloak from 'keycloak-js'

let initPromise: Promise<boolean> | null = null

export function initKeycloak(kc: Keycloak): Promise<boolean> {
  if (initPromise) return initPromise
  initPromise = kc.init({ checkLoginIframe: false, enableLogging: true }).catch(() => false)
  return initPromise
}

export function resetKeycloakInit() {
  initPromise = null
}
