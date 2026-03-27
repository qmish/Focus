import { create } from 'zustand'
import { getAdminAccessToken } from '../lib/authToken'
import { normalizeStats, type AdminStats } from '../lib/adminStats'
import { adminApi, type AdminInvite, type AdminUser } from '../lib/adminApi'

export type User = AdminUser

type Stats = AdminStats

interface Pagination {
  page: number
  per_page: number
  total: number
  total_pages: number
}

interface AdminState {
  users: User[]
  invites: AdminInvite[]
  stats: Stats
  pagination: Pagination
  currentPage: number
  loading: boolean
  error: string | null
  fetchUsers: (page: number) => Promise<void>
  fetchStats: () => Promise<void>
  fetchInvites: () => Promise<void>
  createUser: (payload: { email: string; name: string; password?: string; roles?: string[]; is_active?: boolean }) => Promise<void>
  patchUser: (userId: string, payload: Record<string, unknown>) => Promise<void>
  deleteUser: (userId: string) => Promise<void>
  updateUserRoles: (userId: string, roles: string[]) => Promise<void>
  banUser: (userId: string, reason: string, durationHours?: number) => Promise<void>
  unbanUser: (userId: string) => Promise<void>
  createInvite: (payload: { email: string; roles: string[]; expires_in_hours?: number }) => Promise<string | null>
  resendInvite: (inviteId: string) => Promise<string | null>
}

export const useAdminStore = create<AdminState>((set, get) => ({
  users: [],
  invites: [],
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
      const data = await adminApi.listUsers(page, 20)
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
      const response = await fetch('/api/v1/admin/stats', { headers: { Authorization: `Bearer ${token || ''}` } })
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

  fetchInvites: async () => {
    try {
      const data = await adminApi.listInvites(get().currentPage, 20)
      set({ invites: data.data || [] })
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },

  createUser: async (payload) => {
    try {
      await adminApi.createUser(payload)
      await get().fetchUsers(get().currentPage)
      await get().fetchStats()
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },

  patchUser: async (userId, payload) => {
    try {
      await adminApi.patchUser(userId, payload)
      await get().fetchUsers(get().currentPage)
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },

  deleteUser: async (userId) => {
    try {
      await adminApi.deleteUser(userId)
      await get().fetchUsers(get().currentPage)
      await get().fetchStats()
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },

  updateUserRoles: async (userId, roles) => {
    try {
      await adminApi.updateUserRoles(userId, roles)
      await get().fetchUsers(get().currentPage)
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },

  banUser: async (userId, reason, durationHours = 0) => {
    try {
      await adminApi.banUser(userId, reason, durationHours)
      await get().fetchUsers(get().currentPage)
      await get().fetchStats()
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },

  unbanUser: async (userId) => {
    try {
      await adminApi.unbanUser(userId)
      await get().fetchUsers(get().currentPage)
      await get().fetchStats()
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
    }
  },

  createInvite: async (payload) => {
    try {
      const result = await adminApi.createInvite(payload)
      await get().fetchInvites()
      return result.inviteUrl || null
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
      return null
    }
  },

  resendInvite: async (inviteId) => {
    try {
      const result = await adminApi.resendInvite(inviteId)
      await get().fetchInvites()
      return result.inviteUrl || null
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Unknown error' })
      return null
    }
  },
}))
