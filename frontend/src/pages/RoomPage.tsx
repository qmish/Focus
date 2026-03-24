import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useRoomsStore, useRoomsStore as roomsStore } from '../store/roomsStore'
import { JitsiMeeting } from '../components/JitsiMeeting'

export default function RoomPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const navigate = useNavigate()
  const { currentRoom, setCurrentRoom, fetchRooms } = useRoomsStore()
  const [showVideo, setShowVideo] = useState(false)
  const [jitsiJWT, setJitsiJWT] = useState<string>('')
  const [messages, setMessages] = useState<any[]>([])
  const [messageInput, setMessageInput] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (roomId) {
      loadRoom(roomId)
      loadMessages(roomId)
    }
  }, [roomId])

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const loadRoom = async (id: string) => {
    try {
      const response = await fetch(`/api/v1/rooms/${id}`, {
        headers: {
          'Authorization': `Bearer ${useAuthStore.getState().token}`,
        },
      })
      
      if (!response.ok) throw new Error('Failed to load room')
      
      const room = await response.json()
      setCurrentRoom(room)

      // Получаем Jitsi JWT
      const joinResponse = await fetch(`/api/v1/rooms/${id}/join`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${useAuthStore.getState().token}`,
        },
      })
      
      if (joinResponse.ok) {
        const data = await joinResponse.json()
        setJitsiJWT(data.jitsi_jwt)
      }
    } catch (error) {
      console.error('Failed to load room:', error)
    }
  }

  const loadMessages = async (id: string) => {
    try {
      const response = await fetch(`/api/v1/messages?room_id=${id}`, {
        headers: {
          'Authorization': `Bearer ${useAuthStore.getState().token}`,
        },
      })
      
      if (!response.ok) throw new Error('Failed to load messages')
      
      const data = await response.json()
      setMessages(data.data || [])
    } catch (error) {
      console.error('Failed to load messages:', error)
    }
  }

  const sendMessage = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!messageInput.trim() || !roomId) return

    try {
      const response = await fetch('/api/v1/messages', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${useAuthStore.getState().token}`,
        },
        body: JSON.stringify({
          room_id: roomId,
          content: messageInput,
          type: 'text',
        }),
      })

      if (!response.ok) throw new Error('Failed to send message')

      const message = await response.json()
      setMessages(prev => [...prev, message])
      setMessageInput('')
    } catch (error) {
      console.error('Failed to send message:', error)
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

  if (!currentRoom) {
    return <div>Загрузка...</div>
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
        </div>
        <button onClick={handleVideoToggle} className="video-btn">
          {showVideo ? '💬 Чат' : '🎥 Видеозвонок'}
        </button>
      </div>

      <div className="room-content">
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

// Импортируем useAuthStore
import { useAuthStore } from '../store/authStore'
