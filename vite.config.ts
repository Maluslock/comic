import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

// Container/dev override: 容器内后端地址与宿主机不同，默认保持 localhost 行为不变
const API_TARGET = process.env.VITE_API_TARGET || 'http://127.0.0.1:8088'

export default defineConfig({
  plugins: [uni()],
  server: {
    host: '::', // bind all interfaces incl. IPv6 (dual-stack)
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
      },
      '/static/uploads': {
        target: API_TARGET,
        changeOrigin: true,
      },
      '/static/remote': {
        target: API_TARGET,
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
