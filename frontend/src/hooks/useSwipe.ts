import { useRef, useCallback } from 'react'

export interface SwipeOptions {
  /** Минимальная дистанция в пикселях, чтобы считать жест свайпом */
  minDistance?: number
  /** Максимальное соотношение поперечного смещения к основному */
  maxOffAxisRatio?: number
}

export interface SwipeHandlers {
  onTouchStart: (e: React.TouchEvent) => void
  onTouchEnd: (e: React.TouchEvent) => void
}

export interface SwipeCallbacks {
  onSwipeLeft?: () => void
  onSwipeRight?: () => void
  onSwipeUp?: () => void
  onSwipeDown?: () => void
}

/**
 * Хук для определения жеста свайп. Возвращает обработчики touch-событий.
 */
export function useSwipe(
  callbacks: SwipeCallbacks,
  { minDistance = 50, maxOffAxisRatio = 0.7 }: SwipeOptions = {}
): SwipeHandlers {
  const startRef = useRef<{ x: number; y: number; t: number } | null>(null)

  const onTouchStart = useCallback((e: React.TouchEvent) => {
    const t = e.touches[0]
    if (!t) return
    startRef.current = { x: t.clientX, y: t.clientY, t: Date.now() }
  }, [])

  const onTouchEnd = useCallback(
    (e: React.TouchEvent) => {
      const start = startRef.current
      startRef.current = null
      if (!start) return
      const t = e.changedTouches[0]
      if (!t) return
      const dx = t.clientX - start.x
      const dy = t.clientY - start.y
      const absDx = Math.abs(dx)
      const absDy = Math.abs(dy)

      if (absDx > absDy) {
        if (absDx < minDistance) return
        if (absDy / absDx > maxOffAxisRatio) return
        if (dx < 0) callbacks.onSwipeLeft?.()
        else callbacks.onSwipeRight?.()
      } else {
        if (absDy < minDistance) return
        if (absDx / absDy > maxOffAxisRatio) return
        if (dy < 0) callbacks.onSwipeUp?.()
        else callbacks.onSwipeDown?.()
      }
    },
    [callbacks, minDistance, maxOffAxisRatio]
  )

  return { onTouchStart, onTouchEnd }
}
