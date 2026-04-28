import { useState, useEffect, useRef, useCallback } from 'react'
import { apiClient } from '../lib/apiClient'

interface MentionUser {
  id: string
  name: string
  email: string
  avatar_url?: string
}

interface MentionPopupProps {
  query: string
  roomId: string
  position: { top: number; left: number }
  onSelect: (user: MentionUser) => void
  onClose: () => void
}

export default function MentionPopup({ query, roomId, position, onSelect, onClose }: MentionPopupProps) {
  const [users, setUsers] = useState<MentionUser[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [loading, setLoading] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()
  const popupRef = useRef<HTMLDivElement>(null)

  const fetchUsers = useCallback(async (q: string) => {
    if (!q) {
      setUsers([])
      return
    }
    setLoading(true)
    try {
      const params = new URLSearchParams({ q, room_id: roomId })
      const result = await apiClient.get<MentionUser[]>(`/api/v1/users/search?${params}`)
      setUsers(result || [])
      setActiveIndex(0)
    } catch {
      setUsers([])
    } finally {
      setLoading(false)
    }
  }, [roomId])

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => fetchUsers(query), 200)
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
  }, [query, fetchUsers])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIndex(i => (i + 1) % Math.max(users.length, 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIndex(i => (i - 1 + users.length) % Math.max(users.length, 1))
      } else if (e.key === 'Enter' && users.length > 0) {
        e.preventDefault()
        onSelect(users[activeIndex])
      } else if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [users, activeIndex, onSelect, onClose])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (popupRef.current && !popupRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [onClose])

  if (!query && users.length === 0) return null

  return (
    <div
      ref={popupRef}
      className="mention-popup"
      style={{ bottom: position.top, left: position.left }}
    >
      {loading && <div className="mention-popup-loading">Поиск...</div>}
      {!loading && users.length === 0 && query && (
        <div className="mention-popup-empty">Пользователи не найдены</div>
      )}
      {users.map((u, i) => (
        <div
          key={u.id}
          className={`mention-popup-item${i === activeIndex ? ' active' : ''}`}
          onMouseDown={(e) => { e.preventDefault(); onSelect(u) }}
          onMouseEnter={() => setActiveIndex(i)}
        >
          <span className="mention-popup-avatar">
            {u.name?.charAt(0)?.toUpperCase() || '?'}
          </span>
          <div className="mention-popup-info">
            <span className="mention-popup-name">{u.name}</span>
            <span className="mention-popup-email">{u.email}</span>
          </div>
        </div>
      ))}
    </div>
  )
}
