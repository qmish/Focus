import { useMemo } from 'react'
import type { Room } from '../store/roomsStore'

export interface ScheduledMeetingItem {
  id: string
  subject: string
  start_time: string
  room_id?: string
}

interface RoomSidebarProps<TMeeting extends ScheduledMeetingItem = ScheduledMeetingItem> {
  rooms: Room[]
  activeRoomId?: string
  searchQuery: string
  onSearchChange: (value: string) => void
  onSelectRoom: (room: Room) => void
  onCreateRoom: () => void
  onScheduleMeeting: () => void
  onOpenScheduledMeeting: (m: TMeeting) => void
  onRefreshScheduled: () => void
  onProfileClick: () => void
  onLogout: () => void
  /** Закрыть sidebar — на мобильном после выбора комнаты */
  onCloseMobile?: () => void
  /** Открыт ли sidebar на мобильном (slide-in) */
  isMobileOpen?: boolean
  scheduledMeetings: TMeeting[]
  isLoadingScheduled: boolean
  user?: { name?: string; email?: string } | null
  getInitials: (name?: string) => string
}

export default function RoomSidebar<TMeeting extends ScheduledMeetingItem = ScheduledMeetingItem>({
  rooms,
  activeRoomId,
  searchQuery,
  onSearchChange,
  onSelectRoom,
  onCreateRoom,
  onScheduleMeeting,
  onOpenScheduledMeeting,
  onRefreshScheduled,
  onProfileClick,
  onLogout,
  onCloseMobile,
  isMobileOpen,
  scheduledMeetings,
  isLoadingScheduled,
  user,
  getInitials,
}: RoomSidebarProps<TMeeting>) {
  const filteredRooms = useMemo(
    () =>
      searchQuery
        ? rooms.filter(r => r.name.toLowerCase().includes(searchQuery.toLowerCase()))
        : rooms,
    [rooms, searchQuery]
  )

  const handleSelect = (room: Room) => {
    onSelectRoom(room)
    onCloseMobile?.()
  }

  return (
    <aside
      className={`messenger-sidebar${isMobileOpen ? ' is-open' : ''}`}
      data-testid="room-sidebar"
    >
      <div className="sidebar-top">
        <div className="sidebar-brand">
          <h1>Focus</h1>
          <div className="sidebar-actions">
            <button
              className="icon-btn"
              onClick={onScheduleMeeting}
              title="Запланировать встречу"
              type="button"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="3" y="4" width="18" height="18" rx="2" />
                <line x1="16" y1="2" x2="16" y2="6" />
                <line x1="8" y1="2" x2="8" y2="6" />
                <line x1="3" y1="10" x2="21" y2="10" />
              </svg>
            </button>
            <button
              className="icon-btn"
              onClick={onCreateRoom}
              title="Новый чат"
              type="button"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="12" y1="5" x2="12" y2="19" />
                <line x1="5" y1="12" x2="19" y2="12" />
              </svg>
            </button>
          </div>
        </div>
        <div className="sidebar-search">
          <input
            type="text"
            placeholder="Поиск..."
            value={searchQuery}
            onChange={e => onSearchChange(e.target.value)}
            aria-label="Поиск по комнатам"
          />
        </div>
      </div>

      <div className="rooms-list">
        <div className="scheduled-panel">
          <div className="scheduled-panel-header">
            <span>Запланированные</span>
            <button
              type="button"
              className="scheduled-refresh"
              onClick={onRefreshScheduled}
              title="Обновить"
            >
              ↻
            </button>
          </div>
          {isLoadingScheduled ? (
            <div className="scheduled-loading">Загрузка...</div>
          ) : scheduledMeetings.length === 0 ? (
            <div className="scheduled-empty">Нет встреч</div>
          ) : (
            scheduledMeetings.slice(0, 6).map(item => (
              <div
                key={item.id}
                className="scheduled-item"
                onClick={() => onOpenScheduledMeeting(item)}
              >
                <div className="scheduled-item-title">{item.subject}</div>
                <div className="scheduled-item-time">
                  {new Date(item.start_time).toLocaleString('ru-RU', {
                    day: '2-digit',
                    month: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </div>
              </div>
            ))
          )}
        </div>
        {filteredRooms.length === 0 ? (
          <div className="rooms-empty">
            <p>Нет комнат</p>
            <button onClick={onCreateRoom} type="button">Создать</button>
          </div>
        ) : (
          filteredRooms.map(room => (
            <div
              key={room.id}
              className={`room-item ${room.id === activeRoomId ? 'active' : ''}`}
              onClick={() => handleSelect(room)}
              role="button"
              tabIndex={0}
              onKeyDown={e => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  handleSelect(room)
                }
              }}
            >
              <div className="room-item-avatar">
                {room.type === 'public' ? '#' : room.type === 'meeting' ? '📅' : '🔒'}
              </div>
              <div className="room-item-info">
                <span className="room-item-name">{room.name}</span>
                <span className="room-item-desc">
                  {room.description ||
                    (room.type === 'public'
                      ? 'Публичная комната'
                      : room.type === 'private'
                        ? 'Приватная'
                        : 'Встреча')}
                </span>
              </div>
            </div>
          ))
        )}
      </div>

      <div className="sidebar-bottom">
        <div
          className="sidebar-user"
          onClick={onProfileClick}
          style={{ cursor: 'pointer' }}
        >
          <div className="sidebar-user-avatar">{getInitials(user?.name)}</div>
          <div className="sidebar-user-info">
            <span className="sidebar-user-name">{user?.name}</span>
            <span className="sidebar-user-email">{user?.email}</span>
          </div>
        </div>
        <button
          className="icon-btn logout-icon"
          onClick={onLogout}
          title="Выйти"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
        </button>
      </div>
    </aside>
  )
}
