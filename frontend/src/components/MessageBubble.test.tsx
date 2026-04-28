import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import MessageBubble from './MessageBubble'
import type { Message } from '../store/roomsStore'

const baseMessage: Message = {
  id: 'msg-1',
  room_id: 'room-1',
  user_id: 'user-1',
  content: 'Hello, world!',
  type: 'text',
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  user: { id: 'user-1', name: 'Alice', email: 'alice@test' },
}

const baseProps = {
  isMine: true,
  currentUserId: 'user-1',
  canDelete: true,
  canEdit: true,
  onReplyInThread: vi.fn(),
  onReaction: vi.fn(),
  onEdit: vi.fn(),
  onDelete: vi.fn(),
  formatTime: (s: string) => new Date(s).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' }),
  formatFileSize: (b: number) => `${b} B`,
  getInitials: (name?: string) => (name ? name[0].toUpperCase() : '?'),
}

describe('MessageBubble', () => {
  it('renders message content', () => {
    render(<MessageBubble {...baseProps} message={baseMessage} />)
    expect(screen.getByText('Hello, world!')).toBeTruthy()
  })

  it('renders (ред.) marker when metadata.edited is true', () => {
    const edited: Message = { ...baseMessage, metadata: { edited: true } }
    render(<MessageBubble {...baseProps} message={edited} />)
    expect(screen.getByText(/\(ред\.\)/)).toBeTruthy()
  })

  it('does not render (ред.) marker for non-edited message', () => {
    render(<MessageBubble {...baseProps} message={baseMessage} />)
    expect(screen.queryByText(/\(ред\.\)/)).toBeNull()
  })

  it('renders "Сообщение удалено" for deleted message', () => {
    const deleted: Message = { ...baseMessage, is_deleted: true, content: '' }
    render(<MessageBubble {...baseProps} message={deleted} />)
    expect(screen.getByText('Сообщение удалено')).toBeTruthy()
  })

  it('hides actions for deleted message', () => {
    const deleted: Message = { ...baseMessage, is_deleted: true, content: '' }
    render(<MessageBubble {...baseProps} message={deleted} />)
    expect(screen.queryByTitle('Действия')).toBeNull()
    expect(screen.queryByTitle('Реакция')).toBeNull()
    expect(screen.queryByTitle('Ответить в треде')).toBeNull()
  })

  it('does not show context menu trigger when canDelete and canEdit are both false for foreign message', () => {
    render(<MessageBubble {...baseProps} message={baseMessage} isMine={false} canDelete={false} canEdit={false} />)
    expect(screen.queryByTitle('Действия')).toBeNull()
  })

  it('shows context menu trigger when canDelete=true (admin case for foreign message)', () => {
    render(<MessageBubble {...baseProps} message={baseMessage} isMine={false} canDelete={true} canEdit={false} />)
    expect(screen.getByTitle('Действия')).toBeTruthy()
  })

  it('calls onEdit when "Редактировать" clicked', () => {
    const onEdit = vi.fn()
    render(<MessageBubble {...baseProps} message={baseMessage} onEdit={onEdit} />)
    fireEvent.click(screen.getByTitle('Действия'))
    fireEvent.click(screen.getByText('Редактировать'))
    expect(onEdit).toHaveBeenCalledWith(baseMessage)
  })

  it('calls onDelete when "Удалить" clicked', () => {
    const onDelete = vi.fn()
    render(<MessageBubble {...baseProps} message={baseMessage} onDelete={onDelete} />)
    fireEvent.click(screen.getByTitle('Действия'))
    fireEvent.click(screen.getByText('Удалить'))
    expect(onDelete).toHaveBeenCalledWith(baseMessage)
  })

  it('renders @mentions with class', () => {
    const mentioned: Message = { ...baseMessage, content: 'Hi @bob and @alice' }
    render(<MessageBubble {...baseProps} message={mentioned} />)
    const mentions = document.querySelectorAll('.mention')
    expect(mentions.length).toBeGreaterThanOrEqual(2)
  })

  it('opens emoji picker on emoji button click and calls onReaction on emoji select', () => {
    const onReaction = vi.fn()
    render(<MessageBubble {...baseProps} message={baseMessage} onReaction={onReaction} />)
    fireEvent.click(screen.getByTitle('Реакция'))
    const picker = document.querySelector('.emoji-picker')
    expect(picker).toBeTruthy()
    const items = picker!.querySelectorAll('.emoji-picker-item')
    expect(items.length).toBeGreaterThan(0)
    fireEvent.click(items[0])
    expect(onReaction).toHaveBeenCalledWith('msg-1', expect.any(String))
  })

  it('keeps emoji button highlighted while picker is open (is-open class)', () => {
    render(<MessageBubble {...baseProps} message={baseMessage} />)
    const btn = screen.getByTitle('Реакция')
    expect(btn.className).not.toContain('is-open')
    fireEvent.click(btn)
    expect(btn.className).toContain('is-open')
  })

  it('renders existing reactions chips when reactions_summary is non-empty', () => {
    const withReactions: Message = {
      ...baseMessage,
      reactions_summary: [
        { emoji: '👍', count: 2, user_ids: ['user-1', 'user-2'] },
        { emoji: '🔥', count: 1, user_ids: ['user-3'] },
      ],
    }
    render(<MessageBubble {...baseProps} message={withReactions} />)
    expect(screen.getByText('👍')).toBeTruthy()
    expect(screen.getByText('🔥')).toBeTruthy()
  })
})
