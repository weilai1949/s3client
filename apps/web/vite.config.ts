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
      provider: 'istanbul',
      reporter: ['text', 'text-summary', 'html', 'lcov'],
      include: ['src/**/*.{ts,vue}'],
      exclude: [
        'src/main.ts',
        'src/env.d.ts',
        'src/**/*.test.ts',
        // i18n 曾整体排除（字典庞大、纯数据）。覆盖率去水分（见 docs/features.md §M）：
        // 字典数据由 coverage.test.ts 的键完整性门禁覆盖，故仅排除纯数据模块；
        // index.ts（读/写/回退/循环逻辑）纳入统计。
        'src/i18n/messages/**',
        'src/assets/**',
      ],
      // 门槛：四指标均已达成 100%，设为 100 作为回归护栏
      // （防新增业务代码无测试回落）。
      thresholds: {
        statements: 100,
        functions: 100,
        branches: 100,
        lines: 100,
      },
    },
  },
})
