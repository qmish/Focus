import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import RoomSidebar from './RoomSidebar'
import type { Room } from '../store/roomsStore'

const rooms: Room[] = [
  { id: 'r1', name: 'General', type: 'public', description: 'Общий', jitsi_room_name: 'gen', is_private: false, created_at: '', updated_at: '' },
  { id: 'r2', name: 'Devs', type: 'private', description: '', jitsi_room_name: 'dev', is_private: true, created_at: '', updated_at: '' },
  { id: 'r3', name: 'Marketing', type: 'public', description: '', jitsi_room_name: 'mark', is_private: false, created_at: '', updated_at: '' },
]

const baseProps = {
  rooms,
  activeRoomId: 'r1',
  searchQuery: '',
  onSearchChange: vi.fn(),
  onSelectRoom: vi.fn(),
  onCreateRoom: vi.fn(),
  onScheduleMeeting: vi.fn(),
  onOpenScheduledMeeting: vi.fn(),
  onRefreshScheduled: vi.fn(),
  onProfileClick: vi.fn(),
  onLogout: vi.fn(),
  onCloseMobile: vi.fn(),
  scheduledMeetings: [],
  isLoadingScheduled: false,
  user: { name: 'Test User', email: 'test@focus.local' },
  getInitials: (name?: string) => (name ? name.charAt(0) : '?'),
}

describe('RoomSidebar', () => {
  it('renders all rooms by default', () => {
    render(<RoomSidebar {...baseProps} />)
    expect(screen.getByText('General')).toBeTruthy()
    expect(screen.getByText('Devs')).toBeTruthy()
    expect(screen.getByText('Marketing')).toBeTruthy()
  })

  it('marks active room with active class', () => {
    render(<RoomSidebar {...baseProps} />)
    const generalEl = screen.getByText('General').closest('.room-item')
    expect(generalEl?.className).toContain('active')
    const devsEl = screen.getByText('Devs').closest('.room-item')
    expect(devsEl?.className).not.toContain('active')
  })

  it('filters rooms by search query', () => {
    render(<RoomSidebar {...baseProps} searchQuery="dev" />)
    expect(screen.getByText('Devs')).toBeTruthy()
    expect(screen.queryByText('General')).toBeNull()
    expect(screen.queryByText('Marketing')).toBeNull()
  })

  it('calls onSelectRoom and onCloseMobile when a room is clicked', () => {
    const onSelectRoom = vi.fn()
    const onCloseMobile = vi.fn()
    render(<RoomSidebar {...baseProps} onSelectRoom={onSelectRoom} onCloseMobile={onCloseMobile} />)
    fireEvent.click(screen.getByText('Devs').closest('.room-item')!)
    expect(onSelectRoom).toHaveBeenCalledTimes(1)
    expect(onSelectRoom.mock.calls[0][0].id).toBe('r2')
    expect(onCloseMobile).toHaveBeenCalledTimes(1)
  })

  it('calls onCreateRoom when create button clicked', () => {
    const onCreateRoom = vi.fn()
    render(<RoomSidebar {...baseProps} onCreateRoom={onCreateRoom} />)
    fireEvent.click(screen.getByTitle('Новый чат'))
    expect(onCreateRoom).toHaveBeenCalledTimes(1)
  })

  it('calls onScheduleMeeting when schedule button clicked', () => {
    const onScheduleMeeting = vi.fn()
    render(<RoomSidebar {...baseProps} onScheduleMeeting={onScheduleMeeting} />)
    fireEvent.click(screen.getByTitle('Запланировать встречу'))
    expect(onScheduleMeeting).toHaveBeenCalledTimes(1)
  })

  it('calls onLogout when logout clicked', () => {
    const onLogout = vi.fn()
    render(<RoomSidebar {...baseProps} onLogout={onLogout} />)
    fireEvent.click(screen.getByTitle('Выйти'))
    expect(onLogout).toHaveBeenCalledTimes(1)
  })

  it('applies is-open class when isMobileOpen=true', () => {
    const { container } = render(<RoomSidebar {...baseProps} isMobileOpen={true} />)
    const aside = container.querySelector('aside')
    expect(aside?.className).toContain('is-open')
  })

  it('does not apply is-open class when isMobileOpen=false', () => {
    const { container } = render(<RoomSidebar {...baseProps} isMobileOpen={false} />)
    const aside = container.querySelector('aside')
    expect(aside?.className).not.toContain('is-open')
  })

  it('shows empty state when no rooms', () => {
    render(<RoomSidebar {...baseProps} rooms={[]} />)
    expect(screen.getByText('Нет комнат')).toBeTruthy()
  })

  it('shows scheduled meetings if provided', () => {
    const scheduled = [
      { id: 's1', subject: 'Standup', start_time: '2026-04-28T10:00:00Z' },
    ]
    render(<RoomSidebar {...baseProps} scheduledMeetings={scheduled} />)
    expect(screen.getByText('Standup')).toBeTruthy()
  })
})
