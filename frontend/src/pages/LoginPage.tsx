import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'

export default function LoginPage() {
  const { isAuthenticated, isLoading, login, init } = useAuthStore()
  const navigate = useNavigate()

  useEffect(() => {
    init()
  }, [])

  useEffect(() => {
    if (isAuthenticated && !isLoading) {
      navigate('/rooms')
    }
  }, [isAuthenticated, isLoading, navigate])

  const handleLogin = async () => {
    await login()
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
          <h1 className="login-title">Focus</h1>
          <p className="login-subtitle">Корпоративный мессенджер</p>
          
          <div className="login-actions">
            <button onClick={handleLogin} className="login-btn">
              Войти через корпоративный портал
            </button>
          </div>

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
