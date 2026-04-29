import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  urlBase64ToUint8Array,
  arrayBufferToBase64,
  isPushSupported,
  subscribePush,
  unsubscribePush,
} from './pushSubscribe'

vi.mock('./apiClient', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))
import { apiClient } from './apiClient'

describe('pushSubscribe utils', () => {
  it('urlBase64ToUint8Array decodes web-safe base64', () => {
    const out = urlBase64ToUint8Array('AQID')
    expect(Array.from(out)).toEqual([1, 2, 3])
  })

  it('urlBase64ToUint8Array handles missing padding', () => {
    const out = urlBase64ToUint8Array('AQIDBA') // length 6, padding=2
    expect(Array.from(out)).toEqual([1, 2, 3, 4])
  })

  it('arrayBufferToBase64 round-trip', () => {
    const arr = new Uint8Array([10, 20, 30, 40]).buffer
    const b64 = arrayBufferToBase64(arr)
    const decoded = atob(b64)
    expect(decoded.charCodeAt(0)).toBe(10)
    expect(decoded.charCodeAt(3)).toBe(40)
  })
})

describe('isPushSupported', () => {
  it('returns false when ServiceWorker missing', () => {
    const original = (global as any).navigator
    ;(global as any).navigator = {}
    expect(isPushSupported()).toBe(false)
    ;(global as any).navigator = original
  })
})

describe('subscribePush', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(global as any).window = global
    ;(global as any).Notification = class {
      static permission = 'granted'
      static requestPermission = vi.fn().mockResolvedValue('granted')
    }
    ;(global as any).PushManager = class {}
    ;(global as any).navigator = {
      userAgent: 'test',
      language: 'ru',
      serviceWorker: {
        ready: Promise.resolve({
          pushManager: {
            getSubscription: vi.fn().mockResolvedValue(null),
            subscribe: vi.fn().mockResolvedValue({
              endpoint: 'https://push.example/endpoint',
              toJSON: () => ({
                endpoint: 'https://push.example/endpoint',
                keys: { p256dh: 'p256dh-key', auth: 'auth-key' },
              }),
              unsubscribe: vi.fn().mockResolvedValue(true),
            }),
          },
        }),
      },
    }
    ;(global as any).window.localStorage = {
      data: new Map<string, string>(),
      getItem(k: string) {
        return this.data.get(k) ?? null
      },
      setItem(k: string, v: string) {
        this.data.set(k, v)
      },
      removeItem(k: string) {
        this.data.delete(k)
      },
    }
  })

  afterEach(() => {
    delete (global as any).Notification
    delete (global as any).PushManager
  })

  it('returns "subscribed" and posts to /push/register', async () => {
    ;(apiClient.get as any).mockResolvedValue({ public_key: 'AQIDBA' })
    ;(apiClient.post as any).mockResolvedValue(undefined)

    const result = await subscribePush()
    expect(result.status).toBe('subscribed')
    expect(result.endpoint).toBe('https://push.example/endpoint')
    expect(apiClient.post).toHaveBeenCalledWith(
      '/api/v1/push/register',
      expect.objectContaining({
        platform: 'web',
        endpoint: 'https://push.example/endpoint',
        p256dh: 'p256dh-key',
        auth: 'auth-key',
      })
    )
  })

  it('returns "no-vapid" when server has no key configured', async () => {
    ;(apiClient.get as any).mockRejectedValue(new Error('503'))
    const result = await subscribePush()
    expect(result.status).toBe('no-vapid')
  })

  it('returns "denied" when permission rejected', async () => {
    ;(apiClient.get as any).mockResolvedValue({ public_key: 'AQIDBA' })
    ;(global as any).Notification.permission = 'default'
    ;(global as any).Notification.requestPermission = vi.fn().mockResolvedValue('denied')

    const result = await subscribePush()
    expect(result.status).toBe('denied')
  })
})

describe('unsubscribePush', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(global as any).window = global
    ;(global as any).Notification = class {
      static permission = 'granted'
    }
    ;(global as any).PushManager = class {}
    ;(global as any).window.localStorage = {
      data: new Map<string, string>([
        ['focus.push.endpoint', 'https://push.example/endpoint'],
      ]),
      getItem(k: string) {
        return this.data.get(k) ?? null
      },
      removeItem(k: string) {
        this.data.delete(k)
      },
    }
    ;(global as any).navigator = {
      serviceWorker: {
        ready: Promise.resolve({
          pushManager: {
            getSubscription: vi.fn().mockResolvedValue({
              endpoint: 'https://push.example/endpoint',
              unsubscribe: vi.fn().mockResolvedValue(true),
            }),
          },
        }),
      },
    }
    ;(apiClient.post as any).mockResolvedValue(undefined)
  })

  afterEach(() => {
    delete (global as any).Notification
    delete (global as any).PushManager
  })

  it('calls /push/unregister and clears local cache', async () => {
    await unsubscribePush()
    expect(apiClient.post).toHaveBeenCalledWith(
      '/api/v1/push/unregister',
      expect.objectContaining({ endpoint: 'https://push.example/endpoint' })
    )
    expect((global as any).window.localStorage.getItem('focus.push.endpoint')).toBeNull()
  })
})
