import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { randomUUID } from 'node:crypto'

/**
 * 真实联调浏览器冒烟（KNOWN_ISSUES #37）。
 *
 * 与 `e2e/*.spec.ts` 的根本差异：**本文件不使用 `page.route()` mock 任何 `/api/**`**。
 * 页面由真实 Go 后端托管（`S3C_STATIC_DIR` 指向真实 `vite build` 产物），
 * `/api/*` 打到真实后端，后端再经 SigV4 访问真实 RustFS。因此它验证的是
 * 「构建产物 + 后端 + S3 对端」真正拼在一起时的用户可见行为：
 *
 *   1) 账号创建/持久化（真实 store 落盘，SecretKey 不回传）；
 *   2) 建桶 → 列桶（真实 S3 `CreateBucket` / `ListBuckets`）；
 *   3) **浏览器真实直传**：在 UI 里选文件，前端取预签名 PUT URL 后由浏览器
 *      XHR PUT 到 RustFS（跨源，依赖 RustFS CORS），再列对象、回读校验内容。
 *
 * 第 3 条是本任务的核心价值：`page.route` mock 时代它根本不可能被验证——
 * 预签名直传是**浏览器 → S3** 的跨源请求，只有真对端 + 真 CORS 才能跑通。
 *
 * 运行前提（由 `scripts/e2e-real.sh` / CI job 提供）：
 *   - `PLAYWRIGHT_BASE_URL` 指向**真实后端**（默认 http://127.0.0.1:8080）；
 *   - 后端 `S3C_STATIC_DIR` 指向真实构建产物；
 *   - `S3C_TOKEN`：脚本自动生成并注入，**后端 /api 鉴权开启**（生产同构形态），
 *     本文件负责把它带上——API 请求加 Authorization 头、页面预置 sessionStorage；
 *   - 一个可用的 RustFS（默认 127.0.0.1:9000，rustfsadmin/rustfsadmin），
 *     且 `RUSTFS_CORS_ALLOWED_ORIGINS` 放行页面 Origin。
 *
 * 前端 API base 默认同源（`storage.ts` 的 `defaultBase()` 对非 Tauri 返回 `''`），
 * 故浏览器只需访问后端端口，无需任何代理。
 */

/** 后端地址（页面与 /api 同源）。 */
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:8080'
/** 真实 S3（RustFS）端点：后端用它签名，浏览器用它直传，二者都必须可达。 */
const S3_ENDPOINT = process.env.S3CLIENT_ENDPOINT || 'http://127.0.0.1:9000'
const ACCESS_KEY = process.env.S3CLIENT_ACCESS_KEY || 'rustfsadmin'
const SECRET_KEY = process.env.S3CLIENT_SECRET_KEY || 'rustfsadmin'
/**
 * `/api/*` 鉴权 token（`scripts/e2e-real.sh` 生成后注入环境变量）。
 * 为空 = 后端未开鉴权（本地直接 `pnpm e2e:real` 打无 token 后端的兼容路径），
 * 此时不加 Authorization、也不预置 sessionStorage。
 */
const API_TOKEN = process.env.S3C_TOKEN || ''

/**
 * 后端有 IP 令牌桶限速（`ratelimit.go`：120 req/min、突发 30）。本套用例在
 * 极短时间（数秒）内会打出远超突发的 /api 请求（建账号 / 列桶 / 预签名 / 刷新 …），
 * 命中 429 属**预期内的真实行为**，不是被测功能缺陷。
 *
 * 不关闭限速（那会削弱「真实后端」的保真度），而是给请求加一层「遇 429 按
 * Retry-After 退避重试」的包装——这本身就是真实客户端应有的行为，也让用例
 * 在 CI 抖动下稳定。只对**幂等**调用使用（GET 与预签名签发都是幂等的）。
 */
async function requestWithRetry(
  request: APIRequestContext,
  method: 'get' | 'post' | 'delete',
  url: string,
  opts: Parameters<APIRequestContext['get']>[1] = {},
  attempts = 6,
): Promise<import('@playwright/test').APIResponse> {
  // S3C_TOKEN 开启下所有 /api 调用必须带 Bearer——与生产部署同构地验证鉴权路径。
  const optsWithAuth = API_TOKEN
    ? { ...opts, headers: { ...opts.headers, Authorization: `Bearer ${API_TOKEN}` } }
    : opts
  let res = await request[method](url, optsWithAuth)
  for (let i = 0; i < attempts && res.status() === 429; i++) {
    await new Promise((r) => setTimeout(r, 1500))
    res = await request[method](url, optsWithAuth)
  }
  return res
}

