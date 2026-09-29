// vite_env_guard.test.ts —— 宿主 NODE_ENV 不得渗透进 vitest 进程。
//
// 背景：Vue 的 node 入口 `vue/index.js` 在运行时按 `process.env.NODE_ENV` 二选一
// 加载 CJS 产物（`production` → `dist/vue.cjs.prod.js`，其余 → `dist/vue.cjs.js`）。
// 若宿主带着 `NODE_ENV=production`（CI 镜像 / 容器 / Makefile 链路都可能），
// 而测试进程照单全收，Vue 会被解析到 **prod 构建**；但 `@vitejs/plugin-vue`
// 编译 SFC 产出的绑定符号属于 **dev 构建** ⇒ 进程里同时存在两份 Vue，
// reactive 状态互不连通、`vi.mock` 打不进组件。可观测形态极误导：
//
//	[vitest] No "toasts" export is defined on the "./store" mock
//
// 而 `src/store.ts` 的 `toasts` 导出**确实存在**——只是属于另一份 Vue 实例。
// 当时连带 33 个测试文件 / 246 例全红，排查花了不少时间才定位到环境变量。
//
// 修法在 `vite.config.ts` 顶部（`process.env.NODE_ENV = 'test'`，必须早于任何
// import）。本文件把这条不变量钉住：一旦有人删掉那行、或改成透传宿主值，
// 下面第一条断言立刻变红。
//
// 为什么有效：`process.env.NODE_ENV` 是在** vite 读取配置之前**由 config 文件
// 自身改写的，因此本测试看到的已是改写后的值；若那行被删，看到的就是宿主的
// `production`（在带该变量的环境里）/ `undefined`（干净环境里）。

import { describe, expect, it } from 'vitest'

describe('vite.config.ts 的 NODE_ENV 隔离', () => {
  it('测试进程不得运行在 NODE_ENV=production 下（否则 Vue 走 prod 构建，两份实例使 vi.mock 失效）', () => {
    expect(
      process.env.NODE_ENV,
      'NODE_ENV=production 会让 vue/index.js 加载 prod CJS 构建，与 SFC 编译产出的 dev 绑定不匹配 ⇒ ' +
        'reactive 状态与 vi.mock 双双失效（33 文件 / 246 例全红的根因）。' +
        '修法：在 vite.config.ts 顶部、import 之前钉死 process.env.NODE_ENV。',
    ).not.toBe('production')
  })

  it('NODE_ENV 必须是被显式钉住的值（保证 Vue 走 dev 构建）', () => {
    // 非 production 一律落到 vue.cjs.js（dev 构建），断言具体值是防止「改成透传
    // 宿主值」这类看似无害的改动：干净环境下它会是 undefined 从而侥幸通过，
    // 却在 CI / 容器里复发。
    expect(process.env.NODE_ENV).toBe('test')
  })
})
