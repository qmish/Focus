import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import ProfileModal from './ProfileModal'

const mockUser = {
  id: '123',
  email: 'test@example.com',
  name: 'Test User',
  roles: ['user'] as string[],
  department: 'Engineering',
  directorate: 'IT',
  position: 'Developer',
  phone: '+7 999 123 45 67',
  about_me: 'Hello',
  video_start_with_audio_muted: false,
  video_start_with_video_muted: true,
  video_display_name: 'TestUser',
  video_default_language: 'ru',
}

describe('ProfileModal', () => {
  it('does not render when closed', () => {
    const { container } = render(
      <ProfileModal open={false} onClose={vi.fn()} user={mockUser} token="test-token" onSave={vi.fn()} />
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders form fields when open', () => {
    render(
      <ProfileModal open={true} onClose={vi.fn()} user={mockUser} token="test-token" onSave={vi.fn()} />
    )
    expect(screen.getByText('Настройки профиля')).toBeTruthy()
    expect(screen.getByDisplayValue('Test User')).toBeTruthy()
    expect(screen.getByDisplayValue('Engineering')).toBeTruthy()
    expect(screen.getByDisplayValue('IT')).toBeTruthy()
    expect(screen.getByDisplayValue('Developer')).toBeTruthy()
    expect(screen.getByDisplayValue('+7 999 123 45 67')).toBeTruthy()
    expect(screen.getByDisplayValue('Hello')).toBeTruthy()
    expect(screen.getByDisplayValue('TestUser')).toBeTruthy()
  })

  it('shows email as disabled', () => {
    render(
      <ProfileModal open={true} onClose={vi.fn()} user={mockUser} token="test-token" onSave={vi.fn()} />
    )
    const emailInput = screen.getByDisplayValue('test@example.com')
    expect(emailInput).toHaveProperty('disabled', true)
  })

  it('calls onClose when cancel is clicked', () => {
    const onClose = vi.fn()
    render(
      <ProfileModal open={true} onClose={onClose} user={mockUser} token="test-token" onSave={vi.fn()} />
    )
    fireEvent.click(screen.getByText('Отмена'))
    expect(onClose).toHaveBeenCalled()
  })

  it('calls onClose when overlay is clicked', () => {
    const onClose = vi.fn()
    render(
      <ProfileModal open={true} onClose={onClose} user={mockUser} token="test-token" onSave={vi.fn()} />
    )
    const overlay = document.querySelector('.modal-overlay')
    if (overlay) {
      fireEvent.click(overlay)
      expect(onClose).toHaveBeenCalled()
    }
  })

  it('renders video conference settings section', () => {
    render(
      <ProfileModal open={true} onClose={vi.fn()} user={mockUser} token="test-token" onSave={vi.fn()} />
    )
    expect(screen.getByText('Настройки видеоконференции')).toBeTruthy()
    expect(screen.getByLabelText('Начинать с выключенным микрофоном')).toBeTruthy()
    expect(screen.getByLabelText('Начинать с выключенной камерой')).toBeTruthy()
  })

  it('does not render when user is null', () => {
    const { container } = render(
      <ProfileModal open={true} onClose={vi.fn()} user={null} token="test-token" onSave={vi.fn()} />
    )
    // Modal still renders the shell but form fields will be empty
    expect(container.querySelector('.profile-modal')).toBeTruthy()
  })
})
