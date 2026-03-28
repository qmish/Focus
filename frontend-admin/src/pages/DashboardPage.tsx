import { useEffect, useState } from 'react'
import { useAdminStore } from '../store/adminStore'
import { adminApi } from '../lib/adminApi'

type AuditEntry = { id: string; actor: string; action: string; created_at: string }
type HealthStatus = { name: string; status: 'ok' | 'error' }

export default function DashboardPage() {
  const { stats, error, fetchStats } = useAdminStore()
  const [loading, setLoading] = useState(true)
  const [recentActivity, setRecentActivity] = useState<AuditEntry[]>([])
  const [health, setHealth] = useState<HealthStatus[]>([])

  useEffect(() => {
    const loadAll = async () => {
      try {
        await fetchStats()
      } catch { /* handled by store */ }

      try {
        const audit = await adminApi.listAuditLogs('limit=5&page=1')
        setRecentActivity(audit.data as AuditEntry[])
      } catch {
        setRecentActivity([])
      }

      try {
        const res = await fetch('/api/v1/health')
        if (res.ok) {
          const data = await res.json()
          const services = ['api', 'database', 'jitsi']
          setHealth(services.map(s => ({
            name: s.charAt(0).toUpperCase() + s.slice(1),
            status: (data[s] === 'ok' || data.status === 'ok') ? 'ok' as const : 'error' as const,
          })))
        } else {
          setHealth([{ name: 'API', status: 'error' }])
        }
      } catch {
        setHealth([{ name: 'API', status: 'error' }])
      }

      setLoading(false)
    }
    loadAll()
  }, [])

  if (loading) {
    return <div className="loading">Загрузка...</div>
  }

  return (
    <div className="dashboard-page">
      <h1>Панель управления</h1>
      {error && <p className="error">{error}</p>}
      
      <div className="stats-grid">
        <div className="stat-card">
          <h3>Пользователи</h3>
          <p className="stat-value">{stats.users?.total || 0}</p>
        </div>

        <div className="stat-card">
          <h3>Комнаты</h3>
          <p className="stat-value">{stats.rooms?.total || 0}</p>
        </div>

        <div className="stat-card">
          <h3>Активные конференции</h3>
          <p className="stat-value">{stats.conferences?.active || 0}</p>
        </div>

        <div className="stat-card">
          <h3>Сообщений сегодня</h3>
          <p className="stat-value">{stats.messages?.today || 0}</p>
        </div>
      </div>

      <div className="dashboard-content">
        <div className="recent-activity">
          <h2>Последняя активность</h2>
          {recentActivity.length === 0 ? (
            <p>Нет данных</p>
          ) : (
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {recentActivity.map(a => (
                <li key={a.id} style={{ padding: '0.5rem 0', borderBottom: '1px solid rgba(0,0,0,0.08)' }}>
                  <strong>{a.actor}</strong> — {a.action}
                  <span style={{ float: 'right', color: '#888', fontSize: '0.85em' }}>
                    {new Date(a.created_at).toLocaleString()}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="system-health">
          <h2>Состояние системы</h2>
          {health.length === 0 ? (
            <p>Проверка...</p>
          ) : (
            health.map(h => (
              <div className="health-item" key={h.name}>
                <span className={`health-status ${h.status}`}>●</span>
                <span>{h.name}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
