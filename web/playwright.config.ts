import { defineConfig, devices } from '@playwright/test'

// Playwright 配置：本地 / CI 共用一份；webServer 自动拉起 vite preview（静态构建产物）。
// 注意：与后端 API 的集成在 server/...e2e_test.go 已覆盖（RustFS 真对端），
// 此处只跑浏览器侧核心流程（登录 → 建账号 → 建桶 → 上传 → 共享）。
export default defineConfig({
  testDir: './e2e',
  // 默认串行：避免多 worker 同时改 localStorage / 后端状态造成相互干扰。
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 2 : 0,
  // 单用例 30s 上限；CI 再宽限到 60s。
  timeout: process.env.CI ? 60_000 : 30_000,
  reporter: process.env.CI ? [['list'], ['github']] : 'list',
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  // 不在 CI 自动拉起 webServer；workflow 显式起 vite preview。
  webServer: process.env.PLAYWRIGHT_NO_WEBSERVER
    ? undefined
    : {
        command: 'pnpm preview --port 4173',
        port: 4173,
        timeout: 60_000,
        reuseExistingServer: !process.env.CI,
      },
})
