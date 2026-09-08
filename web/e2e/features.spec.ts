import { expect, test, type Page, type Route } from '@playwright/test'

/**
 * 功能特性 E2E（任务 T-6）：桶管理 / 对象操作 / 版本控制 / 回收站 / 迁移面板。
 *
 * 与 account-flow.spec.ts 相同思路：用 page.route() 拦截 /api/** 并回放
 * 最小合理 JSON（本文件内自带的 in-memory fake），完全不依赖真实后端；
 * vite preview 静态环境即可运行（`pnpm e2e`）。
 *
 * 每个用例先 seed 一个 FakeDb（账号 / 桶 / 对象 / 版本 / 删除标记），再
 * installFake() 挂路由，随后通过真实 UI 驱动流程，最后断言：
 *   1) UI 渲染结果（列表行 / 弹窗内容）；
 *   2) 前端确实调用了目标 API（db.calls 记录 method + path + body）。
 *
 * 说明：不修改 fixture-server.ts（其 stub 为空容器）；helper 全部定义在本文件。
 */

/* ------------------------------------------------------------------ */
/* 类型与 seed helper                                                  */
/* ------------------------------------------------------------------ */

interface FakeAcc {
  id: string
  name: string
  endpoint: string
  region: string
  accessKey: string
  secretKey: string
  bucket: string
  pathStyle: boolean
  useSSL: boolean
}

interface FakeObj {
  key: string
  size: number
  lastModified: string
  etag: string
  contentType: string
  storageClass?: string
  isDir: boolean
}

interface FakeVersion {
  key: string
  versionId: string
  isLatest: boolean
  lastModified: string
  size: number
  etag: string
  storageClass?: string
}

interface FakeMarker {
  key: string
  versionId: string
  isLatest: boolean
  lastModified: string
}

interface Failure {
  method: string
  pathname: string
  status: number
  body: unknown
}

/** in-memory fake 后端状态；key 形如 `${accId}|${bucket}|${objectKey}`。 */
interface FakeDb {
  accounts: FakeAcc[]
  buckets: Map<string, string[]>
  objects: Map<string, FakeObj[]>
  versions: Map<string, FakeVersion[]>
  deleteMarkers: Map<string, FakeMarker[]>
  /** 一次性失败注入：匹配 method+pathname 时返回 status/body 并消费该条。 */
  failures: Failure[]
  /** 已接收到的 API 调用记录：`METHOD /path?query [body]`。 */
  calls: string[]
  restoreSeq: number
  policy: string
  tags: Map<string, { key: string; value: string }[]>
}

const K = (...parts: string[]) => parts.join('|')

function emptyDb(): FakeDb {
  return {
    accounts: [],
    buckets: new Map(),
    objects: new Map(),
    versions: new Map(),
    deleteMarkers: new Map(),
    failures: [],
    calls: [],
    restoreSeq: 1,
    policy: '',
    tags: new Map(),
  }
}

function fakeAccount(id: string, name: string, bucket = ''): FakeAcc {
  return {
    id,
    name,
    endpoint: 'http://127.0.0.1:9000',
    region: 'us-east-1',
    accessKey: 'AKIA' + id.toUpperCase(),
    secretKey: 'SECRET' + id.toUpperCase(),
    bucket,
    pathStyle: true,
    useSSL: false,
  }
}

let objSeq = 0
function fakeObj(key: string, size: number, opts: Partial<FakeObj> = {}): FakeObj {
  objSeq += 1
  return {
    key,
    size,
    lastModified: `2024-03-0${(objSeq % 9) + 1}T10:00:00.000Z`,
    etag: 'etag-' + objSeq,
    contentType: 'application/octet-stream',
    storageClass: 'STANDARD',
    isDir: false,
    ...opts,
  }
}

function fakeVersion(key: string, versionId: string, size: number, opts: Partial<FakeVersion> = {}): FakeVersion {
  return { key, versionId, size, etag: 'etag-' + versionId, storageClass: 'STANDARD', isLatest: false, lastModified: '2024-01-01T00:00:00.000Z', ...opts }
}

