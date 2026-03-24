import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useAuthStore } from '../store/authStore'

// Mock keycloak-js
vi.mock('keycloak-js', () => {
  const mockKeycloak = {
    init: vi.fn(() => Promise.resolve(true)),
    login: vi.fn(() => Promise.resolve()),
    logout: vi.fn(() => Promise.resolve()),
    updateToken: vi.fn(() => Promise.resolve(true)),
    token: 'mock-token',
    subject: 'user-123',
    idTokenParsed: {
      email: 'test@example.com',
      name: 'Test User',
    },
    resourceAccess: {
      'messenger-api': {
        roles: ['user'],
      },
    },
  }
  return { default: vi.fn(() => mockKeycloak) }
})

describe('AuthStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should initialize with loading state', () => {
    const state = useAuthStore.getState()
    expect(state.isLoading).toBe(true)
    expect(state.isAuthenticated).toBe(false)
  })

  it('should initialize Keycloak successfully', async () => {
    const { init } = useAuthStore.getState()
    await init()

    const state = useAuthStore.getState()
    expect(state.isLoading).toBe(false)
    expect(state.isAuthenticated).toBe(true)
    expect(state.user).toBeTruthy()
    expect(state.user?.email).toBe('test@example.com')
    expect(state.user?.name).toBe('Test User')
  })

  it('should call login', async () => {
    const { login } = useAuthStore.getState()
    await login()

    const Keycloak = (await import('keycloak-js')).default
    expect(Keycloak().login).toHaveBeenCalled()
  })

  it('should call logout', async () => {
    const { logout } = useAuthStore.getState()
    await logout()

    const Keycloak = (await import('keycloak-js')).default
    expect(Keycloak().logout).toHaveBeenCalled()
  })

  it('should refresh token', async () => {
    const { refreshToken } = useAuthStore.getState()
    await refreshToken()

    const Keycloak = (await import('keycloak-js')).default
    expect(Keycloak().updateToken).toHaveBeenCalledWith(30)
  })
})
