import { expect, test } from './fixture-server'

/**
 * 账号管理流程：模拟服务端响应，验证前端能正确渲染空状态 → 新增 → 列表。
 *
 * 这是浏览器侧对前端 + 后端契约的最小闭环测试；不依赖真实 S3。
 * 真实 S3 端到端由 apps/server/internal/s3wrap/e2e_test.go 覆盖。
 *
 * 使用 page.route() 拦截 /api/* 请求并回放 fixture 响应；这样本测试不依赖真实后端，
 * 任何能跑 vite preview 的环境都能运行。
 *
 * 注意：侧边栏入口是 App.vue 里的 `<button>`（不是 `<a>`），所以用 role=button 定位。
 * 入口找不到就直接失败——不要用「不可见就 test.skip」把定位器写错伪装成跳过。
 */
test('账号管理：空状态 → 新增 → 列表', async ({ page }) => {
  const accounts: { id: string; name: string; endpoint: string }[] = []
  let posted: Record<string, unknown> | undefined

  await page.route('**/api/accounts', async (route) => {
    const req = route.request()
    if (req.method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ accounts }),
      })
      return
    }
    if (req.method() === 'POST') {
      const body = JSON.parse(req.postData() || '{}') as Record<string, unknown>
      posted = body
      const created = { id: 'acc-' + (accounts.length + 1), name: String(body.name), endpoint: String(body.endpoint) }
      accounts.push(created)
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(created),
      })
      return
    }
    await route.continue()
  })

  await page.goto('/')

  // 1) 侧边栏「账号管理」入口可见可点（无账号时 App 本就停在该面板）。
  const navAccounts = page.getByRole('button', { name: /^账号管理$|^Accounts$/ })
  await expect(navAccounts).toBeVisible()
  await navAccounts.click()

  // 2) 空状态：渲染真实的 i18n 引导文案。
  await expect(page.getByText(/还没有登录配置|No login yet/)).toBeVisible()

  // 3) 新增：打开表单 → 填写 → 保存。
  await page.getByRole('button', { name: /新增登录|Add login/ }).click()
  const dlg = page.getByRole('dialog', { name: /新增登录|Add login/ })
  await expect(dlg).toBeVisible()
  await dlg.getByPlaceholder(/生产环境 MinIO|Production MinIO/).fill('e2e-account')
  await dlg.getByPlaceholder(/myqcloud|127\.0\.0\.1:9000/).fill('127.0.0.1:9000')
  await dlg.getByRole('button', { name: /保存并登录|Save and sign in/ }).click()

  // 4) 列表：表单字段真的进了请求体，新增行渲染出来，弹窗关闭。
  await expect(dlg).toHaveCount(0)
  await expect(page.getByRole('row', { name: /e2e-account/ })).toBeVisible()
  expect(posted).toMatchObject({ name: 'e2e-account', endpoint: '127.0.0.1:9000' })
})

/**
 * 后端不可用 → 健康轮询自动恢复（useHealthPoll / ASSESSMENT S5）。
 *
 * /api/accounts 先返回 500，App 必须渲染错误横幅与「连接异常」徽标；
 * 后端恢复后，健康轮询须自动重新拉取账号并清除错误态——全程无需刷新页面。
 *
 * 原用例只做了一次 page.evaluate(fetch) 并断言 fixture 自己写死的 version，
 * 前端完全不参与，app 全坏也会通过；这里改为断言用户可见的恢复行为。
 */
test('健康检查契约：后端恢复后自动重新拉取账号', async ({ page }) => {
  let backendUp = false
  const account = { id: 'acc-1', name: 'recovered', endpoint: '127.0.0.1:9000' }

  await page.route('**/api/accounts', async (route) => {
    if (!backendUp) {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'boom' }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ accounts: [account] }),
    })
  })

  await page.route('**/api/health', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'ok', version: 'e2e-1.0.0', time: new Date().toISOString(), store: { ok: true } }),
    })
  })

  await page.goto('/')

  // 1) 后端不可用：错误横幅 + 「连接异常」徽标，且后端错误信息透出。
  const banner = page.locator('.msg.err').filter({ hasText: /无法连接后端|Cannot reach backend/ })
  await expect(banner).toBeVisible()
  await expect(banner).toContainText('boom')
  await expect(page.locator('.conn.bad')).toBeVisible()

  // 2) 后端恢复：健康轮询（默认 5s 一次）触发 onRecover → 自动重拉账号、清除错误态。
  backendUp = true
  await expect(banner).toHaveCount(0, { timeout: 20_000 })
  await expect(page.locator('.conn.ok')).toBeVisible()
  await expect(page.getByRole('row', { name: /recovered/ })).toBeVisible()
})
