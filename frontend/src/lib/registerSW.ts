/**
 * Регистрация service worker для PWA / Web Push.
 * vite-plugin-pwa в режиме `injectManifest` создаёт виртуальный модуль
 * `virtual:pwa-register`, через который мы автоматически регистрируем SW
 * и обновляем его при появлении новой версии.
 */
export async function registerSW(): Promise<void> {
  if (typeof window === 'undefined') return
  if (!('serviceWorker' in navigator)) return

  // В тестовом окружении (vitest/jsdom) virtual-модуль недоступен — пропускаем.
  // import.meta.env.MODE === 'test' для vitest
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  if (import.meta.env?.MODE === 'test') return

  try {
    // Динамический импорт, чтобы тесты и Tauri-сборка не падали из-за virtual-модуля.
    const mod = await import(
      /* @vite-ignore */
      'virtual:pwa-register'
    ).catch(() => null)
    if (mod && typeof mod.registerSW === 'function') {
      mod.registerSW({ immediate: true })
    }
  } catch (err) {
    console.warn('SW registration skipped:', err)
  }
}
