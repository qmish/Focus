import { describe, expect, it } from 'vitest'
import { normalizeStats } from '../lib/adminStats'

describe('adminStore normalizeStats', () => {
  it('maps full stats payload', () => {
    const stats = normalizeStats({
      users: { total: 10 },
      rooms: { total: 20 },
      conferences: { active: 3 },
      messages: { today: 99 },
    })

    expect(stats.users.total).toBe(10)
    expect(stats.rooms.total).toBe(20)
    expect(stats.conferences.active).toBe(3)
    expect(stats.messages.today).toBe(99)
  })

  it('falls back to zeros for missing fields', () => {
    const stats = normalizeStats({})
    expect(stats.users.total).toBe(0)
    expect(stats.rooms.total).toBe(0)
    expect(stats.conferences.active).toBe(0)
    expect(stats.messages.today).toBe(0)
  })
})