function fakeMarker(key: string, versionId: string, lastModified = '2024-04-01T00:00:00.000Z'): FakeMarker {
  return { key, versionId, isLatest: true, lastModified }
}

/* ------------------------------------------------------------------ */
/* page.route 分发：最小 fake 后端                                      */
/* ------------------------------------------------------------------ */

function json(route: Route, status: number, body: unknown) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
}

async function handleApi(route: Route, db: FakeDb): Promise<void> {
  const req = route.request()
  const method = req.method()
  const url = new URL(req.url())
  const path = url.pathname
  const postData = req.postData()
  db.calls.push(`${method} ${path}${url.search}${postData ? ' ' + postData : ''}`)

  // 一次性失败注入（负向用例）
  const failIdx = db.failures.findIndex((f) => f.method === method && f.pathname === path)
  if (failIdx >= 0) {
    const f = db.failures.splice(failIdx, 1)[0]
    await json(route, f.status, f.body)
    return
  }

  // 账号列表（App 启动即拉取）
  if (path === '/api/accounts') {
    if (method === 'GET') return json(route, 200, { accounts: db.accounts })
    return json(route, 404, { error: 'unsupported @ /api/accounts: ' + method })
  }

  const m = path.match(/^\/api\/accounts\/([^/]+)(?:\/(.*))?$/)
  if (!m) {
    await json(route, 404, { error: 'not found: ' + method + ' ' + path })
    return
  }
  const accId = m[1]
  const rest = m[2] ?? ''

  if (rest === 'buckets' && method === 'GET') {
    const names = db.buckets.get(accId) ?? []
    return json(route, 200, {
      buckets: names.map((name) => ({ name, creationDate: '2024-01-02T00:00:00.000Z' })),
    })
  }
  if (rest === 'bucket' && method === 'POST') {
    const body = JSON.parse(postData || '{}') as { name?: string; region?: string; acl?: string }
    const name = body.name ?? ''
    const list = db.buckets.get(accId) ?? []
    if (!list.includes(name)) list.push(name)
    db.buckets.set(accId, list)
    return json(route, 201, { created: name, region: body.region ?? 'us-east-1', acl: body.acl ?? 'private' })
  }
  if (rest === 'bucket' && method === 'DELETE') {
    const name = url.searchParams.get('name') ?? ''
    db.buckets.set(accId, (db.buckets.get(accId) ?? []).filter((n) => n !== name))
    return json(route, 200, { deleted: name })
  }
  if (rest === 'objects' && method === 'GET') {
    const bucket = url.searchParams.get('bucket') ?? ''
    const prefix = url.searchParams.get('prefix') ?? ''
    const delimiter = url.searchParams.get('delimiter') ?? ''
    const all = db.objects.get(K(accId, bucket)) ?? []
    const files: FakeObj[] = []
    const prefixes = new Set<string>()
    for (const o of all) {
      if (!o.key.startsWith(prefix)) continue
      const rem = o.key.slice(prefix.length)
      if (delimiter && rem.includes(delimiter)) {
        prefixes.add(prefix + rem.slice(0, rem.indexOf(delimiter) + delimiter.length))
        continue
      }
      files.push(o)
    }
    return json(route, 200, {
      objects: files,
      commonPrefixes: [...prefixes].sort(),
      isTruncated: false,
      nextToken: '',
    })
  }
  if (rest === 'mkdir' && method === 'POST') {
    const body = JSON.parse(postData || '{}') as { bucket?: string; key?: string }
    const key = body.key ?? ''
    const list = db.objects.get(K(accId, body.bucket ?? '')) ?? []
    if (!list.some((o) => o.key === key)) {
      list.push(fakeObj(key, 0, { isDir: true, contentType: 'application/x-directory' }))
    }
    db.objects.set(K(accId, body.bucket ?? ''), list)
    return json(route, 200, { created: key, bucket: body.bucket ?? '' })
  }
  if (rest === 'delete' && method === 'POST') {
    const body = JSON.parse(postData || '{}') as { bucket?: string; keys?: string[] }
    const bucket = body.bucket ?? ''
    const keys = body.keys ?? []
    db.objects.set(K(accId, bucket), (db.objects.get(K(accId, bucket)) ?? []).filter((o) => !keys.includes(o.key)))
    return json(route, 200, { deleted: keys.length })
  }
  if (rest === 'versions' && method === 'GET') {
    const bucket = url.searchParams.get('bucket') ?? ''
    const key = url.searchParams.get('prefix') ?? ''
    return json(route, 200, {
      versions: db.versions.get(K(accId, bucket, key)) ?? [],
      deleteMarkers: [],
      isTruncated: false,
      nextKeyMarker: '',
      nextVersionIdMarker: '',
    })
  }
  if (rest === 'version' && method === 'DELETE') {
    const bucket = url.searchParams.get('bucket') ?? ''
    const key = url.searchParams.get('key') ?? ''
    const versionId = url.searchParams.get('versionId') ?? ''
    const vk = K(accId, bucket, key)
    db.versions.set(vk, (db.versions.get(vk) ?? []).filter((v) => v.versionId !== versionId))
    return json(route, 200, { deleted: key, versionId })
  }
  if (rest === 'version/restore' && method === 'POST') {
    const body = JSON.parse(postData || '{}') as { bucket?: string; key?: string; versionId?: string }
    const bucket = body.bucket ?? ''
    const key = body.key ?? ''
    const vk = K(accId, bucket, key)
    const list = db.versions.get(vk) ?? []
    const src = list.find((v) => v.versionId === body.versionId) ?? list[0]
    const restored: FakeVersion = {
      key,
      versionId: 'restored-' + db.restoreSeq++,
      isLatest: true,
      lastModified: new Date().toISOString(),
      size: src?.size ?? 0,
      etag: src?.etag ?? '',
      storageClass: src?.storageClass ?? 'STANDARD',
    }
    db.versions.set(vk, [...list.map((v) => ({ ...v, isLatest: false })), restored])
    return json(route, 200, { restored: key, versionId: restored.versionId })
  }
  if (rest === 'trash' && method === 'GET') {
    const bucket = url.searchParams.get('bucket') ?? ''
    return json(route, 200, {
      deleteMarkers: db.deleteMarkers.get(K(accId, bucket)) ?? [],
      isTruncated: false,
      nextKeyMarker: '',
      nextVersionIdMarker: '',
    })
  }
  if (rest === 'trash/purge' && method === 'POST') {
    const body = JSON.parse(postData || '{}') as { bucket?: string; key?: string }
    const bucket = body.bucket ?? ''
    const mk = K(accId, bucket)
    const before = (db.deleteMarkers.get(mk) ?? []).length
    db.deleteMarkers.set(mk, (db.deleteMarkers.get(mk) ?? []).filter((x) => x.key !== body.key))
    return json(route, 200, { purged: body.key ?? '', deleted: Math.max(before, 1) })
  }
  if (rest === 'bucket-info' && method === 'GET') {
    const bucket = url.searchParams.get('bucket') ?? ''
    return json(route, 200, { bucket, region: 'us-east-1', createdAt: '2024-01-02T00:00:00.000Z', versioning: '' })
  }
  if (rest === 'bucket/policy' && method === 'GET') {
    return json(route, 200, {
      policy: '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::reports/*"}]}',
    })
  }
  if (rest === 'bucket/policy' && method === 'PUT') {
    const body = JSON.parse(postData || '{}') as { policy?: string }
    db.policy = body.policy ?? ''
    return json(route, 200, { saved: true })
  }
  if (rest === 'bucket/policy' && method === 'DELETE') {
    db.policy = ''
    return json(route, 200, { deleted: true })
  }
  if (rest === 'bucket/tags' && method === 'GET') {
    const bucket = url.searchParams.get('bucket') ?? ''
    return json(route, 200, { bucket, tags: db.tags.get(bucket) ?? [] })
  }
  if (rest === 'bucket/tags' && method === 'PUT') {
    const body = JSON.parse(postData || '{}') as { bucket?: string; tags?: { key: string; value: string }[] }
    db.tags.set(body.bucket ?? '', body.tags ?? [])
    return json(route, 200, { updated: (body.tags ?? []).length })
  }

  await json(route, 404, { error: 'not found: ' + method + ' ' + path })
}

