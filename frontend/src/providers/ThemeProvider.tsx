import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'

type ThemeContextValue = {
  effectiveTheme: 'light' | 'dark'
}

const ThemeContext = createContext<ThemeContextValue>({ effectiveTheme: 'dark' })

function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [effectiveTheme, setEffectiveTheme] = useState<'light' | 'dark'>('dark')

  useEffect(() => {
    const fetchAndApply = async () => {
      try {
        const res = await fetch('/api/v1/settings/appearance')
        if (!res.ok) return
        const data = await res.json()
        const settings = data?.settings
        if (!settings) return

        let mode: 'light' | 'dark' = 'dark'
        if (settings.theme_mode === 'light') mode = 'light'
        else if (settings.theme_mode === 'system') mode = getSystemTheme()
        else mode = 'dark'

        setEffectiveTheme(mode)
        document.documentElement.setAttribute('data-theme', mode)

        const root = document.documentElement
        if (settings.chat_accent_color) root.style.setProperty('--accent', settings.chat_accent_color)
        if (settings.chat_bg_primary) root.style.setProperty('--bg-primary', settings.chat_bg_primary)
        if (settings.chat_bg_secondary) root.style.setProperty('--bg-secondary', settings.chat_bg_secondary)
        if (settings.chat_text_primary) root.style.setProperty('--text-primary', settings.chat_text_primary)
      } catch {
        // keep defaults
      }
    }
    void fetchAndApply()

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => {
      const saved = document.documentElement.getAttribute('data-theme-mode')
      if (saved === 'system') {
        const next = media.matches ? 'dark' : 'light'
        setEffectiveTheme(next)
        document.documentElement.setAttribute('data-theme', next)
      }
    }
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])

  return (
    <ThemeContext.Provider value={{ effectiveTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  return useContext(ThemeContext)
}
