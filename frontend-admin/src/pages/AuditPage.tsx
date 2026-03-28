import { useEffect, useState } from 'react'
import { adminApi } from '../lib/adminApi'

type AuditEntry = {
  id: string
  actor_email: string
  action: string
  resource_type: string
  resource_id: string
  details: string
  created_at: string
}

function escapeCSV(value: string): string {
  if (value == null) return ''
  const str = String(value)
  const escaped = str.replace(/"/g, '""')
  if (/[,"\n\r]/.test(escaped) || /^[=+\-@\t\r]/.test(escaped)) {
    return `"\t${escaped}"`
  }
  if (/[,"\n]/.test(str)) {
    return `"${escaped}"`
  }
  return escaped
}

export default function AuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [actor, setActor] = useState('')
  const [action, setAction] = useState('')
  const [resourceType, setResourceType] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ limit: '100' })
      if (actor) params.set('actor', actor)
      if (action) params.set('action', action)
      if (resourceType) params.set('resource_type', resourceType)
      const data = await adminApi.listAuditLogs(params.toString())
      setEntries((data.data || []) as AuditEntry[])
      setTotal(data.total || 0)
    } catch (err) { console.error('AuditPage:', err) } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [])

  const exportCSV = () => {
    const header = 'Время,Актор,Действие,Тип ресурса,ID ресурса,Детали\n'
    const rows = entries.map(e =>
      [new Date(e.created_at).toLocaleString('ru'), e.actor_email, e.action, e.resource_type, e.resource_id, e.details].map(v => escapeCSV(v)).join(',')
    ).join('\n')
    const blob = new Blob([header + rows], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'audit-log.csv'
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div>
      <div className="page-header">
        <h1>Аудит-лог</h1>
        <div className="flex-row">
          <span className="text-muted-sm">Всего: {total}</span>
          <button onClick={exportCSV}>Экспорт CSV</button>
        </div>
      </div>

      <div className="flex-row-wrap">
        <input placeholder="Фильтр по актору" value={actor} onChange={(e) => setActor(e.target.value)} aria-label="Фильтр по актору" style={{ flex: 1, minWidth: 150 }} />
        <input placeholder="Фильтр по действию" value={action} onChange={(e) => setAction(e.target.value)} aria-label="Фильтр по действию" style={{ flex: 1, minWidth: 150 }} />
        <input placeholder="Фильтр по типу ресурса" value={resourceType} onChange={(e) => setResourceType(e.target.value)} aria-label="Фильтр по типу ресурса" style={{ flex: 1, minWidth: 150 }} />
        <button className="primary" onClick={() => void load()}>Применить</button>
      </div>

      {loading ? (
        <div className="loading">Загрузка...</div>
      ) : (
        <div className="users-table">
          <table>
            <thead>
              <tr>
                <th>Время</th>
                <th>Актор</th>
                <th>Действие</th>
                <th>Тип</th>
                <th>ID ресурса</th>
                <th>Детали</th>
              </tr>
            </thead>
            <tbody>
              {entries.length === 0 ? (
                <tr><td colSpan={6} className="cell-empty">Нет записей</td></tr>
              ) : entries.map((e) => (
                <tr key={e.id}>
                  <td style={{ whiteSpace: 'nowrap', fontSize: '0.85rem' }}>{new Date(e.created_at).toLocaleString('ru')}</td>
                  <td>{e.actor_email}</td>
                  <td><code>{e.action}</code></td>
                  <td>{e.resource_type}</td>
                  <td className="cell-mono-sm">{e.resource_id?.substring(0, 12)}{e.resource_id?.length > 12 ? '...' : ''}</td>
                  <td className="cell-truncate">{e.details || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
