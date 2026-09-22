import { describe, expect, it } from 'vitest'
import {
  POLICY_TEMPLATES,
  normalizeStringArray,
  parsePolicy,
  serializePolicy,
  validateDoc,
  type PolicyDoc,
} from './bucketPolicy'

describe('bucketPolicy', () => {
  it('parses empty string to empty doc', () => {
    expect(parsePolicy('')).toEqual({ Version: '2012-10-17', Statement: [] })
    expect(parsePolicy('   \n  ')).toEqual({ Version: '2012-10-17', Statement: [] })
  })

  it('returns null for valid JSON that is not an object', () => {
    expect(parsePolicy('123')).toBeNull()
    expect(parsePolicy('null')).toBeNull()
    expect(parsePolicy('"s3:GetObject"')).toBeNull()
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

  it('normalizeStringArray rejects non-string/array values', () => {
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

describe('parsePolicy edge branches', () => {
  it('missing Version/Statement fields fall back to defaults', () => {
    expect(parsePolicy('{}')).toEqual({ Version: '2012-10-17', Statement: [] })
    expect(parsePolicy('{"Version":1,"Statement":"x"}')).toEqual({ Version: '2012-10-17', Statement: [] })
  })

  it('rejects malformed statement shapes', () => {
    expect(parsePolicy('{"Statement":["x"]}')).toBeNull() // non-object statement
    expect(parsePolicy('{"Statement":[{"Effect":"Nope","Principal":"*","Action":"s3:*","Resource":"*"}]}')).toBeNull()
    expect(parsePolicy('{"Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}')).toBeNull() // no principal
    expect(parsePolicy('{"Statement":[{"Effect":"Allow","Principal":"*","Resource":"*"}]}')).toBeNull() // no action
    expect(parsePolicy('{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*"}]}')).toBeNull() // no resource
    expect(parsePolicy('{"Statement":[{"Effect":"Allow","Principal":["*"],"Action":"s3:*","Resource":"*"}]}')).toBeNull() // array principal
  })

  it('NotAction/NotResource fallbacks and principal variants', () => {
    const viaNotAction = parsePolicy('{"Statement":[{"Effect":"Allow","Principal":"*","NotAction":"s3:*","Resource":"*"}]}')
    expect(viaNotAction?.Statement[0].actions).toEqual(['s3:*'])
    const viaNotResource = parsePolicy('{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*","NotResource":"b/*"}]}')
    expect(viaNotResource?.Statement[0].resources).toEqual(['b/*'])

    const awsObj = parsePolicy('{"Statement":[{"Effect":"Deny","Principal":{"AWS":"arn:aws:iam::1:root"},"Action":"s3:*","Resource":"*"}]}')
    expect(awsObj?.Statement[0].principal).toBe('arn:aws:iam::1:root')
    const awsStar = parsePolicy('{"Statement":[{"Effect":"Deny","Principal":{"AWS":"*"},"Action":"s3:*","Resource":"*"}]}')
    expect(awsStar?.Statement[0].principal).toBe('*')
    const prefixed = parsePolicy('{"Statement":[{"Effect":"Deny","Principal":"AWS:arn:x","Action":"s3:*","Resource":"*"}]}')
    expect(prefixed?.Statement[0].principal).toBe('arn:x')
    expect(parsePolicy('{"Statement":[{"Effect":"Deny","Principal":123,"Action":"s3:*","Resource":"*"}]}')).toBeNull()
  })

  it('reads Sid string', () => {
    const doc = parsePolicy('{"Statement":[{"Sid":"S1","Effect":"Allow","Principal":"*","Action":"s3:*","Resource":"*"}]}')
    expect(doc?.Statement[0].sid).toBe('S1')
  })
})

describe('serializePolicy', () => {
  it('produces S3-compatible JSON with principal variants and drops undefined Sid', () => {
    const withSid = parsePolicy('{"Statement":[{"Sid":"S1","Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"b/*"}]}')!
    const back1 = JSON.parse(serializePolicy(withSid))
    expect(back1.Statement[0]).toEqual({
      Sid: 'S1', Effect: 'Allow', Principal: '*', Action: ['s3:GetObject'], Resource: ['b/*'],
    })

    const withAws = parsePolicy('{"Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:zzz"},"Action":"s3:*","Resource":"*"}]}')!
    const back2 = JSON.parse(serializePolicy(withAws))
    expect(back2.Statement[0].Principal).toEqual({ AWS: 'arn:zzz' })

    const noSid = parsePolicy('{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*","Resource":"*"}]}')!
    const back3 = JSON.parse(serializePolicy(noSid))
    expect(back3.Statement[0]).not.toHaveProperty('Sid')
  })
})

describe('validateDoc error branches', () => {
  const good = () => ({
    Version: '2012-10-17' as const,
    Statement: [{ sid: 'S', effect: 'Allow' as const, principal: '*', actions: ['s3:GetObject'], resources: ['b/*'] }],
  })

  /** validateDoc 接受 PolicyDoc，但这里要故意传入非法结构，故用 unknown 桥接。 */
  const validateRaw = (doc: unknown): string | null => validateDoc(doc as PolicyDoc)

  it('rejects bad Version/Statement', () => {
    expect(validateRaw({ Version: '2006-03-01', Statement: [] })).toMatch(/Version/)
    expect(validateRaw({ Version: '2012-10-17', Statement: {} })).toMatch(/Statement/)
  })

  it('rejects illegal Effect/empty principal/actions/resources', () => {
    expect(validateRaw({ ...good(), Statement: [{ ...good().Statement[0], effect: 'Bogus' }] })).toMatch(/Effect/)
    expect(validateDoc({ ...good(), Statement: [{ ...good().Statement[0], principal: '' }] })).toMatch(/Principal/)
    expect(validateDoc({ ...good(), Statement: [{ ...good().Statement[0], actions: [] }] })).toMatch(/Action/)
    expect(validateDoc({ ...good(), Statement: [{ ...good().Statement[0], resources: [] }] })).toMatch(/Resource/)
  })

  it('rejects duplicate Sid', () => {
    const dup = {
      Version: '2012-10-17' as const,
      Statement: [
        { sid: 'A', effect: 'Allow' as const, principal: '*', actions: ['s3:*'], resources: ['*'] },
        { sid: 'A', effect: 'Deny' as const, principal: '*', actions: ['s3:*'], resources: ['*'] },
      ],
    }
    expect(validateDoc(dup)).toMatch(/重复/)
  })

  it('accepts valid doc', () => {
    expect(validateDoc(good())).toBeNull()
  })

  it('normalizePrincipal: { AWS: "*" } 归一化为 "*"', () => {
    const raw = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        { Effect: 'Allow', Principal: { AWS: '*' }, Action: 's3:GetObject', Resource: 'arn:aws:s3:::b/*' },
      ],
    })
    const doc = parsePolicy(raw)
    expect(doc).not.toBeNull()
    expect(doc!.Statement[0].principal).toBe('*')
  })

  it('normalizeStringArray: 数组含非字符串元素时返回 null', () => {
    expect(normalizeStringArray(['s3:GetObject', 42])).toBeNull()
    expect(normalizeStringArray(['s3:GetObject', 's3:PutObject'])).toEqual(['s3:GetObject', 's3:PutObject'])
  })
})
