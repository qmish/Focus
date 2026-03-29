import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  root: path.resolve(__dirname, '../frontend'),
  envDir: path.resolve(__dirname),
  plugins: [react()],
  resolve: {
    alias: {
      '@tauri-apps/api': path.resolve(__dirname, 'node_modules/@tauri-apps/api'),
      '@tauri-apps/plugin-notification': path.resolve(__dirname, 'node_modules/@tauri-apps/plugin-notification'),
      '@tauri-apps/plugin-shell': path.resolve(__dirname, 'node_modules/@tauri-apps/plugin-shell'),
      '@tauri-apps/plugin-store': path.resolve(__dirname, 'node_modules/@tauri-apps/plugin-store'),
      '@tauri-apps/plugin-deep-link': path.resolve(__dirname, 'node_modules/@tauri-apps/plugin-deep-link'),
      '@tauri-apps/plugin-autostart': path.resolve(__dirname, 'node_modules/@tauri-apps/plugin-autostart'),
      '@tauri-apps/plugin-updater': path.resolve(__dirname, 'node_modules/@tauri-apps/plugin-updater'),
    },
  },
  build: {
    outDir: path.resolve(__dirname, 'dist'),
    emptyOutDir: true,
  },
  server: {
    port: 1420,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
