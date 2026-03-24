import { useState } from 'react'
import { Link, useLocation, Outlet } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'

export default function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const location = useLocation()
  const { user, logout } = useAuthStore()

  const navItems = [
    { path: '/rooms', label: 'Комнаты' },
    { path: '/profile', label: 'Профиль' },
  ]

  return (
    <div className="app-layout">
      <aside className={`sidebar ${sidebarOpen ? 'open' : 'closed'}`}>
        <div className="sidebar-header">
          <h1>Focus</h1>
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
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="sidebar-footer">
          <div className="user-info">
            <span className="user-name">{user?.name || 'User'}</span>
            <span className="user-email">{user?.email || ''}</span>
          </div>
          <button onClick={logout} className="logout-btn">
            Выйти
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
