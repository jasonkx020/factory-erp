import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  base: '/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@erp/shared': fileURLToPath(new URL('../../packages/shared/src', import.meta.url)),
    },
  },
  optimizeDeps: {
    include: ['pinia', 'vue', 'vue-router', 'element-plus'],
  },
  server: {
    port: 5170,
    host: true,
    proxy: {
      '/api': { target: 'http://127.0.0.1:18080', changeOrigin: true },
    },
  },
})
