import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BucketPolicyVisualEditor from './BucketPolicyVisualEditor.vue'
import { POLICY_TEMPLATES, serializePolicy } from '../bucketPolicy'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

vi.mock('../bucketPolicy', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../bucketPolicy')>()
  return { ...actual, serializePolicy: vi.fn(actual.serializePolicy) }
})

const validRaw = JSON.stringify({
  Version: '2012-10-17',
  Statement: [
    { Sid: 'S1', Effect: 'Allow', Principal: '*', Action: 's3:GetObject', Resource: 'arn:aws:s3:::my-bucket/*' },
  ],
})

function lastUpdate(w: ReturnType<typeof mount>): string {
  const events = w.emitted('update')!
  return events[events.length - 1][0] as string
}

describe('BucketPolicyVisualEditor', () => {
  it('默认可视化模式，渲染模板与语句卡片', () => {
    const w = mount(BucketPolicyVisualEditor, {
      props: { bucket: 'my-bucket', raw: validRaw },
    })
    // 模板按钮（除 clear 外都来自 POLICY_TEMPLATES）
    for (const tpl of POLICY_TEMPLATES) {
      const btn = w.findAll('button').find((b) => b.text() === tpl.label)
      expect(btn, `template button ${tpl.label} should exist`).toBeTruthy()
    }
    expect(w.find('[data-testid="policy-stmt-0"]').exists()).toBe(true)
    expect(w.text()).toContain('policy.parsedOk')
  })

  it('JSON 模式切换并回显 raw，输入触发 update 事件', async () => {
    const w = mount(BucketPolicyVisualEditor, {
      props: { bucket: 'my-bucket', raw: validRaw },
    })
    await w.find('[data-testid="policy-mode-json"]').trigger('click')
    const area = w.find('textarea.policy-area')
    expect(area.exists()).toBe(true)
    expect((area.element as HTMLTextAreaElement).value).toBe(validRaw)
    await area.setValue('{"Version":"2012-10-17","Statement":[]}')
    expect(lastUpdate(w)).toBe('{"Version":"2012-10-17","Statement":[]}')
    // 切回可视化模式
    await w.find('[data-testid="policy-mode-visual"]').trigger('click')
    expect(w.find('textarea.policy-area').exists()).toBe(false)
  })

  it('应用模板生成策略 JSON 并设置 dirty 标记', async () => {
    const w = mount(BucketPolicyVisualEditor, {
      props: { bucket: 'my-bucket', raw: '' },
    })
    const tpl = POLICY_TEMPLATES.find((x) => x.id === 'public-read')!
    const btn = w.findAll('button').find((b) => b.text() === tpl.label)
    expect(btn).toBeTruthy()
    await btn!.trigger('click')
    const emitted = lastUpdate(w)
    const doc = JSON.parse(emitted)
    expect(doc.Statement.length).toBe(1)
    expect(emitted).toContain('my-bucket')
    // 未保存标记
    expect(w.find('.badge.dirty').exists()).toBe(true)
  })

  it('不可解析的 raw 自动切到 JSON 模式并提示', () => {
    const w = mount(BucketPolicyVisualEditor, {
      props: { bucket: 'my-bucket', raw: 'not-json' },
    })
    expect(w.find('textarea.policy-area').exists()).toBe(true)
    expect(w.text()).toContain('policy.parsedFail')
  })
})

