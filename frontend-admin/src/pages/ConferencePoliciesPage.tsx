import { useEffect, useState } from 'react'
import { adminApi } from '../lib/adminApi'

type Policies = {
  max_participants: number
  max_duration_minutes: number
  recording_enabled: boolean
  lobby_enabled: boolean
  auto_mute_on_join: boolean
  require_password: boolean
}

const defaults: Policies = {
  max_participants: 100,
  max_duration_minutes: 480,
  recording_enabled: false,
  lobby_enabled: false,
  auto_mute_on_join: false,
  require_password: false,
}

export default function ConferencePoliciesPage() {
  const [policies, setPolicies] = useState<Policies>(defaults)
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')

  useEffect(() => {
    const load = async () => {
      try {
        const data = await adminApi.getConferencePolicies()
        if (data.policies) setPolicies({ ...defaults, ...data.policies })
      } catch { /* use defaults */ }
    }
    void load()
  }, [])

  const save = async () => {
    setMsg('')
    setErr('')
    try {
      await adminApi.putConferencePolicies(policies as unknown as Record<string, unknown>)
      setMsg('Политики сохранены')
      setTimeout(() => setMsg(''), 4000)
    } catch (e: any) {
      setErr(e.message || 'Не удалось сохранить')
    }
  }

  const Toggle = ({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) => (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '8px 0', borderBottom: '1px solid var(--border-color, #333)' }}>
      <span>{label}</span>
      <button
        onClick={() => onChange(!value)}
        style={{ padding: '4px 16px', borderRadius: 6, border: '1px solid var(--border-color, #444)', background: value ? 'var(--primary-color, #2563eb)' : 'transparent', color: value ? '#fff' : 'inherit', cursor: 'pointer' }}
      >
        {value ? 'Вкл' : 'Выкл'}
      </button>
    </div>
  )

  return (
    <div>
      <h1>Политики конференций</h1>
      {msg && <p style={{ color: 'var(--success-color, #4ade80)', marginBottom: 8 }}>{msg}</p>}
      {err && <p className="error">{err}</p>}

      <div className="settings-section">
        <h2>Лимиты</h2>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <div className="form-group">
            <label>Макс. участников</label>
            <input type="number" value={policies.max_participants} onChange={(e) => setPolicies({ ...policies, max_participants: parseInt(e.target.value) || 100 })} />
          </div>
          <div className="form-group">
            <label>Макс. длительность (минуты)</label>
            <input type="number" value={policies.max_duration_minutes} onChange={(e) => setPolicies({ ...policies, max_duration_minutes: parseInt(e.target.value) || 480 })} />
          </div>
        </div>
      </div>

      <div className="settings-section">
        <h2>Настройки по умолчанию</h2>
        <Toggle label="Запись конференций" value={policies.recording_enabled} onChange={(v) => setPolicies({ ...policies, recording_enabled: v })} />
        <Toggle label="Лобби (зал ожидания)" value={policies.lobby_enabled} onChange={(v) => setPolicies({ ...policies, lobby_enabled: v })} />
        <Toggle label="Автоматический mute при подключении" value={policies.auto_mute_on_join} onChange={(v) => setPolicies({ ...policies, auto_mute_on_join: v })} />
        <Toggle label="Требовать пароль для входа" value={policies.require_password} onChange={(v) => setPolicies({ ...policies, require_password: v })} />
      </div>

      <button className="primary" onClick={() => void save()}>Сохранить политики</button>
    </div>
  )
}
