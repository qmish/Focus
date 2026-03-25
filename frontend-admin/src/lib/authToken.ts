import { useAdminAuthStore } from '../store/adminAuthStore'

export function getAdminAccessToken(): string | null {
  const stateToken = useAdminAuthStore.getState().token
  if (stateToken) {
    return stateToken
  }

  return localStorage.getItem('admin_token')
}
