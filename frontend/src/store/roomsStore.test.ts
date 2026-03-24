import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useRoomsStore } from './roomsStore'

// Mock fetch
global.fetch = vi.fn()

const mockRooms = [
  {
    id: 'room-1',
    name: 'Test Room 1',
    description: 'Test Description',
    type: 'public' as const,
    jitsi_room_name: 'room-1-jitsi',
    is_private: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'room-2',
    name: 'Test Room 2',
    description: '',
    type: 'private' as const,
    jitsi_room_name: 'room-2-jitsi',
    is_private: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
]

describe('RoomsStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useRoomsStore.setState({
      rooms: [],
      currentRoom: null,
      isLoading: false,
      error: null,
    })
  })

  it('should initialize with empty state', () => {
    const state = useRoomsStore.getState()
    expect(state.rooms).toEqual([])
    expect(state.currentRoom).toBeNull()
    expect(state.isLoading).toBe(false)
    expect(state.error).toBeNull()
  })

  it('should fetch rooms successfully', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: mockRooms }),
    } as any)

    const { fetchRooms } = useRoomsStore.getState()
    await fetchRooms()

    const state = useRoomsStore.getState()
    expect(state.rooms).toEqual(mockRooms)
    expect(state.isLoading).toBe(false)
    expect(global.fetch).toHaveBeenCalledWith('/api/v1/rooms', expect.any(Object))
  })

  it('should handle fetch rooms error', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: false,
    } as any)

    const { fetchRooms } = useRoomsStore.getState()
    await fetchRooms()

    const state = useRoomsStore.getState()
    expect(state.rooms).toEqual([])
    expect(state.error).toBeTruthy()
  })

  it('should create room successfully', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRooms[0],
    } as any)

    const { createRoom } = useRoomsStore.getState()
    const room = await createRoom('Test Room', 'public', 'Test Description')

    expect(room).toEqual(mockRooms[0])
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/rooms',
      expect.objectContaining({
        method: 'POST',
        headers: expect.any(Object),
        body: JSON.stringify({
          name: 'Test Room',
          type: 'public',
          description: 'Test Description',
        }),
      })
    )
  })

  it('should set current room', () => {
    const { setCurrentRoom } = useRoomsStore.getState()
    setCurrentRoom(mockRooms[0])

    const state = useRoomsStore.getState()
    expect(state.currentRoom).toEqual(mockRooms[0])
  })

  it('should delete room successfully', async () => {
    useRoomsStore.setState({ rooms: mockRooms })

    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
    } as any)

    const { deleteRoom } = useRoomsStore.getState()
    await deleteRoom('room-1')

    const state = useRoomsStore.getState()
    expect(state.rooms).toEqual([mockRooms[1]])
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/rooms/room-1',
      expect.objectContaining({
        method: 'DELETE',
      })
    )
  })
})