async function installFake(page: Page, db: FakeDb) {
  await page.route('**/api/**', (route) => handleApi(route, db))
}

/* ------------------------------------------------------------------ */
/* 用例                                                                */
/* ------------------------------------------------------------------ */

test('桶管理：创建桶 POST /api/accounts/{id}/bucket 后进入列表', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1')]
  db.buckets.set('acc-1', ['reports'])
  await installFake(page, db)

  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await page.getByRole('button', { name: /桶管理|^Buckets$/ }).click()
  // BucketsPanel 自动选中第一个桶并进入详情页；等 bucket-info 拉取完成后返回列表视图
  await expect.poll(() => db.calls.some((c) => c.includes('GET /api/accounts/acc-1/bucket-info'))).toBe(true)
  await page.getByRole('button', { name: /返回列表|Back to list/ }).click()
  await expect(page.getByRole('row', { name: /reports/ })).toBeVisible()

  // 创建桶
  await page.getByRole('button', { name: /新建桶|New bucket/ }).click()
  const dlg = page.getByRole('dialog', { name: /创建桶|Create bucket/ })
  await expect(dlg).toBeVisible()
  await dlg.locator('input').fill('data')
  await dlg.getByRole('button', { name: /^创建$|^Create$/ }).click()
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('POST /api/accounts/acc-1/bucket') && c.includes('"name":"data"')))
    .toBe(true)
  // 创建后自动刷新桶列表（重新进入详情视图 → 返回列表看到新桶）
  await page.getByRole('button', { name: /返回列表|Back to list/ }).click()
  await expect(page.getByRole('row', { name: /data/ })).toBeVisible()
  await expect(page.getByRole('row', { name: /reports/ })).toBeVisible()
})

