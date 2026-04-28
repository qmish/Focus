import { useEffect, useRef, useState } from 'react'

interface MessageContextMenuProps {
  isMine: boolean
  canDelete: boolean
  canEdit: boolean
  onEdit: () => void
  onDelete: () => void
  onReplyInThread: () => void
}

export default function MessageContextMenu({
  isMine,
  canDelete,
  canEdit,
  onEdit,
  onDelete,
  onReplyInThread,
}: MessageContextMenuProps) {
  const [open, setOpen] = useState(false)
  const wrapperRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handleOutside = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', handleOutside)
    document.addEventListener('keydown', handleEsc)
    return () => {
      document.removeEventListener('mousedown', handleOutside)
      document.removeEventListener('keydown', handleEsc)
    }
  }, [open])

  const handle = (cb: () => void) => () => {
    setOpen(false)
    cb()
  }

  return (
    <div className="msg-ctx-menu-wrapper" ref={wrapperRef}>
      <button
        type="button"
        className="msg-ctx-trigger"
        title="Действия"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen(v => !v)}
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <circle cx="12" cy="5" r="1.5" />
          <circle cx="12" cy="12" r="1.5" />
          <circle cx="12" cy="19" r="1.5" />
        </svg>
      </button>
      {open && (
        <div className="msg-ctx-menu" role="menu">
          <button type="button" role="menuitem" className="msg-ctx-item" onClick={handle(onReplyInThread)}>
            Ответить в треде
          </button>
          {isMine && canEdit && (
            <button type="button" role="menuitem" className="msg-ctx-item" onClick={handle(onEdit)}>
              Редактировать
            </button>
          )}
          {canDelete && (
            <button type="button" role="menuitem" className="msg-ctx-item msg-ctx-item-danger" onClick={handle(onDelete)}>
              Удалить
            </button>
          )}
        </div>
      )}
    </div>
  )
}
