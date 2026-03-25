import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { JitsiMeeting } from './JitsiMeeting'

let capturedProps: Record<string, unknown> | undefined

vi.mock('@jitsi/react-sdk', () => ({
  JitsiMeeting: (props: {
    domain?: string
    configOverwrite?: Record<string, unknown>
    interfaceConfigOverwrite?: Record<string, unknown>
    onVideoConferenceJoined?: () => void
    onVideoConferenceLeft?: () => void
    onParticipantJoined?: (participant: unknown) => void
    onParticipantLeft?: (participant: unknown) => void
  }) => {
    capturedProps = props
    return (
      <div>
        <button onClick={() => props.onVideoConferenceJoined?.()}>join</button>
        <button onClick={() => props.onVideoConferenceLeft?.()}>leave</button>
        <button onClick={() => props.onParticipantJoined?.({ id: 'p1' })}>participant-join</button>
        <button onClick={() => props.onParticipantLeft?.({ id: 'p1' })}>participant-leave</button>
      </div>
    )
  },
}))

describe('JitsiMeeting bridge callbacks', () => {
  it('propagates join/leave and participant events', () => {
    const onJoin = vi.fn()
    const onLeave = vi.fn()
    const onParticipantJoined = vi.fn()
    const onParticipantLeft = vi.fn()

    render(
      <JitsiMeeting
        roomName="room-1"
        jwt="token"
        onJoin={onJoin}
        onLeave={onLeave}
        onParticipantJoined={onParticipantJoined}
        onParticipantLeft={onParticipantLeft}
      />,
    )

    fireEvent.click(screen.getByText('join'))
    fireEvent.click(screen.getByText('leave'))
    fireEvent.click(screen.getByText('participant-join'))
    fireEvent.click(screen.getByText('participant-leave'))

    expect(onJoin).toHaveBeenCalledTimes(1)
    expect(onLeave).toHaveBeenCalledTimes(1)
    expect(onParticipantJoined).toHaveBeenCalledTimes(1)
    expect(onParticipantLeft).toHaveBeenCalledTimes(1)
  })

  it('applies domain and branding overrides', () => {
    render(
      <JitsiMeeting
        roomName="room-1"
        jwt="token"
        domain="meet.focus.local"
        branding={{
          appName: 'Focus Meet',
          defaultLanguage: 'ru',
          dynamicBrandingUrl: '/api/v1/branding/jitsi',
          customTheme: { 'palette.ui01': '#000' },
          customIcons: { mic: '/pics/image16.png' },
          toolbarButtons: ['microphone', 'hangup'],
        }}
      />,
    )

    expect(capturedProps?.domain).toBe('meet.focus.local')
    const configOverwrite = capturedProps?.configOverwrite as Record<string, unknown>
    expect(configOverwrite.defaultLanguage).toBe('ru')
    expect(configOverwrite.dynamicBrandingUrl).toBe('/api/v1/branding/jitsi')
    expect(configOverwrite.customTheme).toEqual({ 'palette.ui01': '#000' })
    expect(configOverwrite.customIcons).toEqual({ mic: '/pics/image16.png' })

    const interfaceConfig = capturedProps?.interfaceConfigOverwrite as Record<string, unknown>
    expect(interfaceConfig.APP_NAME).toBe('Focus Meet')
    expect(interfaceConfig.TOOLBAR_BUTTONS).toEqual(['microphone', 'hangup'])
  })
})
