import { useState } from 'react'
import { Link, Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAdminAuthStore } from '../store/adminAuthStore'

export default function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout, isAuthenticated, isLoading } = useAdminAuthStore()

  const navItems = [
    { path: '/dashboard', label: 'Дашборд', icon: '📊' },
    { path: '/users', label: 'Пользователи', icon: '👥' },
    { path: '/conferences', label: 'Конференции', icon: '🎥' },
    { path: '/observability', label: 'Наблюдаемость', icon: '🩺' },
    { path: '/settings', label: 'Настройки', icon: '⚙️' },
  ]

  if (isLoading) {
    return <div style={{display:'flex',alignItems:'center',justifyContent:'center',height:'100vh',color:'#a6adc8'}}>Загрузка...</div>
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="admin-layout">
      <aside className={`sidebar ${sidebarOpen ? 'open' : 'closed'}`}>
        <div className="sidebar-header">
          <h1>Focus Admin</h1>
          <button onClick={() => setSidebarOpen(!sidebarOpen)}>
            {sidebarOpen ? '◀' : '▶'}
          </button>
        </div>

        <nav className="sidebar-nav">
          {navItems.map(item => (
            <Link
              key={item.path}
              to={item.path}
              className={`nav-item ${location.pathname === item.path ? 'active' : ''}`}
            >
              <span className="nav-icon">{item.icon}</span>
              {sidebarOpen && <span>{item.label}</span>}
            </Link>
          ))}
        </nav>

        <div className="sidebar-footer">
          {sidebarOpen && (
            <div className="user-info">
              <span className="user-name">{user?.name || 'Admin'}</span>
              <span className="user-email">{user?.email || ''}</span>
            </div>
          )}
          <button onClick={() => { logout(); navigate('/login') }} className="logout-btn">
            {sidebarOpen ? 'Выйти' : '🚪'}
          </button>
        </div>
      </aside>

      <main className="main-content">
        <header className="top-bar">
          <div className="breadcrumb">
            {location.pathname.split('/').filter(Boolean).map((segment, index, arr) => (
              <span key={segment} className="breadcrumb-item">
                <Link to={'/' + arr.slice(0, index + 1).join('/')}>
                  {segment.charAt(0).toUpperCase() + segment.slice(1)}
                </Link>
                {index < arr.length - 1 && ' / '}
              </span>
            ))}
          </div>
        </header>

        <div className="page-content">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
