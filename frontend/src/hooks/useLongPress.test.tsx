import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, fireEvent } from '@testing-library/react'
import { useLongPress } from './useLongPress'

function TestComp({ onLong }: { onLong: () => void }) {
  const handlers = useLongPress(onLong, { delay: 200, moveTolerance: 5 })
  return (
    <div data-testid="target" {...handlers}>
      target
    </div>
  )
}

describe('useLongPress', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('triggers callback after delay on touchstart held', () => {
    const onLong = vi.fn()
    const { getByTestId } = render(<TestComp onLong={onLong} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 10, clientY: 10 }] })
    vi.advanceTimersByTime(199)
    expect(onLong).not.toHaveBeenCalled()
    vi.advanceTimersByTime(2)
    expect(onLong).toHaveBeenCalledTimes(1)
  })

  it('does not trigger if touch ends early', () => {
    const onLong = vi.fn()
    const { getByTestId } = render(<TestComp onLong={onLong} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 0, clientY: 0 }] })
    vi.advanceTimersByTime(50)
    fireEvent.touchEnd(el)
    vi.advanceTimersByTime(500)
    expect(onLong).not.toHaveBeenCalled()
  })

  it('cancels when finger moves beyond tolerance', () => {
    const onLong = vi.fn()
    const { getByTestId } = render(<TestComp onLong={onLong} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 0, clientY: 0 }] })
    fireEvent.touchMove(el, { touches: [{ clientX: 30, clientY: 0 }] })
    vi.advanceTimersByTime(500)
    expect(onLong).not.toHaveBeenCalled()
  })

  it('triggers via mouse events (desktop)', () => {
    const onLong = vi.fn()
    const { getByTestId } = render(<TestComp onLong={onLong} />)
    const el = getByTestId('target')
    fireEvent.mouseDown(el, { clientX: 0, clientY: 0 })
    vi.advanceTimersByTime(250)
    expect(onLong).toHaveBeenCalledTimes(1)
  })
})
