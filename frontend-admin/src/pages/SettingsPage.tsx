import { useEffect, useState } from 'react'
import { useAdminUi, type AdminThemeMode } from '../providers/AdminUiProvider'
import { adminApi, type AppearanceSettings } from '../lib/adminApi'

const defaultAppearance: AppearanceSettings = {
  theme_mode: 'system',
  chat_accent_color: '#89b4fa',
  chat_bg_primary: '#1e1e2e',
  chat_bg_secondary: '#181825',
  chat_text_primary: '#cdd6f4',
  conference_theme_json: '{}',
  branding_product_name: 'Focus',
  branding_logo_url: '/logo.png',
}

export default function SettingsPage() {
  const { branding, setBranding, theme, setTheme } = useAdminUi()
  const [productName, setProductName] = useState(branding.productName)
  const [logoUrl, setLogoUrl] = useState(branding.logoUrl)
  const [accentColor, setAccentColor] = useState(branding.accentColor)

  const [appearance, setAppearance] = useState<AppearanceSettings>(defaultAppearance)
  const [conferenceJson, setConferenceJson] = useState('{}')
  const [serverMsg, setServerMsg] = useState('')
  const [serverErr, setServerErr] = useState('')

  useEffect(() => {
    const load = async () => {
      try {
        const data = await adminApi.getAppearanceSettings()
        if (data.settings) {
          setAppearance(data.settings)
          setConferenceJson(data.settings.conference_theme_json || '{}')
        }
      } catch { /* use defaults */ }
    }
    void load()
  }, [])

  const saveBranding = () => {
    setBranding({
      productName: productName.trim() || 'Focus Admin',
      logoUrl: logoUrl.trim() || '/logo.png',
      accentColor: accentColor.trim() || '#2563eb',
    })
  }

  const saveAppearance = async () => {
    setServerMsg('')
    setServerErr('')
    try {
      await adminApi.putAppearanceSettings({
        ...appearance,
        conference_theme_json: conferenceJson,
      })
      setServerMsg('Настройки внешнего вида сохранены')
      setTimeout(() => setServerMsg(''), 4000)
    } catch (err: any) {
      setServerErr(err.message || 'Не удалось сохранить')
    }
  }

  const previewColors = [
    { label: 'Фон', value: appearance.chat_bg_primary },
    { label: 'Фон вторичный', value: appearance.chat_bg_secondary },
    { label: 'Текст', value: appearance.chat_text_primary },
    { label: 'Акцент', value: appearance.chat_accent_color },
  ]

  return (
    <div className="settings-page">
      <h1>Настройки</h1>

      {/* Admin panel local branding */}
      <div className="settings-section">
        <h2>Брендирование админ-панели</h2>
        <div className="form-group">
          <label>Название продукта</label>
          <input type="text" value={productName} onChange={(e) => setProductName(e.target.value)} />
        </div>
        <div className="form-group">
          <label>URL логотипа</label>
          <input type="text" value={logoUrl} onChange={(e) => setLogoUrl(e.target.value)} />
        </div>
        <div className="form-group">
          <label>Акцентный цвет</label>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <input type="color" value={accentColor} onChange={(e) => setAccentColor(e.target.value)} style={{ width: 40, height: 32, padding: 0, border: 'none' }} />
            <input type="text" value={accentColor} onChange={(e) => setAccentColor(e.target.value)} style={{ flex: 1 }} />
          </div>
        </div>
        <button className="primary" onClick={saveBranding}>Сохранить брендирование</button>
      </div>

      {/* Admin panel theme */}
      <div className="settings-section">
        <h2>Тема админ-панели</h2>
        <div className="form-group">
          <label>Режим темы</label>
          <select value={theme} onChange={(e) => setTheme(e.target.value as AdminThemeMode)}>
            <option value="system">Системная</option>
            <option value="light">Светлая</option>
            <option value="dark">Тёмная</option>
          </select>
        </div>
      </div>

      {/* Global chat + conference theme (server-persisted) */}
      <div className="settings-section">
        <h2>Тема чата (глобальная)</h2>
        <p style={{ fontSize: '0.85rem', color: 'var(--muted-color, #888)', marginBottom: 12 }}>
          Эти настройки применяются ко всем пользователям чата.
        </p>

        {serverMsg && <p style={{ color: 'var(--success-color, #4ade80)', marginBottom: 8 }}>{serverMsg}</p>}
        {serverErr && <p className="error">{serverErr}</p>}

        <div className="form-group">
          <label>Режим темы чата</label>
          <select value={appearance.theme_mode} onChange={(e) => setAppearance({ ...appearance, theme_mode: e.target.value })}>
            <option value="system">Системная</option>
            <option value="light">Светлая</option>
            <option value="dark">Тёмная</option>
          </select>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <div className="form-group">
            <label>Акцентный цвет</label>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input type="color" value={appearance.chat_accent_color} onChange={(e) => setAppearance({ ...appearance, chat_accent_color: e.target.value })} style={{ width: 40, height: 32, padding: 0, border: 'none' }} />
              <input type="text" value={appearance.chat_accent_color} onChange={(e) => setAppearance({ ...appearance, chat_accent_color: e.target.value })} style={{ flex: 1 }} />
            </div>
          </div>
          <div className="form-group">
            <label>Основной фон</label>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input type="color" value={appearance.chat_bg_primary} onChange={(e) => setAppearance({ ...appearance, chat_bg_primary: e.target.value })} style={{ width: 40, height: 32, padding: 0, border: 'none' }} />
              <input type="text" value={appearance.chat_bg_primary} onChange={(e) => setAppearance({ ...appearance, chat_bg_primary: e.target.value })} style={{ flex: 1 }} />
            </div>
          </div>
          <div className="form-group">
            <label>Вторичный фон</label>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input type="color" value={appearance.chat_bg_secondary} onChange={(e) => setAppearance({ ...appearance, chat_bg_secondary: e.target.value })} style={{ width: 40, height: 32, padding: 0, border: 'none' }} />
              <input type="text" value={appearance.chat_bg_secondary} onChange={(e) => setAppearance({ ...appearance, chat_bg_secondary: e.target.value })} style={{ flex: 1 }} />
            </div>
          </div>
          <div className="form-group">
            <label>Основной текст</label>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input type="color" value={appearance.chat_text_primary} onChange={(e) => setAppearance({ ...appearance, chat_text_primary: e.target.value })} style={{ width: 40, height: 32, padding: 0, border: 'none' }} />
              <input type="text" value={appearance.chat_text_primary} onChange={(e) => setAppearance({ ...appearance, chat_text_primary: e.target.value })} style={{ flex: 1 }} />
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginBottom: 12 }}>
          <span style={{ fontWeight: 600, fontSize: '0.85rem' }}>Превью:</span>
          {previewColors.map((c) => (
            <div key={c.label} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <div style={{ width: 24, height: 24, borderRadius: 4, background: c.value, border: '1px solid #555' }} />
              <span style={{ fontSize: '0.75rem' }}>{c.label}</span>
            </div>
          ))}
        </div>

        <div className="form-group">
          <label>Название продукта (глобальное)</label>
          <input value={appearance.branding_product_name} onChange={(e) => setAppearance({ ...appearance, branding_product_name: e.target.value })} />
        </div>
        <div className="form-group">
          <label>URL логотипа (глобальное)</label>
          <input value={appearance.branding_logo_url} onChange={(e) => setAppearance({ ...appearance, branding_logo_url: e.target.value })} />
        </div>
      </div>

      <div className="settings-section">
        <h2>Тема конференций</h2>
        <p style={{ fontSize: '0.85rem', color: 'var(--muted-color, #888)', marginBottom: 12 }}>
          JSON-палитра Jitsi (palette.ui01, palette.ui02, palette.action01, palette.text01).
        </p>
        <div className="form-group">
          <label>Conference theme JSON</label>
          <textarea
            value={conferenceJson}
            onChange={(e) => setConferenceJson(e.target.value)}
            rows={5}
            style={{ width: '100%', fontFamily: 'monospace', fontSize: '0.85rem', padding: 8, borderRadius: 6, border: '1px solid var(--border-color, #444)', background: 'var(--input-bg, #1e1e2e)', color: 'var(--text-color, #cdd6f4)' }}
            placeholder={'{\n  "palette.ui01": "#0B1220",\n  "palette.action01": "#0EA5E9"\n}'}
          />
        </div>
        <button className="primary" onClick={() => void saveAppearance()}>Сохранить настройки внешнего вида</button>
      </div>
    </div>
  )
}
