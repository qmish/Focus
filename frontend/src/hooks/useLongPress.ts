import { useCallback, useRef } from 'react'

export interface LongPressOptions {
  /** Задержка в мс перед срабатыванием (по умолчанию 500) */
  delay?: number
  /** Максимальное смещение пальца в пикселях, при котором всё ещё считается long-press */
  moveTolerance?: number
}

export interface LongPressHandlers {
  onTouchStart: (e: React.TouchEvent) => void
  onTouchEnd: (e: React.TouchEvent) => void
  onTouchMove: (e: React.TouchEvent) => void
  onTouchCancel: () => void
  onMouseDown: (e: React.MouseEvent) => void
  onMouseUp: () => void
  onMouseLeave: () => void
}

/**
 * Хук для определения долгого нажатия (long press) на элементе.
 * Используется для контекстного меню сообщений на мобильных устройствах.
 */
export function useLongPress(
  callback: () => void,
  { delay = 500, moveTolerance = 10 }: LongPressOptions = {}
): LongPressHandlers {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startPos = useRef<{ x: number; y: number } | null>(null)
  const triggered = useRef(false)

  const clear = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    startPos.current = null
    triggered.current = false
  }, [])

  const start = useCallback(
    (x: number, y: number) => {
      clear()
      startPos.current = { x, y }
      triggered.current = false
      timerRef.current = setTimeout(() => {
        triggered.current = true
        callback()
      }, delay)
    },
    [callback, delay, clear]
  )

  const move = useCallback(
    (x: number, y: number) => {
      if (!startPos.current || triggered.current) return
      const dx = Math.abs(x - startPos.current.x)
      const dy = Math.abs(y - startPos.current.y)
      if (dx > moveTolerance || dy > moveTolerance) {
        clear()
      }
    },
    [moveTolerance, clear]
  )

  return {
    onTouchStart: (e: React.TouchEvent) => {
      const t = e.touches[0]
      if (t) start(t.clientX, t.clientY)
    },
    onTouchEnd: clear,
    onTouchMove: (e: React.TouchEvent) => {
      const t = e.touches[0]
      if (t) move(t.clientX, t.clientY)
    },
    onTouchCancel: clear,
    onMouseDown: (e: React.MouseEvent) => start(e.clientX, e.clientY),
    onMouseUp: clear,
    onMouseLeave: clear,
  }
}
