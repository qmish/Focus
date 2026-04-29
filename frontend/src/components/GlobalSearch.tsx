import { useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useNavigate } from 'react-router-dom'
import { useSearchStore } from '../store/searchStore'
import { useDebounce } from '../hooks/useDebounce'
import { useHotkey } from '../hooks/useHotkey'
import type {
  FlatSearchResult,
  FileHit,
  MeetingHit,
  MessageHit,
  SearchUser,
} from '../types/search'
import type { Room } from '../store/roomsStore'

function flattenResults(results: ReturnType<typeof useSearchStore.getState>['results']): FlatSearchResult[] {
  if (!results) return []
  const out: FlatSearchResult[] = []
  for (const r of results.rooms) {
    out.push({ kind: 'rooms', id: `room-${r.id}`, title: r.name, subtitle: r.description, payload: r })
  }
  for (const u of results.users) {
    out.push({
      kind: 'users',
      id: `user-${u.id}`,
      title: u.name || u.email,
      subtitle: u.email,
      payload: u,
    })
  }
  for (const m of results.messages) {
    out.push({
      kind: 'messages',
      id: `msg-${m.message?.id ?? Math.random()}`,
      title: m.room_name,
      snippet: m.highlight,
      payload: m,
    })
  }
  for (const f of results.files) {
    out.push({
      kind: 'files',
      id: `file-${f.message_id}`,
      title: f.file_name,
      subtitle: f.room_name,
      payload: f,
    })
  }
  for (const meet of results.meetings) {
    out.push({
      kind: 'meetings',
      id: `meet-${meet.id}`,
      title: meet.subject,
      subtitle: `${new Date(meet.start_at).toLocaleString('ru-RU')} • ${meet.room_name}`,
      payload: meet,
    })
  }
  return out
}

const GROUP_LABEL: Record<FlatSearchResult['kind'], string> = {
  rooms: 'Чаты',
  users: 'Люди',
  messages: 'Сообщения',
  files: 'Файлы',
  meetings: 'Встречи',
}

const GROUP_ORDER: FlatSearchResult['kind'][] = ['rooms', 'users', 'messages', 'files', 'meetings']

export interface GlobalSearchProps {
  open: boolean
  onClose: () => void
}

