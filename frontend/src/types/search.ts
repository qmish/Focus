// Типы данных для глобального и локального поиска (Telegram-style).
// Соответствуют ответам API_Go/internal/search и handlers/search_handler.go.

import type { Message, Room } from '../store/roomsStore'

export interface SearchUser {
  id: string
  name: string
  email: string
  avatar_url?: string
}

export interface MessageHit {
  message: Message
  room_id: string
  room_name: string
  highlight?: string
}

export interface FileHit {
  message_id: string
  room_id: string
  room_name: string
  file_id: string
  file_name: string
  file_mime?: string
  file_size?: number
  uploaded_at: string
  type: 'file' | 'image' | string
}

export interface MeetingHit {
  id: string
  room_id: string
  room_name: string
  subject: string
  organizer_email: string
  start_at: string
  end_at: string
  status: string
}

export interface GlobalSearchResponse {
  users: SearchUser[]
  rooms: Room[]
  messages: MessageHit[]
  files: FileHit[]
  meetings: MeetingHit[]
  took_ms: number
  query: string
}

export interface LocalMessagesResponse {
  messages: MessageHit[]
  next_before?: string
  took_ms: number
  query: string
}

export type SearchTypeKey = 'users' | 'rooms' | 'messages' | 'files' | 'meetings'

export interface FlatSearchResult {
  kind: SearchTypeKey
  id: string
  title: string
  subtitle?: string
  snippet?: string
  payload: SearchUser | Room | MessageHit | FileHit | MeetingHit
}
