export default function SettingsPage() {
  return (
    <div className="settings-page">
      <h1>Настройки</h1>

      <div className="settings-section">
        <h2>Общие настройки</h2>
        <div className="form-group">
          <label>Название системы</label>
          <input type="text" defaultValue="Focus Messenger" />
        </div>
        <div className="form-group">
          <label>Максимум участников в конференции</label>
          <input type="number" defaultValue="100" />
        </div>
        <button className="primary">Сохранить</button>
      </div>

      <div className="settings-section">
        <h2>Интеграции</h2>
        <div className="form-group">
          <label>Keycloak URL</label>
          <input type="text" defaultValue="http://localhost:8180" />
        </div>
        <div className="form-group">
          <label>Jitsi Base URL</label>
          <input type="text" defaultValue="https://meet.company.com" />
        </div>
        <button className="primary">Сохранить</button>
      </div>

      <div className="settings-section">
        <h2>Безопасность</h2>
        <div className="form-group">
          <label>Время жизни сессии (часы)</label>
          <input type="number" defaultValue="24" />
        </div>
        <button className="primary">Сохранить</button>
      </div>
    </div>
  )
}