/** RustFS 桶名：3-63 位小写字母/数字/连字符，且全局唯一。 */
function uniqueBucket(prefix: string): string {
  return `${prefix}-${randomUUID().slice(0, 8)}`
}

/** 中文界面文案（i18n 默认 zh-CN；用例统一在 zh-CN 下断言）。 */
const zh = {
  navAccounts: /^账号管理$|^Accounts$/,
  navObjects: /^对象管理$|^Objects$/,
  navBuckets: /^桶管理$|^Buckets$/,
  addLogin: /新增登录|Add login/,
  namePh: /生产环境 MinIO|Production MinIO/,
  endpointPh: /myqcloud|127\.0\.0\.1:9000/,
  saveLogin: /保存并登录|Save and sign in/,
  healthOk: /^正常$|^OK$/,
  createBucket: /创建桶|Create bucket/,
  bucketNamePh: /3-63 位小写字母|lowercase letters/,
  // BucketsPanel 的工具栏按钮（`buckets.createBtn`）；注意别与 ObjectsPanel 的
  // BucketList 按钮（`buckets.createAction` = 「+ 创建 Bucket」）混淆。
  createBucketBtn: /\+ 新建桶|\+ New bucket/,
  backList: /返回列表|Back to list/,
  enterBucket: /^进入$|^Open$/,
  upload: /^上传文件$|^Upload$/,
  uploadOk: /已上传 1 个文件到当前目录|Uploaded 1 file/,
  testConn: /^测试$|^Test$/,
}

/**
 * 清空后端账号表，保证每个用例从「无账号」这一真实空状态出发。
 *
 * 必要性：CI 失败重试会重跑用例，而账号写在**真实 store** 里（不是浏览器内存），
 * 不清空会让第二个用例看到上一个用例的账号，空状态断言随即失真。
 */
async function resetBackend(request: APIRequestContext) {
  const res = await requestWithRetry(request, 'get', `${BASE_URL}/api/accounts`)
  expect(res.status()).toBe(200)
  const body = (await res.json()) as { accounts: Array<{ id: string }> }
  for (const a of body.accounts) {
    await requestWithRetry(request, 'delete', `${BASE_URL}/api/accounts/${a.id}`)
  }
}

/**
 * 走真实 UI 创建账号（空状态 → 新增 → 列表），返回账号 id。
 * 顺带验证账号 CRUD 真实落库与 SecretKey 不回传。
 */
async function createAccountViaUI(page: Page, name: string): Promise<string> {
  await page.goto('/')
  const navAccounts = page.getByRole('button', { name: zh.navAccounts })
  await expect(navAccounts).toBeVisible()
  await navAccounts.click()

  // 空状态（resetBackend 已保证后端无账号）
  await expect(page.getByText(/还没有登录配置|No login yet/)).toBeVisible()

  await page.getByRole('button', { name: zh.addLogin }).click()
  const dlg = page.getByRole('dialog', { name: zh.addLogin })
  await expect(dlg).toBeVisible()
  await dlg.getByPlaceholder(zh.namePh).fill(name)
  await dlg.getByPlaceholder(zh.endpointPh).fill(S3_ENDPOINT)
  await dlg.locator('input[autocomplete="off"]').fill(ACCESS_KEY)
  await dlg.locator('input[type="password"]').fill(SECRET_KEY)
  await dlg.getByRole('button', { name: zh.saveLogin }).click()

  await expect(dlg).toHaveCount(0)
  await expect(page.getByRole('row', { name: new RegExp(name) })).toBeVisible()

  // 取回真实账号 id（供后续真实 API 调用）。
  const res = await requestWithRetry(page.request, 'get', `${BASE_URL}/api/accounts`)
  const body = (await res.json()) as { accounts: Array<{ id: string; name: string }> }
  const created = body.accounts.find((a) => a.name === name)
  expect(created, `账号 ${name} 应已真实落库`).toBeTruthy()
  return created!.id
}

