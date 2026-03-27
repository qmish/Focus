import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAdminAuthStore } from '../store/adminAuthStore'
import { useAdminUi } from '../providers/AdminUiProvider'

export default function LoginPage() {
  const { isAuthenticated, isLoading, loginLocal, loginKeycloak, keycloakAvailable, init } = useAdminAuthStore()
  const { branding } = useAdminUi()
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => { init() }, [])

  useEffect(() => {
    if (isAuthenticated && !isLoading) {
      navigate('/dashboard')
    }
  }, [isAuthenticated, isLoading, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await loginLocal(email, password)
    } catch (err: any) {
      setError(err.message || 'Ошибка авторизации')
    } finally {
      setSubmitting(false)
    }
  }

  if (isLoading) {
    return <div className="login-page"><div className="loading">Загрузка...</div></div>
  }

  return (
    <div className="login-page">
      <div className="login-container">
        <div className="login-card">
          <img src={branding.logoUrl} alt={branding.productName} className="login-logo" />
          <h1 className="login-title">{branding.productName}</h1>
          <p className="login-subtitle">Панель администратора</p>

          <form onSubmit={handleSubmit} className="login-form">
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="login-input"
              required
            />
            <input
              type="password"
              placeholder="Пароль"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="login-input"
              required
            />

            {error && <div className="login-error">{error}</div>}

            <button type="submit" className="login-btn login-btn-primary" disabled={submitting}>
              {submitting ? 'Подождите...' : 'Войти'}
            </button>
          </form>

          <div className="login-divider"><span>или</span></div>
          <button
            onClick={() => {
              if (keycloakAvailable) {
                loginKeycloak()
              } else {
                setError('SSO (Keycloak/AD) не настроен. Обратитесь к администратору.')
              }
            }}
            className="login-btn login-btn-keycloak"
          >
            Войти через Keycloak / Active Directory
          </button>

          <div className="login-warning">
            <p>Доступ только для пользователей с ролью admin</p>
          </div>
        </div>
      </div>
    </div>
  )
}
