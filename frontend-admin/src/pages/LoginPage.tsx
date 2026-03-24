import { useAdminAuthStore } from '../store/adminAuthStore'

export default function LoginPage() {
  const { login, isLoading } = useAdminAuthStore()

  const handleLogin = async () => {
    await login()
  }

  if (isLoading) {
    return <div className="login-page"><div className="loading">Загрузка...</div></div>
  }

  return (
    <div className="login-page">
      <div className="login-container">
        <h1>Focus Admin</h1>
        <p>Панель администратора</p>
        
        <div className="login-actions">
          <button onClick={handleLogin} className="login-btn">
            Войти через Keycloak
          </button>
        </div>

        <div className="login-warning">
          <p>⚠️ Доступ только для пользователей с ролью admin</p>
        </div>
      </div>
    </div>
  )
}
