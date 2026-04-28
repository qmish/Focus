import { useEffect, useState, useRef, useCallback, useMemo } from 'react'
import { useParams, useNavigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { useRoomsStore, type Room, type Message } from '../store/roomsStore'
import { apiClient } from '../lib/apiClient'
import { ApiError } from '../lib/apiClientCore'
import { buildWebSocketURL, mergeMessageList } from '../lib/roomRealtime'
import { JitsiMeeting } from '../components/JitsiMeeting'
import ProfileModal from '../components/ProfileModal'
import MessageBubble from '../components/MessageBubble'
import ThreadPanel from '../components/ThreadPanel'
import MentionPopup from '../components/MentionPopup'
import { JITSI_DOMAIN } from '../lib/config'

const EDIT_WINDOW_MS = 24 * 60 * 60 * 1000

interface ScheduledMeeting {
  id: string
  subject: string
  description?: string
  start_time: string
  end_time: string
  location?: string
  jitsi_url?: string
  room_id?: string
  sync_status?: string
  exchange_event_id?: string
}

export default function MessengerPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const navigate = useNavigate()
  const { user, token, logout } = useAuthStore()
  const { rooms, fetchRooms, createRoom, deleteRoom } = useRoomsStore()

  const [messages, setMessages] = useState<Message[]>([])
  const [messageInput, setMessageInput] = useState('')
  const [pendingFile, setPendingFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [currentRoom, setCurrentRoom] = useState<Room | null>(null)
  const [isLoadingMessages, setIsLoadingMessages] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [wsConnected, setWsConnected] = useState(false)

  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newRoomName, setNewRoomName] = useState('')
  const [newRoomType, setNewRoomType] = useState<'public' | 'private' | 'meeting'>('public')
  const [showScheduleModal, setShowScheduleModal] = useState(false)
  const [scheduledMeetings, setScheduledMeetings] = useState<ScheduledMeeting[]>([])
  const [isLoadingScheduled, setIsLoadingScheduled] = useState(false)
  const [scheduleForm, setScheduleForm] = useState({
    subject: '',
    description: '',
    start: '',
    end: '',
    attendees: '',
  })

  const [showRoomSettings, setShowRoomSettings] = useState(false)
  const [showProfileModal, setShowProfileModal] = useState(false)
  const [showVideo, setShowVideo] = useState(false)
  const [showVideoChat, setShowVideoChat] = useState(false)
  const [jitsiJWT, setJitsiJWT] = useState('')
  const [jitsiBranding, setJitsiBranding] = useState<Record<string, unknown> | null>(null)

  const [searchQuery, setSearchQuery] = useState('')
  const [activeThread, setActiveThread] = useState<Message | null>(null)
  const [threadReplies, setThreadReplies] = useState<Message[]>([])
  const [mentionQuery, setMentionQuery] = useState('')
  const [showMentionPopup, setShowMentionPopup] = useState(false)
  const [mentionCursorPos, setMentionCursorPos] = useState(0)
  const [editingMessage, setEditingMessage] = useState<Message | null>(null)
  const chatInputRef = useRef<HTMLInputElement>(null)

  const isGlobalAdmin = useMemo(
    () => Boolean(user?.roles?.includes('admin')),
    [user?.roles]
  )

  const canEditMessage = useCallback((msg: Message) => {
    if (msg.is_deleted) return false
    if (msg.user_id !== user?.id) return false
    if (msg.type !== 'text') return false
    return Date.now() - new Date(msg.created_at).getTime() < EDIT_WINDOW_MS
  }, [user?.id])

  const canDeleteMessage = useCallback((msg: Message) => {
    if (msg.is_deleted) return false
    return msg.user_id === user?.id || isGlobalAdmin
  }, [user?.id, isGlobalAdmin])

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<number | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const jitsiDomain = JITSI_DOMAIN

  const pendingFileUrl = useMemo(
    () => (pendingFile && pendingFile.type.startsWith('image/') ? URL.createObjectURL(pendingFile) : ''),
    [pendingFile]
  )
  useEffect(() => {
    return () => { if (pendingFileUrl) URL.revokeObjectURL(pendingFileUrl) }
  }, [pendingFileUrl])

  useEffect(() => {
    fetchRooms()
    fetchScheduledMeetings()
    if ('__TAURI__' in window || '__TAURI_INTERNALS__' in window) {
      import('@tauri-apps/api/event').then(({ listen }) => {
        listen<string>('navigate-to-room', (event) => {
          if (event.payload) navigate(`/rooms/${event.payload}`)
        })
      })
    }
  }, [])

  useEffect(() => {
    if (roomId) {
      loadRoom(roomId)
      loadMessages(roomId)
    } else {
      setCurrentRoom(null)
      setMessages([])
      setShowVideo(false)
      setActiveThread(null)
    }
  }, [roomId])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // WebSocket connection
  useEffect(() => {
    if (!roomId || !token) return

    const connect = () => {
      try {
        const ws = new WebSocket(buildWebSocketURL(window.location.href))
        wsRef.current = ws

        ws.onopen = () => {
          const wasReconnect = reconnectAttemptsRef.current > 0
          reconnectAttemptsRef.current = 0
          setWsConnected(true)
          ws.send(JSON.stringify({ type: 'auth', payload: { token } }))
          ws.send(JSON.stringify({ type: 'subscribe', payload: { room_id: roomId } }))
          if (wasReconnect) {
            refreshMessages(roomId, 500)
          }
        }

        ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data)
            if (data.type === 'message' && data.payload?.room_id === roomId) {
              setMessages(prev => mergeMessageList(prev, data.payload))
              if (('__TAURI__' in window || '__TAURI_INTERNALS__' in window) && !document.hasFocus() && data.payload.user_id !== user?.id) {
                import('@tauri-apps/api/core').then(({ invoke }) => {
                  invoke('show_notification', {
                    title: data.payload.user?.name || 'Новое сообщение',
                    body: data.payload.content?.substring(0, 100) || '',
                    roomId: data.payload.room_id,
                  }).catch(() => {})
                })
              }
            }
            if (data.type === 'thread_reply' && data.payload?.room_id === roomId) {
              const threadRootId = data.payload.thread_root_id
              setMessages(prev =>
                prev.map(m => m.id === threadRootId ? { ...m, thread_count: (m.thread_count ?? 0) + 1 } : m)
              )
              setActiveThread(prev => {
                if (prev && prev.id === threadRootId) {
                  setThreadReplies(r => mergeMessageList(r, data.payload))
                }
                return prev
              })
            }
            if (data.type === 'mention' && data.payload) {
              const p = data.payload
              if (Notification.permission === 'granted') {
                new Notification('Вас упомянули', {
                  body: p.content_preview || 'Новое упоминание',
                  tag: `mention-${p.message_id}`,
                })
              } else if (Notification.permission !== 'denied') {
                Notification.requestPermission()
              }
            }
            if (data.type === 'reaction_added' && data.payload) {
              const { message_id, user_id: ruid, emoji } = data.payload
              setMessages(prev => prev.map(m => {
                if (m.id !== message_id) return m
                const summary = [...(m.reactions_summary || [])]
                const idx = summary.findIndex(s => s.emoji === emoji)
                if (idx >= 0) {
                  if (!summary[idx].user_ids.includes(ruid)) {
                    summary[idx] = { ...summary[idx], count: summary[idx].count + 1, user_ids: [...summary[idx].user_ids, ruid] }
                  }
                } else {
                  summary.push({ emoji, count: 1, user_ids: [ruid] })
                }
                return { ...m, reactions_summary: summary }
              }))
            }
            if (data.type === 'reaction_removed' && data.payload) {
              const { message_id, user_id: ruid, emoji } = data.payload
              setMessages(prev => prev.map(m => {
                if (m.id !== message_id) return m
                const summary = (m.reactions_summary || [])
                  .map(s => s.emoji !== emoji ? s : { ...s, count: s.count - 1, user_ids: s.user_ids.filter(id => id !== ruid) })
                  .filter(s => s.count > 0)
                return { ...m, reactions_summary: summary }
              }))
            }
            if (data.type === 'message_updated' && data.payload) {
              const updated: Message = data.payload
              if (updated.room_id !== roomId) return
              setMessages(prev => prev.map(m => m.id === updated.id ? { ...m, ...updated } : m))
              setThreadReplies(prev => prev.map(m => m.id === updated.id ? { ...m, ...updated } : m))
              setActiveThread(prev =>
                prev && prev.id === updated.id ? { ...prev, ...updated } : prev
              )
            }
            if (data.type === 'message_deleted' && data.payload) {
              const { message_id, room_id: rid, thread_root_id } = data.payload
              if (rid !== roomId) return
              setMessages(prev =>
                prev.map(m => {
                  if (m.id === message_id) {
                    return { ...m, is_deleted: true, content: '' }
                  }
                  if (thread_root_id && m.id === thread_root_id) {
                    return { ...m, thread_count: Math.max(0, (m.thread_count ?? 1) - 1) }
                  }
                  return m
                })
              )
              setThreadReplies(prev =>
                prev.map(m => m.id === message_id ? { ...m, is_deleted: true, content: '' } : m)
              )
            }
          } catch (err) { console.error('WS parse error:', err) }
        }

        ws.onerror = () => {
          setWsConnected(false)
        }

        ws.onclose = () => {
          setWsConnected(false)
          if (!roomId) return
          reconnectAttemptsRef.current += 1
          const delay = Math.min(5000, 500 * reconnectAttemptsRef.current)
          reconnectTimerRef.current = window.setTimeout(connect, delay)
        }
      } catch (err) { console.error('MessengerPage:', err) }
    }

    connect()

    return () => {
      if (reconnectTimerRef.current !== null) window.clearTimeout(reconnectTimerRef.current)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [roomId, token])

  const loadRoom = useCallback(async (id: string) => {
    try {
      setError(null)
      const room = await apiClient.get<Room>(`/api/v1/rooms/${id}`)
      setCurrentRoom(room)
      const data = await apiClient.post<{ jitsi_jwt?: string }>(`/api/v1/rooms/${id}/join`, {})
      setJitsiJWT(data.jitsi_jwt || '')
      try {
        const branding = await apiClient.get<Record<string, unknown>>('/api/v1/branding/jitsi')
        setJitsiBranding(branding)
      } catch (err) { console.error('MessengerPage:', err) }
    } catch {
      setError('Не удалось загрузить комнату')
    }
  }, [])

  const loadMessages = async (id: string) => {
    setIsLoadingMessages(true)
    try {
      const data = await apiClient.get<{ data?: Message[] }>(`/api/v1/messages?room_id=${id}`)
      setMessages(data.data || [])
    } catch {
      setError('Не удалось загрузить сообщения')
    } finally {
      setIsLoadingMessages(false)
    }
  }

  const fetchScheduledMeetings = async () => {
    setIsLoadingScheduled(true)
    try {
      const now = new Date()
      const from = new Date(now.getTime() - 6 * 60 * 60 * 1000).toISOString()
      const to = new Date(now.getTime() + 14 * 24 * 60 * 60 * 1000).toISOString()
      const data = await apiClient.get<{ data?: ScheduledMeeting[] }>(`/api/v1/calendar/events?start=${encodeURIComponent(from)}&end=${encodeURIComponent(to)}`)
      const meetings = (data.data || []).sort((a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime())
      setScheduledMeetings(meetings)
    } catch {
      // optional panel; do not break chat if calendar unavailable
      setScheduledMeetings([])
    } finally {
      setIsLoadingScheduled(false)
    }
  }

  const scheduleMeeting = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!scheduleForm.subject.trim() || !scheduleForm.start || !scheduleForm.end) return
    try {
      const attendeeEmails = scheduleForm.attendees
        .split(/[,\n;]/)
        .map(v => v.trim())
        .filter(Boolean)
      const payload = {
        subject: scheduleForm.subject.trim(),
        description: scheduleForm.description.trim(),
        start_time: new Date(scheduleForm.start).toISOString(),
        end_time: new Date(scheduleForm.end).toISOString(),
        attendee_emails: attendeeEmails,
        create_jitsi_room: true,
      }
      const idempotencyKey = `calendar-${crypto.randomUUID()}`
      const created = await apiClient.post<{ room_id?: string }>('/api/v1/calendar/events', payload, {
        'Idempotency-Key': idempotencyKey,
      })
      setShowScheduleModal(false)
      setScheduleForm({ subject: '', description: '', start: '', end: '', attendees: '' })
      await Promise.all([fetchRooms(), fetchScheduledMeetings()])
      if (created.room_id) {
        navigate(`/rooms/${created.room_id}`)
      }
    } catch {
      setError('Не удалось запланировать встречу в Exchange')
    }
  }

  const refreshMessages = useCallback((id: string, delayMs: number) => {
    setTimeout(async () => {
      try {
        const data = await apiClient.get<{ data?: Message[] }>(`/api/v1/messages?room_id=${id}`)
        if (data.data) {
          setMessages(data.data)
        }
      } catch { /* silent */ }
    }, delayMs)
  }, [])

  const openThread = useCallback((msg: Message) => {
    setActiveThread(msg)
    setThreadReplies([])
  }, [])

  const sendThreadReply = useCallback(async (content: string, threadRootId: string) => {
    if (!roomId) return
    await apiClient.post<Message>('/api/v1/messages', {
      room_id: roomId,
      content,
      type: 'text',
      thread_root_id: threadRootId,
    })
  }, [roomId])

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    const cursorPos = e.target.selectionStart ?? val.length
    setMessageInput(val)

    const textBefore = val.slice(0, cursorPos)
    const atMatch = textBefore.match(/@(\w*)$/)
    if (atMatch) {
      setMentionQuery(atMatch[1])
      setMentionCursorPos(cursorPos)
      setShowMentionPopup(true)
    } else {
      setShowMentionPopup(false)
      setMentionQuery('')
    }
  }, [])

  const handleMentionSelect = useCallback((u: { name: string }) => {
    const textBefore = messageInput.slice(0, mentionCursorPos)
    const atIdx = textBefore.lastIndexOf('@')
    const before = messageInput.slice(0, atIdx)
    const after = messageInput.slice(mentionCursorPos)
    const newVal = `${before}@${u.name} ${after}`
    setMessageInput(newVal)
    setShowMentionPopup(false)
    setMentionQuery('')
    chatInputRef.current?.focus()
  }, [messageInput, mentionCursorPos])

  const startEdit = useCallback((msg: Message) => {
    setEditingMessage(msg)
    setMessageInput(msg.content)
    setError(null)
    setTimeout(() => chatInputRef.current?.focus(), 0)
  }, [])

  const cancelEdit = useCallback(() => {
    setEditingMessage(null)
    setMessageInput('')
    setError(null)
  }, [])

  const handleDeleteMessage = useCallback(async (msg: Message) => {
    if (!confirm('Удалить сообщение?')) return
    try {
      await apiClient.delete(`/api/v1/messages/${msg.id}`)
      setMessages(prev => prev.map(m => m.id === msg.id ? { ...m, is_deleted: true, content: '' } : m))
      if (editingMessage?.id === msg.id) cancelEdit()
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setError('Доступ запрещён: нельзя удалить это сообщение')
      } else {
        setError('Не удалось удалить сообщение')
      }
    }
  }, [editingMessage?.id, cancelEdit])

  const handleReaction = useCallback(async (messageId: string, emoji: string) => {
    const uid = user?.id || ''
    const msg = messages.find(m => m.id === messageId)
    const existing = msg?.reactions_summary?.find(r => r.emoji === emoji)
    const alreadyReacted = uid ? existing?.user_ids?.includes(uid) : false
    const prevSummary = msg?.reactions_summary

    const nextSummary = (() => {
      const base = [...(msg?.reactions_summary || [])]
      if (alreadyReacted) {
        return base
          .map(s =>
            s.emoji !== emoji
              ? s
              : { ...s, count: s.count - 1, user_ids: s.user_ids.filter(id => id !== uid) }
          )
          .filter(s => s.count > 0)
      }
      const idx = base.findIndex(s => s.emoji === emoji)
      if (idx >= 0) {
        const s = base[idx]
        if (s.user_ids.includes(uid)) return base
        const copy = [...base]
        copy[idx] = { ...s, count: s.count + 1, user_ids: [...s.user_ids, uid] }
        return copy
      }
      if (!uid) return base
      return [...base, { emoji, count: 1, user_ids: [uid] }]
    })()

    setMessages(prev => prev.map(m => (m.id === messageId ? { ...m, reactions_summary: nextSummary } : m)))

    try {
      if (alreadyReacted) {
        await apiClient.delete(`/api/v1/messages/${messageId}/reactions/${encodeURIComponent(emoji)}`)
      } else {
        await apiClient.post(`/api/v1/messages/${messageId}/reactions`, { emoji })
      }
    } catch {
      setMessages(prev =>
        prev.map(m =>
          m.id === messageId
            ? { ...m, reactions_summary: prevSummary ? [...prevSummary] : [] }
            : m
        )
      )
      setError('Не удалось обновить реакцию')
    }
  }, [messages, user?.id])

  const sendMessage = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!roomId) return
    if (!messageInput.trim() && !pendingFile) return

    if (editingMessage) {
      const content = messageInput.trim()
      if (!content) return
      const editingId = editingMessage.id
      try {
        const updated = await apiClient.put<Message>(`/api/v1/messages/${editingId}`, { content })
        setMessages(prev => prev.map(m => m.id === editingId ? { ...m, ...updated } : m))
        setEditingMessage(null)
        setMessageInput('')
        setError(null)
      } catch (err) {
        if (err instanceof ApiError && err.status === 410) {
          setError('Истёк срок редактирования сообщения (24 часа)')
        } else if (err instanceof ApiError && err.status === 403) {
          setError('Доступ запрещён: нельзя редактировать чужое сообщение')
        } else {
          setError('Не удалось обновить сообщение')
        }
      }
      return
    }

    if (pendingFile) {
      await sendFileMessage(pendingFile, messageInput.trim())
      return
    }

    const content = messageInput
    setMessageInput('')

    try {
      const created = await apiClient.post<Message>('/api/v1/messages', {
        room_id: roomId,
        content,
        type: 'text',
      })
      if (created?.id) {
        setMessages(prev => mergeMessageList(prev, created))
      }
      if (content.startsWith('/')) {
        refreshMessages(roomId, 1200)
        refreshMessages(roomId, 3000)
      }
    } catch {
      setError('Не удалось отправить сообщение')
      setMessageInput(content)
    }
  }, [roomId, messageInput, pendingFile, refreshMessages, editingMessage])

  const sendFileMessage = async (file: File, caption: string) => {
    if (!roomId) return
    setUploading(true)
    try {
      const upload = await apiClient.uploadFile<{
        file_id: string; file_name: string; file_size: number; file_mime: string; url: string
      }>('/api/v1/files/upload', file)

      const isImage = file.type.startsWith('image/')
      const created = await apiClient.post<Message>('/api/v1/messages', {
        room_id: roomId,
        content: caption || upload.file_name,
        type: isImage ? 'image' : 'file',
        metadata: {
          file_id: upload.file_id,
          file_name: upload.file_name,
          file_size: upload.file_size,
          file_mime: upload.file_mime,
        },
      })
      if (created?.id) {
        setMessages(prev => mergeMessageList(prev, created))
      }
      setPendingFile(null)
      setMessageInput('')
    } catch {
      setError('Не удалось загрузить файл')
    } finally {
      setUploading(false)
    }
  }

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) setPendingFile(file)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const handleCreateRoom = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newRoomName.trim()) return
    try {
      const room = await createRoom(newRoomName, newRoomType)
      setNewRoomName('')
      setNewRoomType('public')
      setShowCreateModal(false)
      navigate(`/rooms/${room.id}`)
    } catch (err) { console.error('MessengerPage:', err) }
  }, [newRoomName, newRoomType, createRoom, navigate])

  const openScheduledMeeting = (meeting: ScheduledMeeting) => {
    if (meeting.room_id) {
      navigate(`/rooms/${meeting.room_id}`)
      return
    }
    if (meeting.jitsi_url) {
      window.open(meeting.jitsi_url, '_blank', 'noopener,noreferrer')
    }
  }

  const handleDeleteRoom = async (id: string) => {
    if (!confirm('Удалить комнату?')) return
    try {
      await deleteRoom(id)
      if (roomId === id) navigate('/rooms')
    } catch (err) { console.error('MessengerPage:', err) }
  }

  const selectRoom = useCallback((room: Room) => {
    setShowVideo(false)
    navigate(`/rooms/${room.id}`)
  }, [navigate])

  const filteredRooms = useMemo(() =>
    searchQuery
      ? rooms.filter(r => r.name.toLowerCase().includes(searchQuery.toLowerCase()))
      : rooms,
    [rooms, searchQuery]
  )

  const formatTime = (dateStr: string) => {
    return new Date(dateStr).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  }

  const getInitials = (name?: string) => {
    if (!name) return '?'
    return name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
  }

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  if (showVideo && roomId && jitsiJWT) {
    return (
      <div className="video-fullscreen">
        <div className="video-fullscreen-header">
          <div className="video-fullscreen-info">
            <h3>{currentRoom?.name}</h3>
            <span className="video-fullscreen-badge">Видеозвонок</span>
          </div>
          <div className="video-fullscreen-actions">
            <button
              className={`video-header-btn ${showVideoChat ? 'active' : ''}`}
              onClick={() => setShowVideoChat(v => !v)}
              title="Чат"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"/></svg>
              <span>Чат</span>
            </button>
            <button
              className="video-header-btn video-header-btn-end"
              onClick={() => { setShowVideo(false); setShowVideoChat(false) }}
              title="Завершить"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              <span>Завершить</span>
            </button>
          </div>
        </div>

        <div className="video-fullscreen-body">
          <div className="video-fullscreen-main">
            <JitsiMeeting
              domain={jitsiDomain}
              branding={jitsiBranding ?? undefined}
              roomName={currentRoom?.jitsi_room_name || ''}
              jwt={jitsiJWT}
              userName={user?.name}
              userEmail={user?.email}
              onLeave={() => { setShowVideo(false); setShowVideoChat(false) }}
            />
          </div>

          {showVideoChat && (
            <div className="video-chat-panel">
              <div className="video-chat-messages">
                {messages.length === 0 ? (
                  <div className="chat-no-messages"><p>Нет сообщений</p></div>
                ) : (
                  messages.map(msg => {
                    const isMine = msg.user_id === user?.id
                    const isEdited = msg.metadata?.edited === true && !msg.is_deleted
                    return (
                      <div key={msg.id} className={`msg ${isMine ? 'msg-mine' : 'msg-other'}`}>
                        {!isMine && <div className="msg-avatar">{getInitials(msg.user?.name)}</div>}
                        <div className="msg-bubble">
                          {!isMine && <div className="msg-author">{msg.user?.name || 'Пользователь'}</div>}
                          {msg.is_deleted
                            ? <div className="msg-text msg-deleted">Сообщение удалено</div>
                            : <div className="msg-text">{msg.content}</div>}
                          <div className="msg-time">
                            {formatTime(msg.created_at)}
                            {isEdited && <span className="msg-edited"> (ред.)</span>}
                          </div>
                        </div>
                      </div>
                    )
                  })
                )}
                <div ref={messagesEndRef} />
              </div>
              <form onSubmit={sendMessage} className="chat-input-area">
                <input
                  type="text"
                  value={messageInput}
                  onChange={e => setMessageInput(e.target.value)}
                  placeholder="Сообщение..."
                  className="chat-input"
                />
                <button type="submit" className="chat-send-btn" disabled={!messageInput.trim()}>
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>
                </button>
              </form>
            </div>
          )}
        </div>

        {/* Modals still need to be accessible */}
        {showCreateModal && renderCreateModal()}
        {showRoomSettings && currentRoom && renderSettingsModal()}
        <Outlet />
      </div>
    )
  }

  return (
    <div className="messenger">
      {/* Left sidebar - rooms list */}
      <aside className="messenger-sidebar">
        <div className="sidebar-top">
          <div className="sidebar-brand">
            <h1>Focus</h1>
            <div className="sidebar-actions">
              <button className="icon-btn" onClick={() => setShowScheduleModal(true)} title="Запланировать встречу">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
              </button>
              <button className="icon-btn" onClick={() => setShowCreateModal(true)} title="Новый чат">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              </button>
            </div>
          </div>
          <div className="sidebar-search">
            <input
              type="text"
              placeholder="Поиск..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
            />
          </div>
        </div>

        <div className="rooms-list">
          <div className="scheduled-panel">
            <div className="scheduled-panel-header">
              <span>Запланированные</span>
              <button type="button" className="scheduled-refresh" onClick={fetchScheduledMeetings} title="Обновить">↻</button>
            </div>
            {isLoadingScheduled ? (
              <div className="scheduled-loading">Загрузка...</div>
            ) : scheduledMeetings.length === 0 ? (
              <div className="scheduled-empty">Нет встреч</div>
            ) : (
              scheduledMeetings.slice(0, 6).map(item => (
                <div key={item.id} className="scheduled-item" onClick={() => openScheduledMeeting(item)}>
                  <div className="scheduled-item-title">{item.subject}</div>
                  <div className="scheduled-item-time">
                    {new Date(item.start_time).toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}
                  </div>
                </div>
              ))
            )}
          </div>
          {filteredRooms.length === 0 ? (
            <div className="rooms-empty">
              <p>Нет комнат</p>
              <button onClick={() => setShowCreateModal(true)}>Создать</button>
            </div>
          ) : (
            filteredRooms.map(room => (
              <div
                key={room.id}
                className={`room-item ${room.id === roomId ? 'active' : ''}`}
                onClick={() => selectRoom(room)}
              >
                <div className="room-item-avatar">
                  {room.type === 'public' ? '#' : room.type === 'meeting' ? '📅' : '🔒'}
                </div>
                <div className="room-item-info">
                  <span className="room-item-name">{room.name}</span>
                  <span className="room-item-desc">{room.description || (room.type === 'public' ? 'Публичная комната' : room.type === 'private' ? 'Приватная' : 'Встреча')}</span>
                </div>
              </div>
            ))
          )}
        </div>

        <div className="sidebar-bottom">
          <div className="sidebar-user" onClick={() => setShowProfileModal(true)} style={{ cursor: 'pointer' }}>
            <div className="sidebar-user-avatar">{getInitials(user?.name)}</div>
            <div className="sidebar-user-info">
              <span className="sidebar-user-name">{user?.name}</span>
              <span className="sidebar-user-email">{user?.email}</span>
            </div>
          </div>
          <button className="icon-btn logout-icon" onClick={logout} title="Выйти">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
          </button>
        </div>
      </aside>

      {/* Right panel - chat or empty state */}
      <main className="messenger-main">
        {!roomId ? (
          <div className="chat-empty">
            <div className="chat-empty-icon">💬</div>
            <h2>Выберите чат</h2>
            <p>Выберите комнату из списка слева или создайте новую</p>
            <button onClick={() => setShowCreateModal(true)} className="btn-primary">Создать комнату</button>
            <button onClick={() => setShowScheduleModal(true)} className="btn-secondary">Запланировать встречу</button>
          </div>
        ) : (
          <>
            <div className="chat-header">
              <div className="chat-header-info">
                <h3>{currentRoom?.name || 'Загрузка...'}</h3>
                <span className="chat-header-status">
                  {wsConnected ? 'онлайн' : 'подключение...'}
                </span>
              </div>
              <div className="chat-header-actions">
                <button className="icon-btn" onClick={() => setShowVideo(true)} title="Видеозвонок">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>
                </button>
                <button className="icon-btn" onClick={() => setShowRoomSettings(true)} title="Настройки">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>
                </button>
              </div>
            </div>

            <div className="chat-messages">
              {isLoadingMessages ? (
                <div className="chat-loading">Загрузка сообщений...</div>
              ) : messages.length === 0 ? (
                <div className="chat-no-messages">
                  <p>Нет сообщений. Начните диалог!</p>
                </div>
              ) : (
                messages.map(msg => (
                  <MessageBubble
                    key={msg.id}
                    message={msg}
                    isMine={msg.user_id === user?.id}
                    currentUserId={user?.id}
                    canDelete={canDeleteMessage(msg)}
                    canEdit={canEditMessage(msg)}
                    onReplyInThread={openThread}
                    onReaction={handleReaction}
                    onEdit={startEdit}
                    onDelete={handleDeleteMessage}
                    formatTime={formatTime}
                    formatFileSize={formatFileSize}
                    getInitials={getInitials}
                  />
                ))
              )}
              <div ref={messagesEndRef} />
            </div>

            {error && <div className="chat-error">{error}</div>}

            {pendingFile && (
              <div className="chat-file-preview">
                {pendingFileUrl
                  ? <img src={pendingFileUrl} alt="" className="file-preview-thumb" />
                  : <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
                }
                <span className="file-preview-name">{pendingFile.name}</span>
                <span className="file-preview-size">{formatFileSize(pendingFile.size)}</span>
                <button className="file-preview-remove" onClick={() => setPendingFile(null)} type="button">&times;</button>
              </div>
            )}
            {editingMessage && (
              <div className="chat-edit-banner" role="status">
                <span className="chat-edit-banner-text">
                  Редактирование сообщения
                </span>
                <button type="button" className="chat-edit-banner-cancel" onClick={cancelEdit}>
                  Отмена
                </button>
              </div>
            )}
            <form onSubmit={sendMessage} className="chat-input-area">
              <input
                type="file"
                ref={fileInputRef}
                onChange={handleFileSelect}
                className="chat-file-input-hidden"
              />
              <button
                type="button"
                className="chat-attach-btn"
                onClick={() => fileInputRef.current?.click()}
                title="Прикрепить файл"
              >
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"/></svg>
              </button>
              <div className="chat-input-wrapper">
                {showMentionPopup && roomId && (
                  <MentionPopup
                    query={mentionQuery}
                    roomId={roomId}
                    position={{ top: 8, left: 0 }}
                    onSelect={handleMentionSelect}
                    onClose={() => setShowMentionPopup(false)}
                  />
                )}
                <input
                  ref={chatInputRef}
                  type="text"
                  value={messageInput}
                  onChange={handleInputChange}
                  placeholder={editingMessage ? 'Редактируйте сообщение...' : pendingFile ? 'Добавьте подпись...' : 'Введите сообщение...'}
                  className="chat-input"
                />
              </div>
              <button type="submit" className="chat-send-btn" disabled={(!messageInput.trim() && !pendingFile) || uploading}>
                {uploading
                  ? <span className="chat-send-spinner" />
                  : <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>
                }
              </button>
            </form>
          </>
        )}
      </main>

      {activeThread && roomId && (
        <ThreadPanel
          rootMessage={activeThread}
          currentUserId={user?.id}
          onClose={() => setActiveThread(null)}
          onSendReply={sendThreadReply}
          threadReplies={threadReplies}
          formatTime={formatTime}
          getInitials={getInitials}
        />
      )}

      {showCreateModal && renderCreateModal()}
      {showScheduleModal && renderScheduleModal()}
      {showRoomSettings && currentRoom && renderSettingsModal()}

      <ProfileModal
        open={showProfileModal}
        onClose={() => setShowProfileModal(false)}
        user={user}
        onSave={(updated) => {
          useAuthStore.setState({
            user: {
              ...user,
              id: updated.id,
              email: updated.email,
              name: updated.name,
              roles: updated.roles || user?.roles || [],
              department: updated.department,
              directorate: updated.directorate,
              position: updated.position,
              phone: updated.phone,
              about_me: updated.about_me,
              video_start_with_audio_muted: updated.video_start_with_audio_muted,
              video_start_with_video_muted: updated.video_start_with_video_muted,
              video_display_name: updated.video_display_name,
              video_default_language: updated.video_default_language,
            },
          })
        }}
      />

      <Outlet />
    </div>
  )

  function renderCreateModal() {
    return (
      <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
        <div className="modal" role="dialog" aria-modal="true" onClick={e => e.stopPropagation()}>
          <div className="modal-header">
            <h3>Создать комнату</h3>
            <button className="icon-btn" onClick={() => setShowCreateModal(false)}>✕</button>
          </div>
          <form onSubmit={handleCreateRoom}>
            <div className="form-group">
              <label>Название</label>
              <input type="text" value={newRoomName} onChange={e => setNewRoomName(e.target.value)} placeholder="Название комнаты" autoFocus />
            </div>
            <div className="form-group">
              <label>Тип</label>
              <select value={newRoomType} onChange={e => setNewRoomType(e.target.value as any)}>
                <option value="public">Публичная</option>
                <option value="private">Приватная</option>
                <option value="meeting">Встреча</option>
              </select>
            </div>
            <div className="modal-actions">
              <button type="button" className="btn-secondary" onClick={() => setShowCreateModal(false)}>Отмена</button>
              <button type="submit" className="btn-primary" disabled={!newRoomName.trim()}>Создать</button>
            </div>
          </form>
        </div>
      </div>
    )
  }

  function renderScheduleModal() {
    return (
      <div className="modal-overlay" onClick={() => setShowScheduleModal(false)}>
        <div className="modal" role="dialog" aria-modal="true" onClick={e => e.stopPropagation()}>
          <div className="modal-header">
            <h3>Запланировать встречу</h3>
            <button className="icon-btn" onClick={() => setShowScheduleModal(false)}>✕</button>
          </div>
          <form onSubmit={scheduleMeeting}>
            <div className="form-group">
              <label>Тема</label>
              <input
                type="text"
                value={scheduleForm.subject}
                onChange={e => setScheduleForm(prev => ({ ...prev, subject: e.target.value }))}
                placeholder="Планёрка команды"
                required
                autoFocus
              />
            </div>
            <div className="form-group">
              <label>Описание</label>
              <input
                type="text"
                value={scheduleForm.description}
                onChange={e => setScheduleForm(prev => ({ ...prev, description: e.target.value }))}
                placeholder="Повестка встречи"
              />
            </div>
            <div className="form-group">
              <label>Начало</label>
              <input
                type="datetime-local"
                value={scheduleForm.start}
                onChange={e => setScheduleForm(prev => ({ ...prev, start: e.target.value }))}
                required
              />
            </div>
            <div className="form-group">
              <label>Окончание</label>
              <input
                type="datetime-local"
                value={scheduleForm.end}
                onChange={e => setScheduleForm(prev => ({ ...prev, end: e.target.value }))}
                required
              />
            </div>
            <div className="form-group">
              <label>Участники (email через запятую)</label>
              <input
                type="text"
                value={scheduleForm.attendees}
                onChange={e => setScheduleForm(prev => ({ ...prev, attendees: e.target.value }))}
                placeholder="user1@company.ru, user2@company.ru"
              />
            </div>
            <div className="modal-actions">
              <button type="button" className="btn-secondary" onClick={() => setShowScheduleModal(false)}>Отмена</button>
              <button type="submit" className="btn-primary">Создать в Exchange</button>
            </div>
          </form>
        </div>
      </div>
    )
  }

  function renderSettingsModal() {
    if (!currentRoom) return null
    return (
      <div className="modal-overlay" onClick={() => setShowRoomSettings(false)}>
        <div className="modal" role="dialog" aria-modal="true" onClick={e => e.stopPropagation()}>
          <div className="modal-header">
            <h3>Настройки комнаты</h3>
            <button className="icon-btn" onClick={() => setShowRoomSettings(false)}>✕</button>
          </div>
          <div className="settings-info">
            <div className="form-group">
              <label>Название</label>
              <p>{currentRoom.name}</p>
            </div>
            <div className="form-group">
              <label>Тип</label>
              <p>{currentRoom.type === 'public' ? 'Публичная' : currentRoom.type === 'private' ? 'Приватная' : 'Встреча'}</p>
            </div>
            <div className="form-group">
              <label>Создана</label>
              <p>{new Date(currentRoom.created_at).toLocaleDateString('ru-RU')}</p>
            </div>
          </div>
          <div className="modal-actions">
            <button className="btn-danger" onClick={() => { handleDeleteRoom(currentRoom.id); setShowRoomSettings(false) }}>Удалить комнату</button>
            <button className="btn-secondary" onClick={() => setShowRoomSettings(false)}>Закрыть</button>
          </div>
        </div>
      </div>
    )
  }
}
