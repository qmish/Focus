import { useEffect, useRef } from 'react'
import { JitsiMeeting as JitsiReactMeeting } from '@jitsi/react-sdk'

type BrandingConfig = {
  appName?: string
  defaultLanguage?: string
  dynamicBrandingUrl?: string
  customTheme?: Record<string, string>
  customIcons?: Record<string, string>
  toolbarButtons?: string[]
}

interface JitsiMeetingProps {
  roomName: string
  jwt: string
  domain?: string
  branding?: BrandingConfig
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
  domain,
  branding,
  userName,
  userEmail,
  onJoin,
  onLeave,
  onParticipantJoined,
  onParticipantLeft,
}: JitsiMeetingProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const toolbarButtons = branding?.toolbarButtons ?? [
    'microphone',
    'camera',
    'desktop',
    'fullscreen',
    'fodeviceselection',
    'hangup',
    'chat',
    'participants-pane',
    'raisehand',
    'tileview',
    'settings',
    'select-background',
    'videoquality',
  ]

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
        domain={domain || 'meet.company.com'}
        roomName={roomName}
        jwt={jwt}
        configOverwrite={{
          prejoinPageEnabled: true,
          enableWelcomePage: false,
          disableThirdPartyRequests: true,
          defaultLanguage: branding?.defaultLanguage || 'ru',
          dynamicBrandingUrl: branding?.dynamicBrandingUrl,
          customTheme: branding?.customTheme,
          customIcons: branding?.customIcons,
          p2p: { enabled: false },
          startWithAudioMuted: false,
          startWithVideoMuted: false,
        }}
        interfaceConfigOverwrite={{
          APP_NAME: branding?.appName || 'Focus Messenger',
          SHOW_JITSI_WATERMARK: false,
          SHOW_BRAND_WATERMARK: false,
          TOOLBAR_BUTTONS: toolbarButtons,
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