test('桶管理：删除桶 失败保留（409 反向）→ 重试成功移除（正向）', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1')]
  db.buckets.set('acc-1', ['reports', 'temp-1'])
  await installFake(page, db)

  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await page.getByRole('button', { name: /桶管理|^Buckets$/ }).click()
  await expect.poll(() => db.calls.some((c) => c.includes('GET /api/accounts/acc-1/bucket-info'))).toBe(true)
  await page.getByRole('button', { name: /返回列表|Back to list/ }).click()
  const row = page.getByRole('row', { name: /temp-1/ })
  await expect(row).toBeVisible()

  // 反向：后端 409 → 桶保留 + 错误横幅
  db.failures.push({ method: 'DELETE', pathname: '/api/accounts/acc-1/bucket', status: 409, body: { error: 'BucketNotEmpty' } })
  await row.getByRole('button', { name: /^删除$|^Delete$/ }).click()
  const confirm = page.getByRole('alertdialog')
  await expect(confirm).toBeVisible()
  await confirm.getByRole('button', { name: /^删除$|^Delete$/ }).click()
  await expect(page.getByText(/BucketNotEmpty/)).toBeVisible()
  await expect(page.getByRole('row', { name: /temp-1/ })).toBeVisible()

  // 正向：重试成功 → 从列表移除
  await page.getByRole('row', { name: /temp-1/ }).getByRole('button', { name: /^删除$|^Delete$/ }).click()
  await expect(page.getByRole('alertdialog')).toBeVisible()
  await page.getByRole('alertdialog').getByRole('button', { name: /^删除$|^Delete$/ }).click()
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('DELETE /api/accounts/acc-1/bucket?name=temp-1')))
    .toBe(true)
  await expect(page.getByRole('row', { name: /temp-1/ })).toHaveCount(0)
  await expect(page.getByRole('row', { name: /reports/ })).toBeVisible()
})