describe('BucketPolicyVisualEditor extra branches', () => {
  it('添加/删除语句卡片 + 编辑字段触发 update 与 dirty', async () => {
    const w = mount(BucketPolicyVisualEditor, { props: { raw: '', bucket: 'my-bucket' } })
    // 添加语句 → update + dirty
    await w.findAll('button').find((b) => b.text() === 'policy.addStatement')!.trigger('click')
    expect(w.emitted('update')).toHaveLength(1)
    expect(w.text()).toContain('policy.dirty')

    // 删除第一条 → 再次 update
    await w.find('[data-testid="policy-stmt-0"]').find('button.danger').trigger('click')
    expect(w.emitted('update')).toHaveLength(2)
    expect(w.findAll('.stmt-card')).toHaveLength(0)
  })

  it('编辑 actions/resources 文本域解析多行', async () => {
    const w = mount(BucketPolicyVisualEditor, {
      props: { raw: validRaw, bucket: 'my-bucket' },
    })
    const stmt = w.find('[data-testid="policy-stmt-0"]')
    // actions textarea 是卡内第一个 textarea
    const ta = stmt.findAll('textarea')
    await ta[0].setValue('s3:GetObject\ns3:PutObject\n\n')
    const updated = JSON.parse(lastUpdate(w))
    expect(updated.Statement[0].Action).toEqual(['s3:GetObject', 's3:PutObject'])

    await ta[1].setValue('arn:aws:s3:::my-bucket/*\narn:aws:s3:::my-bucket/cat')
    const updated2 = JSON.parse(lastUpdate(w))
    expect(updated2.Statement[0].Resource).toEqual(['arn:aws:s3:::my-bucket/*', 'arn:aws:s3:::my-bucket/cat'])
  })

  it('修改 principal/sid/effect/资源等字段', async () => {
    const w = mount(BucketPolicyVisualEditor, {
      props: { raw: validRaw, bucket: 'my-bucket' },
    })
    const stmt = w.find('[data-testid="policy-stmt-0"]')
    const sel = stmt.find('select') // effect
    await sel.setValue('Deny')
    expect(JSON.parse(lastUpdate(w)).Statement[0].Effect).toBe('Deny')

    const inputs = stmt.findAll('input')
    // sid 输入 + principal 输入
    await inputs[0].setValue('S2')
    expect(JSON.parse(lastUpdate(w)).Statement[0].Sid).toBe('S2')
    await inputs[1].setValue('arn:aws:iam::1:root')
    expect(JSON.parse(lastUpdate(w)).Statement[0].Principal).toEqual({ AWS: 'arn:aws:iam::1:root' })
  })

  it('模板按钮应用策略并 emit update', async () => {
    const w = mount(BucketPolicyVisualEditor, { props: { raw: '', bucket: 'tb' } })
    const publicReadBtn = w.findAll('button').find((b) => b.text() === 'policy.tplPublicRead')!
    await publicReadBtn.trigger('click')
    const upd = JSON.parse(lastUpdate(w))
    expect(upd.Statement[0].Effect).toBe('Allow')
    expect(upd.Statement[0].Resource).toEqual(['arn:aws:s3:::tb/*'])
  })

  it('raw 变化解析：不同内容重载 doc、等价内容跳过、空清空、非法切 JSON', async () => {
    const w = mount(BucketPolicyVisualEditor, { props: { raw: '', bucket: 'b' } })
    // 非法 JSON → json 模式 + parseError
    await w.setProps({ raw: 'not-json' })
    expect(w.text()).toContain('policy.parsedFail')
    expect(w.findAll('[data-testid="policy-mode-json"]')[0].classes()).toContain('primary')

    // 空串 → parsePolicy('') 返回空 doc（bucketPolicy.ts:32 对 trim 为空直接返回空 doc），
    // 走「解析成功」分支；组件 49-54 行的空清空分支（parsePolicy 返回 null 且 trim 为空）
    // 覆盖不可达：trim 为空 ⇒ parsePolicy 必然返回非 null 空 doc，属防御代码。
    await w.setProps({ raw: '' })
    expect(w.text()).not.toContain('policy.parsedFail')

    // 回到可视化模式（parseError 不清除 mode）
    await w.find('[data-testid="policy-mode-visual"]').trigger('click')

    // 合法内容
    await w.setProps({ raw: validRaw })
    expect(w.find('[data-testid="policy-stmt-0"]').exists()).toBe(true)

    // 相同 raw 值再喂一次：Vue 对相同 prop 值去重，不会再次触发 watcher；
    // 「等价内容不重解析」由下方独立用例覆盖（不同文本、相同语义）。
    await w.setProps({ raw: validRaw })
    expect(w.emitted('update') ?? []).toHaveLength(0)
  })

  it('等价内容（序列化相同）不重解析 doc', async () => {
    const w = mount(BucketPolicyVisualEditor, { props: { raw: validRaw, bucket: 'my-bucket' } })
    const docBefore = (w.vm as any).doc
    // 同一语义、不同文本：键序与空白变化
    const equivalent = JSON.stringify(
      {
        Statement: [
          { Resource: 'arn:aws:s3:::my-bucket/*', Action: 's3:GetObject', Principal: '*', Effect: 'Allow', Sid: 'S1' },
        ],
        Version: '2012-10-17',
      },
      null,
      4,
    )
    await w.setProps({ raw: equivalent })
    // 序列化结果相同 → doc 未被重新赋值（同一引用）
    expect((w.vm as any).doc).toBe(docBefore)
    expect(w.emitted('update') ?? []).toHaveLength(0)
  })

  it('预览 JSON 失败回退错误消息', async () => {
    const w = mount(BucketPolicyVisualEditor, { props: { raw: '', bucket: 'b' } })
    // 正常 doc 序列化成功
    expect((w.vm as any).previewJSON).toBeTypeOf('string')
    // 构造循环引用使 serializePolicy → JSON.stringify 抛 TypeError → catch 回退错误消息
    const circular: Record<string, unknown> = {}
    circular.self = circular
    ;(w.vm as any).doc = {
      Version: '2012-10-17',
      Statement: [
        { sid: circular, effect: 'Allow', principal: '*', actions: ['s3:GetObject'], resources: ['arn:aws:s3:::b/*'] },
      ],
    }
    expect((w.vm as any).previewJSON).toBeTypeOf('string')
    expect(String((w.vm as any).previewJSON)).toMatch(/circular/i)
  })

  it('未知模板 id 时 applyTemplate 走守卫直接返回（不 emit update）', () => {
    const w = mount(BucketPolicyVisualEditor, { props: { raw: '', bucket: 'my-bucket' } })
    ;(w.vm as unknown as { applyTemplate: (id: string) => void }).applyTemplate('no-such-template')
    expect(w.emitted('update')).toBeUndefined()
  })

  it('序列化抛出无 message 的异常 → previewJSON 回退 String(e)', async () => {
    vi.mocked(serializePolicy).mockImplementationOnce(() => {
      throw undefined
    })
    const w = mount(BucketPolicyVisualEditor, { props: { raw: '', bucket: 'b' } })
    // (e as Error)?.message 短路 + ?? e 右分支 → String(undefined)
    expect((w.vm as any).previewJSON).toBe('undefined')
  })

  it('Sid 缺失的语句渲染空输入的 ?? 分支', () => {
    const rawNoSid = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        { Effect: 'Allow', Principal: '*', Action: 's3:GetObject', Resource: 'arn:aws:s3:::my-bucket/*' },
      ],
    })
    const w = mount(BucketPolicyVisualEditor, { props: { raw: rawNoSid, bucket: 'my-bucket' } })
    const stmt = w.find('[data-testid="policy-stmt-0"]')
    const sidInput = stmt.findAll('input')[0]
    expect((sidInput.element as HTMLInputElement).value).toBe('')
  })

  it('无效策略（空 Resource）显示校验错误徽章', () => {
    const rawInvalid = JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        { Effect: 'Allow', Principal: '*', Action: 's3:GetObject', Resource: [] },
      ],
    })
    const w = mount(BucketPolicyVisualEditor, { props: { raw: rawInvalid, bucket: 'my-bucket' } })
    expect(w.find('.badge.error').exists()).toBe(true)
    expect(w.text()).toContain('Resource')
  })
})
