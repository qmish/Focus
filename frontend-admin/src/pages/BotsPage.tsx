import { useEffect, useState } from 'react'
import { adminApi } from '../lib/adminApi'

type BotRow = {
  id: string
  name: string
  description: string
  is_enabled: boolean
  rate_limit_ms: number
  allowed_rooms: string[]
  commands_json: string
}

type BotError = {
  id: string
  room_id: string
  user_id: string
  command: string
  status: string
  error: string
  created_at: string
}

export default function BotsPage() {
  const [bots, setBots] = useState<BotRow[]>([])
  const [errors, setErrors] = useState<BotError[]>([])
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [editData, setEditData] = useState<Partial<BotRow>>({})
  const [tab, setTab] = useState<'list' | 'errors'>('list')
  const [stats, setStats] = useState<{ total_events_24h: number; errors_24h: number } | null>(null)

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await adminApi.listBots()
      setBots((data.data || []) as BotRow[])
    } catch (err: any) {
      setError(err.message || 'Не удалось загрузить ботов')
    } finally {
      setLoading(false)
    }
  }

  const loadErrors = async () => {
    try {
      const data = await adminApi.getBotErrors(50)
      setErrors((data.data || []) as BotError[])
    } catch { /* ignore */ }
  }

  const loadStats = async () => {
    try {
      const data = await adminApi.getBotStats('all')
      setStats(data)
    } catch { /* ignore */ }
  }

  useEffect(() => {
    void load()
    void loadStats()
  }, [])

  useEffect(() => {
    if (tab === 'errors') void loadErrors()
  }, [tab])

  const createBot = async () => {
    if (!name.trim()) return
    try {
      await adminApi.createBot({
        name: name.trim(),
        description: description.trim(),
        is_enabled: true,
        rate_limit_ms: 2000,
        allowed_rooms: [],
        commands_json: '[]',
      })
      setName('')
      setDescription('')
      await load()
      setSuccess('Бот создан')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: any) {
      setError(err.message || 'Не удалось создать бота')
    }
  }

  const toggle = async (bot: BotRow) => {
    try {
      await adminApi.toggleBot(bot.id, !bot.is_enabled)
      await load()
    } catch (err: any) {
      setError(err.message || 'Не удалось обновить бота')
    }
  }

  const deleteBot = async (bot: BotRow) => {
    if (!confirm(`Удалить бота "${bot.name}"?`)) return
    try {
      await adminApi.deleteBot(bot.id)
      setExpandedId(null)
      await load()
      setSuccess('Бот удалён')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: any) {
      setError(err.message || 'Не удалось удалить бота')
    }
  }

  const saveEdit = async (botId: string) => {
    try {
      const payload: Record<string, unknown> = {}
      if (editData.name !== undefined) payload.name = editData.name
      if (editData.description !== undefined) payload.description = editData.description
      if (editData.rate_limit_ms !== undefined) payload.rate_limit_ms = editData.rate_limit_ms
      if (editData.allowed_rooms !== undefined) payload.allowed_rooms = editData.allowed_rooms
      if (editData.commands_json !== undefined) payload.commands_json = editData.commands_json
      await adminApi.patchBot(botId, payload)
      setExpandedId(null)
      setEditData({})
      await load()
      setSuccess('Бот обновлён')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: any) {
      setError(err.message || 'Не удалось обновить бота')
    }
  }

  const reloadConfig = async () => {
    try {
      await adminApi.reloadBotConfig()
      setSuccess('Конфигурация перезагружена')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: any) {
      setError(err.message || 'Не удалось перезагрузить конфигурацию')
    }
  }

  const expand = (bot: BotRow) => {
    if (expandedId === bot.id) {
      setExpandedId(null)
      setEditData({})
    } else {
      setExpandedId(bot.id)
      setEditData({
        name: bot.name,
        description: bot.description,
        rate_limit_ms: bot.rate_limit_ms,
        allowed_rooms: bot.allowed_rooms || [],
        commands_json: bot.commands_json || '[]',
      })
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h1>Боты</h1>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          {stats && (
            <span style={{ fontSize: '0.85rem', color: 'var(--muted-color, #888)' }}>
              За 24ч: {stats.total_events_24h} вызовов, {stats.errors_24h} ошибок
            </span>
          )}
          <button onClick={() => void reloadConfig()}>Перезагрузить конфиг</button>
        </div>
      </div>

      {error && <p className="error">{error}</p>}
      {success && <p style={{ color: 'var(--success-color, #4ade80)', marginBottom: 12 }}>{success}</p>}

      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <button className={tab === 'list' ? 'primary' : ''} onClick={() => setTab('list')}>Список ботов</button>
        <button className={tab === 'errors' ? 'primary' : ''} onClick={() => setTab('errors')}>Ошибки</button>
      </div>

      {tab === 'list' && (
        <>
          <div className="settings-section">
            <h2>Добавить бота</h2>
            <div style={{ display: 'flex', gap: 8, alignItems: 'flex-end' }}>
              <div className="form-group" style={{ flex: 1, marginBottom: 0 }}>
                <label>Название</label>
                <input value={name} onChange={(e) => setName(e.target.value)} placeholder="my-bot" />
              </div>
              <div className="form-group" style={{ flex: 2, marginBottom: 0 }}>
                <label>Описание</label>
                <input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Описание бота" />
              </div>
              <button className="primary" onClick={() => void createBot()}>Создать</button>
            </div>
          </div>

          {loading ? (
            <div className="loading">Загрузка...</div>
          ) : (
            <div className="users-table">
              <table>
                <thead>
                  <tr>
                    <th>Имя</th>
                    <th>Описание</th>
                    <th>Rate limit</th>
                    <th>Статус</th>
                    <th>Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {bots.map((bot) => (
                    <>
                      <tr key={bot.id} style={{ cursor: 'pointer' }} onClick={() => expand(bot)}>
                        <td>{bot.name}</td>
                        <td>{bot.description || '—'}</td>
                        <td>{bot.rate_limit_ms} ms</td>
                        <td>{bot.is_enabled ? 'Включен' : 'Отключен'}</td>
                        <td>
                          <div style={{ display: 'flex', gap: 4 }}>
                            <button onClick={(e) => { e.stopPropagation(); void toggle(bot) }}>
                              {bot.is_enabled ? 'Выключить' : 'Включить'}
                            </button>
                            <button style={{ color: 'var(--error-color, #f38ba8)' }} onClick={(e) => { e.stopPropagation(); void deleteBot(bot) }}>
                              Удалить
                            </button>
                          </div>
                        </td>
                      </tr>
                      {expandedId === bot.id && (
                        <tr key={bot.id + '-edit'}>
                          <td colSpan={5}>
                            <div className="settings-section" style={{ margin: '8px 0' }}>
                              <h3>Редактирование: {bot.name}</h3>
                              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                                <div className="form-group">
                                  <label>Название</label>
                                  <input value={editData.name || ''} onChange={(e) => setEditData({ ...editData, name: e.target.value })} />
                                </div>
                                <div className="form-group">
                                  <label>Описание</label>
                                  <input value={editData.description || ''} onChange={(e) => setEditData({ ...editData, description: e.target.value })} />
                                </div>
                                <div className="form-group">
                                  <label>Rate limit (мс)</label>
                                  <input type="number" value={editData.rate_limit_ms || 2000} onChange={(e) => setEditData({ ...editData, rate_limit_ms: parseInt(e.target.value) || 2000 })} />
                                </div>
                                <div className="form-group">
                                  <label>Разрешённые комнаты (UUID через запятую)</label>
                                  <input
                                    value={(editData.allowed_rooms || []).join(', ')}
                                    onChange={(e) => setEditData({ ...editData, allowed_rooms: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })}
                                    placeholder="Пусто = все комнаты"
                                  />
                                </div>
                              </div>
                              <div className="form-group">
                                <label>Команды (JSON)</label>
                                <textarea
                                  value={editData.commands_json || '[]'}
                                  onChange={(e) => setEditData({ ...editData, commands_json: e.target.value })}
                                  rows={6}
                                  style={{ width: '100%', fontFamily: 'monospace', fontSize: '0.85rem', padding: 8, borderRadius: 6, border: '1px solid var(--border-color, #444)', background: 'var(--input-bg, #1e1e2e)', color: 'var(--text-color, #cdd6f4)' }}
                                  placeholder={'[\n  {"command": "/ping", "handler": "static-reply", "description": "Pong!", "is_active": true}\n]'}
                                />
                                <small style={{ color: 'var(--muted-color, #888)' }}>
                                  Формат: [{'{'}command, handler: "static-reply", description: "текст ответа", is_active: true{'}'}]
                                </small>
                              </div>
                              <div style={{ display: 'flex', gap: 8 }}>
                                <button className="primary" onClick={() => void saveEdit(bot.id)}>Сохранить</button>
                                <button onClick={() => { setExpandedId(null); setEditData({}) }}>Отмена</button>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {tab === 'errors' && (
        <div className="users-table">
          <table>
            <thead>
              <tr>
                <th>Время</th>
                <th>Команда</th>
                <th>Статус</th>
                <th>Ошибка</th>
                <th>Room ID</th>
              </tr>
            </thead>
            <tbody>
              {errors.length === 0 ? (
                <tr><td colSpan={5} style={{ textAlign: 'center', color: 'var(--muted-color, #888)' }}>Нет ошибок</td></tr>
              ) : errors.map((e) => (
                <tr key={e.id}>
                  <td>{new Date(e.created_at).toLocaleString('ru')}</td>
                  <td>/{e.command}</td>
                  <td>{e.status}</td>
                  <td style={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{e.error || '—'}</td>
                  <td style={{ fontSize: '0.8rem' }}>{e.room_id?.substring(0, 8)}...</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
