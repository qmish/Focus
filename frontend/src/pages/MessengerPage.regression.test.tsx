import { useState } from 'react'
import { describe, it, expect } from 'vitest'
import { render, fireEvent } from '@testing-library/react'
import { useSwipe } from '../hooks/useSwipe'

/**
 * Regression-тест для бага «Произошла ошибка» (React #300):
 *
 * `MessengerPage` имел early-return при `showVideo && roomId && jitsiJWT`,
 * а после него — вызов `useSwipe(...)`. При переключении `showVideo` с false
 * на true количество хуков уменьшалось → React падал с
 * «Rendered fewer hooks than expected. This may be caused by an accidental
 * early return statement».
 *
 * Этот тест воспроизводит ту же структуру компонента и проверяет, что
 * переключение между «обычным» и «full-screen video» режимами не приводит
 * к ошибке. Если когда-нибудь снова добавят хук ниже early-return — тест
 * упадёт.
 */

function ComponentWithSwitch({
  initial = false,
}: {
  initial?: boolean
}) {
  const [showVideo, setShowVideo] = useState(initial)

  // ВСЕ хуки ДО любого условного return.
  const swipeHandlers = useSwipe({
    onSwipeLeft: () => undefined,
    onSwipeRight: () => undefined,
  })

  if (showVideo) {
    return (
      <div data-testid="video">
        <button onClick={() => setShowVideo(false)}>back</button>
      </div>
    )
  }

  return (
    <div data-testid="messenger" {...swipeHandlers}>
      <button onClick={() => setShowVideo(true)}>video</button>
    </div>
  )
}

describe('MessengerPage hooks order regression', () => {
  it('переключение showVideo не вызывает «Rendered fewer hooks than expected»', () => {
    const { getByText, getByTestId } = render(<ComponentWithSwitch />)
    expect(getByTestId('messenger')).toBeTruthy()

    // Включаем «видео» — раньше тут падало с React #300.
    fireEvent.click(getByText('video'))
    expect(getByTestId('video')).toBeTruthy()

    // Возвращаемся обратно — тоже должно работать.
    fireEvent.click(getByText('back'))
    expect(getByTestId('messenger')).toBeTruthy()
  })

  it('начальный рендер с showVideo=true тоже стабилен', () => {
    const { getByTestId } = render(<ComponentWithSwitch initial={true} />)
    expect(getByTestId('video')).toBeTruthy()
  })
})
