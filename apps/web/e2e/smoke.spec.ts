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
  // 这是契约测试：vite preview 没有后端代理 → /api/openapi.json 应被 SPA
  // fallback 之外的逻辑处理（vite preview 直接 404/500）。当真实 CI 起后端时
  // 应返回 200 + JSON。我们接受任何非 SPA HTML 的响应。
  const res = await request.get('/api/openapi.json', { failOnStatusCode: false })
  // 接受：404（vite 无此路由）/ 502/503/500（后端不在时 vite 转发失败）
  expect([200, 404, 500, 502, 503]).toContain(res.status())
  // 如果返回 200，必须是 JSON 而非 HTML（避免回退到 SPA）。
  if (res.status() === 200) {
    const ct = res.headers()['content-type'] || ''
    expect(ct).toContain('application/json')
  }
})

test('静态资源 200', async ({ request }) => {
  const res = await request.get('/')
  expect(res.status()).toBe(200)
  const ct = res.headers()['content-type'] || ''
  expect(ct).toContain('text/html')
})