export default function GlobalSearch({ open, onClose }: GlobalSearchProps) {
  const navigate = useNavigate()
  const { query, results, loading, error, search, setQuery, reset } = useSearchStore()
  const debounced = useDebounce(query, 250)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [activeIdx, setActiveIdx] = useState(0)

  // Сбросить поле и подсветку при открытии.
  useEffect(() => {
    if (open) {
      reset()
      setActiveIdx(0)
      // Фокус через RAF, чтобы portal успел смонтироваться.
      requestAnimationFrame(() => inputRef.current?.focus())
    }
  }, [open, reset])

  // Обновить запрос при debounce.
  useEffect(() => {
    if (!open) return
    void search(debounced)
  }, [debounced, open, search])

  // Сбросить указатель при смене результатов.
  useEffect(() => {
    setActiveIdx(0)
  }, [results])

  const flat = useMemo(() => flattenResults(results), [results])

  const grouped = useMemo(() => {
    const map: Record<string, FlatSearchResult[]> = {}
    for (const r of flat) (map[r.kind] ||= []).push(r)
    return map
  }, [flat])

  // Хоткеи внутри overlay (allowInInput: true для стрелок и Enter).
  useHotkey(
    [
      { key: 'ArrowDown', allowInInput: true, preventDefault: true },
      { key: 'ArrowUp', allowInInput: true, preventDefault: true },
      { key: 'Enter', allowInInput: true, preventDefault: true },
      { key: 'Escape', allowInInput: true, preventDefault: true },
    ],
    (_e, spec) => {
      if (spec.key === 'Escape') {
        onClose()
        return
      }
      if (spec.key === 'ArrowDown') {
        if (flat.length === 0) return
        setActiveIdx((i) => Math.min(i + 1, flat.length - 1))
      } else if (spec.key === 'ArrowUp') {
        if (flat.length === 0) return
        setActiveIdx((i) => Math.max(i - 1, 0))
      } else if (spec.key === 'Enter') {
        const target = flat[activeIdx]
        if (target) handleSelect(target)
      }
    },
    open,
  )

  function handleSelect(item: FlatSearchResult) {
    switch (item.kind) {
      case 'rooms': {
        const room = item.payload as Room
        navigate(`/rooms/${room.id}`)
        break
      }
      case 'users': {
        // Открытие диалога/профиля выходит за рамки PR — просто закроем.
        const u = item.payload as SearchUser
        console.info('[search] user clicked', u.id)
        break
      }
      case 'messages': {
        const m = item.payload as MessageHit
        if (m.message?.id) {
          navigate(`/rooms/${m.room_id}?messageId=${m.message.id}`)
        } else {
          navigate(`/rooms/${m.room_id}`)
        }
        break
      }
      case 'files': {
        const f = item.payload as FileHit
        navigate(`/rooms/${f.room_id}?messageId=${f.message_id}`)
        break
      }
      case 'meetings': {
        const meet = item.payload as MeetingHit
        navigate(`/rooms/${meet.room_id}`)
        break
      }
    }
    onClose()
  }

  if (!open) return null

  // Скользящий индекс для подсветки активного элемента.
  let runningIndex = -1

  const node = (
    <div className="global-search-overlay" role="dialog" aria-modal="true" onClick={onClose}>
      <div
        className="global-search-modal"
        role="combobox"
        aria-expanded={true}
        aria-controls="global-search-results"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="global-search-input-wrap">
          <span className="global-search-icon" aria-hidden="true">
            🔎
          </span>
          <input
            ref={inputRef}
            className="global-search-input"
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Поиск по чатам, людям, сообщениям, файлам, встречам"
            aria-label="Глобальный поиск"
          />
          <button className="global-search-close" onClick={onClose} aria-label="Закрыть поиск">
            ✕
          </button>
        </div>

        <div id="global-search-results" className="global-search-results" data-testid="global-search-results">
          {loading && <div className="global-search-loading">Идёт поиск…</div>}
          {error && !loading && <div className="global-search-error">{error}</div>}
          {!loading && !error && query.trim().length < 2 && (
            <div className="global-search-empty">Введите минимум 2 символа для поиска</div>
          )}
          {!loading && !error && query.trim().length >= 2 && flat.length === 0 && results && (
            <div className="global-search-empty">Ничего не найдено</div>
          )}

          {!loading && !error && flat.length > 0 && (
            <>
              {GROUP_ORDER.filter((k) => grouped[k]?.length).map((kind) => (
                <div className="global-search-group" key={kind}>
                  <div className="global-search-group-title">{GROUP_LABEL[kind]}</div>
                  {grouped[kind].map((item) => {
                    runningIndex++
                    const isActive = runningIndex === activeIdx
                    return (
                      <button
                        key={item.id}
                        type="button"
                        className={`global-search-item${isActive ? ' is-active' : ''}`}
                        onClick={() => handleSelect(item)}
                        onMouseEnter={() => setActiveIdx(runningIndex)}
                        data-testid="global-search-item"
                      >
                        <span className="global-search-item-icon" aria-hidden="true">
                          {iconForKind(item.kind)}
                        </span>
                        <span className="global-search-item-body">
                          <span className="global-search-item-title">{item.title}</span>
                          {item.snippet ? (
                            <span
                              className="global-search-item-snippet"
                              dangerouslySetInnerHTML={{ __html: item.snippet }}
                            />
                          ) : item.subtitle ? (
                            <span className="global-search-item-subtitle">{item.subtitle}</span>
                          ) : null}
                        </span>
                      </button>
                    )
                  })}
                </div>
              ))}
            </>
          )}
        </div>

        <div className="global-search-footer">
          <span>
            <kbd>↑</kbd> <kbd>↓</kbd> навигация
          </span>
          <span>
            <kbd>Enter</kbd> открыть
          </span>
          <span>
            <kbd>Esc</kbd> закрыть
          </span>
        </div>
      </div>
    </div>
  )

  return createPortal(node, document.body)
}

function iconForKind(kind: FlatSearchResult['kind']): string {
  switch (kind) {
    case 'rooms':
      return '#'
    case 'users':
      return '@'
    case 'messages':
      return '💬'
    case 'files':
      return '📎'
    case 'meetings':
      return '📅'
  }
}
