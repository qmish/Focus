import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useRoomsStore, type Message, type Room } from '../store/roomsStore'
import { JitsiMeeting } from '../components/JitsiMeeting'
import { apiClient } from '../lib/apiClient'
import { buildWebSocketURL, mergeMessageList } from '../lib/roomRealtime'
import { useAuthStore } from '../store/authStore'

export default function RoomPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const navigate = useNavigate()
  const { currentRoom, setCurrentRoom } = useRoomsStore()
  const token = useAuthStore((state) => state.token)
  const refreshToken = useAuthStore((state) => state.refreshToken)
  const [showVideo, setShowVideo] = useState(false)
  const [jitsiJWT, setJitsiJWT] = useState<string>('')
  const [messages, setMessages] = useState<Message[]>([])
  const [messageInput, setMessageInput] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isRealtimeConnected, setIsRealtimeConnected] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimerRef = useRef<number | null>(null)

  useEffect(() => {
    if (roomId) {
      loadRoom(roomId)
      loadMessages(roomId)
    }
    return () => {
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current)
      }
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [roomId])

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  useEffect(() => {
    if (!roomId || !token) {
      return
    }

    const connect = () => {
      try {
        const ws = new WebSocket(buildWebSocketURL(window.location.href, token))
        wsRef.current = ws

        ws.onopen = () => {
          reconnectAttemptsRef.current = 0
          setIsRealtimeConnected(true)
          ws.send(JSON.stringify({
            type: 'subscribe',
            payload: { room_id: roomId },
          }))
        }

        ws.onmessage = (event) => {
          try {
            const wsMessage = JSON.parse(event.data) as {
              type?: string
              payload?: Message & { code?: string; message?: string }
            }
            if (wsMessage.type === 'message' && wsMessage.payload?.room_id === roomId) {
              setMessages((prev) => mergeMessageList(prev, wsMessage.payload as Message))
              return
            }
            if (wsMessage.type === 'error' && wsMessage.payload?.message) {
              setError(wsMessage.payload.message)
            }
          } catch {
            setError('Ошибка realtime-события')
          }
        }

        ws.onclose = async (event) => {
          setIsRealtimeConnected(false)
          if (!roomId) {
            return
          }
          if (event.reason === 'token_expired') {
            await refreshToken()
          }
          reconnectAttemptsRef.current += 1
          const delayMs = Math.min(5000, 500 * reconnectAttemptsRef.current)
          reconnectTimerRef.current = window.setTimeout(connect, delayMs)
        }
      } catch {
        setError('Не удалось подключить realtime')
      }
    }

    connect()

    return () => {
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current)
      }
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [roomId, token])

  const loadRoom = async (id: string) => {
    try {
      setError(null)
      const room = await apiClient.get<Room>(`/api/v1/rooms/${id}`)
      setCurrentRoom(room)

      // Получаем Jitsi JWT
      const data = await apiClient.post<{ jitsi_jwt?: string }>(`/api/v1/rooms/${id}/join`, {})
      setJitsiJWT(data.jitsi_jwt || '')
    } catch (error) {
      console.error('Failed to load room:', error)
      setError('Не удалось загрузить комнату')
    }
  }

  const loadMessages = async (id: string) => {
    try {
      const data = await apiClient.get<{ data?: Message[] }>(`/api/v1/messages?room_id=${id}`)
      setMessages(data.data || [])
    } catch (error) {
      console.error('Failed to load messages:', error)
      setError('Не удалось загрузить сообщения')
    } finally {
      setIsLoading(false)
    }
  }

  const sendMessage = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!messageInput.trim() || !roomId) return

    try {
      const message = await apiClient.post<Message>('/api/v1/messages', {
          room_id: roomId,
          content: messageInput,
          type: 'text',
      })
      setMessages(prev => [...prev, message])
      setMessageInput('')
    } catch (error) {
      console.error('Failed to send message:', error)
      setError('Не удалось отправить сообщение')
    }
  }

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const handleVideoToggle = () => {
    setShowVideo(!showVideo)
  }

  const handleJitsiLeave = () => {
    setShowVideo(false)
  }

  if (isLoading) {
    return <div>Загрузка...</div>
  }
  if (!currentRoom) {
    return <div>{error || 'Комната не найдена'}</div>
  }

  return (
    <div className="room-page">
      <div className="room-header">
        <button onClick={() => navigate('/rooms')} className="back-btn">
          ← Назад
        </button>
        <div className="room-title">
          <h2>{currentRoom.name}</h2>
          <span className={`room-type-badge room-type-${currentRoom.type}`}>
            {currentRoom.type}
          </span>
          <span style={{ marginLeft: 8, fontSize: 12, opacity: 0.8 }}>
            {isRealtimeConnected ? 'WS: online' : 'WS: reconnecting...'}
          </span>
        </div>
        <button onClick={handleVideoToggle} className="video-btn">
          {showVideo ? '💬 Чат' : '🎥 Видеозвонок'}
        </button>
      </div>

      <div className="room-content">
        {error && <p className="error">{error}</p>}
        {showVideo && jitsiJWT ? (
          <div className="video-container">
            <JitsiMeeting
              roomName={currentRoom.jitsi_room_name}
              jwt={jitsiJWT}
              onLeave={handleJitsiLeave}
            />
          </div>
        ) : (
          <div className="chat-container">
            <div className="messages">
              {messages.map(msg => (
                <div key={msg.id} className="message">
                  <div className="message-avatar">
                    {msg.user?.name?.charAt(0) || 'U'}
                  </div>
                  <div className="message-content">
                    <div className="message-header">
                      <span className="message-author">{msg.user?.name || 'Unknown'}</span>
                      <span className="message-time">
                        {new Date(msg.created_at).toLocaleTimeString('ru-RU', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                    </div>
                    <div className="message-text">{msg.content}</div>
                  </div>
                </div>
              ))}
              <div ref={messagesEndRef} />
            </div>

            <form onSubmit={sendMessage} className="message-form">
              <input
                type="text"
                value={messageInput}
                onChange={e => setMessageInput(e.target.value)}
                placeholder="Введите сообщение..."
                className="message-input"
              />
              <button type="submit" className="send-btn">
                ➤
              </button>
            </form>
          </div>
        )}
      </div>
    </div>
  )
}
