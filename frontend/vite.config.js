import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@locales': path.resolve(__dirname, '../locales'),
      '@docs': path.resolve(__dirname, '../docs')
    }
  },
  server: {
    fs: {
      allow: [path.resolve(__dirname, '..')]
    },
    port: 3000,
    open: true,
    proxy: {
      '/api': {
        target: process.env.VITE_GATEWAY_PROXY_TARGET || 'http://127.0.0.1:3001',
        changeOrigin: true,
        secure: false
      }
    }
  }
})
