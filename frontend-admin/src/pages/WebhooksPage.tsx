import { useEffect, useState } from 'react'
import { adminApi } from '../lib/adminApi'

type Delivery = {
  id: string
  event_type: string
  target_url: string
  status_code: number
  success: boolean
  error: string
  created_at: string
}

export default function WebhooksPage() {
  const [deliveries, setDeliveries] = useState<Delivery[]>([])
  const [errors, setErrors] = useState<Delivery[]>([])
  const [loading, setLoading] = useState(false)
  const [tab, setTab] = useState<'deliveries' | 'errors'>('deliveries')

  const loadDeliveries = async () => {
    setLoading(true)
    try {
      const data = await adminApi.listWebhookDeliveries(100)
      setDeliveries((data.data || []) as Delivery[])
    } catch { /* ignore */ } finally {
      setLoading(false)
    }
  }

  const loadErrors = async () => {
    setLoading(true)
    try {
      const data = await adminApi.listWebhookErrors(100)
      setErrors((data.data || []) as Delivery[])
    } catch { /* ignore */ } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (tab === 'deliveries') void loadDeliveries()
    else void loadErrors()
  }, [tab])

  const items = tab === 'deliveries' ? deliveries : errors

  return (
    <div>
      <h1>Вебхуки</h1>

      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <button className={tab === 'deliveries' ? 'primary' : ''} onClick={() => setTab('deliveries')}>Доставки</button>
        <button className={tab === 'errors' ? 'primary' : ''} onClick={() => setTab('errors')}>Ошибки</button>
      </div>

      {loading ? (
        <div className="loading">Загрузка...</div>
      ) : (
        <div className="users-table">
          <table>
            <thead>
              <tr>
                <th>Время</th>
                <th>Тип события</th>
                <th>URL</th>
                <th>Код</th>
                <th>Статус</th>
                <th>Ошибка</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr><td colSpan={6} style={{ textAlign: 'center', color: 'var(--muted-color, #888)' }}>Нет записей</td></tr>
              ) : items.map((d) => (
                <tr key={d.id}>
                  <td style={{ whiteSpace: 'nowrap', fontSize: '0.85rem' }}>{new Date(d.created_at).toLocaleString('ru')}</td>
                  <td><code>{d.event_type}</code></td>
                  <td style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: '0.85rem' }}>{d.target_url || '—'}</td>
                  <td>{d.status_code || '—'}</td>
                  <td>{d.success ? 'OK' : 'Ошибка'}</td>
                  <td style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{d.error || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
