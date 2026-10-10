import { expect, test, type Page } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

/**
 * 自动化无障碍（a11y）审计：用 axe-core 在真实浏览器里扫真实构建产物。
 *
 * 为什么需要它：`src/a11y_gate.test.ts` 只能做**源码形态**断言（媒体查询是否存在、
 * `:focus-visible` 列表是否含 `textarea`）——它证明不了「渲染出来的 DOM 真的可访问」。
 * 2026-09-29 修掉的三条缺陷（KNOWN_ISSUES #67）就是靠人读代码发现的。本 spec 补上
 * **渲染态**的机械探测，两者互补（见 docs/accessibility.md）。
 *
 * 判定口径：只把 axe 的 **serious / critical** 违规判为失败——moderate / minor 违规
 * 在这些面板上仍会打印出来供人工判断，但不阻塞 CI（避免为了规则洁癖而伪装通过或
 * 加一堆无依据的屏蔽项）。任何**新增的** serious / critical 违规都会让 CI 红灯点名。
 *
 * 空跑防护：`axe 有效性自检` 用例故意注入一个必然违规的 DOM，断言 axe 真的能报出来——
 * 否则「0 违规」可能只是扫描器没生效。
 *
 * 界面状态：前四态只桩 `/api/health` 与 `/api/accounts`（口径同 screenshots.spec.ts），
 * 因此扫的是「全新安装、尚未添加账号」的真实初始态，不掺入「无法连接后端」这类环境噪声；
 * **网格视图**一态另用 `installObjectsStub` 桩出账号 + 对象（否则进不了对象面板）。
 */

test.use({ viewport: { width: 1440, height: 900 }, reducedMotion: 'reduce' })

/** 阻塞级影响面：serious / critical 一律不得出现。 */
const BLOCKING_IMPACTS = new Set(['serious', 'critical'])

/** axe 规则集：WCAG 2.0 / 2.1 的 A + AA。 */
const WCAG_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']

/** 把违规压成一行，作为断言失败时的可读证据。 */
function formatViolations(violations: { id: string; impact?: string | null; help: string; nodes: unknown[] }[]): string {
  return violations
    .map((v) => `  [${v.impact ?? 'unknown'}] ${v.id}: ${v.help}（${v.nodes.length} 个节点）`)
    .join('\n')
}

/**
 * 桩：健康检查通过 + 账号列表为空（形状逐字取自 docs/api.md）。
 * 网格态用例另用 installObjectsStub（见下方）。
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
    if (route.request().method() !== 'GET') return route.continue()
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ accounts: [] }) })
  })
}

/**
 * 桩：一个带默认桶的账号 + 三个对象（形状逐字取自 `model.AccountView` 与 `docs/api.md` 的
 * `GET /api/accounts/{id}/objects`）。有账号且账号带默认桶时，应用会直接进对象面板并列出对象
 * ——这是进入**网格视图**的最短路径（口径同 `features.spec.ts`）。
 */
async function installObjectsStub(page: Page): Promise<void> {
  const account = {
    id: 'acc-1',
    name: 'account-1',
    endpoint: 'http://127.0.0.1:9000',
    publicEndpoint: '',
    region: 'us-east-1',
    accessKey: 'AKIAEXAMPLE',
    secretSet: true,
    bucket: 'photos',
    pathStyle: true,
    useSSL: false,
    createdAt: '2024-01-02T00:00:00.000Z',
    updatedAt: '2024-01-02T00:00:00.000Z',
  }
  const objects = [
    { key: 'img/logo.png', size: 2048, lastModified: '2024-03-01T10:00:00.000Z', etag: 'e1', contentType: 'image/png', storageClass: 'STANDARD', isDir: false },
    { key: 'data.csv', size: 120, lastModified: '2024-03-02T10:00:00.000Z', etag: 'e2', contentType: 'text/csv', storageClass: 'STANDARD', isDir: false },
    { key: 'readme.txt', size: 14, lastModified: '2024-03-03T10:00:00.000Z', etag: 'e3', contentType: 'text/plain', storageClass: 'STANDARD', isDir: false },
  ]
  await page.route('**/api/health', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'ok', version: 'v1.0.0', time: new Date().toISOString(), store: { ok: true } }),
    }),
  )
  await page.route((url) => url.pathname === '/api/accounts', (route) => {
    if (route.request().method() !== 'GET') return route.continue()
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ accounts: [account] }) })
  })
  await page.route((url) => url.pathname === '/api/accounts/acc-1/buckets', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ buckets: [{ name: 'photos', creationDate: '2024-01-02T00:00:00.000Z' }] }),
    }),
  )
  await page.route((url) => url.pathname === '/api/accounts/acc-1/bucket-info', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ bucket: 'photos', region: 'us-east-1', createdAt: '2024-01-02T00:00:00.000Z', versioning: '' }),
    }),
  )
  await page.route((url) => url.pathname === '/api/accounts/acc-1/objects', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ objects, commonPrefixes: ['img/'], isTruncated: false, nextToken: '' }),
    }),
  )
}

