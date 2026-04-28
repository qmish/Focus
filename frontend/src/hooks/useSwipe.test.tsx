import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/react'
import { useSwipe, type SwipeCallbacks } from './useSwipe'

function TestComp(props: SwipeCallbacks) {
  const handlers = useSwipe(props, { minDistance: 30 })
  return (
    <div data-testid="target" {...handlers} style={{ width: 200, height: 200 }}>
      target
    </div>
  )
}

describe('useSwipe', () => {
  it('detects left swipe', () => {
    const onSwipeLeft = vi.fn()
    const { getByTestId } = render(<TestComp onSwipeLeft={onSwipeLeft} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 100, clientY: 100 }] })
    fireEvent.touchEnd(el, { changedTouches: [{ clientX: 30, clientY: 105 }] })
    expect(onSwipeLeft).toHaveBeenCalledTimes(1)
  })

  it('detects right swipe', () => {
    const onSwipeRight = vi.fn()
    const { getByTestId } = render(<TestComp onSwipeRight={onSwipeRight} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 50, clientY: 100 }] })
    fireEvent.touchEnd(el, { changedTouches: [{ clientX: 150, clientY: 100 }] })
    expect(onSwipeRight).toHaveBeenCalledTimes(1)
  })

  it('detects up swipe', () => {
    const onSwipeUp = vi.fn()
    const { getByTestId } = render(<TestComp onSwipeUp={onSwipeUp} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 100, clientY: 200 }] })
    fireEvent.touchEnd(el, { changedTouches: [{ clientX: 100, clientY: 50 }] })
    expect(onSwipeUp).toHaveBeenCalledTimes(1)
  })

  it('detects down swipe', () => {
    const onSwipeDown = vi.fn()
    const { getByTestId } = render(<TestComp onSwipeDown={onSwipeDown} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 100, clientY: 50 }] })
    fireEvent.touchEnd(el, { changedTouches: [{ clientX: 100, clientY: 200 }] })
    expect(onSwipeDown).toHaveBeenCalledTimes(1)
  })

  it('ignores movement below minDistance', () => {
    const onSwipeLeft = vi.fn()
    const { getByTestId } = render(<TestComp onSwipeLeft={onSwipeLeft} />)
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 100, clientY: 100 }] })
    fireEvent.touchEnd(el, { changedTouches: [{ clientX: 80, clientY: 100 }] })
    expect(onSwipeLeft).not.toHaveBeenCalled()
  })

  it('does not trigger horizontal swipe when off-axis ratio is too high', () => {
    const onSwipeLeft = vi.fn()
    const onSwipeUp = vi.fn()
    const { getByTestId } = render(
      <TestComp onSwipeLeft={onSwipeLeft} onSwipeUp={onSwipeUp} />
    )
    const el = getByTestId('target')
    fireEvent.touchStart(el, { touches: [{ clientX: 100, clientY: 100 }] })
    fireEvent.touchEnd(el, { changedTouches: [{ clientX: 150, clientY: 0 }] })
    expect(onSwipeLeft).not.toHaveBeenCalled()
    // диагональный жест с явным вертикальным преобладанием — должен попасть в onSwipeUp
    expect(onSwipeUp).toHaveBeenCalledTimes(1)
  })
})
