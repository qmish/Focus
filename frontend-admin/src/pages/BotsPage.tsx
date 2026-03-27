import { useEffect, useState } from 'react'
import { adminApi } from '../lib/adminApi'

type BotRow = {
  id: string
  name: string
  description: string
  is_enabled: boolean
  rate_limit_ms: number
}

export default function BotsPage() {
  const [bots, setBots] = useState<BotRow[]>([])
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

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

  useEffect(() => {
    void load()
  }, [])

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

  return (
    <div>
      <h1>Боты</h1>
      {error && <p className="error">{error}</p>}

      <div className="settings-section">
        <h2>Добавить бота</h2>
        <div className="form-group">
          <label>Название</label>
          <input value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="form-group">
          <label>Описание</label>
          <input value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <button className="primary" onClick={() => void createBot()}>Создать</button>
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
                <tr key={bot.id}>
                  <td>{bot.name}</td>
                  <td>{bot.description}</td>
                  <td>{bot.rate_limit_ms} ms</td>
                  <td>{bot.is_enabled ? 'Включен' : 'Отключен'}</td>
                  <td>
                    <button onClick={() => void toggle(bot)}>
                      {bot.is_enabled ? 'Выключить' : 'Включить'}
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
