import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAdminAuthStore } from './store/adminAuthStore'
import Layout from './components/Layout'
import DashboardPage from './pages/DashboardPage'
import UsersPage from './pages/UsersPage'
import ConferencesPage from './pages/ConferencesPage'
import SettingsPage from './pages/SettingsPage'
import ObservabilityPage from './pages/ObservabilityPage'
import LoginPage from './pages/LoginPage'
import BotsPage from './pages/BotsPage'
import IntegrationsPage from './pages/IntegrationsPage'

function AdminApp() {
  const init = useAdminAuthStore(s => s.init)
  useEffect(() => { init() }, [init])

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="conferences" element={<ConferencesPage />} />
          <Route path="observability" element={<ObservabilityPage />} />
          <Route path="bots" element={<BotsPage />} />
          <Route path="integrations" element={<IntegrationsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default AdminApp
