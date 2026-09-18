import { describe, expect, it, vi } from 'vitest'
import { cycleLocale, i18nKeyCount, locale, setLocale, t, tf } from './index'

describe('i18n', () => {
  it('translates nav keys', () => {
    setLocale('en-US')
    expect(t('nav.objects')).toBe('Objects')
    setLocale('zh-CN')
    expect(t('nav.objects')).toBe('对象管理')
  })

  it('covers common/toolbar keys in both locales', () => {
    setLocale('en-US')
    expect(t('common.cancel')).toBe('Cancel')
    expect(t('toolbar.upload')).toBe('Upload')
    expect(t('migrate.cancel')).toBe('Cancel migrate')
    expect(t('upload.start')).toBe('Start upload')
    expect(t('buckets.tabLifecycle')).toBe('Lifecycle')
    setLocale('zh-CN')
    expect(t('common.cancel')).toBe('取消')
    expect(t('toolbar.upload')).toBe('上传文件')
    expect(t('migrate.cancel')).toBe('取消迁移')
    expect(t('upload.start')).toBe('开始上传')
    expect(t('buckets.tabLifecycle')).toBe('生命周期')
  })

  it('covers preview/compare/server/trash/versions/dest keys', () => {
    setLocale('zh-CN')
    expect(t('preview.unsupported')).toContain('预览')
    expect(t('compare.run')).toBe('比较')
    expect(t('server.persistent')).toContain('跨会话')
    expect(t('trash.restore')).toBe('还原')
    expect(t('versions.compare')).toContain('比较')
    expect(t('dest.errBucket')).toContain('Bucket')
    expect(tf('dest.selected', { n: 3 })).toContain('3')
    setLocale('en-US')
    expect(t('preview.unsupported')).toContain('preview')
    expect(t('compare.run')).toBe('Compare')
    expect(t('server.persistent')).toContain('Persist')
    expect(t('trash.restore')).toBe('Restore')
    expect(t('versions.compare')).toContain('Compare')
    expect(t('dest.errBucket')).toContain('bucket')
    expect(tf('dest.selected', { n: 3 })).toContain('3')
  })

  it('covers accounts/overview/website/lifecycle/acl keys', () => {
    setLocale('zh-CN')
    expect(t('accounts.saveLogin')).toContain('登录')
    expect(t('overview.enableTitle')).toContain('版本控制')
    expect(t('website.disableBtn')).toContain('网站托管')
    expect(tf('lifecycle.toastSaved', { n: 2 })).toContain('2')
    expect(t('acl.publicRead')).toContain('公共读')
    expect(t('encryption.disableBtn')).toContain('加密')
    expect(t('policy.removeBtn')).toContain('策略')
    expect(t('cors.clearAll')).toContain('清空')
    setLocale('en-US')
    expect(t('accounts.saveLogin')).toContain('sign in')
    expect(t('overview.enableTitle')).toContain('versioning')
    expect(t('website.disableBtn')).toContain('website')
    expect(tf('lifecycle.toastSaved', { n: 2 })).toContain('2')
    expect(t('acl.publicRead')).toContain('Public read')
    expect(t('encryption.disableBtn')).toContain('encryption')
    expect(t('policy.removeBtn')).toContain('policy')
    expect(t('cors.clearAll')).toContain('Clear')
  })

  it('covers ctx/tags/detail/headers/storage/bucketTags keys', () => {
    setLocale('zh-CN')
    expect(t('ctx.openFolder')).toContain('文件夹')
    expect(t('tags.title')).toContain('标签')
    expect(t('detail.editHeaders')).toContain('HTTP')
    expect(t('headers.customMeta')).toContain('元数据')
    expect(t('storage.switch')).toBe('切换')
    expect(t('bucketTags.clearAll')).toContain('清空')
    expect(t('buckets.enter')).toBe('进入')
    setLocale('en-US')
    expect(t('ctx.openFolder')).toContain('folder')
    expect(t('tags.title')).toContain('tags')
    expect(t('detail.editHeaders')).toContain('HTTP')
    expect(t('headers.customMeta')).toContain('metadata')
    expect(t('storage.switch')).toBe('Change')
    expect(t('bucketTags.clearAll')).toContain('Clear')
    expect(t('buckets.enter')).toBe('Open')
  })

  it('keeps zh/en key parity above 640', () => {
    const zh = i18nKeyCount('zh-CN')
    const en = i18nKeyCount('en-US')
    expect(zh).toBe(en)
    expect(zh).toBeGreaterThanOrEqual(640)
    // 缺省参数分支：不传 locale 时统计 zh-CN。
    expect(i18nKeyCount()).toBe(zh)
  })

  it('tf replaces placeholders', () => {
    setLocale('zh-CN')
    expect(tf('objects.noMatch', { q: 'abc' })).toContain('abc')
    expect(tf('server.active', { name: 'local' })).toContain('local')
    expect(tf('compare.error', { msg: 'CORS' })).toContain('CORS')
  })

  // 回归：这 3 个键曾被引用但未定义，用户会看到原始 key（2026-09-16 评估 L4 / R1）。
  // 断言「解析结果不等于 key 本身」，而非断言具体文案，兼顾文案微调与缺失检测。
  it('resolves previously-missing keys instead of echoing the raw key', () => {
    for (const loc of ['zh-CN', 'en-US'] as const) {
      setLocale(loc)
      for (const key of ['objects.toastCopyFailed', 'batchEdit.tagsNeedKey', 'common.working']) {
        expect(t(key), `${loc} ${key}`).not.toBe(key)
        expect(t(key).length).toBeGreaterThan(0)
      }
    }
    setLocale('zh-CN')
  })

  // 覆盖率门禁去水分（roadmap #11）：i18n 纳入统计后，这些分支必须有行为断言，
  // 而非靠排除目录「注水」。
  it('locale() 反映 setLocale 的当前语言', () => {
    setLocale('en-US')
    expect(locale()).toBe('en-US')
    setLocale('zh-CN')
    expect(locale()).toBe('zh-CN')
  })

  it('cycleLocale 在 zh-CN → en-US → zh-CN 之间循环', () => {
    setLocale('zh-CN')
    expect(cycleLocale()).toBe('en-US')
    expect(locale()).toBe('en-US')
    expect(cycleLocale()).toBe('zh-CN')
    expect(locale()).toBe('zh-CN')
  })

  it('t() 对未知 key 回退为 key 本身（缺省兜底分支）', () => {
    setLocale('zh-CN')
    expect(t('definitely.not.defined')).toBe('definitely.not.defined')
    setLocale('en-US')
    expect(t('definitely.not.defined')).toBe('definitely.not.defined')
  })

  it('readLocale：localStorage 抛错时回退默认 zh-CN（异常分支）', async () => {
    vi.stubGlobal('localStorage', {
      getItem: () => {
        throw new Error('blocked')
      },
      setItem: () => {},
      removeItem: () => {},
    })
    vi.resetModules()
    const mod = await import('./index')
    expect(mod.locale()).toBe('zh-CN')
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('readLocale：localStorage 返回非法值时回退默认 zh-CN（校验分支）', async () => {
    vi.stubGlobal('localStorage', {
      getItem: () => 'fr-FR',
      setItem: () => {},
      removeItem: () => {},
    })
    vi.resetModules()
    const mod = await import('./index')
    expect(mod.locale()).toBe('zh-CN')
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('readLocale：localStorage 中的合法语言被采用', async () => {
    vi.stubGlobal('localStorage', {
      getItem: () => 'en-US',
      setItem: () => {},
      removeItem: () => {},
    })
    vi.resetModules()
    const mod = await import('./index')
    expect(mod.locale()).toBe('en-US')
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('setLocale：localStorage 写入抛错被吞掉，内存态仍生效', () => {
    vi.stubGlobal('localStorage', {
      getItem: () => null,
      setItem: () => {
        throw new Error('quota')
      },
      removeItem: () => {},
    })
    setLocale('en-US')
    expect(locale()).toBe('en-US')
    vi.unstubAllGlobals()
    setLocale('zh-CN')
  })
})
