import { useEffect, useState, useCallback, Fragment } from 'react'
import { adminApi, type CommandStat, type BotTemplate } from '../lib/adminApi'

type BotCommand = {
  command: string
  handler: string
  description: string
  is_active: boolean
  webhook_url?: string
  rate_limit_ms?: number
}

type BotRow = {
  id: string
  name: string
  description: string
  is_enabled: boolean
  rate_limit_ms: number
  allowed_rooms: string[]
  commands_json: string
  schedule_json: string
  avatar_url: string
}

type BotError = {
  id: string
  room_id: string
  user_id: string
  command: string
  args: string
  status: string
  error: string
  created_at: string
}

const HANDLER_TYPES = [
  { value: 'static-reply', label: 'Статический ответ' },
  { value: 'template', label: 'Шаблон (переменные)' },
  { value: 'random', label: 'Случайный ответ' },
  { value: 'alias', label: 'Алиас (перенаправление)' },
  { value: 'webhook', label: 'Вебхук (HTTP)' },
]

function parseCommands(json: string): BotCommand[] {
  try {
    return JSON.parse(json) || []
  } catch {
    return []
  }
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
  const [editCommands, setEditCommands] = useState<BotCommand[]>([])
  const [tab, setTab] = useState<'list' | 'errors' | 'analytics' | 'history' | 'templates'>('list')
  const [stats, setStats] = useState<{ total_events_24h: number; errors_24h: number } | null>(null)
  const [commandStats, setCommandStats] = useState<CommandStat[]>([])
  const [history, setHistory] = useState<BotError[]>([])
  const [historyTotal, setHistoryTotal] = useState(0)
  const [historyFilter, setHistoryFilter] = useState({ command: '', status: '', limit: 50 })
  const [templates, setTemplates] = useState<BotTemplate[]>([])
  const [testResult, setTestResult] = useState<string | null>(null)

  const showSuccess = (msg: string) => { setSuccess(msg); setTimeout(() => setSuccess(''), 3000) }

  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const data = await adminApi.listBots()
      setBots((data.data || []) as BotRow[])
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось загрузить ботов') }
    finally { setLoading(false) }
  }, [])

  const loadErrors = async () => {
    try { const data = await adminApi.getBotErrors(50); setErrors((data.data || []) as BotError[]) } catch (err) { console.error('BotsPage:', err) }
  }
  const loadStats = async () => {
    try { const data = await adminApi.getBotStats('all'); setStats(data) } catch (err) { console.error('BotsPage:', err) }
  }
  const loadCommandStats = async () => {
    try { const data = await adminApi.getCommandStats(7); setCommandStats(data.data || []) } catch (err) { console.error('BotsPage:', err) }
  }
  const loadHistory = async () => {
    try {
      const params = new URLSearchParams({ limit: String(historyFilter.limit) })
      if (historyFilter.command) params.set('command', historyFilter.command)
      if (historyFilter.status) params.set('status', historyFilter.status)
      const data = await adminApi.getCommandHistory(params.toString())
      setHistory((data.data || []) as BotError[]); setHistoryTotal(data.total || 0)
    } catch (err) { console.error('BotsPage:', err) }
  }
  const loadTemplates = async () => {
    try { const data = await adminApi.listBotTemplates(); setTemplates(data.data || []) } catch (err) { console.error('BotsPage:', err) }
  }

  useEffect(() => { void load(); void loadStats() }, [load])
  useEffect(() => {
    if (tab === 'errors') void loadErrors()
    if (tab === 'analytics') void loadCommandStats()
    if (tab === 'history') void loadHistory()
    if (tab === 'templates') void loadTemplates()
  }, [tab])

  const createBot = async () => {
    if (!name.trim()) return
    try {
      await adminApi.createBot({ name: name.trim(), description: description.trim(), is_enabled: true, rate_limit_ms: 2000, allowed_rooms: [], commands_json: '[]', schedule_json: '[]' })
      setName(''); setDescription(''); await load(); showSuccess('Бот создан')
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось создать бота') }
  }

  const toggle = async (bot: BotRow) => {
    try { await adminApi.toggleBot(bot.id, !bot.is_enabled); await load() }
    catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось обновить бота') }
  }

  const deleteBot = async (bot: BotRow) => {
    if (!confirm(`Удалить бота "${bot.name}"?`)) return
    try { await adminApi.deleteBot(bot.id); setExpandedId(null); await load(); showSuccess('Бот удалён') }
    catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось удалить бота') }
  }

  const saveEdit = async (botId: string) => {
    try {
      const payload: Record<string, unknown> = {}
      if (editData.name !== undefined) payload.name = editData.name
      if (editData.description !== undefined) payload.description = editData.description
      if (editData.rate_limit_ms !== undefined) payload.rate_limit_ms = editData.rate_limit_ms
      if (editData.allowed_rooms !== undefined) payload.allowed_rooms = editData.allowed_rooms
      if (editData.avatar_url !== undefined) payload.avatar_url = editData.avatar_url
      if (editData.schedule_json !== undefined) payload.schedule_json = editData.schedule_json
      payload.commands_json = JSON.stringify(editCommands)
      await adminApi.patchBot(botId, payload)
      setExpandedId(null); setEditData({}); setEditCommands([]); await load(); showSuccess('Бот обновлён')
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось обновить бота') }
  }

  const reloadConfig = async () => {
    try { await adminApi.reloadBotConfig(); showSuccess('Конфигурация перезагружена') }
    catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось перезагрузить конфигурацию') }
  }

  const expand = (bot: BotRow) => {
    if (expandedId === bot.id) { setExpandedId(null); setEditData({}); setEditCommands([]) }
    else {
      setExpandedId(bot.id)
      setEditData({ name: bot.name, description: bot.description, rate_limit_ms: bot.rate_limit_ms, allowed_rooms: bot.allowed_rooms || [], avatar_url: bot.avatar_url || '', schedule_json: bot.schedule_json || '[]' })
      setEditCommands(parseCommands(bot.commands_json))
    }
  }

  const addCommand = () => {
    setEditCommands([...editCommands, { command: '/new', handler: 'static-reply', description: '', is_active: true }])
  }

  const updateCommand = (idx: number, field: string, value: any) => {
    const updated = [...editCommands]
    updated[idx] = { ...updated[idx], [field]: value }
    setEditCommands(updated)
  }

  const removeCommand = (idx: number) => {
    setEditCommands(editCommands.filter((_, i) => i !== idx))
  }

  const testCommand = async (cmd: BotCommand) => {
    try {
      const data = await adminApi.testBotCommand({ handler: cmd.handler, description: cmd.description, webhook_url: cmd.webhook_url, args: 'test' })
      setTestResult(data.result)
      setTimeout(() => setTestResult(null), 5000)
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось протестировать') }
  }

  const exportBot = async (bot: BotRow) => {
    try {
      const data = await adminApi.exportBot(bot.id)
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a'); a.href = url; a.download = `bot-${bot.name}.json`; a.click()
      URL.revokeObjectURL(url)
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось экспортировать') }
  }

  const importBot = async () => {
    const input = document.createElement('input'); input.type = 'file'; input.accept = '.json'
    input.onchange = async () => {
      const file = input.files?.[0]
      if (!file) return
      try {
        const text = await file.text()
        const data = JSON.parse(text)
        await adminApi.importBot(data); await load(); showSuccess('Бот импортирован')
      } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось импортировать') }
    }
    input.click()
  }

  const createFromTemplate = async (tmpl: BotTemplate) => {
    try {
      await adminApi.createBot({
        name: tmpl.name + '-' + Date.now().toString(36),
        description: tmpl.description, is_enabled: true, rate_limit_ms: 2000,
        allowed_rooms: [], commands_json: tmpl.commands_json, schedule_json: tmpl.schedule_json || '[]',
      })
      await load(); setTab('list'); showSuccess(`Бот "${tmpl.name}" создан из шаблона`)
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Не удалось создать из шаблона') }
  }

  const exportCSV = () => {
    const header = 'Время,Команда,Аргументы,Статус,Ошибка,Room ID,User ID\n'
    const rows = history.map(e => [new Date(e.created_at).toLocaleString('ru'), '/' + e.command, e.args || '', e.status, e.error || '', e.room_id, e.user_id].map(v => escapeCSV(v)).join(',')).join('\n')
    const blob = new Blob([header + rows], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a'); a.href = url; a.download = 'bot-history.csv'; a.click()
    URL.revokeObjectURL(url)
  }

  const groupedStats = commandStats.reduce<Record<string, Record<string, number>>>((acc, s) => {
    if (!acc[s.command]) acc[s.command] = {}
    acc[s.command][s.status] = s.count
    return acc
  }, {})

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
          <button onClick={importBot}>Импорт</button>
        </div>
      </div>

      {error && <p className="error">{error}</p>}
      {success && <p style={{ color: 'var(--success-color, #4ade80)', marginBottom: 12 }}>{success}</p>}
      {testResult && <div style={{ padding: 12, borderRadius: 8, background: 'var(--card-bg, #313244)', marginBottom: 12, fontFamily: 'monospace', fontSize: '0.85rem', whiteSpace: 'pre-wrap' }}>Результат теста: {testResult}</div>}

      <div style={{ display: 'flex', gap: 8, marginBottom: 16, flexWrap: 'wrap' }}>
        {(['list', 'errors', 'analytics', 'history', 'templates'] as const).map(t => (
          <button key={t} className={tab === t ? 'primary' : ''} onClick={() => setTab(t)}>
            {{ list: 'Список ботов', errors: 'Ошибки', analytics: 'Аналитика', history: 'История', templates: 'Шаблоны' }[t]}
          </button>
        ))}
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

          {loading ? <div className="loading">Загрузка...</div> : (
            <div className="users-table">
              <table>
                <thead><tr><th>Имя</th><th>Описание</th><th>Rate limit</th><th>Команд</th><th>Статус</th><th>Действия</th></tr></thead>
                <tbody>
                  {bots.map((bot) => (
                    <Fragment key={bot.id}>
                      <tr style={{ cursor: 'pointer' }} onClick={() => expand(bot)}>
                        <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          {bot.avatar_url && <img src={bot.avatar_url} alt="" style={{ width: 24, height: 24, borderRadius: '50%' }} />}
                          {bot.name}
                        </td>
                        <td>{bot.description || '—'}</td>
                        <td>{bot.rate_limit_ms} ms</td>
                        <td>{parseCommands(bot.commands_json).length}</td>
                        <td>{bot.is_enabled ? 'Включен' : 'Отключен'}</td>
                        <td>
                          <div style={{ display: 'flex', gap: 4 }}>
                            <button onClick={(e) => { e.stopPropagation(); void toggle(bot) }}>{bot.is_enabled ? 'Выключить' : 'Включить'}</button>
                            <button onClick={(e) => { e.stopPropagation(); void exportBot(bot) }} title="Экспорт">📤</button>
                            <button style={{ color: 'var(--error-color, #f38ba8)' }} onClick={(e) => { e.stopPropagation(); void deleteBot(bot) }}>Удалить</button>
                          </div>
                        </td>
                      </tr>
                      {expandedId === bot.id && (
                        <tr>
                          <td colSpan={6}>
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
                                  <label>URL аватара</label>
                                  <input value={editData.avatar_url || ''} onChange={(e) => setEditData({ ...editData, avatar_url: e.target.value })} placeholder="https://..." />
                                </div>
                                <div className="form-group" style={{ gridColumn: 'span 2' }}>
                                  <label>Разрешённые комнаты (UUID через запятую)</label>
                                  <input
                                    value={(editData.allowed_rooms || []).join(', ')}
                                    onChange={(e) => setEditData({ ...editData, allowed_rooms: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })}
                                    placeholder="Пусто = все комнаты"
                                  />
                                </div>
                              </div>

                              <div style={{ marginTop: 16 }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                                  <h4 style={{ margin: 0 }}>Команды ({editCommands.length})</h4>
                                  <button onClick={addCommand}>+ Добавить команду</button>
                                </div>
                                {editCommands.map((cmd, idx) => (
                                  <div key={idx} style={{ display: 'grid', gridTemplateColumns: '120px 160px 1fr auto auto auto', gap: 8, marginBottom: 8, alignItems: 'start', padding: 10, borderRadius: 8, background: 'var(--input-bg, #1e1e2e)', border: '1px solid var(--border-color, #444)' }}>
                                    <input value={cmd.command} onChange={(e) => updateCommand(idx, 'command', e.target.value)} placeholder="/cmd" style={{ fontFamily: 'monospace', fontSize: '0.85rem' }} />
                                    <select value={cmd.handler} onChange={(e) => updateCommand(idx, 'handler', e.target.value)} style={{ fontSize: '0.85rem' }}>
                                      {HANDLER_TYPES.map(h => <option key={h.value} value={h.value}>{h.label}</option>)}
                                    </select>
                                    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                                      <textarea
                                        value={cmd.description}
                                        onChange={(e) => updateCommand(idx, 'description', e.target.value)}
                                        rows={2}
                                        placeholder={cmd.handler === 'template' ? 'Привет, {{user_name}}!' : cmd.handler === 'random' ? 'ответ1||ответ2||ответ3' : cmd.handler === 'alias' ? 'help' : cmd.handler === 'webhook' ? 'URL или описание' : 'Текст ответа'}
                                        style={{ fontFamily: 'monospace', fontSize: '0.8rem', padding: 6, borderRadius: 4, border: '1px solid var(--border-color, #555)', background: 'var(--card-bg, #313244)', color: 'var(--text-color, #cdd6f4)', resize: 'vertical' }}
                                      />
                                      {cmd.handler === 'webhook' && (
                                        <input value={cmd.webhook_url || ''} onChange={(e) => updateCommand(idx, 'webhook_url', e.target.value)} placeholder="https://example.com/hook" style={{ fontSize: '0.8rem' }} />
                                      )}
                                      {cmd.rate_limit_ms !== undefined && cmd.rate_limit_ms > 0 ? (
                                        <div style={{ display: 'flex', gap: 4, alignItems: 'center', fontSize: '0.8rem' }}>
                                          <span>Rate limit:</span>
                                          <input type="number" value={cmd.rate_limit_ms} onChange={(e) => updateCommand(idx, 'rate_limit_ms', parseInt(e.target.value) || 0)} style={{ width: 80, fontSize: '0.8rem' }} /> ms
                                        </div>
                                      ) : (
                                        <button style={{ fontSize: '0.75rem', padding: '2px 6px', opacity: 0.6 }} onClick={() => updateCommand(idx, 'rate_limit_ms', 2000)}>+ Rate limit</button>
                                      )}
                                    </div>
                                    <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: '0.8rem', cursor: 'pointer' }}>
                                      <input type="checkbox" checked={cmd.is_active} onChange={(e) => updateCommand(idx, 'is_active', e.target.checked)} />
                                      Вкл
                                    </label>
                                    <button onClick={() => void testCommand(cmd)} title="Тест" style={{ padding: '4px 8px' }}>▶️</button>
                                    <button onClick={() => removeCommand(idx)} style={{ color: 'var(--error-color, #f38ba8)', padding: '4px 8px' }}>✕</button>
                                  </div>
                                ))}
                              </div>

                              <div className="form-group" style={{ marginTop: 12 }}>
                                <label>Расписание (JSON)</label>
                                <textarea
                                  value={editData.schedule_json || '[]'}
                                  onChange={(e) => setEditData({ ...editData, schedule_json: e.target.value })}
                                  rows={3}
                                  style={{ width: '100%', fontFamily: 'monospace', fontSize: '0.85rem', padding: 8, borderRadius: 6, border: '1px solid var(--border-color, #444)', background: 'var(--input-bg, #1e1e2e)', color: 'var(--text-color, #cdd6f4)' }}
                                  placeholder='[{"cron": "0 10 * * 1-5", "room_id": "...", "message": "Доброе утро!"}]'
                                />
                                <small style={{ color: 'var(--muted-color, #888)' }}>Cron-формат: минута час день месяц день_недели</small>
                              </div>

                              <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
                                <button className="primary" onClick={() => void saveEdit(bot.id)}>Сохранить</button>
                                <button onClick={() => { setExpandedId(null); setEditData({}); setEditCommands([]) }}>Отмена</button>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </Fragment>
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
            <thead><tr><th>Время</th><th>Команда</th><th>Статус</th><th>Ошибка</th><th>Room ID</th></tr></thead>
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

      {tab === 'analytics' && (
        <div className="settings-section">
          <h2>Аналитика по командам (7 дней)</h2>
          {Object.keys(groupedStats).length === 0 ? (
            <p style={{ color: 'var(--muted-color, #888)' }}>Нет данных</p>
          ) : (
            <div className="users-table">
              <table>
                <thead><tr><th>Команда</th><th>Отправлено</th><th>Ошибки</th><th>Rate-limited</th><th>Отключено</th><th>Нет доступа</th><th>Всего</th></tr></thead>
                <tbody>
                  {Object.entries(groupedStats).map(([cmd, statuses]) => {
                    const total = Object.values(statuses).reduce((a, b) => a + b, 0)
                    return (
                      <tr key={cmd}>
                        <td style={{ fontFamily: 'monospace' }}>/{cmd}</td>
                        <td style={{ color: 'var(--success-color, #4ade80)' }}>{statuses['sent'] || 0}</td>
                        <td style={{ color: 'var(--error-color, #f38ba8)' }}>{statuses['failed'] || 0}</td>
                        <td>{statuses['rate_limited'] || 0}</td>
                        <td>{statuses['disabled'] || 0}</td>
                        <td>{(statuses['permission_denied'] || 0) + (statuses['room_not_allowed'] || 0)}</td>
                        <td><strong>{total}</strong></td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {tab === 'history' && (
        <div>
          <div className="settings-section" style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', gap: 8, alignItems: 'flex-end', flexWrap: 'wrap' }}>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label>Команда</label>
                <input value={historyFilter.command} onChange={e => setHistoryFilter({ ...historyFilter, command: e.target.value })} placeholder="help" style={{ width: 120 }} />
              </div>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label>Статус</label>
                <select value={historyFilter.status} onChange={e => setHistoryFilter({ ...historyFilter, status: e.target.value })} style={{ width: 140 }}>
                  <option value="">Все</option>
                  <option value="sent">Отправлено</option>
                  <option value="failed">Ошибка</option>
                  <option value="rate_limited">Rate limited</option>
                  <option value="disabled">Отключено</option>
                  <option value="permission_denied">Нет доступа</option>
                </select>
              </div>
              <button className="primary" onClick={() => void loadHistory()}>Найти</button>
              <button onClick={exportCSV}>Экспорт CSV</button>
              <span style={{ fontSize: '0.85rem', color: 'var(--muted-color, #888)' }}>Найдено: {historyTotal}</span>
            </div>
          </div>
          <div className="users-table">
            <table>
              <thead><tr><th>Время</th><th>Команда</th><th>Аргументы</th><th>Статус</th><th>Ошибка</th><th>Room ID</th><th>User ID</th></tr></thead>
              <tbody>
                {history.length === 0 ? (
                  <tr><td colSpan={7} style={{ textAlign: 'center', color: 'var(--muted-color, #888)' }}>Нет записей</td></tr>
                ) : history.map((e) => (
                  <tr key={e.id}>
                    <td style={{ fontSize: '0.8rem' }}>{new Date(e.created_at).toLocaleString('ru')}</td>
                    <td style={{ fontFamily: 'monospace' }}>/{e.command}</td>
                    <td style={{ maxWidth: 150, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{e.args || '—'}</td>
                    <td><span style={{ padding: '2px 6px', borderRadius: 4, fontSize: '0.8rem', background: e.status === 'sent' ? 'rgba(74,222,128,0.2)' : e.status === 'failed' ? 'rgba(243,139,168,0.2)' : 'rgba(136,136,136,0.2)' }}>{e.status}</span></td>
                    <td style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: '0.8rem' }}>{e.error || '—'}</td>
                    <td style={{ fontSize: '0.75rem', fontFamily: 'monospace' }}>{e.room_id?.substring(0, 8)}</td>
                    <td style={{ fontSize: '0.75rem', fontFamily: 'monospace' }}>{e.user_id?.substring(0, 8)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'templates' && (
        <div className="settings-section">
          <h2>Библиотека шаблонов</h2>
          <p style={{ color: 'var(--muted-color, #888)', marginBottom: 16 }}>Создайте бота из готового шаблона — настройте под себя после создания.</p>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 16 }}>
            {templates.map((tmpl) => (
              <div key={tmpl.name} style={{ padding: 16, borderRadius: 12, background: 'var(--card-bg, #313244)', border: '1px solid var(--border-color, #444)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <h3 style={{ margin: 0 }}>{tmpl.name}</h3>
                  <span style={{ fontSize: '0.75rem', padding: '2px 8px', borderRadius: 10, background: 'var(--input-bg, #1e1e2e)' }}>{tmpl.category}</span>
                </div>
                <p style={{ fontSize: '0.85rem', color: 'var(--muted-color, #888)', marginBottom: 12 }}>{tmpl.description}</p>
                <p style={{ fontSize: '0.8rem', marginBottom: 12 }}>
                  Команд: {parseCommands(tmpl.commands_json).length}
                  {tmpl.schedule_json && tmpl.schedule_json !== '[]' && ' | С расписанием'}
                </p>
                <button className="primary" onClick={() => void createFromTemplate(tmpl)}>Создать из шаблона</button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
