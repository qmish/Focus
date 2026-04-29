import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import GlobalSearch from './GlobalSearch'
import { useSearchStore } from '../store/searchStore'

const navigateMock = vi.fn()
vi.mock('react-router-dom', async (orig) => {
  const actual = await orig<typeof import('react-router-dom')>()
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock('../lib/searchClient', () => ({
  SearchAbortError: class extends Error {},
  searchGlobal: vi.fn(),
}))

import { searchGlobal } from '../lib/searchClient'
const mockedSearch = searchGlobal as unknown as ReturnType<typeof vi.fn>

const baseResp = {
  users: [{ id: 'u1', name: 'Alice', email: 'alice@focus.local' }],
  rooms: [
    {
      id: 'r1',
      name: 'general',
      description: '',
      type: 'public' as const,
      jitsi_room_name: 'g',
      is_private: false,
      created_at: '',
      updated_at: '',
    },
  ],
  messages: [
    {
      message: { id: 'm1', room_id: 'r1', user_id: 'u1', content: 'hi', type: 'text' as const, created_at: '', updated_at: '' },
      room_id: 'r1',
      room_name: 'general',
      highlight: '<mark>hi</mark>',
    },
  ],
  files: [],
  meetings: [],
  took_ms: 1,
  query: 'hi',
}

function renderOverlay() {
  return render(
    <MemoryRouter>
      <GlobalSearch open={true} onClose={() => useSearchStore.getState().close()} />
    </MemoryRouter>,
  )
}

describe('GlobalSearch', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    useSearchStore.setState({
      isOpen: true,
      query: '',
      results: null,
      loading: false,
      error: null,
      _abort: null,
      _lastQuery: '',
    })
    mockedSearch.mockReset()
    navigateMock.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders empty hint when query is short', () => {
    renderOverlay()
    expect(screen.getByText(/2 символа/i)).toBeTruthy()
  })

  it('debounces query and calls API once', async () => {
    mockedSearch.mockResolvedValue(baseResp)
    renderOverlay()
    const input = screen.getByPlaceholderText(/Поиск/i) as HTMLInputElement

    fireEvent.change(input, { target: { value: 'h' } })
    fireEvent.change(input, { target: { value: 'hi' } })

    await act(async () => {
      vi.advanceTimersByTime(300)
    })
    await act(async () => {
      await Promise.resolve()
    })
    expect(mockedSearch).toHaveBeenCalled()
    expect(mockedSearch.mock.calls.at(-1)?.[0]?.q).toBe('hi')
  })

  it('renders results grouped by type', async () => {
    mockedSearch.mockResolvedValue(baseResp)
    renderOverlay()
    const input = screen.getByPlaceholderText(/Поиск/i)
    fireEvent.change(input, { target: { value: 'hi' } })
    await act(async () => {
      vi.advanceTimersByTime(300)
      await Promise.resolve()
    })
    expect(screen.getByText('Чаты')).toBeTruthy()
    expect(screen.getByText('Люди')).toBeTruthy()
    expect(screen.getByText('Сообщения')).toBeTruthy()
    expect(screen.getAllByText('general').length).toBeGreaterThan(0)
    expect(screen.getByText('Alice')).toBeTruthy()
  })

  it('Enter on first item navigates to room', async () => {
    mockedSearch.mockResolvedValue(baseResp)
    renderOverlay()
    const input = screen.getByPlaceholderText(/Поиск/i)
    fireEvent.change(input, { target: { value: 'hi' } })
    await act(async () => {
      vi.advanceTimersByTime(300)
      await Promise.resolve()
    })
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(navigateMock).toHaveBeenCalledWith('/rooms/r1')
  })

  it('ArrowDown moves selection', async () => {
    mockedSearch.mockResolvedValue(baseResp)
    renderOverlay()
    const input = screen.getByPlaceholderText(/Поиск/i)
    fireEvent.change(input, { target: { value: 'hi' } })
    await act(async () => {
      vi.advanceTimersByTime(300)
      await Promise.resolve()
    })
    fireEvent.keyDown(window, { key: 'ArrowDown' })
    fireEvent.keyDown(window, { key: 'Enter' })
    // 1й — комната, 2й — пользователь. Для пользователя navigate не вызывается.
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it('Escape closes the overlay', async () => {
    renderOverlay()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(useSearchStore.getState().isOpen).toBe(false)
  })
})
