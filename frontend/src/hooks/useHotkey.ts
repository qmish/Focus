import { useEffect } from 'react'

export interface HotkeySpec {
  // Имя клавиши из KeyboardEvent.key (case-insensitive).
  key: string
  ctrl?: boolean
  cmd?: boolean
  alt?: boolean
  shift?: boolean
  // Если true, хоткей сработает даже когда фокус в input/textarea/contenteditable.
  // По умолчанию игнорируется в полях ввода (как в Telegram/Slack).
  allowInInput?: boolean
  preventDefault?: boolean
}

function isInInput(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  const tag = target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  if (target.isContentEditable) return true
  return false
}

// useHotkey — подписка на window keydown с проверкой модификаторов и
// фильтрацией по target (поля ввода).
//
// Несколько спецификаций можно передать массивом — любая из них активирует
// handler (например, Ctrl+K и Cmd+K, или `/` для быстрого фокуса).
export function useHotkey(
  specs: HotkeySpec | HotkeySpec[],
  handler: (e: KeyboardEvent, matched: HotkeySpec) => void,
  enabled = true,
): void {
  useEffect(() => {
    if (!enabled) return
    const list = Array.isArray(specs) ? specs : [specs]
    const onKey = (e: KeyboardEvent) => {
      for (const spec of list) {
        if (e.key.toLowerCase() !== spec.key.toLowerCase()) continue
        if (spec.ctrl !== undefined && spec.ctrl !== e.ctrlKey) continue
        if (spec.cmd !== undefined && spec.cmd !== e.metaKey) continue
        if (spec.alt !== undefined && spec.alt !== e.altKey) continue
        if (spec.shift !== undefined && spec.shift !== e.shiftKey) continue
        if (!spec.allowInInput && isInInput(e.target)) continue
        if (spec.preventDefault !== false) e.preventDefault()
        handler(e, spec)
        return
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [specs, handler, enabled])
}
