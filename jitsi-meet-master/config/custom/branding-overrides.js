/*
 * Focus corporate branding baseline for Jitsi fork.
 * Apply from upstream config.js / interface_config.js according to deployment method.
 */
module.exports = {
  dynamicBrandingUrl: '/api/v1/branding/jitsi',
  customTheme: {
    palette: {
      ui01: '#0B1220',
      ui02: '#111827',
      action01: '#0EA5E9',
      text01: '#F9FAFB',
    },
  },
  customIcons: {
    mic: '/pics/image16.png',
    camera: '/pics/image17.png',
    hangup: '/pics/image16.png',
    participants: '/pics/image17.png',
  },
  // Policy-driven disables for corporate deployment.
  toolbarButtons: [
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
  ],
  disabledNotifications: ['dialog.kickTitle', 'dialog.kickMessage'],
}
