import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'

export default function LoginPage() {
  const { isAuthenticated, isLoading, loginLocal, registerLocal, loginKeycloak, keycloakAvailable, init } = useAuthStore()
  const navigate = useNavigate()

  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => { init() }, [])

  useEffect(() => {
    if (isAuthenticated && !isLoading) {
      navigate('/rooms')
    }
  }, [isAuthenticated, isLoading, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      if (mode === 'register') {
        await registerLocal(email, password, name)
      } else {
        await loginLocal(email, password)
      }
    } catch (err: any) {
      setError(err.message || 'Authentication failed')
    } finally {
      setSubmitting(false)
    }
  }

  if (isLoading) {
    return (
      <div className="login-page">
        <div className="login-container">
          <h1>Focus</h1>
          <p>Загрузка...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="login-page">
      <div className="login-container">
        <div className="login-card">
          <img src="/logo.png" alt="Focus" className="login-logo" />
          <h1 className="login-title">Focus</h1>
          <p className="login-subtitle">Корпоративный мессенджер</p>

          <div className="login-tabs">
            <button
              className={`login-tab ${mode === 'login' ? 'active' : ''}`}
              onClick={() => { setMode('login'); setError('') }}
            >
              Вход
            </button>
            <button
              className={`login-tab ${mode === 'register' ? 'active' : ''}`}
              onClick={() => { setMode('register'); setError('') }}
            >
              Регистрация
            </button>
          </div>

          <form onSubmit={handleSubmit} className="login-form">
            {mode === 'register' && (
              <input
                type="text"
                placeholder="Имя"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="login-input"
                required
              />
            )}
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
              minLength={6}
            />

            {error && <div className="login-error">{error}</div>}

            <button type="submit" className="login-btn login-btn-primary" disabled={submitting}>
              {submitting
                ? 'Подождите...'
                : mode === 'register'
                  ? 'Зарегистрироваться'
                  : 'Войти'}
            </button>
          </form>

          <div className="login-divider">
            <span>или</span>
          </div>
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
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{verticalAlign: 'middle', marginRight: 8}}><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
            Войти через Keycloak / Active Directory
          </button>

          <div className="login-features">
            <div className="feature">
              <span className="feature-icon">💬</span>
              <span>Мгновенные сообщения</span>
            </div>
            <div className="feature">
              <span className="feature-icon">🎥</span>
              <span>Видеоконференции</span>
            </div>
            <div className="feature">
              <span className="feature-icon">📅</span>
              <span>Календари</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
