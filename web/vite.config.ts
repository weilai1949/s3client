/// <reference types="vitest/config" />
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

// 前端构建。输出到 dist，Go 后端把 dist 作为静态资源托管。
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 1949,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  test: {
    environment: 'happy-dom',
    exclude: ['node_modules/**', 'e2e/**', 'dist/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary', 'html', 'lcov'],
      include: ['src/**/*.{ts,vue}'],
      exclude: [
        'src/main.ts',
        'src/env.d.ts',
        'src/**/*.test.ts',
        'src/i18n/**',
        'src/assets/**',
      ],
      // 当前基线极低（全局 ~11%，组件几乎为 0%）。此处不设全局门槛，
      // 以免 CI 一接入就红；等组件/ composables 单测站上来后再逐步抬门槛。
    },
  },
})
