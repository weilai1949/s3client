import { describe, expect, it } from 'vitest'
import {
  POLICY_TEMPLATES,
  normalizeStringArray,
  parsePolicy,
  serializePolicy,
  validateDoc,
} from './bucketPolicy'

describe('bucketPolicy', () => {
  it('parses empty string to empty doc', () => {
    expect(parsePolicy('')).toEqual({ Version: '2012-10-17', Statement: [] })
  })

  it('round-trips a simple policy', () => {
    const raw = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Sid: 'AllowPublicRead',
          Effect: 'Allow',
          Principal: '*',
          Action: 's3:GetObject',
          Resource: 'arn:aws:s3:::bucket/*',
        },
      ],
    })
    const doc = parsePolicy(raw)
    expect(doc).not.toBeNull()
    expect(validateDoc(doc!)).toBeNull()
    const back = serializePolicy(doc!)
    // parse 归一化为数组；AWS 同时接受字符串与数组两种形式，
    // 此处断言「再次解析后语义不变」即可。
    const backParsed = JSON.parse(back)
    expect(backParsed.Statement[0].Effect).toBe('Allow')
    expect(backParsed.Statement[0].Principal).toBe('*')
    expect(backParsed.Statement[0].Sid).toBe('AllowPublicRead')
    expect(backParsed.Statement[0].Action).toEqual(['s3:GetObject'])
    expect(backParsed.Statement[0].Resource).toEqual(['arn:aws:s3:::bucket/*'])
  })

  it('accepts array Action/Resource', () => {
    const raw = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Effect: 'Allow',
          Principal: '*',
          Action: ['s3:GetObject', 's3:PutObject'],
          Resource: ['arn:aws:s3:::b/*', 'arn:aws:s3:::b'],
        },
      ],
    })
    expect(parsePolicy(raw)).not.toBeNull()
  })

  it('strips AWS: prefix in Principal', () => {
    const raw = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Effect: 'Allow',
          Principal: { AWS: 'arn:aws:iam::123:root' },
          Action: 's3:*',
          Resource: 'arn:aws:s3:::b/*',
        },
      ],
    })
    const doc = parsePolicy(raw)
    expect(doc).not.toBeNull()
    expect(doc!.Statement[0].principal).toBe('arn:aws:iam::123:root')
  })

  it('rejects invalid JSON', () => {
    expect(parsePolicy('not-json')).toBeNull()
  })

  it('rejects unsupported structure (nested object)', () => {
    const raw = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Effect: 'Allow',
          Principal: { AWS: ['arn:1', 'arn:2'] }, // 数组 Principal 暂不支持
          Action: 's3:GetObject',
          Resource: 'arn:aws:s3:::b/*',
        },
      ],
    })
    expect(parsePolicy(raw)).toBeNull()
  })

  it('validates duplicate sids', () => {
    const raw = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        { Sid: 'S1', Effect: 'Allow', Principal: '*', Action: 's3:GetObject', Resource: 'arn:aws:s3:::b/*' },
        { Sid: 'S1', Effect: 'Allow', Principal: '*', Action: 's3:PutObject', Resource: 'arn:aws:s3:::b/*' },
      ],
    })
    const doc = parsePolicy(raw)!
    expect(validateDoc(doc)).toMatch(/重复/)
  })

  it('templates generate a non-empty Statement', () => {
    for (const t of POLICY_TEMPLATES) {
      if (t.id === 'clear') continue
      const doc = t.build('my-bucket')
      expect(doc.Statement.length).toBeGreaterThan(0)
      expect(validateDoc(doc)).toBeNull()
    }
  })

  it('serialize omits undefined Sid', () => {
    const raw = serializePolicy({
      Version: '2012-10-17',
      Statement: [{ effect: 'Allow', principal: '*', actions: ['s3:GetObject'], resources: ['arn:aws:s3:::b/*'] }],
    })
    const parsed = JSON.parse(raw)
    expect(parsed.Statement[0]).not.toHaveProperty('Sid')
  })

  it('normalizeStringArray rejects non-string/array values (line 111)', () => {
    expect(normalizeStringArray(42)).toBeNull()
    expect(normalizeStringArray(true)).toBeNull()
    expect(normalizeStringArray({})).toBeNull()
  })

  it('clear template generates empty Statement', () => {
    const clearTemplate = POLICY_TEMPLATES.find((t) => t.id === 'clear')
    expect(clearTemplate).toBeDefined()
    const doc = clearTemplate!.build('my-bucket')
    expect(doc.Statement).toEqual([])
    expect(validateDoc(doc)).toBeNull()
  })
})
