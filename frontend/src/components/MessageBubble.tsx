import type { Message } from '../store/roomsStore'

interface MessageBubbleProps {
  message: Message
  isMine: boolean
  onReplyInThread: (msg: Message) => void
  formatTime: (dateStr: string) => string
  formatFileSize: (bytes: number) => string
  getInitials: (name?: string) => string
}

export default function MessageBubble({
  message: msg,
  isMine,
  onReplyInThread,
  formatTime,
  formatFileSize,
  getInitials,
}: MessageBubbleProps) {
  const renderContent = () => {
    const meta = msg.metadata
    if (msg.type === 'image' && meta?.file_id) {
      const url = `/api/v1/files/${meta.file_id}?inline=1`
      return (
        <div className="msg-attachment">
          <a href={url} target="_blank" rel="noopener noreferrer">
            <img src={url} alt={meta.file_name || 'image'} className="msg-image" />
          </a>
          {msg.content && msg.content !== meta.file_name && (
            <div className="msg-text">{msg.content}</div>
          )}
        </div>
      )
    }
    if (msg.type === 'file' && meta?.file_id) {
      const url = `/api/v1/files/${meta.file_id}`
      return (
        <div className="msg-attachment">
          <a href={url} className="msg-file-link" download>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
              <polyline points="14 2 14 8 20 8" />
              <line x1="12" y1="18" x2="12" y2="12" />
              <polyline points="9 15 12 18 15 15" />
            </svg>
            <div className="msg-file-info">
              <span className="msg-file-name">{meta.file_name}</span>
              {meta.file_size && meta.file_size > 0 && (
                <span className="msg-file-size">{formatFileSize(meta.file_size)}</span>
              )}
            </div>
          </a>
          {msg.content && msg.content !== meta.file_name && (
            <div className="msg-text">{msg.content}</div>
          )}
        </div>
      )
    }
    return <div className="msg-text">{msg.content}</div>
  }

  return (
    <div className={`msg ${isMine ? 'msg-mine' : 'msg-other'}`}>
      {!isMine && <div className="msg-avatar">{getInitials(msg.user?.name)}</div>}
      <div className="msg-bubble">
        {!isMine && <div className="msg-author">{msg.user?.name || 'Пользователь'}</div>}
        {renderContent()}
        <div className="msg-footer">
          <span className="msg-time">{formatTime(msg.created_at)}</span>
          <button
            className="msg-thread-btn"
            onClick={() => onReplyInThread(msg)}
            title="Ответить в треде"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
            </svg>
            {(msg.thread_count ?? 0) > 0 && (
              <span className="msg-thread-count">{msg.thread_count}</span>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
