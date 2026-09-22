import { defineConfig, devices } from '@playwright/test'

/**
 * 真实联调（todolist #37）专用 Playwright 配置。
 *
 * 与 `playwright.config.ts`（mock `/api/**` 的浏览器侧用例）分开的原因：
 *   - **被测对象不同**：这里必须打到真实 Go 后端 + 真实 RustFS，任何
 *     `page.route()` mock 都会让结论失真；
 *   - **环境由外部提供**：后端（托管真实 `vite build` 产物）与 RustFS 由
 *     `scripts/e2e-real.sh` 或 CI job 拉起，本配置**不**自拉 webServer
 *     （vite preview 无法提供 /api，真实后端才能）；
 *   - **串行执行**：账号 / 桶 / 对象都写在共享的后端 store 与同一个 RustFS 上，
 *     并行会互相干扰。
 *
 * `PLAYWRIGHT_BASE_URL` 缺省指向真实后端 http://127.0.0.1:8080。
 */
export default defineConfig({
  testDir: './e2e-real',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 2 : 0,
  timeout: process.env.CI ? 90_000 : 45_000,
  // 真实网络 + 容器启动比 mock 慢，给足断言超时。
  expect: { timeout: 15_000 },
  reporter: process.env.CI ? [['list'], ['github']] : 'list',
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:8080',
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
  // 真实后端由外部脚本/CI job 提供，Playwright 不自拉 webServer。
})
