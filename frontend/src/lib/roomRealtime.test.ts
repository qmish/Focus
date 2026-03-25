import { describe, expect, it } from 'vitest'
import { buildWebSocketURL, mergeMessageList } from './roomRealtime'
import type { Message } from '../store/roomsStore'

describe('roomRealtime helpers', () => {
  it('builds websocket url with token', () => {
    const wsURL = buildWebSocketURL('https://focus.company.com/rooms/1', 'abc')
    expect(wsURL).toContain('wss://focus.company.com/api/v1/ws')
    expect(wsURL).toContain('access_token=abc')
  })

  it('merges incoming message without duplicates', () => {
    const base: Message = {
      id: '1',
      room_id: 'r',
      user_id: 'u',
      content: 'hello',
      type: 'text',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }
    const merged = mergeMessageList([], base)
    expect(merged).toHaveLength(1)
    expect(mergeMessageList(merged, base)).toHaveLength(1)
  })
})