/** 建桶（真实 UI）并停留在桶列表，返回桶名。 */
async function createBucketViaUI(page: Page, bucket: string) {
  await page.getByRole('button', { name: zh.navBuckets }).click()
  // BucketsPanel 的工具栏（无账号/无桶时是空状态 + 「+ 新建桶」）。
  await expect(page.getByRole('button', { name: zh.createBucketBtn })).toBeVisible()

  await page.getByRole('button', { name: zh.createBucketBtn }).click()
  const dlg = page.getByRole('dialog', { name: zh.createBucket })
  await expect(dlg).toBeVisible()
  await dlg.getByPlaceholder(zh.bucketNamePh).fill(bucket)
  // 建桶成功路径必然触发一次 BucketsPanel 的 GET /buckets 刷新（onCreateBucket →
  // loadBuckets）。等这次刷新落定再回列表：否则「点返回列表」可能抢在刷新 resolve
  // 之前，loadBuckets 随后又把空 selectedBucket 自动选中 buckets[0] 钻进详情页，
  // 断言随即闪断——竞态成败，不能留。
  const listRefreshed = page.waitForResponse(
    (r) => r.ok() && r.request().method() === 'GET' && r.url().includes('/buckets'),
  )
  await dlg.getByRole('button', { name: /^创建$|^Create$/ }).click()
  await listRefreshed

  // 刷新完成后建桶落定的页面是**桶详情**（loadBuckets 在 selectedBucket 为空时自动
  // 选中 buckets[0]；复用有残留桶的 RustFS 时更是先进旧桶详情）。必须返回列表再断言：
  // 直接按行断言会命中详情页概览表里含桶名的那行，测的是别的东西（误中假绿）。
  await page.getByRole('button', { name: zh.backList }).click()
  await expect(page.getByRole('row', { name: new RegExp(bucket) })).toBeVisible()
}

/** 清空并删除真实桶（清理，失败仅告警，不掩盖用例结论）。 */
async function cleanupBucket(request: APIRequestContext, accId: string, bucket: string) {
  try {
    // 先清空对象。不能走 delete-prefix：handler 有意拒绝空前缀
    // （objects.go「拒绝空前缀以免误删全桶」，空 prefix 必 400），只能对象级批量删除。
    const list = await requestWithRetry(
      request,
      'get',
      `${BASE_URL}/api/accounts/${accId}/objects?bucket=${encodeURIComponent(bucket)}`,
    )
    expect(list.status()).toBe(200)
    const body = (await list.json()) as { objects: Array<{ key: string }> }
    // 本套用例每桶最多 1 个对象（直传用例 1 个 hello.txt），单页列举足够；
    // 将来用例若批量灌对象，此处需按 nextToken 翻页。
    const keys = body.objects.map((o) => o.key)
    if (keys.length > 0) {
      const del = await requestWithRetry(request, 'post', `${BASE_URL}/api/accounts/${accId}/delete`, {
        data: { bucket, keys },
        timeout: 20_000,
      })
      expect(del.status()).toBe(200)
    }
    const rm = await requestWithRetry(
      request,
      'delete',
      `${BASE_URL}/api/accounts/${accId}/bucket?name=${encodeURIComponent(bucket)}`,
    )
    expect(rm.status()).toBe(200)
  } catch (e) {
    console.warn(`cleanup ${bucket} failed: ${String(e)}`)
  }
}

/** 创建账号 → 建桶 → 返回 (accId, bucket)；统一登记 finally 清理。 */
async function seedAccountAndBucket(page: Page, name: string, bucketPrefix: string) {
  await resetBackend(page.request)
  const accId = await createAccountViaUI(page, name)
  const bucket = uniqueBucket(bucketPrefix)
  await createBucketViaUI(page, bucket)
  return { accId, bucket }
}

