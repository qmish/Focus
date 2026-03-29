import { create } from 'zustand'
import Keycloak from 'keycloak-js'
import { initKeycloak } from '../lib/keycloakInit'

const ACCESS_TOKEN_KEY = 'focus_access_token'
const KEYCLOAK_URL = import.meta.env.VITE_KEYCLOAK_URL || ''
const isTauri = '__TAURI__' in window

function generateCodeVerifier(): string {
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  return Array.from(array, (b) => b.toString(16).padStart(2, '0')).join('')
}

async function generateCodeChallenge(verifier: string): Promise<string> {
  const data = new TextEncoder().encode(verifier)
  const digest = await crypto.subtle.digest('SHA-256', data)
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

const keycloak = KEYCLOAK_URL
  ? new Keycloak({
      url: KEYCLOAK_URL,
      realm: import.meta.env.VITE_KEYCLOAK_REALM || 'company',
      clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'messenger-frontend',
    })
  : null

interface AuthUser {
  id: string
  email: string
  name: string
  roles: string[]
  department?: string
  directorate?: string
  position?: string
  phone?: string
  about_me?: string
  video_start_with_audio_muted?: boolean
  video_start_with_video_muted?: boolean
  video_display_name?: string
  video_default_language?: string
}

interface AuthState {
  keycloak: Keycloak | null
  isAuthenticated: boolean
  isLoading: boolean
  user: AuthUser | null
  token: string | null
  keycloakAvailable: boolean
  init: () => Promise<void>
  loginLocal: (email: string, password: string) => Promise<void>
  registerLocal: (email: string, password: string, name: string) => Promise<void>
  loginKeycloak: () => Promise<void>
  logout: () => Promise<void>
  refreshToken: () => Promise<void>
}

const API_BASE = import.meta.env.VITE_API_URL || '/api'

let storeInitPromise: Promise<void> | null = null

export const useAuthStore = create<AuthState>((set, get) => ({
  keycloak: null,
  isAuthenticated: false,
  isLoading: true,
  user: null,
  token: null,
  keycloakAvailable: !!KEYCLOAK_URL,

  init: async () => {
    if (storeInitPromise) return storeInitPromise
    storeInitPromise = (async () => {

    if (isTauri) {
      try {
        const { listen } = await import('@tauri-apps/api/event')
        const { invoke } = await import('@tauri-apps/api/core')
        listen<string>('auth-deep-link', async (event) => {
          try {
            const url = new URL(event.payload)
            const code = url.searchParams.get('code')
            if (!code) return
            const kcUrl = KEYCLOAK_URL
            const realm = import.meta.env.VITE_KEYCLOAK_REALM || 'company'
            const clientId = import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'messenger-frontend'
            const tokens = await invoke<{ id_token?: string; access_token?: string }>('exchange_auth_code', {
              keycloakUrl: kcUrl,
              realm,
              clientId,
              redirectUri: 'focus://auth/callback',
              code,
            })
            const idToken = tokens.id_token
            if (idToken) {
              const exchangeRes = await fetch(`${API_BASE}/v1/auth/token-exchange`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token: idToken }),
              })
              if (exchangeRes.ok) {
                const data = await exchangeRes.json()
                localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token)
                set({
                  isAuthenticated: true,
                  isLoading: false,
                  user: data.user,
                  token: data.access_token,
                })
              }
            }
          } catch (err) {
            console.error('Tauri auth callback error:', err)
          }
        })
      } catch (err) {
        console.error('Tauri event listen failed:', err)
      }
    }

    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    if (code) {
      window.history.replaceState({}, '', window.location.pathname)
      try {
        const kcUrl = KEYCLOAK_URL
        const realm = import.meta.env.VITE_KEYCLOAK_REALM || 'company'
        const clientId = import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'messenger-frontend'
        const codeVerifier = sessionStorage.getItem('pkce_code_verifier') || ''
        sessionStorage.removeItem('pkce_code_verifier')
        const tokenUrl = `${kcUrl}/realms/${realm}/protocol/openid-connect/token`
        const tokenBody: Record<string, string> = {
          grant_type: 'authorization_code',
          client_id: clientId,
          code,
          redirect_uri: window.location.origin + '/',
        }
        if (codeVerifier) {
          tokenBody.code_verifier = codeVerifier
        }
        const tokenRes = await fetch(tokenUrl, {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: new URLSearchParams(tokenBody),
        })
        if (tokenRes.ok) {
          const tokens = await tokenRes.json()
          const idToken = tokens.id_token
          if (idToken) {
            const exchangeRes = await fetch(`${API_BASE}/v1/auth/token-exchange`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ token: idToken }),
            })
            if (exchangeRes.ok) {
              const data = await exchangeRes.json()
              localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token)
              set({
                keycloak,
                isAuthenticated: true,
                isLoading: false,
                user: data.user,
                token: data.access_token,
              })
              return
            }
          }
        }
      } catch (err) {
        console.error('OIDC callback handling failed:', err)
      }
    }

    const saved = localStorage.getItem(ACCESS_TOKEN_KEY)
    if (saved) {
      try {
        const res = await fetch(`${API_BASE}/v1/auth/me`, {
          headers: { Authorization: `Bearer ${saved}` },
        })
        if (res.ok) {
          const user = await res.json()
          set({
            isAuthenticated: true,
            isLoading: false,
            user,
            token: saved,
          })
          return
        }
      } catch { /* token expired or invalid */ }
      localStorage.removeItem(ACCESS_TOKEN_KEY)
    }

    if (keycloak) {
      try {
        const authenticated = await initKeycloak(keycloak)

        if (authenticated && keycloak.idToken) {
          try {
            const res = await fetch(`${API_BASE}/v1/auth/token-exchange`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ token: keycloak.idToken }),
            })
            if (res.ok) {
              const data = await res.json()
              localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token)
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
            console.error('Token exchange failed:', err)
          }
        }

        set({ keycloak, isAuthenticated: false, isLoading: false, user: null, token: null })
        return
      } catch (err) {
        console.error('Keycloak init failed:', err)
      }
    }

    set({ isLoading: false })
    })()
    return storeInitPromise!
  },

  loginLocal: async (email: string, password: string) => {
    const res = await fetch(`${API_BASE}/v1/auth/local/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })

    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || 'Ошибка входа')
    }

    const data = await res.json()
    localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token)
    set({
      isAuthenticated: true,
      isLoading: false,
      user: data.user,
      token: data.access_token,
    })
  },

  registerLocal: async (email: string, password: string, name: string) => {
    const res = await fetch(`${API_BASE}/v1/auth/local/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, name }),
    })

    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || 'Ошибка регистрации')
    }

    const data = await res.json()
    localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token)
    set({
      isAuthenticated: true,
      isLoading: false,
      user: data.user,
      token: data.access_token,
    })
  },

  loginKeycloak: async () => {
    const kcUrl = KEYCLOAK_URL
    if (!kcUrl) {
      throw new Error('Keycloak не сконфигурирован (VITE_KEYCLOAK_URL пуст)')
    }
    const realm = import.meta.env.VITE_KEYCLOAK_REALM || 'company'
    const clientId = import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'messenger-frontend'

    if (isTauri) {
      const { invoke } = await import('@tauri-apps/api/core')
      await invoke('open_keycloak_auth', {
        keycloakUrl: kcUrl,
        realm,
        clientId,
        redirectUri: 'focus://auth/callback',
      })
      return
    }

    const redirectUri = window.location.origin + '/'
    const codeVerifier = generateCodeVerifier()
    const codeChallenge = await generateCodeChallenge(codeVerifier)
    sessionStorage.setItem('pkce_code_verifier', codeVerifier)
    const authUrl =
      `${kcUrl}/realms/${realm}/protocol/openid-connect/auth` +
      `?client_id=${encodeURIComponent(clientId)}` +
      `&redirect_uri=${encodeURIComponent(redirectUri)}` +
      `&response_type=code` +
      `&scope=openid` +
      `&code_challenge=${encodeURIComponent(codeChallenge)}` +
      `&code_challenge_method=S256`
    window.location.href = authUrl
  },

  logout: async () => {
    const { token } = get()
    if (token && token !== 'bypass-token') {
      try {
        await fetch(`${API_BASE}/v1/auth/logout`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
        })
      } catch { /* ignore */ }
    }

    if (!isTauri && keycloak?.authenticated) {
      await keycloak.logout({ redirectUri: window.location.origin })
    }

    set({ isAuthenticated: false, user: null, token: null })
    localStorage.removeItem(ACCESS_TOKEN_KEY)
  },

  refreshToken: async () => {
    if (keycloak?.authenticated) {
      try {
        await keycloak.updateToken(30)
        const nextToken = keycloak.token || null
        set({ token: nextToken, isAuthenticated: Boolean(nextToken) })
        if (nextToken) localStorage.setItem(ACCESS_TOKEN_KEY, nextToken)
        else localStorage.removeItem(ACCESS_TOKEN_KEY)
      } catch {
        get().logout()
      }
    }
  },
}))
