import { useEffect, useState } from 'react'
import { adminApi } from '../lib/adminApi'

type ExchangeSettings = {
  ews_url: string
  username: string
  password: string
  domain: string
  auth_mode: string
  insecure_tls: boolean
  timeout_seconds: number
  sync_enabled: boolean
  sync_interval_seconds: number
}

export default function IntegrationsPage() {
  const [settings, setSettings] = useState<ExchangeSettings>({
    ews_url: '',
    username: '',
    password: '',
    domain: '',
    auth_mode: 'basic',
    insecure_tls: false,
    timeout_seconds: 15,
    sync_enabled: false,
    sync_interval_seconds: 120,
  })
  const [result, setResult] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    const load = async () => {
      try {
        const data: any = await adminApi.getExchangeSettings()
        if (data?.configured && data.settings) {
          setSettings((prev) => ({
            ...prev,
            ...data.settings,
            password: '',
          }))
        }
      } catch (err: any) {
        setError(err.message || 'Не удалось загрузить настройки Exchange')
      }
    }
    void load()
  }, [])

  const save = async () => {
    setError('')
    setResult('')
    try {
      await adminApi.putExchangeSettings(settings as unknown as Record<string, unknown>)
      setResult('Настройки сохранены')
      setSettings((prev) => ({ ...prev, password: '' }))
    } catch (err: any) {
      setError(err.message || 'Не удалось сохранить настройки')
    }
  }

  const testConnection = async () => {
    setError('')
    setResult('')
    try {
      const data: any = await adminApi.testExchangeConnection({})
      setResult(data?.ok ? 'Подключение успешно' : 'Подключение завершилось с ошибкой')
    } catch (err: any) {
      setError(err.message || 'Проверка подключения не удалась')
    }
  }

  return (
    <div>
      <h1>Интеграции</h1>
      {error && <p className="error">{error}</p>}
      {result && <p>{result}</p>}
      <div className="settings-section">
        <h2>Exchange (on-prem EWS)</h2>
        <div className="form-group">
          <label>EWS URL</label>
          <input value={settings.ews_url} onChange={(e) => setSettings({ ...settings, ews_url: e.target.value })} />
        </div>
        <div className="form-group">
          <label>Username</label>
          <input value={settings.username} onChange={(e) => setSettings({ ...settings, username: e.target.value })} />
        </div>
        <div className="form-group">
          <label>Password (оставьте пустым чтобы не менять)</label>
          <input type="password" value={settings.password} onChange={(e) => setSettings({ ...settings, password: e.target.value })} />
        </div>
        <div className="form-group">
          <label>Domain</label>
          <input value={settings.domain} onChange={(e) => setSettings({ ...settings, domain: e.target.value })} />
        </div>
        <div className="form-group">
          <label>Auth mode</label>
          <select value={settings.auth_mode} onChange={(e) => setSettings({ ...settings, auth_mode: e.target.value })}>
            <option value="basic">basic</option>
            <option value="ntlm">ntlm</option>
            <option value="kerberos">kerberos</option>
          </select>
        </div>
        <div className="modal-actions">
          <button className="primary" onClick={() => void save()}>Сохранить</button>
          <button onClick={() => void testConnection()}>Проверить подключение</button>
        </div>
      </div>
    </div>
  )
}