test('对象操作：列出对象 GET /api/accounts/{id}/objects + 新建目录 POST /mkdir', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1', 'photos')]
  db.buckets.set('acc-1', ['photos'])
  db.objects.set(K('acc-1', 'photos'), [
    fakeObj('img/logo.png', 2048),
    fakeObj('data.csv', 120),
    fakeObj('readme.txt', 14),
  ])
  await installFake(page, db)

  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  // 有账号 → 默认进入对象管理；账号默认桶 photos → 自动进入桶并列出对象
  await expect.poll(() => db.calls.some((c) => c.startsWith('GET /api/accounts/acc-1/objects'))).toBe(true)
  await expect(page.locator('tbody')).toContainText('data.csv')
  await expect(page.locator('tbody')).toContainText('readme.txt')
  await expect(page.locator('tbody')).toContainText('img') // img/logo.png 的目录前缀

  // 新建文件夹：toolbar「+ 新建文件夹」→ prompt 弹窗 → POST /mkdir
  await page.getByRole('button', { name: /新建文件夹|New folder/ }).click()
  const prompt = page.getByRole('dialog', { name: /新建文件夹|New folder/ })
  await expect(prompt).toBeVisible()
  await prompt.getByRole('textbox').fill('docs')
  await prompt.getByRole('button', { name: /^创建$|^Create$/ }).click()
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('POST /api/accounts/acc-1/mkdir') && c.includes('"key":"docs/"')))
    .toBe(true)
  await expect(page.locator('tbody')).toContainText('docs')
})

test('对象操作：删除对象 POST /api/accounts/{id}/delete 后从列表移除', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1', 'photos')]
  db.buckets.set('acc-1', ['photos'])
  db.objects.set(K('acc-1', 'photos'), [
    fakeObj('readme.txt', 14),
    fakeObj('keep.txt', 5),
  ])
  await installFake(page, db)

  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await expect.poll(() => db.calls.some((c) => c.startsWith('GET /api/accounts/acc-1/objects'))).toBe(true)
  await expect(page.locator('tbody')).toContainText('readme.txt')

  // 选中 readme.txt → 工具栏「删除」→ 确认 → POST /delete
  await page.getByRole('row', { name: /readme\.txt/ }).getByRole('checkbox').check()
  await page.getByRole('button', { name: /^删除$|^Delete$/ }).click()
  const confirm = page.getByRole('alertdialog')
  await expect(confirm).toBeVisible()
  await confirm.getByRole('button', { name: /^删除$|^Delete$/ }).click()
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('POST /api/accounts/acc-1/delete') && c.includes('"keys":["readme.txt"]')))
    .toBe(true)
  await expect(page.locator('tbody')).not.toContainText('readme.txt')
  await expect(page.locator('tbody')).toContainText('keep.txt')
})