/** 打开首屏并确认「渲染成功、无后端错误横幅」，再返回。 */
async function openClean(page: Page): Promise<void> {
  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await expect(page.getByText(/无法连接后端|Service Unavailable/)).toHaveCount(0)
  await page.waitForLoadState('networkidle')
}

/**
 * 对当前页面跑一次 axe 扫描：阻塞级违规 → 失败；同时做「扫描面非空」自检，
 * 防止 axe 没注入成功时静默全绿。
 */
async function scanPage(page: Page, label: string): Promise<void> {
  const results = await new AxeBuilder({ page }).withTags(WCAG_TAGS).analyze()
  // 自检：axe 引擎确实跑了、且确实检查了东西。
  expect(results.testEngine.name).toBe('axe-core')
  expect(results.passes.length, `${label}：axe 没有产出任何通过项，扫描面疑似为空`).toBeGreaterThan(5)

  const blocking = results.violations.filter((v) => BLOCKING_IMPACTS.has(v.impact ?? ''))
  // moderate / minor 不阻塞 CI，但按本文件口径**必须打印出来供人工判断**——
  // 静默丢弃会让它们无限累积，「0 阻塞」也就分不清是真干净还是没人看。
  const nonBlocking = results.violations.filter((v) => !BLOCKING_IMPACTS.has(v.impact ?? ''))
  if (nonBlocking.length > 0) {
    console.log(`${label}：${nonBlocking.length} 个非阻塞级违规（不阻塞 CI）：\n${formatViolations(nonBlocking)}`)
  }
  expect(
    blocking,
    `${label}：出现 ${blocking.length} 个 serious/critical 无障碍违规：\n${formatViolations(blocking)}`,
  ).toEqual([])
}

test('a11y：账号管理初始态（无 serious/critical 违规）', async ({ page }) => {
  await installMinimalStub(page)
  await openClean(page)
  // 断言落在账号面板（空态引导可见），确认扫的是目标界面而不是加载中骨架。
  await expect(page.getByText(/还没有登录配置|新增登录/).first()).toBeVisible()
  await scanPage(page, '账号管理初始态')
})

test('a11y：新增登录对话框（无 serious/critical 违规）', async ({ page }) => {
  await installMinimalStub(page)
  await openClean(page)
  await page.getByRole('button', { name: /新增登录|Add login/ }).click()
  const dlg = page.getByRole('dialog', { name: /新增登录|Add login/ })
  await expect(dlg).toBeVisible()
  // 模态是 a11y 高风险面：焦点陷阱 / aria-modal / 标签关联都只在打开时才暴露。
  await scanPage(page, '新增登录对话框')
})

test('a11y：服务器设置面板（无 serious/critical 违规）', async ({ page }) => {
  await installMinimalStub(page)
  await openClean(page)
  await page
    .getByRole('button', { name: /服务器设置|^Server$/ })
    .first()
    .click()
  await expect(page.getByRole('button', { name: /全部检测|Probe all/ })).toBeVisible()
  await page.waitForLoadState('networkidle')
  await scanPage(page, '服务器设置面板')
})

test('a11y：对象网格视图（无 serious/critical 违规）', async ({ page }) => {
  // A4（2026-10-10）：accessibility §4.9 记「网格单元格 role=button、名称靠文本节点派生，
  // 未加 aria-label」，但 axe 此前**四个扫描态都不含网格视图**——网格态从没被扫过，
  // 真伪未判。本用例补上该状态，先让 axe 判定（TDD：先红后判），成立才改组件。
  await installObjectsStub(page)
  await openClean(page)
  // 确认真的进了对象面板并列出对象（网格视图只在对象面板里存在）。
  await expect(page.locator('tbody')).toContainText('readme.txt')
  // 切到网格视图：该按钮的可访问名来自 aria-label（切换视图 / Toggle view）。
  await page.getByRole('button', { name: /切换视图|Toggle view/ }).click()
  await expect(page.locator('.grid-item').first()).toBeVisible()
  await scanPage(page, '对象网格视图')
})

test('a11y：深色主题初始态（无 serious/critical 违规）', async ({ page }) => {
  // 主题默认 auto → 跟随系统；`data-theme` 由 theme.ts 在模块初始化时写入。
  // 深色下语义色是另一套 token，对比度必须单独验证（浅色通过不代表深色通过）。
  await page.emulateMedia({ colorScheme: 'dark' })
  await installMinimalStub(page)
  await openClean(page)
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await scanPage(page, '深色主题初始态')
})

test('axe 自身有效性自检：注入已知违规必须被报出', async ({ page }) => {
  // 防「扫描器没生效 → 永远 0 违规」的空跑：主动制造一个 image-alt 违规。
  await page.setContent('<main><h1>probe</h1><img src="data:image/gif;base64,R0lGODlhAQABAAAAACw="></main>')
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze()
  expect(
    results.violations.map((v) => v.id),
    'axe 未报出注入的 image-alt 违规——扫描器/规则集失效，其它用例的「0 违规」不可信',
  ).toContain('image-alt')
})
