import { JitsiMeeting as JitsiReactMeeting } from '@jitsi/react-sdk'
import { JITSI_DOMAIN } from '../lib/config'

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
}

export function JitsiMeeting({
  roomName,
  jwt,
  domain,
  branding,
  userName,
  userEmail,
  onLeave,
}: JitsiMeetingProps) {
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

  return (
    <div className="jitsi-meeting-container">
      <JitsiReactMeeting
        domain={domain || JITSI_DOMAIN}
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
          displayName: userName || '',
          email: userEmail || '',
        }}
        onApiReady={(api) => {
          api.addEventListener('videoConferenceLeft', () => onLeave?.())
        }}
        onReadyToClose={() => onLeave?.()}
        getIFrameRef={(node) => {
          node.style.height = '100%'
          node.style.width = '100%'
        }}
      />
    </div>
  )
}
