import { describe, expect, it, vi } from 'vitest'
import { usePreview } from './usePreview'

vi.mock('../preview', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../preview')>()
  return {
    ...actual,
    previewKind: vi.fn((_key: string) => 'text'),
  }
})

vi.mock('../proxy', () => ({
  proxyUrl: vi.fn(() => 'http://proxy/inline'),
}))

vi.mock('../api', () => ({
  api: { base: 'http://localhost' },
}))

describe('usePreview', () => {
  it('showPreview sets preview state', () => {
    const ctx = {
      account: { value: { id: 'a1' } },
      currentBucket: { value: 'b1' },
      getCtxEntry: vi.fn(),
      closeCtx: vi.fn(),
      download: vi.fn(),
    }
    const result = usePreview(ctx as any)
    const o = { key: 'test.txt' } as any
    result.showPreview(o)
    expect(result.preview.value).not.toBeNull()
    expect(result.preview.value!.key).toBe('test.txt')
  })

  it('showPreview sets empty url for unknown kind', async () => {
    const mod = await import('../preview')
    vi.mocked(mod.previewKind).mockReturnValue('none')
    const ctx = {
      account: { value: { id: 'a1' } },
      currentBucket: { value: 'b1' },
      getCtxEntry: vi.fn(),
      closeCtx: vi.fn(),
      download: vi.fn(),
    }
    const result = usePreview(ctx as any)
    const o = { key: 'unknown.xyz' } as any
    result.showPreview(o)
    expect(result.preview.value!.url).toBe('')
  })

  it('previewOrDownload calls download for unknown kind', async () => {
    const mod = await import('../preview')
    vi.mocked(mod.previewKind).mockReturnValue('none')
    const download = vi.fn()
    const ctx = {
      account: { value: { id: 'a1' } },
      currentBucket: { value: 'b1' },
      getCtxEntry: vi.fn(),
      closeCtx: vi.fn(),
      download,
    }
    const result = usePreview(ctx as any)
    result.previewOrDownload({ key: 'unknown.xyz' } as any)
    expect(download).toHaveBeenCalled()
  })

  it('previewOrDownload calls showPreview for known kind', async () => {
    const mod = await import('../preview')
    vi.mocked(mod.previewKind).mockReturnValue('text')
    const download = vi.fn()
    const ctx = {
      account: { value: { id: 'a1' } },
      currentBucket: { value: 'b1' },
      getCtxEntry: vi.fn(),
      closeCtx: vi.fn(),
      download,
    }
    const result = usePreview(ctx as any)
    result.previewOrDownload({ key: 'test.txt' } as any)
    expect(download).not.toHaveBeenCalled()
    expect(result.preview.value).not.toBeNull()
  })

  it('ctxPreview calls showPreview for file entries', () => {
    const ctx = {
      account: { value: { id: 'a1' } },
      currentBucket: { value: 'b1' },
      getCtxEntry: vi.fn(() => ({ kind: 'file', object: {} })),
      closeCtx: vi.fn(),
      download: vi.fn(),
    }
    const result = usePreview(ctx as any)
    result.ctxPreview()
    expect(ctx.closeCtx).toHaveBeenCalled()
  })

  it('ctxPreview returns early for non-file entries', () => {
    const ctx = {
      account: { value: { id: 'a1' } },
      currentBucket: { value: 'b1' },
      getCtxEntry: vi.fn(() => ({ kind: 'folder' })),
      closeCtx: vi.fn(),
      download: vi.fn(),
    }
    const result = usePreview(ctx as any)
    result.ctxPreview()
    expect(ctx.closeCtx).toHaveBeenCalled()
  })
})
