import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import ChatHeader from './ChatHeader'

const baseProps = {
  roomName: 'General',
  wsConnected: true,
  onVideoCall: vi.fn(),
  onSettings: vi.fn(),
  onMenuClick: vi.fn(),
}

describe('ChatHeader', () => {
  it('renders room name', () => {
    render(<ChatHeader {...baseProps} />)
    expect(screen.getByText('General')).toBeTruthy()
  })

  it('shows online status when wsConnected=true', () => {
    render(<ChatHeader {...baseProps} />)
    expect(screen.getByText('онлайн')).toBeTruthy()
  })

  it('shows connecting status when wsConnected=false', () => {
    render(<ChatHeader {...baseProps} wsConnected={false} />)
    expect(screen.getByText('подключение...')).toBeTruthy()
  })

  it('shows fallback name when roomName is empty', () => {
    render(<ChatHeader {...baseProps} roomName={undefined} />)
    expect(screen.getByText('Загрузка...')).toBeTruthy()
  })

  it('calls onVideoCall when video button clicked', () => {
    const onVideoCall = vi.fn()
    render(<ChatHeader {...baseProps} onVideoCall={onVideoCall} />)
    fireEvent.click(screen.getByTitle('Видеозвонок'))
    expect(onVideoCall).toHaveBeenCalledTimes(1)
  })

  it('calls onSettings when settings button clicked', () => {
    const onSettings = vi.fn()
    render(<ChatHeader {...baseProps} onSettings={onSettings} />)
    fireEvent.click(screen.getByTitle('Настройки'))
    expect(onSettings).toHaveBeenCalledTimes(1)
  })

  it('calls onMenuClick when menu button clicked', () => {
    const onMenuClick = vi.fn()
    render(<ChatHeader {...baseProps} onMenuClick={onMenuClick} />)
    fireEvent.click(screen.getByLabelText('Открыть меню комнат'))
    expect(onMenuClick).toHaveBeenCalledTimes(1)
  })

  it('hides menu button when showMenu=false', () => {
    render(<ChatHeader {...baseProps} showMenu={false} />)
    expect(screen.queryByLabelText('Открыть меню комнат')).toBeNull()
  })
})
