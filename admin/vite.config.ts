import process from 'node:process';
import { URL, fileURLToPath } from 'node:url';
import { defineConfig, loadEnv } from 'vite';
import { setupVitePlugins } from './build/plugins';
import { createViteProxy, getBuildTime } from './build/config';

// 容器/远程开发：后端地址可由环境变量覆盖，默认保持 localhost 行为
const ADMIN_API_TARGET = process.env.ADMIN_API_TARGET || 'http://localhost:8088';

export default defineConfig(configEnv => {
  const viteEnv = loadEnv(configEnv.mode, process.cwd()) as unknown as Env.ImportMeta;

  const buildTime = getBuildTime();

  const enableProxy = configEnv.command === 'serve' && !configEnv.isPreview;

  return {
    base: viteEnv.VITE_BASE_URL,
    resolve: {
      alias: {
        '~': fileURLToPath(new URL('./', import.meta.url)),
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: 'modern-compiler',
          additionalData: `@use "@/styles/scss/global.scss" as *;`
        }
      }
    },
    plugins: setupVitePlugins(viteEnv, buildTime),
    define: {
      BUILD_TIME: JSON.stringify(buildTime)
    },
    server: {
      host: '::',
      port: 5174,
      open: false,
      proxy: {
        ...createViteProxy(viteEnv, enableProxy),
        // mira admin API: relative /api/admin/v1 proxied to Go backend (works for localhost AND LAN IP access)
        '/api/admin': {
          target: ADMIN_API_TARGET,
          changeOrigin: true
        },
        // cached local images served by the Go backend (server/static/img)
        '/static': {
          target: ADMIN_API_TARGET,
          changeOrigin: true
        }
      }
    },
    preview: {
      port: 9725
    },
    build: {
      reportCompressedSize: false,
      sourcemap: viteEnv.VITE_SOURCE_MAP === 'Y',
      commonjsOptions: {
        ignoreTryCatch: false
      },
      rollupOptions: {
        output: {
          // Split the heavy vendors so the login screen does not pay for echarts
          // and so vendor hashes stay stable across feature deploys (long cache).
          manualChunks(id: string) {
            if (!id.includes('node_modules')) return;
            if (id.includes('echarts') || id.includes('zrender')) return 'vendor-echarts';
            if (id.includes('naive-ui') || id.includes('vueuc') || id.includes('seemly') || id.includes('css-render')) return 'vendor-naive';
            if (id.includes('/vue/') || id.includes('vue-router') || id.includes('pinia') || id.includes('vue-i18n') || id.includes('@vue')) return 'vendor-vue';
            return 'vendor';
          }
        }
      }
    }
  };
});
