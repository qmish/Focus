import { useEffect, useState } from 'react'

interface Conference {
  id: string
  room_name: string
  participants_count: number
  started_at: string
  duration_seconds: number
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
          'Authorization': `Bearer ${localStorage.getItem('admin_token')}`,
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
          'Authorization': `Bearer ${localStorage.getItem('admin_token')}`,
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
                  <td>{Math.floor(conf.duration_seconds / 60)} мин</td>
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
