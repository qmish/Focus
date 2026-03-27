import { useState } from 'react'
import { useAdminUi, type AdminThemeMode } from '../providers/AdminUiProvider'

export default function SettingsPage() {
  const { branding, setBranding, theme, setTheme } = useAdminUi()
  const [productName, setProductName] = useState(branding.productName)
  const [logoUrl, setLogoUrl] = useState(branding.logoUrl)
  const [accentColor, setAccentColor] = useState(branding.accentColor)

  const saveBranding = () => {
    setBranding({
      productName: productName.trim() || 'Focus Admin',
      logoUrl: logoUrl.trim() || '/logo.png',
      accentColor: accentColor.trim() || '#2563eb',
    })
  }

  return (
    <div className="settings-page">
      <h1>Настройки</h1>

      <div className="settings-section">
        <h2>Брендирование</h2>
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
          <input type="text" value={accentColor} onChange={(e) => setAccentColor(e.target.value)} />
        </div>
        <button className="primary" onClick={saveBranding}>Сохранить брендирование</button>
      </div>

      <div className="settings-section">
        <h2>Тема</h2>
        <div className="form-group">
          <label>Режим темы</label>
          <select value={theme} onChange={(e) => setTheme(e.target.value as AdminThemeMode)}>
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </div>
      </div>
    </div>
  )
}
