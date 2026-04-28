import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import MessageContextMenu from './MessageContextMenu'

describe('MessageContextMenu', () => {
  const baseProps = {
    isMine: true,
    canDelete: true,
    canEdit: true,
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onReplyInThread: vi.fn(),
  }

  it('hides menu by default', () => {
    render(<MessageContextMenu {...baseProps} />)
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('opens menu on trigger click', () => {
    render(<MessageContextMenu {...baseProps} />)
    fireEvent.click(screen.getByTitle('Действия'))
    expect(screen.getByRole('menu')).toBeTruthy()
  })

  it('shows Edit/Delete/Reply for own editable message', () => {
    render(<MessageContextMenu {...baseProps} />)
    fireEvent.click(screen.getByTitle('Действия'))
    expect(screen.getByText('Редактировать')).toBeTruthy()
    expect(screen.getByText('Удалить')).toBeTruthy()
    expect(screen.getByText('Ответить в треде')).toBeTruthy()
  })

  it('hides Edit when canEdit=false (edit window expired)', () => {
    render(<MessageContextMenu {...baseProps} canEdit={false} />)
    fireEvent.click(screen.getByTitle('Действия'))
    expect(screen.queryByText('Редактировать')).toBeNull()
    expect(screen.getByText('Удалить')).toBeTruthy()
  })

  it('hides Edit when isMine=false (foreign message)', () => {
    render(<MessageContextMenu {...baseProps} isMine={false} />)
    fireEvent.click(screen.getByTitle('Действия'))
    expect(screen.queryByText('Редактировать')).toBeNull()
  })

  it('hides Delete when canDelete=false', () => {
    render(<MessageContextMenu {...baseProps} canDelete={false} />)
    fireEvent.click(screen.getByTitle('Действия'))
    expect(screen.queryByText('Удалить')).toBeNull()
  })

  it('shows Delete for foreign message when canDelete=true (admin/moderator case)', () => {
    render(<MessageContextMenu {...baseProps} isMine={false} canEdit={false} canDelete={true} />)
    fireEvent.click(screen.getByTitle('Действия'))
    expect(screen.queryByText('Редактировать')).toBeNull()
    expect(screen.getByText('Удалить')).toBeTruthy()
  })

  it('calls onEdit and closes menu', () => {
    const onEdit = vi.fn()
    render(<MessageContextMenu {...baseProps} onEdit={onEdit} />)
    fireEvent.click(screen.getByTitle('Действия'))
    fireEvent.click(screen.getByText('Редактировать'))
    expect(onEdit).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('calls onDelete and closes menu', () => {
    const onDelete = vi.fn()
    render(<MessageContextMenu {...baseProps} onDelete={onDelete} />)
    fireEvent.click(screen.getByTitle('Действия'))
    fireEvent.click(screen.getByText('Удалить'))
    expect(onDelete).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('calls onReplyInThread and closes menu', () => {
    const onReplyInThread = vi.fn()
    render(<MessageContextMenu {...baseProps} onReplyInThread={onReplyInThread} />)
    fireEvent.click(screen.getByTitle('Действия'))
    fireEvent.click(screen.getByText('Ответить в треде'))
    expect(onReplyInThread).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('closes menu on Escape', () => {
    render(<MessageContextMenu {...baseProps} />)
    fireEvent.click(screen.getByTitle('Действия'))
    expect(screen.getByRole('menu')).toBeTruthy()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
  })
})
