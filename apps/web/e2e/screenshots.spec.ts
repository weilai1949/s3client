import { expect, test, type Page } from '@playwright/test'
import { mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

/**
 * README 截图生成（`docs/images/*.png`）。
 *
 * 为什么放在 e2e/ 而不另起一套：`e2e/**` 已被 `tsconfig.e2e.json` 与 `eslint` 覆盖，
 * 放这里**零配置改动**即可纳入类型检查与 lint 门禁。
 *
 * 为什么只 mock 两个端点（而不复用 features.spec.ts 的大 fake）：
 * 直接跑 vite preview 时 `/api/*` 全部失败，界面会挂出红色的「无法连接后端：503」横幅——
 * 那是**环境产物，不是产品长相**，拿它当 README 配图是误导。这里只需让
 * `/api/health` 与 `/api/accounts` 返回健康 / 空列表，界面即为「全新安装、尚未添加账号」
 * 的真实初始态，且无需维护一整套 in-memory 后端。
 *
 * 截图是**副产物**：用例先断言界面确实渲染成功（含「错误横幅不存在」），再截图——
 * 所以它在 CI 里是一个有效的冒烟用例，而不是「只为产出图片」的空跑。
 *
 * 重新生成（需先 `pnpm build`）：
 *   cd apps/web && pnpm exec playwright test screenshots.spec.ts
 * 本地开发态：
 *   PLAYWRIGHT_BASE_URL=http://127.0.0.1:5173 pnpm exec playwright test screenshots.spec.ts
 */

// apps/web/e2e → apps/web → apps → 仓库根
const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..', '..')
const OUT_DIR = join(REPO_ROOT, 'docs', 'images')

/**
 * 固定视口，保证每次产出尺寸一致（否则图片 diff 全是噪声）。
 *
 * `reducedMotion: 'reduce'` 是 2026-09-29 起的**关键**设置：应用此时已响应
 * `prefers-reduced-motion`（`styles.css` 末尾的媒体查询，由 `src/a11y_gate.test.ts` 钉住），
 * 于是面板淡入 / 模态过渡被关闭，截图拿到的是**稳定终态**而非过渡中间态。
 * 在此之前这里靠一个 `waitForTimeout(500)` 硬等——那既是 flaky 来源（机器慢就截到按钮
 * 半透明、文字重叠），也掩盖了「应用当时不尊重减少动效偏好」这一事实（KNOWN_ISSUES #67②）。
 */
test.use({ viewport: { width: 1440, height: 900 }, reducedMotion: 'reduce' })

test.beforeAll(() => {
  mkdirSync(OUT_DIR, { recursive: true })
})

/**
 * 挂最小桩：健康检查通过 + 账号列表为空。
 * 形状逐字取自 docs/api.md（`/api/health` 与 `AccountView` 列表）。
 */
async function installMinimalStub(page: Page): Promise<void> {
  await page.route('**/api/health', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'ok', version: 'v1.0.0', time: new Date().toISOString(), store: { ok: true } }),
    }),
  )
  await page.route('**/api/accounts', (route) => {
    // 只桩 GET 列表；其它方法（本 spec 不触发）原样放行，避免掩盖真实问题。
    if (route.request().method() !== 'GET') return route.continue()
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ accounts: [] }) })
  })
}

/** 打开页面并等待首屏稳定；先断言「没有后端错误横幅」再返回。 */
async function openClean(page: Page): Promise<void> {
  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await expect(page.locator('aside, nav').first()).toContainText(/账号|Accounts/)
  // 关键断言：配图里不允许出现「无法连接后端」这类环境噪声。
  await expect(page.getByText(/无法连接后端|Service Unavailable/)).toHaveCount(0)
  await page.waitForLoadState('networkidle')
}

test('截图：账号管理（全新安装的空态）', async ({ page }) => {
  await installMinimalStub(page)
  await openClean(page)
  // 默认即落在账号管理面板；断言空态引导文案存在，确认截到的是「引导态」而非加载中。
  await expect(page.getByText(/还没有登录配置|新增登录/).first()).toBeVisible()
  await page.screenshot({ path: join(OUT_DIR, 'accounts-panel.png') })
})

test('截图：服务器设置面板', async ({ page }) => {
  await installMinimalStub(page)
  await openClean(page)
  await page
    .getByRole('button', { name: /服务器设置|^Server$/ })
    .first()
    .click()
  // 断言**目标面板独有的控件**（`server.probeAll` = 全部检测）已出现：
  // 既证明导航真的发生了（只点侧栏按钮时曾截到「没切过去」的图），
  // 也证明面板内容已挂载。
  await expect(page.getByRole('button', { name: /全部检测|Probe all/ })).toBeVisible()
  await page.waitForLoadState('networkidle')
  // 不再需要 `waitForTimeout(500)` 等淡入动画：`reducedMotion: 'reduce'`（见文件上方
  // `test.use`）已让应用关闭过渡，这里拿到的是终态。
  await page.screenshot({ path: join(OUT_DIR, 'server-panel.png') })
})
