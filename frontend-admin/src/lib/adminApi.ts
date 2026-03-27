import { getAdminAccessToken } from './authToken'

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = getAdminAccessToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (init.headers) Object.assign(headers, init.headers as Record<string, string>)
  if (token) headers['Authorization'] = `Bearer ${token}`

  const response = await fetch(path, { ...init, headers })
  if (!response.ok) {
    const text = await response.text()
    throw new Error(text || `Request failed: ${response.status}`)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export type AdminUser = {
  id: string
  email: string
  name: string
  roles: string[]
  is_active: boolean
  created_at: string
}

export type AdminInvite = {
  id: string
  email: string
  roles: string[]
  status: string
  expires_at: string
  invited_by: string
}

export const adminApi = {
  listUsers: (page: number, perPage = 20) =>
    request<{ data: AdminUser[]; pagination: { page: number; per_page: number; total: number; total_pages: number } }>(
      `/api/v1/admin/users?page=${page}&per_page=${perPage}`,
    ),
  createUser: (payload: { email: string; name: string; password?: string; roles?: string[]; is_active?: boolean }) =>
    request<AdminUser>('/api/v1/admin/users', { method: 'POST', body: JSON.stringify(payload) }),
  patchUser: (userId: string, payload: Record<string, unknown>) =>
    request<AdminUser>(`/api/v1/admin/users/${userId}`, { method: 'PATCH', body: JSON.stringify(payload) }),
  deleteUser: (userId: string) =>
    request<void>(`/api/v1/admin/users/${userId}`, { method: 'DELETE' }),
  updateUserRoles: (userId: string, roles: string[]) =>
    request<AdminUser>(`/api/v1/admin/users/${userId}/roles`, { method: 'PUT', body: JSON.stringify({ roles }) }),
  banUser: (userId: string, reason: string, durationHours = 0) =>
    request(`/api/v1/admin/users/${userId}/ban`, { method: 'POST', body: JSON.stringify({ reason, duration_hours: durationHours }) }),
  unbanUser: (userId: string) =>
    request(`/api/v1/admin/users/${userId}/unban`, { method: 'POST' }),
  listInvites: (page: number, perPage = 20) =>
    request<{ data: AdminInvite[] }>(`/api/v1/admin/invites?page=${page}&per_page=${perPage}`),
  createInvite: (payload: { email: string; roles: string[]; expires_in_hours?: number }) =>
    request<{ invite: AdminInvite; inviteUrl: string; mailSent: boolean }>('/api/v1/admin/invites', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  resendInvite: (inviteId: string) =>
    request<{ invite: AdminInvite; inviteUrl: string; mailSent: boolean }>(`/api/v1/admin/invites/${inviteId}/resend`, { method: 'POST' }),
  listBots: () => request<{ data: any[] }>('/api/v1/admin/bots'),
  createBot: (payload: Record<string, unknown>) => request('/api/v1/admin/bots', { method: 'POST', body: JSON.stringify(payload) }),
  patchBot: (botId: string, payload: Record<string, unknown>) =>
    request(`/api/v1/admin/bots/${botId}`, { method: 'PATCH', body: JSON.stringify(payload) }),
  toggleBot: (botId: string, enabled: boolean) =>
    request(`/api/v1/admin/bots/${botId}/${enabled ? 'enable' : 'disable'}`, { method: 'POST' }),
  getExchangeSettings: () => request('/api/v1/admin/exchange/settings'),
  putExchangeSettings: (payload: Record<string, unknown>) =>
    request('/api/v1/admin/exchange/settings', { method: 'PUT', body: JSON.stringify(payload) }),
  testExchangeConnection: (payload: { test_email?: string }) =>
    request('/api/v1/admin/exchange/test-connection', { method: 'POST', body: JSON.stringify(payload) }),
}
