import { useEffect, useState } from 'react'

// useDebounce — возвращает значение, обновляющееся не чаще, чем раз в `delay` мс.
// Используется для отложенного запуска поиска при наборе.
export function useDebounce<T>(value: T, delay = 250): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(value), delay)
    return () => window.clearTimeout(id)
  }, [value, delay])

  return debounced
}
