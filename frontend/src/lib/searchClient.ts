import { getApiBaseUrl } from './apiBase'
import { useAuthStore } from '../store/authStore'
import type { GlobalSearchResponse, LocalMessagesResponse, SearchTypeKey } from '../types/search'

// SearchAbortError — пробрасывается, когда AbortController отменил запрос.
// Стор использует это, чтобы отличить отмену от настоящей ошибки.
export class SearchAbortError extends Error {
  constructor() {
    super('search aborted')
    this.name = 'SearchAbortError'
  }
}

export interface GlobalSearchParams {
  q: string
  types?: SearchTypeKey[]
  limit?: number
  signal?: AbortSignal
}

export interface LocalSearchParams {
  roomId: string
  q: string
  before?: string
  limit?: number
  signal?: AbortSignal
}

async function fetchJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const token = useAuthStore.getState().token
  const url = `${getApiBaseUrl()}${path}`
  let res: Response
  try {
    res = await fetch(url, {
      method: 'GET',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      signal,
    })
  } catch (err) {
    if (signal?.aborted) throw new SearchAbortError()
    throw err
  }
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `Ошибка поиска (${res.status})`)
  }
  return (await res.json()) as T
}

export async function searchGlobal(params: GlobalSearchParams): Promise<GlobalSearchResponse> {
  const sp = new URLSearchParams()
  sp.set('q', params.q)
  if (params.types && params.types.length > 0) sp.set('types', params.types.join(','))
  if (params.limit) sp.set('limit', String(params.limit))
  return fetchJSON<GlobalSearchResponse>(`/api/v1/search?${sp.toString()}`, params.signal)
}

export async function searchLocalMessages(params: LocalSearchParams): Promise<LocalMessagesResponse> {
  const sp = new URLSearchParams()
  sp.set('q', params.q)
  if (params.before) sp.set('before', params.before)
  if (params.limit) sp.set('limit', String(params.limit))
  return fetchJSON<LocalMessagesResponse>(
    `/api/v1/rooms/${encodeURIComponent(params.roomId)}/messages/search?${sp.toString()}`,
    params.signal,
  )
}
