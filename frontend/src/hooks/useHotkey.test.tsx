import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useHotkey } from './useHotkey'

function fireKey(opts: KeyboardEventInit & { key: string }, target?: EventTarget | null) {
  const ev = new KeyboardEvent('keydown', { bubbles: true, cancelable: true, ...opts })
  if (target) {
    Object.defineProperty(ev, 'target', { value: target, configurable: true })
  }
  window.dispatchEvent(ev)
  return ev
}

describe('useHotkey', () => {
  it('calls handler on plain key press', () => {
    const fn = vi.fn()
    renderHook(() => useHotkey({ key: 'a' }, fn))
    fireKey({ key: 'a' })
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('matches case-insensitive', () => {
    const fn = vi.fn()
    renderHook(() => useHotkey({ key: 'k' }, fn))
    fireKey({ key: 'K' })
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('respects modifiers', () => {
    const fn = vi.fn()
    renderHook(() => useHotkey({ key: 'k', ctrl: true }, fn))
    fireKey({ key: 'k' })
    expect(fn).not.toHaveBeenCalled()
    fireKey({ key: 'k', ctrlKey: true })
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('matches any of multiple specs', () => {
    const fn = vi.fn()
    renderHook(() =>
      useHotkey(
        [
          { key: 'k', ctrl: true },
          { key: 'k', cmd: true },
        ],
        fn,
      ),
    )
    fireKey({ key: 'k', metaKey: true })
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('ignores in input by default', () => {
    const fn = vi.fn()
    renderHook(() => useHotkey({ key: '/' }, fn))
    const input = document.createElement('input')
    document.body.appendChild(input)
    fireKey({ key: '/' }, input)
    expect(fn).not.toHaveBeenCalled()
    input.remove()
  })

  it('allowInInput=true overrides input filter', () => {
    const fn = vi.fn()
    renderHook(() => useHotkey({ key: 'Escape', allowInInput: true }, fn))
    const input = document.createElement('input')
    document.body.appendChild(input)
    fireKey({ key: 'Escape' }, input)
    expect(fn).toHaveBeenCalledTimes(1)
    input.remove()
  })

  it('does not fire when disabled', () => {
    const fn = vi.fn()
    renderHook(({ enabled }: { enabled: boolean }) => useHotkey({ key: 'a' }, fn, enabled), {
      initialProps: { enabled: false },
    })
    fireKey({ key: 'a' })
    expect(fn).not.toHaveBeenCalled()
  })
})
