import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import InChatSearch from './InChatSearch'

vi.mock('../lib/searchClient', () => ({
  SearchAbortError: class extends Error {},
  searchLocalMessages: vi.fn(),
}))

import { searchLocalMessages } from '../lib/searchClient'
const mockedLocal = searchLocalMessages as unknown as ReturnType<typeof vi.fn>

const sample = {
  query: 'hi',
  took_ms: 1,
  messages: [
    { message: { id: 'm1', room_id: 'r1', user_id: 'u1', content: 'hi 1', type: 'text' as const, created_at: '', updated_at: '' }, room_id: 'r1', room_name: 'general', highlight: '<mark>hi</mark>' },
    { message: { id: 'm2', room_id: 'r1', user_id: 'u1', content: 'hi 2', type: 'text' as const, created_at: '', updated_at: '' }, room_id: 'r1', room_name: 'general', highlight: '<mark>hi</mark>' },
    { message: { id: 'm3', room_id: 'r1', user_id: 'u1', content: 'hi 3', type: 'text' as const, created_at: '', updated_at: '' }, room_id: 'r1', room_name: 'general', highlight: '<mark>hi</mark>' },
  ],
}

beforeEach(() => {
  vi.useFakeTimers()
  mockedLocal.mockReset()
  document.body.innerHTML = ''
  for (const id of ['m1', 'm2', 'm3']) {
    const el = document.createElement('div')
    el.id = `message-${id}`
    el.scrollIntoView = vi.fn()
    document.body.appendChild(el)
  }
})

afterEach(() => {
  vi.useRealTimers()
})

async function flush() {
  await act(async () => {
    vi.advanceTimersByTime(300)
    await Promise.resolve()
  })
}

describe('InChatSearch', () => {
  it('shows empty hint until 2+ chars', async () => {
    render(<InChatSearch roomId="r1" onClose={() => {}} />)
    const input = screen.getByPlaceholderText(/Поиск в чате/i) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'a' } })
    await flush()
    expect(mockedLocal).not.toHaveBeenCalled()
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('')
  })

  it('shows 1/3 after first hit', async () => {
    mockedLocal.mockResolvedValue(sample)
    render(<InChatSearch roomId="r1" onClose={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText(/Поиск в чате/i), { target: { value: 'hi' } })
    await flush()
    expect(mockedLocal).toHaveBeenCalledTimes(1)
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('1/3')
  })

  it('Enter advances next, Shift+Enter goes prev', async () => {
    mockedLocal.mockResolvedValue(sample)
    render(<InChatSearch roomId="r1" onClose={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText(/Поиск в чате/i), { target: { value: 'hi' } })
    await flush()

    fireEvent.keyDown(window, { key: 'Enter' })
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('2/3')
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('3/3')
    // wraps around
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('1/3')

    fireEvent.keyDown(window, { key: 'Enter', shiftKey: true })
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('3/3')
  })

  it('prev/next buttons navigate', async () => {
    mockedLocal.mockResolvedValue(sample)
    render(<InChatSearch roomId="r1" onClose={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText(/Поиск в чате/i), { target: { value: 'hi' } })
    await flush()

    fireEvent.click(screen.getByTestId('inchat-search-next'))
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('2/3')
    fireEvent.click(screen.getByTestId('inchat-search-prev'))
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('1/3')
  })

  it('Escape closes via onClose', async () => {
    const onClose = vi.fn()
    render(<InChatSearch roomId="r1" onClose={onClose} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('shows 0/0 when no hits', async () => {
    mockedLocal.mockResolvedValue({ ...sample, messages: [] })
    render(<InChatSearch roomId="r1" onClose={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText(/Поиск в чате/i), { target: { value: 'xx' } })
    await flush()
    expect(screen.getByTestId('inchat-search-counter').textContent).toBe('0/0')
  })

  it('scrolls to active message and toggles highlight class', async () => {
    mockedLocal.mockResolvedValue(sample)
    render(<InChatSearch roomId="r1" onClose={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText(/Поиск в чате/i), { target: { value: 'hi' } })
    await flush()
    const target = document.getElementById('message-m1')!
    expect(target.scrollIntoView).toHaveBeenCalled()
    expect(target.classList.contains('msg--search-target')).toBe(true)
  })
})
