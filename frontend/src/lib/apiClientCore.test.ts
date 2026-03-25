import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiRequest } from './apiClientCore'

global.fetch = vi.fn()

describe('apiClientCore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('sends auth and json headers', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: '1' }),
    } as Response)

    await apiRequest<{ id: string }>('/api/v1/rooms', {
      method: 'POST',
      token: 'token-1',
      body: { name: 'Room' },
    })

    expect(global.fetch).toHaveBeenCalledWith('/api/v1/rooms', expect.objectContaining({
      method: 'POST',
      headers: expect.objectContaining({
        'Content-Type': 'application/json',
        Authorization: 'Bearer token-1',
      }),
      body: JSON.stringify({ name: 'Room' }),
    }))
  })

  it('retries get requests on server error', async () => {
    vi.mocked(global.fetch)
      .mockResolvedValueOnce({
        ok: false,
        status: 503,
        text: async () => 'unavailable',
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: [] }),
      } as Response)

    const result = await apiRequest<{ data: unknown[] }>('/api/v1/rooms', { method: 'GET', retry: 1 })
    expect(result.data).toEqual([])
    expect(global.fetch).toHaveBeenCalledTimes(2)
  })

  it('throws ApiError for non-ok responses', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: false,
      status: 400,
      text: async () => 'bad request',
    } as Response)

    await expect(apiRequest('/api/v1/rooms', { method: 'POST', body: {} })).rejects.toBeInstanceOf(ApiError)
  })
})
