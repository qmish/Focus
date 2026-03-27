import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'

export type AdminThemeMode = 'light' | 'dark' | 'system'

type BrandingConfig = {
  productName: string
  logoUrl: string
  accentColor: string
}

type AdminUiContextValue = {
  theme: AdminThemeMode
  effectiveTheme: 'light' | 'dark'
  setTheme: (theme: AdminThemeMode) => void
  branding: BrandingConfig
  setBranding: (next: BrandingConfig) => void
}

const THEME_KEY = 'focus_admin_theme'
const BRANDING_KEY = 'focus_admin_branding'

const defaultBranding: BrandingConfig = {
  productName: 'Focus Admin',
  logoUrl: '/logo.png',
  accentColor: '#2563eb',
}

const AdminUiContext = createContext<AdminUiContextValue | null>(null)

function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function AdminUiProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<AdminThemeMode>(() => {
    const saved = localStorage.getItem(THEME_KEY)
    if (saved === 'light' || saved === 'dark' || saved === 'system') return saved
    return 'system'
  })
  const [branding, setBrandingState] = useState<BrandingConfig>(() => {
    const saved = localStorage.getItem(BRANDING_KEY)
    if (!saved) return defaultBranding
    try {
      const parsed = JSON.parse(saved) as Partial<BrandingConfig>
      return {
        productName: parsed.productName?.trim() || defaultBranding.productName,
        logoUrl: parsed.logoUrl?.trim() || defaultBranding.logoUrl,
        accentColor: parsed.accentColor?.trim() || defaultBranding.accentColor,
      }
    } catch {
      return defaultBranding
    }
  })
  const [effectiveTheme, setEffectiveTheme] = useState<'light' | 'dark'>(() => (theme === 'system' ? getSystemTheme() : theme))

  useEffect(() => {
    localStorage.setItem(THEME_KEY, theme)
    if (theme !== 'system') {
      setEffectiveTheme(theme)
      return
    }
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => setEffectiveTheme(media.matches ? 'dark' : 'light')
    onChange()
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [theme])

  useEffect(() => {
    localStorage.setItem(BRANDING_KEY, JSON.stringify(branding))
    document.documentElement.style.setProperty('--primary-color', branding.accentColor || defaultBranding.accentColor)
  }, [branding])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', effectiveTheme)
  }, [effectiveTheme])

  const value = useMemo<AdminUiContextValue>(() => ({
    theme,
    effectiveTheme,
    setTheme: setThemeState,
    branding,
    setBranding: setBrandingState,
  }), [theme, effectiveTheme, branding])

  return <AdminUiContext.Provider value={value}>{children}</AdminUiContext.Provider>
}

export function useAdminUi() {
  const context = useContext(AdminUiContext)
  if (!context) throw new Error('useAdminUi must be used within AdminUiProvider')
  return context
}
