import { create } from 'zustand'
import Keycloak from 'keycloak-js'

const ADMIN_TOKEN_KEY = 'admin_token'
const KEYCLOAK_URL = import.meta.env.VITE_KEYCLOAK_URL || ''

const keycloak = KEYCLOAK_URL
  ? new Keycloak({
      url: KEYCLOAK_URL,
      realm: import.meta.env.VITE_KEYCLOAK_REALM || 'company',
      clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'messenger-admin',
    })
  : null

interface AdminUser {
  id: string
  email: string
  name: string
  roles: string[]
}

interface AdminAuthState {
  keycloak: Keycloak | null
  isAuthenticated: boolean
  isLoading: boolean
  user: AdminUser | null
  token: string | null
  keycloakAvailable: boolean
  init: () => Promise<void>
  loginLocal: (email: string, password: string) => Promise<void>
  loginKeycloak: () => Promise<void>
  logout: () => Promise<void>
  checkAdmin: () => boolean
  getAccessToken: () => string | null
}

const API_BASE = import.meta.env.VITE_API_URL || '/api'

let initPromise: Promise<void> | null = null

export const useAdminAuthStore = create<AdminAuthState>((set, get) => ({
  keycloak: null,
  isAuthenticated: false,
  isLoading: true,
  user: null,
  token: null,
  keycloakAvailable: !!KEYCLOAK_URL,

  init: async () => {
    if (initPromise) return initPromise
    initPromise = (async () => {
    const saved = localStorage.getItem(ADMIN_TOKEN_KEY)
    if (saved) {
      try {
        const res = await fetch(`${API_BASE}/v1/auth/me`, {
          headers: { Authorization: `Bearer ${saved}` },
        })
        if (res.ok) {
          const user = await res.json()
          const isAdmin = user.roles?.includes('admin')
          if (isAdmin) {
            set({ isAuthenticated: true, isLoading: false, user, token: saved })
            return
          }
        }
      } catch { /* token expired or invalid */ }
      localStorage.removeItem(ADMIN_TOKEN_KEY)
    }

    if (keycloak) {
      try {
        const authenticated = await keycloak.init({
          pkceMethod: 'S256',
          checkLoginIframe: false,
        })

        if (authenticated && keycloak.idToken) {
          try {
            const res = await fetch(`${API_BASE}/v1/auth/token-exchange`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ token: keycloak.idToken }),
            })
            if (res.ok) {
              const data = await res.json()
              const isAdmin = data.user?.roles?.includes('admin')
              if (!isAdmin) {
                await keycloak.logout()
                set({ keycloak, isAuthenticated: false, isLoading: false, user: null, token: null })
                alert('Доступ запрещён: требуется роль администратора')
                return
              }
              localStorage.setItem(ADMIN_TOKEN_KEY, data.access_token)
              set({
                keycloak,
                isAuthenticated: true,
                isLoading: false,
                user: data.user,
                token: data.access_token,
              })
              return
            }
          } catch (err) {
            console.error('Ошибка обмена токена:', err)
          }
        }

        set({ keycloak, isAuthenticated: false, isLoading: false, user: null, token: null })
        return
      } catch (err) {
        console.error('Ошибка инициализации Keycloak:', err)
      }
    }

    set({ isLoading: false })
    })()
    return initPromise!
  },

  loginLocal: async (email: string, password: string) => {
    const res = await fetch(`${API_BASE}/v1/auth/local/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || 'Login failed')
    }
    const data = await res.json()
    const isAdmin = data.user?.roles?.includes('admin')
    if (!isAdmin) {
      throw new Error('Доступ запрещён: требуется роль администратора')
    }
    localStorage.setItem(ADMIN_TOKEN_KEY, data.access_token)
    set({ isAuthenticated: true, isLoading: false, user: data.user, token: data.access_token })
  },

  loginKeycloak: async () => {
    if (keycloak) {
      await keycloak.login()
    }
  },

  logout: async () => {
    const { token } = get()
    if (token) {
      try {
        await fetch(`${API_BASE}/v1/auth/logout`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
        })
      } catch { /* ignore */ }
    }

    if (keycloak?.authenticated) {
      await keycloak.logout({ redirectUri: window.location.origin })
    }

    set({ isAuthenticated: false, user: null, token: null })
    localStorage.removeItem(ADMIN_TOKEN_KEY)
  },

  checkAdmin: () => {
    const { user } = get()
    return user?.roles?.includes('admin') || false
  },

  getAccessToken: () => {
    const token = get().token
    if (token) return token
    return localStorage.getItem(ADMIN_TOKEN_KEY)
  },
}))
