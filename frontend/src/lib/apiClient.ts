import { useAuthStore } from '../store/authStore'
import { apiRequest, ApiError } from './apiClientCore'

function getToken(): string | null {
  return useAuthStore.getState().token
}

export const apiClient = {
  get<T>(url: string, retry = 1, headers?: Record<string, string>): Promise<T> {
    return apiRequest<T>(url, { method: 'GET', token: getToken(), retry, headers })
  },
  post<T>(url: string, body: unknown, headers?: Record<string, string>): Promise<T> {
    return apiRequest<T>(url, { method: 'POST', token: getToken(), body, retry: 0, headers })
  },
  put<T>(url: string, body: unknown): Promise<T> {
    return apiRequest<T>(url, { method: 'PUT', token: getToken(), body, retry: 0 })
  },
  delete(url: string): Promise<void> {
    return apiRequest<void>(url, { method: 'DELETE', token: getToken(), retry: 0 })
  },
  async uploadFile<T>(url: string, file: File): Promise<T> {
    const token = getToken()
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(url, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    })
    if (!res.ok) {
      const text = await res.text()
      throw new ApiError(text || `HTTP ${res.status}`, res.status, text)
    }
    return res.json() as Promise<T>
  },
}
