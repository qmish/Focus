import { create } from 'zustand'
import { searchGlobal, SearchAbortError } from '../lib/searchClient'
import type { GlobalSearchResponse } from '../types/search'

interface SearchState {
  isOpen: boolean
  query: string
  results: GlobalSearchResponse | null
  loading: boolean
  error: string | null
  // Внутреннее: контроллер последнего запроса. Любой новый запрос отменяет
  // предыдущий, чтобы устаревшие ответы не перетёрли актуальные.
  _abort: AbortController | null
  _lastQuery: string

  open: () => void
  close: () => void
  setQuery: (q: string) => void
  search: (q: string) => Promise<void>
  reset: () => void
}

const empty: GlobalSearchResponse = {
  users: [],
  rooms: [],
  messages: [],
  files: [],
  meetings: [],
  took_ms: 0,
  query: '',
}

export const useSearchStore = create<SearchState>((set, get) => ({
  isOpen: false,
  query: '',
  results: null,
  loading: false,
  error: null,
  _abort: null,
  _lastQuery: '',

  open: () => set({ isOpen: true }),

  close: () => {
    const cur = get()._abort
    cur?.abort()
    set({ isOpen: false, _abort: null, loading: false })
  },

  setQuery: (q) => set({ query: q }),

  reset: () =>
    set({ query: '', results: null, error: null, _lastQuery: '', loading: false }),

  search: async (q) => {
    const trimmed = q.trim()
    if (trimmed.length < 2) {
      const cur = get()._abort
      cur?.abort()
      set({ results: empty, error: null, loading: false, _abort: null, _lastQuery: trimmed })
      return
    }
    // Отменим предыдущий запрос.
    const prev = get()._abort
    prev?.abort()
    const ctrl = new AbortController()
    set({ loading: true, error: null, _abort: ctrl, _lastQuery: trimmed })
    try {
      const res = await searchGlobal({ q: trimmed, signal: ctrl.signal })
      // Если за время запроса появился более новый — игнорируем результат.
      if (get()._abort !== ctrl) return
      set({ results: res, loading: false, _abort: null })
    } catch (err) {
      if (err instanceof SearchAbortError) return
      const aborted = err instanceof DOMException && err.name === 'AbortError'
      if (aborted) return
      if (get()._abort !== ctrl) return
      set({
        error: err instanceof Error ? err.message : 'Ошибка поиска',
        loading: false,
        _abort: null,
      })
    }
  },
}))
