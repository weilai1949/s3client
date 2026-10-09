import { expect, test } from '@playwright/test'

/**
 * 浏览器侧冒烟：跑通「打开页面 → 渲染顶栏 → 没有账号时显示引导」。
 * 不依赖后端（vite preview 即可，纯静态 SPA 资源）。
 *
 * 设计意图：在没有真实 RustFS 的环境里也能跑，作为「前端构建产物可被浏览器加载」
 * 的最小验证。完整流程（建账号/上传/共享）的浏览器测试见 bucket-flow.spec.ts。
 */
test('SPA 渲染：顶栏 + 无账号空状态', async ({ page }) => {
  await page.goto('/')
  // 顶栏：标题
  await expect(page.locator('header')).toBeVisible()
  // 左侧菜单：账号管理 / 对象管理 / 桶管理 / 回收站 / 上传 / 服务器设置（至少其中几个）
  // 用一个宽松匹配：菜单中含「账号」字样
  await expect(page.locator('aside, nav').first()).toContainText(/账号|Accounts/)
})

test('OpenAPI 规范可被前端获取', async ({ request }) => {
  // 评审 R9：原断言接受 `[200,404,500,502,503]` 且仅在 200 分支校验 content-type——
  // 端点彻底坏掉也绿（永真断言）。静态预览把 /api 代理到 127.0.0.1:8080：无后端时
  // 确定性返回 503（本套用例的常态），此时**条件跳过**；一旦代理打到了真实后端
  //（任意其它状态码 = 有真进程应答），必须 200 + JSON 合法 spec。
  // 「真实后端模式断言 200 + JSON」的常驻版本在 e2e-real/real-backend.spec.ts
  //（配合 scripts/e2e-real.sh 的 S3C_EXPOSE_OPENAPI=1）。
  const res = await request.get('/api/openapi.json', { failOnStatusCode: false })
  test.skip(
    res.status() === 503 || res.status() === 502,
    '静态预览无后端（/api 代理无目标）——真实契约断言见 e2e-real/real-backend.spec.ts',
  )
  expect(res.status(), '有后端应答时 /api/openapi.json 必须可用').toBe(200)
  expect(res.headers()['content-type'] || '').toContain('application/json')
  const spec = (await res.json()) as { openapi?: string }
  expect(typeof spec.openapi).toBe('string')
})

test('静态资源 200', async ({ request }) => {
  const res = await request.get('/')
  expect(res.status()).toBe(200)
  const ct = res.headers()['content-type'] || ''
  expect(ct).toContain('text/html')
})
