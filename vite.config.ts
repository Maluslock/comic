import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

export default defineConfig({
  plugins: [uni()],
  server: {
    host: '::', // bind all interfaces incl. IPv6 (dual-stack)
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8088',
        changeOrigin: true,
      },
      '/static/uploads': {
        target: 'http://127.0.0.1:8088',
        changeOrigin: true,
      },
      '/static/remote': {
        target: 'http://127.0.0.1:8088',
        changeOrigin: true,
      }
    }
  },
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: '@import "@/styles/variables.scss";'
      }
    }
  }
})
