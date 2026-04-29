import { useEffect, useRef, useState, type RefObject } from 'react'

export interface PullToRefreshOptions {
  /** Дистанция в пикселях, после которой запускается onRefresh */
  threshold?: number
  /** Только если scrollTop <= 0 */
  enabled?: boolean
}

export interface PullToRefreshState {
  pullDistance: number
  isRefreshing: boolean
}

/**
 * Pull-to-refresh для скроллируемого контейнера. Возвращает состояние —
 * текущее смещение и флаг refresh, которые компонент может использовать
 * для отрисовки индикатора.
 */
export function usePullToRefresh(
  containerRef: RefObject<HTMLElement | null>,
  onRefresh: () => Promise<void> | void,
  { threshold = 70, enabled = true }: PullToRefreshOptions = {}
): PullToRefreshState {
  const [pullDistance, setPullDistance] = useState(0)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const startYRef = useRef<number | null>(null)
  const isPullingRef = useRef(false)

  useEffect(() => {
    const el = containerRef.current
    if (!el || !enabled) return

    const onTouchStart = (e: TouchEvent) => {
      if (el.scrollTop > 0) return
      const t = e.touches[0]
      if (!t) return
      startYRef.current = t.clientY
      isPullingRef.current = true
    }

    const onTouchMove = (e: TouchEvent) => {
      if (!isPullingRef.current || startYRef.current === null) return
      const t = e.touches[0]
      if (!t) return
      const dy = t.clientY - startYRef.current
      if (dy > 0 && el.scrollTop <= 0) {
        const damped = Math.min(dy * 0.5, threshold * 1.5)
        setPullDistance(damped)
      } else {
        setPullDistance(0)
      }
    }

    const onTouchEnd = async () => {
      if (!isPullingRef.current) return
      isPullingRef.current = false
      startYRef.current = null
      if (pullDistance >= threshold) {
        setIsRefreshing(true)
        try {
          await onRefresh()
        } finally {
          setIsRefreshing(false)
          setPullDistance(0)
        }
      } else {
        setPullDistance(0)
      }
    }

    el.addEventListener('touchstart', onTouchStart, { passive: true })
    el.addEventListener('touchmove', onTouchMove, { passive: true })
    el.addEventListener('touchend', onTouchEnd)
    el.addEventListener('touchcancel', onTouchEnd)

    return () => {
      el.removeEventListener('touchstart', onTouchStart)
      el.removeEventListener('touchmove', onTouchMove)
      el.removeEventListener('touchend', onTouchEnd)
      el.removeEventListener('touchcancel', onTouchEnd)
    }
  }, [containerRef, onRefresh, threshold, enabled, pullDistance])

  return { pullDistance, isRefreshing }
}
