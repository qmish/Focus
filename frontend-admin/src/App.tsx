import { useEffect, lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAdminAuthStore } from './store/adminAuthStore'
import ErrorBoundary from './components/ErrorBoundary'
import Layout from './components/Layout'

const DashboardPage = lazy(() => import('./pages/DashboardPage'))
const UsersPage = lazy(() => import('./pages/UsersPage'))
const ConferencesPage = lazy(() => import('./pages/ConferencesPage'))
const SettingsPage = lazy(() => import('./pages/SettingsPage'))
const ObservabilityPage = lazy(() => import('./pages/ObservabilityPage'))
const LoginPage = lazy(() => import('./pages/LoginPage'))
const BotsPage = lazy(() => import('./pages/BotsPage'))
const IntegrationsPage = lazy(() => import('./pages/IntegrationsPage'))
const AuditPage = lazy(() => import('./pages/AuditPage'))
const WebhooksPage = lazy(() => import('./pages/WebhooksPage'))
const ConferencePoliciesPage = lazy(() => import('./pages/ConferencePoliciesPage'))
const AnalyticsPage = lazy(() => import('./pages/AnalyticsPage'))

function AdminApp() {
  const init = useAdminAuthStore(s => s.init)
  useEffect(() => { init() }, [init])

  return (
    <BrowserRouter>
      <ErrorBoundary>
      <Suspense fallback={<div className="loading">Загрузка...</div>}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="conferences" element={<ConferencesPage />} />
          <Route path="conferences/policies" element={<ConferencePoliciesPage />} />
          <Route path="observability" element={<ObservabilityPage />} />
          <Route path="bots" element={<BotsPage />} />
          <Route path="integrations" element={<IntegrationsPage />} />
          <Route path="webhooks" element={<WebhooksPage />} />
          <Route path="audit" element={<AuditPage />} />
          <Route path="analytics" element={<AnalyticsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
      </Suspense>
      </ErrorBoundary>
    </BrowserRouter>
  )
}

export default AdminApp
