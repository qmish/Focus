import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { JitsiMeeting } from './JitsiMeeting'

vi.mock('@jitsi/react-sdk', () => ({
  JitsiMeeting: (props: {
    onVideoConferenceJoined?: () => void
    onVideoConferenceLeft?: () => void
    onParticipantJoined?: (participant: unknown) => void
    onParticipantLeft?: (participant: unknown) => void
  }) => (
    <div>
      <button onClick={() => props.onVideoConferenceJoined?.()}>join</button>
      <button onClick={() => props.onVideoConferenceLeft?.()}>leave</button>
      <button onClick={() => props.onParticipantJoined?.({ id: 'p1' })}>participant-join</button>
      <button onClick={() => props.onParticipantLeft?.({ id: 'p1' })}>participant-leave</button>
    </div>
  ),
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
})
