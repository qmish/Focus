import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

const QUICK_EMOJIS = ['👍', '❤️', '😂', '😮', '🔥', '👎']

interface EmojiPickerProps {
  onSelect: (emoji: string) => void
  onClose: () => void
  /**
   * Якорный элемент, относительно которого позиционируется picker.
   * Если задан, picker рендерится в `document.body` через Portal с
   * `position: fixed`, чтобы не обрезаться родительским `overflow`
   * и не зависеть от `position` родителя (`.msg-bubble`/`.chat-messages`).
   */
  anchorRef?: React.RefObject<HTMLElement | null>
}

export default function EmojiPicker({ onSelect, onClose, anchorRef }: EmojiPickerProps) {
  const ref = useRef<HTMLDivElement>(null)
  const onCloseRef = useRef(onClose)
  onCloseRef.current = onClose
  const [coords, setCoords] = useState<{ top: number; left: number } | null>(null)

  useLayoutEffect(() => {
    if (!anchorRef?.current) return
    const updatePosition = () => {
      const anchor = anchorRef.current
      if (!anchor) return
      const rect = anchor.getBoundingClientRect()
      setCoords({
        top: rect.top - 4,
        left: rect.right,
      })
    }
    updatePosition()
    window.addEventListener('resize', updatePosition)
    window.addEventListener('scroll', updatePosition, true)
    return () => {
      window.removeEventListener('resize', updatePosition)
      window.removeEventListener('scroll', updatePosition, true)
    }
  }, [anchorRef])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as Node
      if (ref.current && ref.current.contains(target)) return
      if (anchorRef?.current && anchorRef.current.contains(target)) return
      onCloseRef.current()
    }
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCloseRef.current()
    }
    // Отложить подписку на mousedown на следующий тик, чтобы клик,
    // открывший пикер, не закрыл его сразу тем же событием (capture/bubble).
    const t = window.setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
    }, 0)
    document.addEventListener('keydown', handleEsc)
    return () => {
      window.clearTimeout(t)
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEsc)
    }
  }, [anchorRef])

  const usePortal = !!anchorRef
  const style: React.CSSProperties | undefined = usePortal && coords
    ? { position: 'fixed', top: `${coords.top}px`, left: `${coords.left}px`, transform: 'translate(-100%, -100%)' }
    : undefined

  if (usePortal && !coords) return null

  const picker = (
    <div ref={ref} className="emoji-picker" style={style} role="menu">
      {QUICK_EMOJIS.map(emoji => (
        <button
          key={emoji}
          className="emoji-picker-item"
          onClick={() => onSelect(emoji)}
          type="button"
        >
          {emoji}
        </button>
      ))}
    </div>
  )

  if (usePortal) {
    return createPortal(picker, document.body)
  }
  return picker
}
