import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BucketPolicyVisualEditor from './BucketPolicyVisualEditor.vue'
import { POLICY_TEMPLATES } from '../bucketPolicy'

vi.mock('../i18n', () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
}))

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
