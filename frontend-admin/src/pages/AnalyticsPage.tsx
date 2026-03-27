import { useEffect, useState } from 'react'
import { adminApi } from '../lib/adminApi'

type DayData = { date: string; messages: number }
type Summary = {
  total_users: number
  total_rooms: number
  active_meetings: number
  messages_today: number
}

export default function AnalyticsPage() {
  const [days, setDays] = useState(7)
  const [messagesByDay, setMessagesByDay] = useState<DayData[]>([])
  const [summary, setSummary] = useState<Summary | null>(null)
  const [loading, setLoading] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const data = await adminApi.getAnalytics(days)
      setMessagesByDay(data.messages_by_day || [])
      setSummary(data.summary || null)
    } catch { /* ignore */ } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [days])

  const maxMessages = Math.max(1, ...messagesByDay.map(d => d.messages))

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h1>Аналитика</h1>
        <div style={{ display: 'flex', gap: 8 }}>
          {[7, 14, 30].map(d => (
            <button key={d} className={days === d ? 'primary' : ''} onClick={() => setDays(d)}>{d}д</button>
          ))}
        </div>
      </div>

      {summary && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12, marginBottom: 24 }}>
          <StatCard label="Пользователей" value={summary.total_users} />
          <StatCard label="Комнат" value={summary.total_rooms} />
          <StatCard label="Активных встреч" value={summary.active_meetings} />
          <StatCard label="Сообщений за сегодня" value={summary.messages_today} />
        </div>
      )}

      <div className="settings-section">
        <h2>Сообщения по дням</h2>
        {loading ? (
          <div className="loading">Загрузка...</div>
        ) : messagesByDay.length === 0 ? (
          <p style={{ color: 'var(--muted-color, #888)' }}>Нет данных</p>
        ) : (
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: 4, height: 200, paddingTop: 8 }}>
            {messagesByDay.map((d) => {
              const pct = (d.messages / maxMessages) * 100
              return (
                <div key={d.date} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4 }}>
                  <span style={{ fontSize: '0.7rem', color: 'var(--muted-color, #888)' }}>{d.messages}</span>
                  <div
                    style={{
                      width: '100%',
                      maxWidth: 40,
                      height: `${Math.max(pct, 2)}%`,
                      background: 'var(--primary-color, #2563eb)',
                      borderRadius: '4px 4px 0 0',
                      transition: 'height 0.3s',
                    }}
                  />
                  <span style={{ fontSize: '0.65rem', color: 'var(--muted-color, #888)', whiteSpace: 'nowrap' }}>
                    {new Date(d.date).toLocaleDateString('ru', { day: '2-digit', month: '2-digit' })}
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div style={{
      padding: 16,
      borderRadius: 10,
      background: 'var(--card-bg, rgba(255,255,255,0.05))',
      border: '1px solid var(--border-color, #333)',
      textAlign: 'center',
    }}>
      <div style={{ fontSize: '1.8rem', fontWeight: 700, color: 'var(--primary-color, #2563eb)' }}>{value}</div>
      <div style={{ fontSize: '0.85rem', color: 'var(--muted-color, #888)', marginTop: 4 }}>{label}</div>
    </div>
  )
}
