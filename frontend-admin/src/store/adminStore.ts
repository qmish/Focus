import { create } from 'zustand'
import { getAdminAccessToken } from '../lib/authToken'
import { normalizeStats, type AdminStats } from '../lib/adminStats'

export interface User {
  id: string
  email: string
  name: string
  roles: string[]
  is_active: boolean
  created_at: string
}

type Stats = AdminStats

interface Pagination {
  page: number
  per_page: number
  total: number
  total_pages: number
}

interface AdminState {
  users: User[]
  stats: Stats
  pagination: Pagination
  currentPage: number
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
  pagination: {
    page: 1,
    per_page: 20,
    total: 0,
    total_pages: 1,
  },
  currentPage: 1,
  loading: false,
  error: null,

  fetchUsers: async (page) => {
    set({ loading: true, error: null, currentPage: page })
    try {
      const token = getAdminAccessToken()
      const response = await fetch(`/api/v1/admin/users?page=${page}&per_page=20`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })
      
      if (!response.ok) throw new Error('Failed to fetch users')
      
      const data = await response.json()
      set({
        users: data.data || [],
        pagination: data.pagination || { page, per_page: 20, total: 0, total_pages: 1 },
        loading: false,
      })
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
      set({ stats: normalizeStats(data), loading: false })
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
      
      await get().fetchUsers(get().currentPage)
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
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
      
      await get().fetchUsers(get().currentPage)
      await get().fetchStats()
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
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
      
      await get().fetchUsers(get().currentPage)
      await get().fetchStats()
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },
}))
