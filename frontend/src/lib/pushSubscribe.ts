import { apiClient } from './apiClient'

/**
 * Управление Web Push-подпиской браузера.
 *
 * Поток:
 *   1. После логина вызывается subscribePush()
 *   2. Получаем VAPID public key с сервера
 *   3. Регистрируем service worker (если ещё не зарегистрирован)
 *   4. Запрашиваем разрешение Notification.requestPermission()
 *   5. Создаём подписку через PushManager.subscribe(...)
 *   6. Отправляем endpoint + ключи на /api/v1/push/register
 *
 * При unsubscribe — отписываемся локально и шлём DELETE/unregister на сервер.
 */

const STORAGE_KEY = 'focus.push.endpoint'

interface VAPIDResponse {
  public_key: string
}

/** Конвертация Base64URL public key в Uint8Array, как требует PushManager.subscribe. */
export function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64)
  const buffer = new ArrayBuffer(raw.length)
  const out = new Uint8Array(buffer)
  for (let i = 0; i < raw.length; ++i) out[i] = raw.charCodeAt(i)
  return out
}

/** Кодирование ArrayBuffer ключей подписки в base64 (без url-safe).
 *  Сервер сохраняет p256dh/auth «как есть» и передаёт в webpush-go,
 *  которая принимает обе формы.
 */
export function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
}

export interface SubscribeOptions {
  /** Если true и подписка уже есть с тем же endpoint — повторно ничего не делаем. */
  skipIfAlreadyRegistered?: boolean
}

export interface SubscribeResult {
  status: 'subscribed' | 'denied' | 'unsupported' | 'no-vapid' | 'error'
  endpoint?: string
  error?: unknown
}

/** Поддерживается ли Web Push в этом окружении. */
export function isPushSupported(): boolean {
  if (typeof window === 'undefined') return false
  if (!('serviceWorker' in navigator)) return false
  if (!('PushManager' in window)) return false
  if (!('Notification' in window)) return false
  return true
}

/** Подписаться на push (идемпотентно). Возвращает результат. */
export async function subscribePush(opts: SubscribeOptions = {}): Promise<SubscribeResult> {
  if (!isPushSupported()) {
    return { status: 'unsupported' }
  }

  let publicKey = ''
  try {
    const data = await apiClient.get<VAPIDResponse>('/api/v1/push/vapid-public-key')
    publicKey = (data.public_key || '').trim()
  } catch (err) {
    return { status: 'no-vapid', error: err }
  }
  if (!publicKey) {
    return { status: 'no-vapid' }
  }

  const reg = await navigator.serviceWorker.ready

  let permission = Notification.permission
  if (permission === 'default') {
    permission = await Notification.requestPermission()
  }
  if (permission !== 'granted') {
    return { status: 'denied' }
  }

  let sub = await reg.pushManager.getSubscription()
  if (sub) {
    if (opts.skipIfAlreadyRegistered) {
      const stored = window.localStorage.getItem(STORAGE_KEY)
      if (stored === sub.endpoint) {
        return { status: 'subscribed', endpoint: sub.endpoint }
      }
    }
    // Удалим старую и подпишемся заново — так гарантируем актуальный VAPID.
    await sub.unsubscribe().catch(() => undefined)
  }

  try {
    sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(publicKey),
    })
  } catch (err) {
    return { status: 'error', error: err }
  }

  const json = sub.toJSON()
  const p256dh = json?.keys?.p256dh ?? ''
  const auth = json?.keys?.auth ?? ''

  try {
    await apiClient.post('/api/v1/push/register', {
      platform: 'web',
      endpoint: sub.endpoint,
      p256dh,
      auth,
      user_agent: navigator.userAgent,
      locale: navigator.language,
    })
    window.localStorage.setItem(STORAGE_KEY, sub.endpoint)
    return { status: 'subscribed', endpoint: sub.endpoint }
  } catch (err) {
    return { status: 'error', error: err }
  }
}

/** Отписаться от push на этом устройстве. */
export async function unsubscribePush(): Promise<void> {
  if (!isPushSupported()) return
  const reg = await navigator.serviceWorker.ready
  const sub = await reg.pushManager.getSubscription()
  if (!sub) return
  try {
    await apiClient.post('/api/v1/push/unregister', { endpoint: sub.endpoint })
  } catch {
    // serverside cleanup best-effort
  }
  await sub.unsubscribe().catch(() => undefined)
  window.localStorage.removeItem(STORAGE_KEY)
}