test('版本控制：列出 GET /versions、删除版本、恢复版本', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1', 'docs')]
  db.buckets.set('acc-1', ['docs'])
  db.objects.set(K('acc-1', 'docs'), [fakeObj('report.pdf', 4096)])
  db.versions.set(K('acc-1', 'docs', 'report.pdf'), [
    fakeVersion('report.pdf', 'v1', 1024, { lastModified: '2024-01-01T10:00:00.000Z' }),
    fakeVersion('report.pdf', 'v2', 4096, { isLatest: true, lastModified: '2024-02-01T10:00:00.000Z' }),
  ])
  await installFake(page, db)

  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await expect(page.locator('tbody')).toContainText('report.pdf')

  // 打开「版本」对话框 → GET /versions
  // 说明：上下文菜单 position:fixed 且高于视口，低处项（版本/删除）在视口外无法直接 click；
  // 用菜单的键盘导航（聚焦 + Enter）触发，与真实用户操作等价。
  await page.getByRole('button', { name: /更多操作|More actions/ }).click()
  const versionsItem = page.getByRole('menuitem', { name: /版本|Versions/ })
  await versionsItem.focus()
  await page.keyboard.press('Enter')
  const dlg = page.getByRole('dialog', { name: /版本 ·|Versions ·/ })
  await expect(dlg).toBeVisible()
  await expect.poll(() => db.calls.some((c) => c.startsWith('GET /api/accounts/acc-1/versions'))).toBe(true)
  await expect(dlg).toContainText('v1')
  await expect(dlg).toContainText('v2')

  // 删除版本 v1：DELETE /version?key=…&versionId=v1
  // 说明：ConfirmDialog 与 ModalDialog 同为 z-index:200，且确认框的 Teleport 锚点在 DOM
  // 中早于版本弹窗，导致在版本弹窗内再弹确认框时确认框被遮住（应用现有行为，非本测试引入，
  // 已在回复中注明）。确认按钮会被自动聚焦，按 Enter 即确认（键盘用户真实路径）。
  await dlg.locator('tr', { hasText: 'v1' }).first().getByRole('button', { name: /^删除$|^Delete$/ }).click()
  const confirm = page.getByRole('alertdialog')
  await expect(confirm).toBeVisible()
  await expect(confirm.getByRole('button', { name: /^删除$|^Delete$/ })).toBeFocused()
  await page.keyboard.press('Enter')
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('DELETE /api/accounts/acc-1/version?key=report.pdf&versionId=v1')))
    .toBe(true)
  await expect(dlg).not.toContainText('v1')

  // 恢复版本 v2：POST /version/restore → 产生新当前版本 restored-1（确认框同样被遮，按 Enter）
  await dlg.locator('tr', { hasText: 'v2' }).first().getByRole('button', { name: /^恢复$|^Restore$/ }).click()
  const confirmRestore = page.getByRole('alertdialog')
  await expect(confirmRestore).toBeVisible()
  await expect(confirmRestore.getByRole('button', { name: /^恢复$|^Restore$/ })).toBeFocused()
  await page.keyboard.press('Enter')
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('POST /api/accounts/acc-1/version/restore') && c.includes('"versionId":"v2"')))
    .toBe(true)
  await expect(dlg).toContainText('restored-1')
})

test('回收站：列出删除标记 GET /api/accounts/{id}/trash + 彻底清除 POST /trash/purge', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1')]
  db.buckets.set('acc-1', ['docs'])
  db.deleteMarkers.set(K('acc-1', 'docs'), [
    fakeMarker('gone.txt', 'dm-1'),
    fakeMarker('old/notes.md', 'dm-2'),
  ])
  await installFake(page, db)

  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await page.getByRole('button', { name: /回收站|^Trash$/ }).click()
  await expect(page.getByRole('heading', { name: /回收站|Trash/ })).toBeVisible()
  // 面板自动选择第一个桶并拉取删除标记
  await expect.poll(() => db.calls.some((c) => c.startsWith('GET /api/accounts/acc-1/trash'))).toBe(true)
  await expect(page.locator('tbody')).toContainText('gone.txt')
  await expect(page.locator('tbody')).toContainText('old/notes.md')

  // 彻底清除 gone.txt → POST /trash/purge → 该标记从列表移除，其余保留
  await page.getByRole('row', { name: /gone\.txt/ }).getByRole('button', { name: /彻底清除|^Purge$/ }).click()
  const confirm = page.getByRole('alertdialog')
  await expect(confirm).toBeVisible()
  await confirm.getByRole('button', { name: /彻底清除|^Purge$/ }).click()
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('POST /api/accounts/acc-1/trash/purge') && c.includes('"key":"gone.txt"')))
    .toBe(true)
  await expect(page.locator('tbody')).not.toContainText('gone.txt')
  await expect(page.locator('tbody')).toContainText('old/notes.md')
})

test('迁移面板：渲染 + 列出源对象（弱断言，不发起真实迁移）', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1', 'data'), fakeAccount('acc-2', 'account-2', 'archive')]
  db.buckets.set('acc-1', ['data'])
  db.buckets.set('acc-2', ['archive'])
  db.objects.set(K('acc-1', 'data'), [fakeObj('a.txt', 10), fakeObj('b.csv', 20)])
  await installFake(page, db)

  await page.goto('/')
  await expect(page.locator('header')).toBeVisible()
  await page.getByRole('button', { name: /文件迁移|^Migrate$/ }).click()
  await expect(page.getByRole('heading', { name: /文件迁移|Migrate/ })).toBeVisible()
  // 目标账号默认自动选另一个账号
  await expect(page.getByRole('combobox', { name: /目标账号|Target account/ })).toContainText('account-2')

  // 选择源桶并列出对象（仅做面板渲染 + 列表弱断言，不点「开始迁移」）
  await page.getByRole('combobox', { name: /源 Bucket|Source bucket/ }).selectOption('data')
  await page.getByRole('button', { name: /列出对象|List objects/ }).click()
  await expect.poll(() => db.calls.some((c) => c.startsWith('GET /api/accounts/acc-1/objects?bucket=data'))).toBe(true)
  await expect(page.locator('table')).toContainText('a.txt')
  await expect(page.locator('table')).toContainText('b.csv')

  // 弱断言：从未发起真实迁移任务
  expect(db.calls.filter((c) => c.startsWith('POST /api/migrate')).length).toBe(0)
})

