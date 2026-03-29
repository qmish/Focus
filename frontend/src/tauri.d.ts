declare module '@tauri-apps/api/core' {
  export function invoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T>
}

declare module '@tauri-apps/api/event' {
  export interface Event<T> {
    payload: T
  }
  export function listen<T>(event: string, handler: (event: Event<T>) => void): Promise<() => void>
  export function emit(event: string, payload?: unknown): Promise<void>
}
