import { useCallback, useRef, useState } from 'react'
import type { Message } from '../store/roomsStore'
import EmojiPicker from './EmojiPicker'
import ReactionsBar from './ReactionsBar'
import MessageContextMenu from './MessageContextMenu'

const mentionPattern = /(@\w+)/g

function renderTextWithMentions(text: string) {
  const parts = text.split(mentionPattern)
  return parts.map((part, i) =>
    mentionPattern.test(part)
      ? <span key={i} className="mention">{part}</span>
      : part
  )
}

interface MessageBubbleProps {
  message: Message
  isMine: boolean
  currentUserId?: string
  canDelete: boolean
  canEdit: boolean
  onReplyInThread: (msg: Message) => void
  onReaction: (messageId: string, emoji: string) => void
  onEdit: (msg: Message) => void
  onDelete: (msg: Message) => void
  formatTime: (dateStr: string) => string
  formatFileSize: (bytes: number) => string
  getInitials: (name?: string) => string
}

export default function MessageBubble({
  message: msg,
  isMine,
  currentUserId,
  canDelete,
  canEdit,
  onReplyInThread,
  onReaction,
  onEdit,
  onDelete,
  formatTime,
  formatFileSize,
  getInitials,
}: MessageBubbleProps) {
  const [showPicker, setShowPicker] = useState(false)
  const emojiBtnRef = useRef<HTMLButtonElement>(null)

  const closeEmojiPicker = useCallback(() => setShowPicker(false), [])

  const renderContent = () => {
    if (msg.is_deleted) {
      return <div className="msg-text msg-deleted">Сообщение удалено</div>
    }
    const meta = msg.metadata
    if (msg.type === 'image' && meta?.file_id) {
      const url = `/api/v1/files/${meta.file_id}?inline=1`
      return (
        <div className="msg-attachment">
          <a href={url} target="_blank" rel="noopener noreferrer">
            <img src={url} alt={meta.file_name || 'image'} className="msg-image" />
          </a>
          {msg.content && msg.content !== meta.file_name && (
            <div className="msg-text">{renderTextWithMentions(msg.content)}</div>
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
            <div className="msg-text">{renderTextWithMentions(msg.content)}</div>
          )}
        </div>
      )
    }
    return <div className="msg-text">{renderTextWithMentions(msg.content)}</div>
  }

  const handleEmojiSelect = (emoji: string) => {
    setShowPicker(false)
    onReaction(msg.id, emoji)
  }

  const isEdited = msg.metadata?.edited === true && !msg.is_deleted

  return (
    <div className={`msg ${isMine ? 'msg-mine' : 'msg-other'}`}>
      {!isMine && <div className="msg-avatar">{getInitials(msg.user?.name)}</div>}
      <div className="msg-bubble">
        {!isMine && <div className="msg-author">{msg.user?.name || 'Пользователь'}</div>}
        {renderContent()}
        {!msg.is_deleted && (
          <ReactionsBar
            reactions={msg.reactions_summary || []}
            currentUserId={currentUserId}
            onToggle={(emoji) => onReaction(msg.id, emoji)}
          />
        )}
        <div className="msg-footer">
          <span className="msg-time">
            {formatTime(msg.created_at)}
            {isEdited && (
              <span className="msg-edited" title="Сообщение отредактировано"> (ред.)</span>
            )}
          </span>
          {!msg.is_deleted && (
            <div className="msg-actions">
              <button
                ref={emojiBtnRef}
                className={`msg-emoji-btn${showPicker ? ' is-open' : ''}`}
                onClick={() => setShowPicker(!showPicker)}
                title="Реакция"
                type="button"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <path d="M8 14s1.5 2 4 2 4-2 4-2" />
                  <line x1="9" y1="9" x2="9.01" y2="9" />
                  <line x1="15" y1="9" x2="15.01" y2="9" />
                </svg>
              </button>
              <button
                className="msg-thread-btn"
                onClick={() => onReplyInThread(msg)}
                title="Ответить в треде"
                type="button"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
                </svg>
                {(msg.thread_count ?? 0) > 0 && (
                  <span className="msg-thread-count">{msg.thread_count}</span>
                )}
              </button>
              {(canDelete || (isMine && canEdit)) && (
                <MessageContextMenu
                  isMine={isMine}
                  canDelete={canDelete}
                  canEdit={canEdit}
                  onEdit={() => onEdit(msg)}
                  onDelete={() => onDelete(msg)}
                  onReplyInThread={() => onReplyInThread(msg)}
                />
              )}
            </div>
          )}
        </div>
        {showPicker && (
          <EmojiPicker
            onSelect={handleEmojiSelect}
            onClose={closeEmojiPicker}
            anchorRef={emojiBtnRef}
          />
        )}
      </div>
    </div>
  )
}
