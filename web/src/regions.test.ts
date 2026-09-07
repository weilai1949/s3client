import { describe, expect, it, vi } from 'vitest'
import {
  PROVIDER_GROUPS,
  PROVIDERS,
  providersInGroup,
  inferProvider,
  regionsFor,
  syncsPublicEndpoint,
  providerGroupLabel,
  providerLabel,
  providerDesc,
  regionLabel,
  type Provider,
} from './regions'

vi.mock('./i18n', () => ({
  locale: vi.fn(() => 'zh-CN'),
  t: vi.fn((k: string) => k),
}))

describe('PROVIDER_GROUPS', () => {
  it('has three groups', () => {
    expect(PROVIDER_GROUPS).toHaveLength(3)
    expect(PROVIDER_GROUPS.map(g => g.id)).toEqual(['compatible', 'domestic', 'overseas'])
  })
})

describe('providersInGroup', () => {
  it('returns only providers in the group', () => {
    const compatible = providersInGroup('compatible')
    expect(compatible.every(p => p.group === 'compatible')).toBe(true)
    expect(compatible.map(p => p.value)).toEqual(['s3', 'rustfs'])

    const domestic = providersInGroup('domestic')
    expect(domestic.every(p => p.group === 'domestic')).toBe(true)
    expect(domestic.map(p => p.value)).toEqual(['oss', 'cos', 'obs', 'tos', 'bos', 'jd', 'qiniu'])

    const overseas = providersInGroup('overseas')
    expect(overseas.every(p => p.group === 'overseas')).toBe(true)
    expect(overseas.map(p => p.value)).toEqual(['aws', 'r2', 'wasabi', 'b2', 'do', 'linode', 'scaleway', 'hetzner'])
  })
})

describe('inferProvider', () => {
  it('infers provider from endpoint domain', () => {
    expect(inferProvider('oss-cn-hangzhou.aliyuncs.com')).toBe('oss')
    expect(inferProvider('cos.ap-guangzhou.myqcloud.com')).toBe('cos')
    expect(inferProvider('obs.cn-north-4.myhuaweicloud.com')).toBe('obs')
    expect(inferProvider('tos-s3-cn-beijing.volces.com')).toBe('tos')
    expect(inferProvider('s3.bj.bcebos.com')).toBe('bos')
    expect(inferProvider('s3.cn-north-1.jdcloud-oss.com')).toBe('jd')
    expect(inferProvider('s3.cn-east-1.qiniucs.com')).toBe('qiniu')
    expect(inferProvider('ACCOUNT_ID.r2.cloudflarestorage.com')).toBe('r2')
    expect(inferProvider('s3.us-east-1.wasabisys.com')).toBe('wasabi')
    expect(inferProvider('s3.us-west-004.backblazeb2.com')).toBe('b2')
    expect(inferProvider('nyc3.digitaloceanspaces.com')).toBe('do')
    expect(inferProvider('us-east-1.linodeobjects.com')).toBe('linode')
    expect(inferProvider('s3.fr-par.scw.cloud')).toBe('scaleway')
    expect(inferProvider('fsn1.your-objectstorage.com')).toBe('hetzner')
    expect(inferProvider('s3.amazonaws.com')).toBe('aws')
    expect(inferProvider('custom.endpoint.com')).toBe('s3')
  })

  it('handles empty endpoint', () => {
    expect(inferProvider('')).toBe('s3')
    expect(inferProvider('   ')).toBe('s3')
  })
})

describe('regionsFor', () => {
  it('returns regions for each provider', () => {
    expect(regionsFor('oss')).toEqual(expect.any(Array))
    expect(regionsFor('cos')).toEqual(expect.any(Array))
    expect(regionsFor('aws')).toEqual(expect.any(Array))
    expect(regionsFor('s3')).toEqual([])
    expect(regionsFor('rustfs')).toEqual([])
  })
})

describe('syncsPublicEndpoint', () => {
  it('returns false for aws and s3', () => {
    expect(syncsPublicEndpoint('aws')).toBe(false)
    expect(syncsPublicEndpoint('s3')).toBe(false)
  })

  it('returns true for all other providers', () => {
    const providers: Provider[] = ['oss', 'cos', 'obs', 'tos', 'bos', 'jd', 'qiniu', 'r2', 'wasabi', 'b2', 'do', 'linode', 'scaleway', 'hetzner']
    providers.forEach(p => expect(syncsPublicEndpoint(p)).toBe(true))
  })
})

describe('providerGroupLabel', () => {
  it('calls t with correct key', async () => {
    const { t } = await import('./i18n')
    providerGroupLabel('compatible')
    expect(t).toHaveBeenCalledWith('provider.group.compatible')
  })
})

describe('providerLabel', () => {
  it('calls t with correct key', async () => {
    const { t } = await import('./i18n')
    providerLabel('oss')
    expect(t).toHaveBeenCalledWith('provider.oss.label')
  })
})

describe('providerDesc', () => {
  it('calls t with correct key', async () => {
    const { t } = await import('./i18n')
    providerDesc('oss')
    expect(t).toHaveBeenCalledWith('provider.oss.desc')
  })
})

describe('regionLabel', () => {
  it('returns zhLabel for zh-CN', async () => {
    const { locale } = await import('./i18n')
    locale.mockReturnValue('zh-CN')
    expect(regionLabel('oss-cn-hangzhou', '华东1（杭州）')).toBe('华东1（杭州）')
  })

  it('returns code for non-zh-CN', async () => {
    const { locale } = await import('./i18n')
    locale.mockReturnValue('en-US')
    expect(regionLabel('oss-cn-hangzhou', '华东1（杭州）')).toBe('oss-cn-hangzhou')
  })
})
