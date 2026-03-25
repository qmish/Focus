import { useEffect, useState } from 'react'
import { getAdminAccessToken } from '../lib/authToken'
import {
  authAuditStatusLabel,
  botStatusLabel,
  calendarOperationLabel,
  isDeliveryFailed,
} from '../lib/observability'

interface WebhookDelivery {
  id: string
  webhook_id: string
  response_code: number
  response_body: string
  success: boolean
  retry_count: number
  created_at: string
}

interface BotErrorEvent {
  id: string
  room_id: string
  user_id: string
  command: string
  args: string
  status: string
  error?: string
  created_at: string
}

interface AuthAuditEvent {
  id: string
  action: string
  status: string
  user_id?: string
  user_email?: string
  error?: string
  created_at: string
}

interface CalendarAuditEvent {
  id: string
  operation: string
  status: string
  event_id?: string
  user_email?: string
  details?: string
  created_at: string
}

export default function ObservabilityPage() {
  const [loading, setLoading] = useState(true)
  const [webhookErrors, setWebhookErrors] = useState<WebhookDelivery[]>([])
  const [botErrors, setBotErrors] = useState<BotErrorEvent[]>([])
  const [authAuditErrors, setAuthAuditErrors] = useState<AuthAuditEvent[]>([])
  const [calendarAuditErrors, setCalendarAuditErrors] = useState<CalendarAuditEvent[]>([])

  useEffect(() => {
    void load()
  }, [])

  async function load() {
    setLoading(true)
    try {
      const token = getAdminAccessToken()
      const [webhookRes, botRes, authAuditRes, calendarAuditRes] = await Promise.all([
        fetch('/api/v1/admin/webhooks/errors?limit=50', {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch('/api/v1/admin/bots/errors?limit=50', {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch('/api/v1/admin/auth/audit?limit=50&failed=true', {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch('/api/v1/admin/calendar/audit?limit=50&failed=true', {
          headers: { Authorization: `Bearer ${token}` },
        }),
      ])
      const webhookData = webhookRes.ok ? await webhookRes.json() : { data: [] }
      const botData = botRes.ok ? await botRes.json() : { data: [] }
      const authAuditData = authAuditRes.ok ? await authAuditRes.json() : { data: [] }
      const calendarAuditData = calendarAuditRes.ok ? await calendarAuditRes.json() : { data: [] }
      setWebhookErrors((webhookData.data || []).filter((d: WebhookDelivery) => isDeliveryFailed(d.success)))
      setBotErrors(botData.data || [])
      setAuthAuditErrors(authAuditData.data || [])
      setCalendarAuditErrors(calendarAuditData.data || [])
    } catch (error) {
      console.error('Failed to load observability data:', error)
      setWebhookErrors([])
      setBotErrors([])
      setAuthAuditErrors([])
      setCalendarAuditErrors([])
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div className="loading">Загрузка...</div>
  }

  return (
    <div className="observability-page">
      <h1>Наблюдаемость webhook/bot ошибок</h1>

      <section>
        <h2>Ошибки webhook-доставок</h2>
        {webhookErrors.length === 0 ? (
          <p>Ошибок webhook-доставок не найдено.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Webhook</th>
                <th>Код</th>
                <th>Retry</th>
                <th>Причина</th>
                <th>Время</th>
              </tr>
            </thead>
            <tbody>
              {webhookErrors.map((item) => (
                <tr key={item.id}>
                  <td>{item.webhook_id.slice(0, 8)}...</td>
                  <td>{item.response_code}</td>
                  <td>{item.retry_count}</td>
                  <td>{item.response_body || 'n/a'}</td>
                  <td>{new Date(item.created_at).toLocaleString('ru-RU')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section>
        <h2>Ошибки bot-команд</h2>
        {botErrors.length === 0 ? (
          <p>Ошибок bot-команд не найдено.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Команда</th>
                <th>Статус</th>
                <th>Ошибка</th>
                <th>Время</th>
              </tr>
            </thead>
            <tbody>
              {botErrors.map((item) => (
                <tr key={item.id}>
                  <td>/{item.command}</td>
                  <td>{botStatusLabel(item.status)}</td>
                  <td>{item.error || '-'}</td>
                  <td>{new Date(item.created_at).toLocaleString('ru-RU')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section>
        <h2>Ошибки авторизации</h2>
        {authAuditErrors.length === 0 ? (
          <p>Ошибок авторизации не найдено.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Действие</th>
                <th>Статус</th>
                <th>Пользователь</th>
                <th>Причина</th>
                <th>Время</th>
              </tr>
            </thead>
            <tbody>
              {authAuditErrors.map((item) => (
                <tr key={item.id}>
                  <td>{item.action}</td>
                  <td>{authAuditStatusLabel(item.status)}</td>
                  <td>{item.user_email || item.user_id || '-'}</td>
                  <td>{item.error || '-'}</td>
                  <td>{new Date(item.created_at).toLocaleString('ru-RU')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section>
        <h2>Ошибки календарных операций</h2>
        {calendarAuditErrors.length === 0 ? (
          <p>Ошибок календарных операций не найдено.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Операция</th>
                <th>Статус</th>
                <th>Событие</th>
                <th>Пользователь</th>
                <th>Причина</th>
                <th>Время</th>
              </tr>
            </thead>
            <tbody>
              {calendarAuditErrors.map((item) => (
                <tr key={item.id}>
                  <td>{calendarOperationLabel(item.operation)}</td>
                  <td>{authAuditStatusLabel(item.status)}</td>
                  <td>{item.event_id || '-'}</td>
                  <td>{item.user_email || '-'}</td>
                  <td>{item.details || '-'}</td>
                  <td>{new Date(item.created_at).toLocaleString('ru-RU')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  )
}
