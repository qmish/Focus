import { create } from 'zustand'
import { apiClient } from '../lib/apiClient'

export interface Room {
  id: string
  name: string
  description: string
  type: 'public' | 'private' | 'meeting'
  jitsi_room_name: string
  is_private: boolean
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  room_id: string
  user_id: string
  content: string
  type: 'text' | 'image' | 'file' | 'system'
  created_at: string
  updated_at: string
  user?: {
    id: string
    name: string
    email: string
  }
}

interface RoomsState {
  rooms: Room[]
  currentRoom: Room | null
  isLoading: boolean
  error: string | null
  fetchRooms: () => Promise<void>
  createRoom: (name: string, type: string, description?: string) => Promise<Room>
  setCurrentRoom: (room: Room | null) => void
  deleteRoom: (roomId: string) => Promise<void>
}

export const useRoomsStore = create<RoomsState>((set, get) => ({
  rooms: [],
  currentRoom: null,
  isLoading: false,
  error: null,

  fetchRooms: async () => {
    set({ isLoading: true, error: null })
    try {
      const data = await apiClient.get<{ data?: Room[] }>('/api/v1/rooms')
      set({ rooms: data.data || [], isLoading: false })
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error',
        isLoading: false 
      })
    }
  },

  createRoom: async (name, type, description = '') => {
    set({ isLoading: true, error: null })
    try {
      const room = await apiClient.post<Room>('/api/v1/rooms', { name, type, description })
      set(state => ({ 
        rooms: [...state.rooms, room],
        isLoading: false 
      }))
      return room
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error',
        isLoading: false 
      })
      throw error
    }
  },

  setCurrentRoom: (room) => {
    set({ currentRoom: room })
  },

  deleteRoom: async (roomId) => {
    set({ isLoading: true, error: null })
    try {
      await apiClient.delete(`/api/v1/rooms/${roomId}`)
      
      set(state => ({
        rooms: state.rooms.filter(r => r.id !== roomId),
        currentRoom: state.currentRoom?.id === roomId ? null : state.currentRoom,
        isLoading: false,
      }))
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error',
        isLoading: false 
      })
      throw error
    }
  },
}))
