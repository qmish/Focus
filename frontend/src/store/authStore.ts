import { create } from 'zustand'
import Keycloak from 'keycloak-js'

const ACCESS_TOKEN_KEY = 'focus_access_token'

// Инициализация Keycloak
const keycloak = new Keycloak({
  url: import.meta.env.VITE_KEYCLOAK_URL || 'http://localhost:8180',
  realm: import.meta.env.VITE_KEYCLOAK_REALM || 'company',
  clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'messenger-frontend',
})

interface AuthState {
  keycloak: typeof keycloak | null
  isAuthenticated: boolean
  isLoading: boolean
  user: {
    id: string
    email: string
    name: string
    roles: string[]
  } | null
  token: string | null
  init: () => Promise<void>
  login: () => Promise<void>
  logout: () => Promise<void>
  refreshToken: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => ({
  keycloak: null,
  isAuthenticated: false,
  isLoading: true,
  user: null,
  token: null,

  init: async () => {
    try {
      const authenticated = await keycloak.init({
        onLoad: 'check-sso',
        silentCheckSsoRedirectUri: window.location.origin + '/silent-check-sso.html',
        pkceMethod: 'S256',
      })

      const token = keycloak.token || localStorage.getItem(ACCESS_TOKEN_KEY) || null
      const user = token ? {
        id: keycloak.subject || '',
        email: keycloak.idTokenParsed?.email || '',
        name: keycloak.idTokenParsed?.name || '',
        roles: keycloak.resourceAccess?.['messenger-api']?.roles || [],
      } : null

      if (token) {
        localStorage.setItem(ACCESS_TOKEN_KEY, token)
      } else {
        localStorage.removeItem(ACCESS_TOKEN_KEY)
      }

      set({
        keycloak,
        isAuthenticated: authenticated || Boolean(token),
        isLoading: false,
        user,
        token,
      })

      // Настраиваем автоматическое обновление токена
      if (authenticated) {
        setInterval(async () => {
          try {
            await keycloak.updateToken(30)
            const nextToken = keycloak.token || null
            set({
              token: nextToken,
              isAuthenticated: Boolean(nextToken),
            })
            if (nextToken) {
              localStorage.setItem(ACCESS_TOKEN_KEY, nextToken)
            } else {
              localStorage.removeItem(ACCESS_TOKEN_KEY)
            }
          } catch {
            get().logout()
          }
        }, 60000)
      }
    } catch (error) {
      console.error('Failed to initialize Keycloak:', error)
      set({ isLoading: false })
    }
  },

  login: async () => {
    await keycloak.login()
  },

  logout: async () => {
    await keycloak.logout({ redirectUri: window.location.origin })
    set({
      isAuthenticated: false,
      user: null,
      token: null,
    })
    localStorage.removeItem(ACCESS_TOKEN_KEY)
  },

  refreshToken: async () => {
    try {
      await keycloak.updateToken(30)
      const nextToken = keycloak.token || null
      set({
        token: nextToken,
        isAuthenticated: Boolean(nextToken),
      })
      if (nextToken) {
        localStorage.setItem(ACCESS_TOKEN_KEY, nextToken)
      } else {
        localStorage.removeItem(ACCESS_TOKEN_KEY)
      }
    } catch (error) {
      console.error('Failed to refresh token:', error)
      get().logout()
    }
  },
}))
