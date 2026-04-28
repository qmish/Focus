import { useState, useEffect, useRef } from 'react'
import type { Message } from '../store/roomsStore'
import { apiClient } from '../lib/apiClient'

interface ThreadPanelProps {
  rootMessage: Message
  currentUserId?: string
  onClose: () => void
  onSendReply: (content: string, threadRootId: string) => Promise<void>
  threadReplies: Message[]
  formatTime: (dateStr: string) => string
  getInitials: (name?: string) => string
}

export default function ThreadPanel({
  rootMessage,
  currentUserId,
  onClose,
  onSendReply,
  threadReplies,
  formatTime,
  getInitials,
}: ThreadPanelProps) {
  const [replyInput, setReplyInput] = useState('')
  const [sending, setSending] = useState(false)
  const [replies, setReplies] = useState<Message[]>(threadReplies)
  const [loading, setLoading] = useState(false)
  const repliesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    loadThread()
  }, [rootMessage.id])

  useEffect(() => {
    setReplies(threadReplies)
  }, [threadReplies])

  useEffect(() => {
    repliesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [replies])

  const loadThread = async () => {
    setLoading(true)
    try {
      const data = await apiClient.get<{ root: Message; replies: Message[]; total: number }>(
        `/api/v1/messages/${rootMessage.id}/thread`
      )
      setReplies(data.replies || [])
    } catch {
      /* silent */
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!replyInput.trim() || sending) return
    setSending(true)
    try {
      await onSendReply(replyInput.trim(), rootMessage.id)
      setReplyInput('')
    } catch {
      /* silent */
    } finally {
      setSending(false)
    }
  }

  return (
    <aside className="thread-panel">
      <div className="thread-panel-header">
        <h4>Тред</h4>
        <button className="icon-btn" onClick={onClose} title="Закрыть">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>

      <div className="thread-panel-root">
        <div className="msg msg-other">
          <div className="msg-avatar">{getInitials(rootMessage.user?.name)}</div>
          <div className="msg-bubble">
            <div className="msg-author">{rootMessage.user?.name || 'Пользователь'}</div>
            <div className="msg-text">{rootMessage.content}</div>
            <span className="msg-time">{formatTime(rootMessage.created_at)}</span>
          </div>
        </div>
      </div>

      <div className="thread-panel-divider">
        <span>{replies.length} {replies.length === 1 ? 'ответ' : 'ответов'}</span>
      </div>

      <div className="thread-panel-replies">
        {loading ? (
          <div className="thread-panel-loading">Загрузка...</div>
        ) : replies.length === 0 ? (
          <div className="thread-panel-empty">Нет ответов. Начните обсуждение!</div>
        ) : (
          replies.map(reply => {
            const isMine = reply.user_id === currentUserId
            return (
              <div key={reply.id} className={`msg ${isMine ? 'msg-mine' : 'msg-other'}`}>
                {!isMine && <div className="msg-avatar">{getInitials(reply.user?.name)}</div>}
                <div className="msg-bubble">
                  {!isMine && <div className="msg-author">{reply.user?.name || 'Пользователь'}</div>}
                  <div className="msg-text">{reply.content}</div>
                  <span className="msg-time">{formatTime(reply.created_at)}</span>
                </div>
              </div>
            )
          })
        )}
        <div ref={repliesEndRef} />
      </div>

      <form onSubmit={handleSubmit} className="thread-panel-input">
        <input
          type="text"
          value={replyInput}
          onChange={e => setReplyInput(e.target.value)}
          placeholder="Ответить в треде..."
          className="chat-input"
        />
        <button type="submit" className="chat-send-btn" disabled={!replyInput.trim() || sending}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z" />
          </svg>
        </button>
      </form>
    </aside>
  )
}
