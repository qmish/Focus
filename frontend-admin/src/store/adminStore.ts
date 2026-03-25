import { create } from 'zustand'
import { getAdminAccessToken } from '../lib/authToken'

interface User {
  id: string
  email: string
  name: string
  roles: string[]
  is_active: boolean
  created_at: string
}

interface Stats {
  users: { total: number }
  rooms: { total: number }
  conferences: { active: number }
  messages: { today: number }
}

interface AdminState {
  users: User[]
  stats: Stats
  loading: boolean
  error: string | null
  fetchUsers: (page: number) => Promise<void>
  fetchStats: () => Promise<void>
  updateUserRoles: (userId: string, roles: string[]) => Promise<void>
  banUser: (userId: string, reason: string) => Promise<void>
  unbanUser: (userId: string) => Promise<void>
}

export const useAdminStore = create<AdminState>((set, get) => ({
  users: [],
  stats: {
    users: { total: 0 },
    rooms: { total: 0 },
    conferences: { active: 0 },
    messages: { today: 0 },
  },
  loading: false,
  error: null,

  fetchUsers: async (page) => {
    set({ loading: true, error: null })
    try {
      const token = getAdminAccessToken()
      const response = await fetch(`/api/v1/admin/users?page=${page}&per_page=20`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })
      
      if (!response.ok) throw new Error('Failed to fetch users')
      
      const data = await response.json()
      set({ users: data.data || [], loading: false })
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error',
        loading: false 
      })
    }
  },

  fetchStats: async () => {
    set({ loading: true, error: null })
    try {
      const token = getAdminAccessToken()
      const response = await fetch('/api/v1/admin/stats', {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })
      
      if (!response.ok) throw new Error('Failed to fetch stats')
      
      const data = await response.json()
      set({ stats: data, loading: false })
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error',
        loading: false 
      })
    }
  },

  updateUserRoles: async (userId, roles) => {
    try {
      const token = getAdminAccessToken()
      const response = await fetch(`/api/v1/admin/users/${userId}/roles`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ roles }),
      })
      
      if (!response.ok) throw new Error('Failed to update roles')
      
      // Обновляем список пользователей
      get().fetchUsers(1)
    } catch (error) {
      console.error('Failed to update roles:', error)
      alert('Не удалось обновить роли')
    }
  },

  banUser: async (userId, reason) => {
    try {
      const token = getAdminAccessToken()
      const response = await fetch(`/api/v1/admin/users/${userId}/ban`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ reason, duration_hours: 0 }),
      })
      
      if (!response.ok) throw new Error('Failed to ban user')
      
      get().fetchUsers(1)
    } catch (error) {
      console.error('Failed to ban user:', error)
      alert('Не удалось заблокировать пользователя')
    }
  },

  unbanUser: async (userId) => {
    try {
      const token = getAdminAccessToken()
      const response = await fetch(`/api/v1/admin/users/${userId}/unban`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })
      
      if (!response.ok) throw new Error('Failed to unban user')
      
      get().fetchUsers(1)
    } catch (error) {
      console.error('Failed to unban user:', error)
      alert('Не удалось разблокировать пользователя')
    }
  },
}))
