import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@tauri-apps/api/event': path.resolve(__dirname, 'src/tests/tauriStub.ts'),
      '@tauri-apps/api/core': path.resolve(__dirname, 'src/tests/tauriStub.ts'),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/tests/setup.ts',
    coverage: {
      reporter: ['text', 'json', 'html'],
    },
  },
})
