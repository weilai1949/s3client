import { describe, expect, it } from 'vitest'
import { proxyUrl } from './proxy'

describe('proxyUrl', () => {
  it('constructs proxy URL with required params', () => {
    const url = proxyUrl('acc1', 'mybucket', 'download', 'key/to/file', 'https://api.example.com')
    expect(url).toBe('https://api.example.com/api/accounts/acc1/proxy?bucket=mybucket&key=key%2Fto%2Ffile&mode=download')
  })

  it('includes versionId when provided', () => {
    const url = proxyUrl('acc1', 'mybucket', 'inline', 'key', 'https://api.example.com', 'v123')
    expect(url).toContain('versionId=v123')
  })

  it('excludes versionId when not provided', () => {
    const url = proxyUrl('acc1', 'mybucket', 'text', 'key', 'https://api.example.com')
    expect(url).not.toContain('versionId')
  })

  it('supports all modes', () => {
    expect(proxyUrl('a', 'b', 'download', 'k', 'https://api.example.com')).toContain('mode=download')
    expect(proxyUrl('a', 'b', 'inline', 'k', 'https://api.example.com')).toContain('mode=inline')
    expect(proxyUrl('a', 'b', 'text', 'k', 'https://api.example.com')).toContain('mode=text')
  })
})