test.describe('真实后端 + 真实 RustFS 联调冒烟', () => {
  // token 注入（C1）：页面侧没有登录界面，与真实部署同构地把 token 放进
  // sessionStorage（前端 storage.ts 的读取路径）；addInitScript 在每次导航的页面
  // 脚本之前执行，覆盖首屏就发起的 /api 调用（列账号等）。同时预置一条「已配置
  // 同源服务器」清单——否则首启走默认 server 分支会 applyProfile('') 把刚写入的
  // token 从 sessionStorage 清掉（storage.ts readServers 一次性初始化路径）。
  test.beforeEach(async ({ page }) => {
    if (!API_TOKEN) return
    await page.addInitScript((token) => {
      if (!localStorage.getItem('s3c.servers')) {
        localStorage.setItem('s3c.servers', JSON.stringify([{ id: 'e2e-real', name: 'e2e-real', base: '' }]))
        localStorage.setItem('s3c.activeServerId', 'e2e-real')
      }
      sessionStorage.setItem('s3c.token', token)
    }, API_TOKEN)
  })

  test('账号 CRUD 真实落库：新增 → 列表 → 后端可见 → 测试连接', async ({ page }) => {
    await resetBackend(page.request)
    const name = `e2e-real-${randomUUID().slice(0, 8)}`
    const accId = await createAccountViaUI(page, name)

    // 后端真实 store 里确实存在（不是前端内存态）
    const one = await requestWithRetry(page.request, 'get', `${BASE_URL}/api/accounts/${accId}`)
    expect(one.status()).toBe(200)
    const acc = (await one.json()) as { name: string; endpoint: string; secretSet: boolean }
    expect(acc.name).toBe(name)
    expect(acc.endpoint).toBe(S3_ENDPOINT)
    // SecretKey 绝不回传（契约：AccountView 无 secretKey 字段，只有 secretSet 布尔）
    expect(acc).not.toHaveProperty('secretKey')
    expect(acc.secretSet).toBe(true)

    // UI「测试连接」走真实 S3（ListBuckets），必须成功
    const row = page.getByRole('row', { name: new RegExp(name) })
    await row.getByRole('button', { name: zh.testConn }).click()
    await expect(row.getByText(zh.healthOk)).toBeVisible()

    const del = await requestWithRetry(page.request, 'delete', `${BASE_URL}/api/accounts/${accId}`)
    expect(del.status()).toBe(200)
  })

  test('建桶 → 列桶：真实 S3 CreateBucket/ListBuckets', async ({ page }) => {
    const name = `e2e-real-${randomUUID().slice(0, 8)}`
    const { accId, bucket } = await seedAccountAndBucket(page, name, 's3c-e2e-bucket')
    try {
      // 后端真实列举到该桶
      const res = await requestWithRetry(page.request, 'get', `${BASE_URL}/api/accounts/${accId}/buckets`)
      const body = (await res.json()) as { buckets: Array<{ name: string }> }
      expect(body.buckets.map((b) => b.name)).toContain(bucket)
    } finally {
      await cleanupBucket(page.request, accId, bucket)
      await requestWithRetry(page.request, 'delete', `${BASE_URL}/api/accounts/${accId}`)
    }
  })

  test('浏览器真实直传 → 列对象 → 回读内容', async ({ page }) => {
    const name = `e2e-real-${randomUUID().slice(0, 8)}`
    const { accId, bucket } = await seedAccountAndBucket(page, name, 's3c-e2e-object')
    const key = 'hello.txt'
    const payload = `hello from real rustfs ${randomUUID()}`

    // 捕获浏览器对 RustFS 的真实 PUT（预签名直传），证明字节不经 Go 服务。
    const browserPut = page.waitForRequest(
      (r) => r.method() === 'PUT' && r.url().includes(`/${key}`) && r.url().includes(S3_ENDPOINT),
      { timeout: 30_000 },
    )

    try {
      // 切到「对象管理」（BucketList 里才有「进入」按钮），进入该桶
      await page.getByRole('button', { name: zh.navObjects }).click()
      await page.getByRole('row', { name: new RegExp(bucket) }).getByRole('button', { name: zh.enterBucket }).click()
      await expect(page.getByRole('button', { name: zh.upload })).toBeVisible()

      // 通过 UI 选文件：直接给隐藏 input 设文件，触发应用真实的 onPickUpload →
      // 预签名 → 浏览器 XHR PUT → 刷新列表 流程（不 mock 任何请求）。
      await page.locator('main input[type="file"]').setInputFiles({
        name: key,
        mimeType: 'text/plain',
        buffer: Buffer.from(payload),
      })

      // 真实直传成功（跨源 PUT 若被 CORS 拦下，这里会先超时）
      const putReq = await browserPut
      expect(putReq.url()).toContain(S3_ENDPOINT)

      // UI 完成提示 + 对象列表出现该文件（真实 ListObjects）
      await expect(page.getByText(zh.uploadOk)).toBeVisible({ timeout: 30_000 })
      await expect(page.getByRole('row', { name: new RegExp(key) })).toBeVisible()

      // 浏览器侧用后端签发的预签名 GET 回读，逐字节校验（证明签名 URL 真实可用）。
      const presign = await requestWithRetry(page.request, 'post', `${BASE_URL}/api/accounts/${accId}/presign`, {
        data: { method: 'get', bucket, key, expiresIn: 600 },
      })
      expect(presign.status()).toBe(200)
      const getUrl = ((await presign.json()) as { url: string }).url
      const text = await page.evaluate(async (url) => {
        const r = await fetch(url)
        if (!r.ok) throw new Error(`presigned GET ${r.status}`)
        return r.text()
      }, getUrl)
      expect(text).toBe(payload)
    } finally {
      await cleanupBucket(page.request, accId, bucket)
      await requestWithRetry(page.request, 'delete', `${BASE_URL}/api/accounts/${accId}`)
    }
  })
})
