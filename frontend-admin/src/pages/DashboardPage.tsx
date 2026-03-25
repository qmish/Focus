import { useEffect, useState } from 'react'
import { useAdminStore } from '../store/adminStore'

export default function DashboardPage() {
  const { stats, error, fetchStats } = useAdminStore()
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchStats().finally(() => setLoading(false))
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
          <p>Нет данных</p>
        </div>

        <div className="system-health">
          <h2>Состояние системы</h2>
          <div className="health-item">
            <span className="health-status ok">●</span>
            <span>API Server</span>
          </div>
          <div className="health-item">
            <span className="health-status ok">●</span>
            <span>Database</span>
          </div>
          <div className="health-item">
            <span className="health-status ok">●</span>
            <span>Redis</span>
          </div>
          <div className="health-item">
            <span className="health-status ok">●</span>
            <span>Jitsi</span>
          </div>
        </div>
      </div>
    </div>
  )
}
