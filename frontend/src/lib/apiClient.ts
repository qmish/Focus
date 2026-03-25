import { useAuthStore } from '../store/authStore'
import { apiRequest } from './apiClientCore'

function getToken(): string | null {
  return useAuthStore.getState().token
}

export const apiClient = {
  get<T>(url: string, retry = 1): Promise<T> {
    return apiRequest<T>(url, { method: 'GET', token: getToken(), retry })
  },
  post<T>(url: string, body: unknown): Promise<T> {
    return apiRequest<T>(url, { method: 'POST', token: getToken(), body, retry: 0 })
  },
  put<T>(url: string, body: unknown): Promise<T> {
    return apiRequest<T>(url, { method: 'PUT', token: getToken(), body, retry: 0 })
  },
  delete(url: string): Promise<void> {
    return apiRequest<void>(url, { method: 'DELETE', token: getToken(), retry: 0 })
  },
}
