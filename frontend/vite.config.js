import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

const gatewayTarget = process.env.VITE_GATEWAY_PROXY_TARGET || 'http://127.0.0.1:3000'

const proxyToGateway = {
  target: gatewayTarget,
  changeOrigin: true,
  secure: false,
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@locales': path.resolve(__dirname, '../locales'),
      '@docs': path.resolve(__dirname, '../docs'),
      /** Shipped benchmark JSON for /docs/benchmarks (see docs/bundled-benchmarks/README.md) */
      '@data': path.resolve(__dirname, '../docs/bundled-benchmarks'),
    },
  },
  server: {
    fs: {
      allow: [path.resolve(__dirname, '..')],
    },
    // Canonical local UI port; Docker gateway listens on 3000 — see repo README / installation.md
    port: 5173,
    open: true,
    proxy: {
      '/api': proxyToGateway,
      '/health': proxyToGateway,
      '/ready': proxyToGateway,
      '/metrics': proxyToGateway,
    },
  },
})
