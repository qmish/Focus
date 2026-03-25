import { create } from 'zustand'
import Keycloak from 'keycloak-js'

const keycloak = new Keycloak({
  url: import.meta.env.VITE_KEYCLOAK_URL || 'http://localhost:8180',
  realm: import.meta.env.VITE_KEYCLOAK_REALM || 'company',
  clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'messenger-admin',
})

interface AdminUser {
  id: string
  email: string
  name: string
  roles: string[]
}

interface AdminAuthState {
  keycloak: typeof keycloak | null
  isAuthenticated: boolean
  isLoading: boolean
  user: AdminUser | null
  token: string | null
  init: () => Promise<void>
  login: () => Promise<void>
  logout: () => Promise<void>
  checkAdmin: () => boolean
  getAccessToken: () => string | null
}

export const useAdminAuthStore = create<AdminAuthState>((set, get) => ({
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

      const token = keycloak.token || null
      const user = token ? {
        id: keycloak.subject || '',
        email: keycloak.idTokenParsed?.email || '',
        name: keycloak.idTokenParsed?.name || '',
        roles: keycloak.resourceAccess?.['messenger-api']?.roles || [],
      } : null

      // Проверка роли администратора
      const isAdmin = user?.roles?.includes('admin') || false

      if (!isAdmin && authenticated) {
        // Нет роли администратора - logout
        await keycloak.logout()
        set({
          keycloak,
          isAuthenticated: false,
          isLoading: false,
          user: null,
          token: null,
        })
        alert('Доступ запрещён: требуется роль администратора')
        return
      }

      set({
        keycloak,
        isAuthenticated: authenticated && isAdmin,
        isLoading: false,
        user,
        token,
      })
      if (token) {
        localStorage.setItem('admin_token', token)
      } else {
        localStorage.removeItem('admin_token')
      }

      // Авто-обновление токена
      if (authenticated) {
        setInterval(async () => {
          try {
            await keycloak.updateToken(30)
            set({ token: keycloak.token })
            if (keycloak.token) {
              localStorage.setItem('admin_token', keycloak.token)
            } else {
              localStorage.removeItem('admin_token')
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
    localStorage.removeItem('admin_token')
  },

  checkAdmin: () => {
    const { user } = get()
    return user?.roles?.includes('admin') || false
  },

  getAccessToken: () => {
    const token = get().token
    if (token) {
      return token
    }
    return localStorage.getItem('admin_token')
  },
}))
