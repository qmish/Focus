import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useRoomsStore, type Room } from '../store/roomsStore'

export default function RoomsPage() {
  const navigate = useNavigate()
  const { rooms, isLoading, error, fetchRooms, createRoom, deleteRoom } = useRoomsStore()
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newRoomName, setNewRoomName] = useState('')
  const [newRoomType, setNewRoomType] = useState<'public' | 'private' | 'meeting'>('public')

  useEffect(() => {
    fetchRooms()
  }, [])

  const handleCreateRoom = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newRoomName.trim()) return

    try {
      const room = await createRoom(newRoomName, newRoomType)
      setNewRoomName('')
      setShowCreateModal(false)
      navigate(`/rooms/${room.id}`)
    } catch (error) {
      console.error('Failed to create room:', error)
    }
  }

  const handleDeleteRoom = async (roomId: string) => {
    if (!confirm('Вы уверены, что хотите удалить эту комнату?')) return
    
    try {
      await deleteRoom(roomId)
    } catch (error) {
      console.error('Failed to delete room:', error)
    }
  }

  const handleJoinRoom = (room: Room) => {
    navigate(`/rooms/${room.id}`)
  }

  return (
    <div className="rooms-page">
      <div className="rooms-header">
        <h2>Комнаты</h2>
        <button onClick={() => setShowCreateModal(true)} className="create-btn">
          + Создать комнату
        </button>
      </div>
      {error && <p className="error">{error}</p>}

      <div className="rooms-list">
        {isLoading ? (
          <div className="loading">Загрузка комнат...</div>
        ) : rooms.length === 0 ? (
          <div className="empty-state">
            <p>Нет комнат</p>
            <button onClick={() => setShowCreateModal(true)}>
              Создать первую комнату
            </button>
          </div>
        ) : (
          rooms.map(room => (
            <div key={room.id} className="room-card">
              <div className="room-info">
                <h3>{room.name}</h3>
                <p className="room-description">{room.description || 'Нет описания'}</p>
                <div className="room-meta">
                  <span className={`room-type room-type-${room.type}`}>
                    {room.type === 'public' ? '🌍 Публичная' : 
                     room.type === 'private' ? '🔒 Приватная' : '📅 Встреча'}
                  </span>
                  <span className="room-date">
                    {new Date(room.created_at).toLocaleDateString('ru-RU')}
                  </span>
                </div>
              </div>
              <div className="room-actions">
                <button onClick={() => handleJoinRoom(room)} className="join-btn">
                  Войти
                </button>
                <button onClick={() => handleDeleteRoom(room.id)} className="delete-btn">
                  🗑️
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>Создать комнату</h3>
            <form onSubmit={handleCreateRoom}>
              <div className="form-group">
                <label>Название</label>
                <input
                  type="text"
                  value={newRoomName}
                  onChange={e => setNewRoomName(e.target.value)}
                  placeholder="Введите название комнаты"
                  autoFocus
                />
              </div>

              <div className="form-group">
                <label>Тип</label>
                <select value={newRoomType} onChange={e => setNewRoomType(e.target.value as any)}>
                  <option value="public">🌍 Публичная</option>
                  <option value="private">🔒 Приватная</option>
                  <option value="meeting">📅 Встреча</option>
                </select>
              </div>

              <div className="modal-actions">
                <button type="button" onClick={() => setShowCreateModal(false)}>
                  Отмена
                </button>
                <button type="submit" disabled={!newRoomName.trim()}>
                  Создать
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
