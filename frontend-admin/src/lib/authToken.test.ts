import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getAdminAccessToken } from './authToken'
import { useAdminAuthStore } from '../store/adminAuthStore'

vi.mock('keycloak-js', () => {
  class KeycloakMock {
    token: string | null = null
    subject = ''
    idTokenParsed = {}
    resourceAccess = {}
    init = vi.fn()
    login = vi.fn()
    logout = vi.fn()
    updateToken = vi.fn()
  }
  return {
    default: KeycloakMock,
  }
})

class LocalStorageMock {
  private store: Record<string, string> = {}

  getItem(key: string): string | null {
    return this.store[key] ?? null
  }

  setItem(key: string, value: string): void {
    this.store[key] = value
  }

  removeItem(key: string): void {
    delete this.store[key]
  }

  clear(): void {
    this.store = {}
  }
}

describe('getAdminAccessToken', () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, 'localStorage', {
      value: new LocalStorageMock(),
      writable: true,
      configurable: true,
    })
    useAdminAuthStore.setState({ token: null })
  })

  it('returns token from auth store first', () => {
    useAdminAuthStore.setState({ token: 'store-token' })
    localStorage.setItem('admin_token', 'local-token')
    expect(getAdminAccessToken()).toBe('store-token')
  })

  it('falls back to localStorage token', () => {
    localStorage.setItem('admin_token', 'local-token')
    expect(getAdminAccessToken()).toBe('local-token')
  })
})