test('桶策略：编辑器加载 + 应用模板保存 PUT /bucket/policy', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1')]
  db.buckets.set('acc-1', ['reports'])
  await installFake(page, db)

  await page.goto('/')
  await page.getByRole('button', { name: /桶管理|^Buckets$/ }).click()
  await expect.poll(() => db.calls.some((c) => c.includes('GET /api/accounts/acc-1/bucket-info'))).toBe(true)
  await page.getByRole('button', { name: /桶策略|^Policy$/ }).click()
  // 编辑器加载既有策略（JSON 预览 pre 隐藏，断言可见的语句卡片）
  await expect(page.locator('[data-testid="policy-stmt-0"]')).toBeVisible()

  // 应用「公开读」模板 → 保存
  await page.getByRole('button', { name: /^公共读（GetObject）$|^Public read \(GetObject\)$/ }).click()
  await page.getByRole('button', { name: /保存|Save/ }).click()
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('PUT /api/accounts/acc-1/bucket/policy') && c.includes('"policy":')))
    .toBe(true)
  expect(db.policy).toContain('s3:GetObject')
})

test('桶标签：添加行 + 保存 PUT /bucket/tags', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1')]
  db.buckets.set('acc-1', ['reports'])
  await installFake(page, db)

  await page.goto('/')
  await page.getByRole('button', { name: /桶管理|^Buckets$/ }).click()
  await expect.poll(() => db.calls.some((c) => c.includes('GET /api/accounts/acc-1/bucket-info'))).toBe(true)
  await page.getByRole('button', { name: /标签|Tags/ }).click()
  await page.getByRole('button', { name: /添加标签|Add tag/ }).click()
  const row = page.locator('tbody tr').last()
  const inputs = row.locator('input')
  await inputs.nth(0).fill('env')
  await inputs.nth(1).fill('prod')
  await page.getByRole('button', { name: /保存|Save/ }).click()
  await expect
    .poll(() => db.calls.some((c) => c.startsWith('PUT /api/accounts/acc-1/bucket/tags') && c.includes('"key":"env"')))
    .toBe(true)
})

test('服务器设置：添加服务器并测试连通', async ({ page }) => {
  const db = emptyDb()
  db.accounts = [fakeAccount('acc-1', 'account-1')]
  await installFake(page, db)

  // 健康检查：拦截 /api/health（任意源），返回版本号供连通性断言
  await page.route('**/api/health', (route) => {
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ version: '9.9.9' }) })
  })

  await page.goto('/')
  await page.getByRole('button', { name: /服务器|^Server$/ }).click()
  await page.getByRole('button', { name: /新增服务端|Add server/ }).click()
  const dlg = page.getByRole('dialog', { name: /新增服务端|Add server/ })
  const inputs = dlg.locator('input')
  await inputs.nth(0).fill('dev-server')
  await inputs.nth(1).fill('http://127.0.0.1:9000')
  await dlg.getByRole('button', { name: /保存|Save/ }).click()
  // upsert 后服务器列表展示新条目
  await expect(page.getByRole('cell', { name: 'http://127.0.0.1:9000' })).toBeVisible()

  // 全部检测（probeAll）→ 每行健康列显示后端版本号
  await page.getByRole('button', { name: /全部检测|Probe all/ }).click()
  await expect(page.getByRole('cell', { name: /9\.9\.9/ })).toHaveCount(2)
})
