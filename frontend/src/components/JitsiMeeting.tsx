import { useEffect, useRef } from 'react'
import { JitsiMeeting as JitsiReactMeeting } from '@jitsi/react-sdk'

interface JitsiMeetingProps {
  roomName: string
  jwt: string
  userName?: string
  userEmail?: string
  onJoin?: () => void
  onLeave?: () => void
  onParticipantJoined?: (participant: unknown) => void
  onParticipantLeft?: (participant: unknown) => void
}

export function JitsiMeeting({
  roomName,
  jwt,
  userName,
  userEmail,
  onJoin,
  onLeave,
  onParticipantJoined,
  onParticipantLeft,
}: JitsiMeetingProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  const handleVideoConferenceJoined = () => {
    console.log('User joined the conference')
    onJoin?.()
  }

  const handleVideoConferenceLeft = () => {
    console.log('User left the conference')
    onLeave?.()
  }

  const handleParticipantJoined = (participant: unknown) => {
    console.log('Participant joined:', participant)
    onParticipantJoined?.(participant)
  }

  const handleParticipantLeft = (participant: unknown) => {
    console.log('Participant left:', participant)
    onParticipantLeft?.(participant)
  }

  return (
    <div ref={containerRef} className="jitsi-meeting-container">
      <JitsiReactMeeting
        roomName={roomName}
        jwt={jwt}
        configOverwrite={{
          prejoinPageEnabled: true,
          enableWelcomePage: false,
          disableThirdPartyRequests: true,
          defaultLanguage: 'ru',
          p2p: { enabled: false },
          startWithAudioMuted: false,
          startWithVideoMuted: false,
        }}
        interfaceConfigOverwrite={{
          APP_NAME: 'Focus Messenger',
          SHOW_JITSI_WATERMARK: false,
          SHOW_BRAND_WATERMARK: false,
          TOOLBAR_BUTTONS: [
            'microphone',
            'camera',
            'closedcaptions',
            'desktop',
            'fullscreen',
            'fodeviceselection',
            'hangup',
            'chat',
            'settings',
            'raisehand',
            'videoquality',
            'filmstrip',
          ],
        }}
        userInfo={{
          displayName: userName,
          email: userEmail,
        }}
        onVideoConferenceJoined={handleVideoConferenceJoined}
        onVideoConferenceLeft={handleVideoConferenceLeft}
        onParticipantJoined={handleParticipantJoined}
        onParticipantLeft={handleParticipantLeft}
        style={{ height: '100%', width: '100%' }}
      />
    </div>
  )
}
