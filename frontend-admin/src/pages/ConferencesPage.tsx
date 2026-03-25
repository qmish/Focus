import { useEffect, useState } from 'react'
import { getAdminAccessToken } from '../lib/authToken'

interface Conference {
  id: string
  room_name: string
  participants_count: number
  started_at: string
  last_activity_at?: string
  status?: string
}

export default function ConferencesPage() {
  const [conferences, setConferences] = useState<Conference[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchConferences()
  }, [])

  const fetchConferences = async () => {
    try {
      const response = await fetch('/api/v1/admin/conferences', {
        headers: {
          'Authorization': `Bearer ${getAdminAccessToken()}`,
        },
      })
      const data = await response.json()
      setConferences(data.data || [])
    } catch (error) {
      console.error('Failed to fetch conferences:', error)
    } finally {
      setLoading(false)
    }
  }

  const endConference = async (id: string) => {
    if (!confirm('Завершить конференцию?')) return

    try {
      await fetch(`/api/v1/admin/conferences/${id}/end`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${getAdminAccessToken()}`,
        },
      })
      fetchConferences()
    } catch (error) {
      console.error('Failed to end conference:', error)
    }
  }

  if (loading) {
    return <div className="loading">Загрузка...</div>
  }

  return (
    <div className="conferences-page">
      <h1>Активные конференции</h1>

      {conferences.length === 0 ? (
        <div className="empty-state">
          <p>Нет активных конференций</p>
        </div>
      ) : (
        <div className="conferences-table">
          <table>
            <thead>
              <tr>
                <th>Комната</th>
                <th>Участников</th>
                <th>Начало</th>
                <th>Длительность</th>
                <th>Действия</th>
              </tr>
            </thead>
            <tbody>
              {conferences.map(conf => (
                <tr key={conf.id}>
                  <td>{conf.room_name}</td>
                  <td>{conf.participants_count}</td>
                  <td>{new Date(conf.started_at).toLocaleString('ru-RU')}</td>
                  <td>{formatDurationMinutes(conf.started_at, conf.last_activity_at)} мин</td>
                  <td>
                    <button onClick={() => endConference(conf.id)} className="danger">
                      Завершить
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function formatDurationMinutes(startedAt: string, lastActivityAt?: string): number {
  const start = new Date(startedAt).getTime()
  const end = lastActivityAt ? new Date(lastActivityAt).getTime() : Date.now()
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    return 0
  }
  return Math.floor((end - start) / 60000)
}
