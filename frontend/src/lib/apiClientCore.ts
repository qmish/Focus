export class ApiError extends Error {
  status: number
  body: unknown

  constructor(message: string, status: number, body: unknown) {
    super(message)
    this.status = status
    this.body = body
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  token?: string | null
  retry?: number
  headers?: Record<string, string>
}

export async function apiRequest<T>(url: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, token, retry = method === 'GET' ? 1 : 0, headers = {} } = options

  let attempt = 0
  while (true) {
    attempt++
    try {
      const response = await fetch(url, {
        method,
        headers: {
          ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
          ...headers,
        },
        ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
      })

      if (!response.ok) {
        const text = await response.text()
        if (method === 'GET' && attempt <= retry + 1 && response.status >= 500) {
          continue
        }
        throw new ApiError(text || `Ошибка сервера (${response.status})`, response.status, text)
      }

      if (response.status === 204) {
        return undefined as unknown as T
      }

      if (typeof response.json !== 'function') {
        return undefined as unknown as T
      }

      return await response.json() as T
    } catch (error) {
      if (method === 'GET' && attempt <= retry + 1) {
        continue
      }
      throw error
    }
  }
}
