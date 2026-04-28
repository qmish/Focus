interface ChatHeaderProps {
  roomName?: string
  wsConnected: boolean
  onVideoCall: () => void
  onSettings: () => void
  /** Кнопка-гамбургер для открытия sidebar на мобильном */
  onMenuClick?: () => void
  showMenu?: boolean
}

export default function ChatHeader({
  roomName,
  wsConnected,
  onVideoCall,
  onSettings,
  onMenuClick,
  showMenu = true,
}: ChatHeaderProps) {
  return (
    <div className="chat-header" data-testid="chat-header">
      {showMenu && (
        <button
          className="icon-btn chat-header-menu"
          onClick={onMenuClick}
          title="Меню"
          type="button"
          aria-label="Открыть меню комнат"
        >
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="3" y1="6" x2="21" y2="6" />
            <line x1="3" y1="12" x2="21" y2="12" />
            <line x1="3" y1="18" x2="21" y2="18" />
          </svg>
        </button>
      )}
      <div className="chat-header-info">
        <h3>{roomName || 'Загрузка...'}</h3>
        <span className="chat-header-status">
          {wsConnected ? 'онлайн' : 'подключение...'}
        </span>
      </div>
      <div className="chat-header-actions">
        <button
          className="icon-btn"
          onClick={onVideoCall}
          title="Видеозвонок"
          type="button"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polygon points="23 7 16 12 23 17 23 7" />
            <rect x="1" y="5" width="15" height="14" rx="2" ry="2" />
          </svg>
        </button>
        <button
          className="icon-btn"
          onClick={onSettings}
          title="Настройки"
          type="button"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
          </svg>
        </button>
      </div>
    </div>
  )
}
