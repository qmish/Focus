import { useEffect, useRef, useState, useCallback } from 'react'
import { searchLocalMessages, SearchAbortError } from '../lib/searchClient'
import type { MessageHit } from '../types/search'
import { useDebounce } from '../hooks/useDebounce'
import { useHotkey } from '../hooks/useHotkey'

export interface InChatSearchProps {
  roomId: string
  onClose: () => void
}

// InChatSearch — Telegram-style локальный поиск внутри открытой комнаты:
// поле ввода + prev/next + счётчик «N / total» + крестик. При клике/нажатии
// «вверх/вниз» подсвечивает совпавшее сообщение через
// document.getElementById(`message-:id`) и плавный scrollIntoView.
export default function InChatSearch({ roomId, onClose }: InChatSearchProps) {
  const [query, setQuery] = useState('')
  const debounced = useDebounce(query, 250)
  const [hits, setHits] = useState<MessageHit[]>([])
  const [activeIdx, setActiveIdx] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  // Автофокус при монтировании.
  useEffect(() => {
    inputRef.current?.focus()
    return () => abortRef.current?.abort()
  }, [])

  // Запрос при debounced изменении query/roomId.
  useEffect(() => {
    const trimmed = debounced.trim()
    abortRef.current?.abort()
    if (trimmed.length < 2) {
      setHits([])
      setActiveIdx(0)
      setError(null)
      setLoading(false)
      return
    }
    const ctrl = new AbortController()
    abortRef.current = ctrl
    setLoading(true)
    setError(null)
    searchLocalMessages({ roomId, q: trimmed, limit: 50, signal: ctrl.signal })
      .then((res) => {
        if (abortRef.current !== ctrl) return
        setHits(res.messages || [])
        setActiveIdx(0)
        setLoading(false)
      })
      .catch((err) => {
        if (err instanceof SearchAbortError) return
        if (err instanceof DOMException && err.name === 'AbortError') return
        if (abortRef.current !== ctrl) return
        setError(err instanceof Error ? err.message : 'Ошибка поиска')
        setLoading(false)
      })
  }, [debounced, roomId])

  // Подсветка активного хита: scrollIntoView + класс msg--search-target.
  useEffect(() => {
    if (hits.length === 0) return
    const hit = hits[activeIdx]
    if (!hit?.message?.id) return
    const el = document.getElementById(`message-${hit.message.id}`)
    if (!el) return
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    el.classList.add('msg--search-target')
    const t = window.setTimeout(() => el.classList.remove('msg--search-target'), 1600)
    return () => {
      window.clearTimeout(t)
      el.classList.remove('msg--search-target')
    }
  }, [hits, activeIdx])

  const goNext = useCallback(() => {
    setActiveIdx((i) => (hits.length ? (i + 1) % hits.length : 0))
  }, [hits.length])
  const goPrev = useCallback(() => {
    setActiveIdx((i) => (hits.length ? (i - 1 + hits.length) % hits.length : 0))
  }, [hits.length])

  // Хоткеи внутри активного режима. Enter — следующий, Shift+Enter — предыдущий.
  useHotkey(
    [
      { key: 'Enter', shift: false, allowInInput: true, preventDefault: true },
      { key: 'Enter', shift: true, allowInInput: true, preventDefault: true },
      { key: 'Escape', allowInInput: true, preventDefault: true },
    ],
    (_e, spec) => {
      if (spec.key === 'Escape') {
        onClose()
        return
      }
      if (spec.shift) goPrev()
      else goNext()
    },
  )

  return (
    <div className="inchat-search" role="search" data-testid="inchat-search">
      <input
        ref={inputRef}
        className="inchat-search-input"
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Поиск в чате..."
        aria-label="Поиск в чате"
      />
      <span className="inchat-search-counter" data-testid="inchat-search-counter">
        {loading
          ? '…'
          : query.trim().length < 2
            ? ''
            : hits.length === 0
              ? '0/0'
              : `${activeIdx + 1}/${hits.length}`}
      </span>
      <button
        type="button"
        className="icon-btn"
        onClick={goPrev}
        title="Предыдущее (Shift+Enter)"
        aria-label="Предыдущее совпадение"
        disabled={hits.length < 2}
        data-testid="inchat-search-prev"
      >
        ↑
      </button>
      <button
        type="button"
        className="icon-btn"
        onClick={goNext}
        title="Следующее (Enter)"
        aria-label="Следующее совпадение"
        disabled={hits.length < 2}
        data-testid="inchat-search-next"
      >
        ↓
      </button>
      <button
        type="button"
        className="icon-btn"
        onClick={onClose}
        title="Закрыть (Esc)"
        aria-label="Закрыть поиск в чате"
        data-testid="inchat-search-close"
      >
        ✕
      </button>
      {error && <span className="inchat-search-error">{error}</span>}
    </div>
  )
}
