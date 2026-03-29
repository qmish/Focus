const isTauri = typeof window !== 'undefined' && ('__TAURI__' in window || '__TAURI_INTERNALS__' in window)

let _baseUrl = ''

if (isTauri) {
  _baseUrl = import.meta.env.VITE_API_URL
    || localStorage.getItem('focus_api_base_url')
    || 'https://chat.focus.local:30443'
}

export function getApiBaseUrl(): string {
  return _baseUrl
}

export function setApiBaseUrl(url: string) {
  _baseUrl = url.replace(/\/+$/, '')
  localStorage.setItem('focus_api_base_url', _baseUrl)
}

export function getWsBaseUrl(): string {
  if (!_baseUrl) return ''
  const url = new URL(_baseUrl)
  const protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${url.host}`
}
