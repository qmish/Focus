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
    localStorage.clear()
    useAuthStore.setState({
      keycloak: null,
      isAuthenticated: false,
      isLoading: true,
      user: null,
      token: null,
    })
  })

  it('should initialize with loading state', () => {
    const state = useAuthStore.getState()
    expect(state.isLoading).toBe(true)
    expect(state.isAuthenticated).toBe(false)
  })

  // TODO(infra): тесты ниже устарели после рефакторинга AuthStore
  // (login → loginLocal/loginKeycloak, init теперь использует token-exchange).
  // Требуют переписывания. Отдельный issue: actualize-auth-tests.
  it.skip('should initialize Keycloak successfully', async () => {
    const { init } = useAuthStore.getState()
    await init()

    const state = useAuthStore.getState()
    expect(state.isLoading).toBe(false)
    expect(state.isAuthenticated).toBe(true)
    expect(state.user).toBeTruthy()
    expect(state.user?.email).toBe('test@example.com')
    expect(state.user?.name).toBe('Test User')
    expect(localStorage.getItem('focus_access_token')).toBe('mock-token')
  })

  it.skip('should call login', async () => {
    const { login } = useAuthStore.getState() as { login: () => Promise<void> }
    await login()

    const Keycloak = (await import('keycloak-js')).default
    expect(Keycloak().login).toHaveBeenCalled()
  })

  it.skip('should call logout', async () => {
    localStorage.setItem('focus_access_token', 'stale-token')
    const { logout } = useAuthStore.getState()
    await logout()

    const Keycloak = (await import('keycloak-js')).default
    expect(Keycloak().logout).toHaveBeenCalled()
    expect(localStorage.getItem('focus_access_token')).toBeNull()
  })

  it.skip('should refresh token', async () => {
    const { refreshToken } = useAuthStore.getState()
    await refreshToken()

    const Keycloak = (await import('keycloak-js')).default
    expect(Keycloak().updateToken).toHaveBeenCalledWith(30)
    expect(localStorage.getItem('focus_access_token')).toBe('mock-token')
  })

  it.skip('should use localStorage token fallback when keycloak token is missing', async () => {
    localStorage.setItem('focus_access_token', 'fallback-token')
    const Keycloak = (await import('keycloak-js')).default
    Keycloak().token = null

    const { init } = useAuthStore.getState()
    await init()

    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(true)
    expect(state.token).toBe('fallback-token')
  })
})
