/// <reference types="vitest/config" />

// 必须在 import 任何依赖之前执行：Vue 的 node 入口（`vue/index.js`）是运行时按
// `process.env.NODE_ENV` 二选一加载 CJS 产物的——
//   production → dist/vue.cjs.prod.js     development/其它 → dist/vue.cjs.js
// 宿主环境若带着 NODE_ENV=production（常见于 CI 镜像 / 容器 / Makefile 链路），
// vitest 会把 Vue 解析到 **prod 构建**，而 ` @vitejs/plugin-vue` 编译 SFC 生成的
// 绑定符号属于 **dev 构建** ⇒ 进程里同时存在两份 Vue，reactive 状态互不连通。
// 可观测形态极具误导性：不是报错缺失模块，而是 mock 打不进组件——
//   `[vitest] No "toasts" export is defined on the "./store" mock`
//   （导出确实存在，只是属于另一份 Vue 实例），连带 30+ 组件测试全红
//   （`Cannot call text on an empty DOMWrapper` / `w.vm.xxx is not a function` 等）。
// 因此这里把值钉成 'test'（非 production 即走 dev 构建），使测试结果与宿主环境解耦。
// 值本身用 'test' 而非 'development'：两者都落到 dev 构建、都能修复本问题，但 'test'
// 是 vitest 惯用值，且不会顺手开启 dev-only 的调试分支（如 devtools 钩子），少一层
// 与生产行为的差异。
//
// 注意：不能写成 vitest 的 `test.env`——注入时机晚于依赖解析，实测无效。
//（早前曾误判 'development' 在 --coverage 下更慢：后续补跑证实该 5s 超时是
//  并发跑批时的资源竞争，与取值无关，已纠正。）
//
// `process.env.VITEST` 是 vitest 注入的标记，用来区分「跑测试」与「跑
// `vite build`」——**这一步不能省**：生产构建读的是同一个 config 文件，若把
// NODE_ENV 无条件钉成非 production，产物会连 Vue 的 dev/warn 分支一起打进包，
// 实测 bundle 从 366.66 kB 膨胀到 424.72 kB（+58 kB）。只在 VITEST 下改写，
// 生产打包路径不受影响。
if (process.env.VITEST) {
  process.env.NODE_ENV = 'test'
}
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

// 前端构建。输出到 dist，Go 后端把 dist 作为静态资源托管。
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 1949,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:5000',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  test: {
    environment: 'happy-dom',
    // e2e/**（mock /api 的浏览器侧用例）与 e2e-real/**（真实后端 + RustFS，KNOWN_ISSUES #37）
    // 都由 Playwright 跑，不是 vitest 单测；不排除会被 vitest 当作 *.spec.ts 收走。
    exclude: ['node_modules/**', 'e2e/**', 'e2e-real/**', 'dist/**'],
    coverage: {
      provider: 'istanbul',
      reporter: ['text', 'text-summary', 'html', 'lcov'],
      include: ['src/**/*.{ts,vue}'],
      exclude: [
        'src/main.ts',
        'src/env.d.ts',
        'src/**/*.test.ts',
        // i18n 曾整体排除（字典庞大、纯数据）。覆盖率去水分（见 docs/FEATURES.md §M）：
        // 字典数据由 coverage.test.ts 的键完整性门禁覆盖，故仅排除纯数据模块；
        // index.ts（读/写/回退/循环逻辑）纳入统计。
        'src/i18n/messages/**',
        'src/assets/**',
      ],
      // 门槛：四指标均已达成 100%，设为 100 作为回归护栏
      // （防新增业务代码无测试回落）。
      thresholds: {
        statements: 100,
        functions: 100,
        branches: 100,
        lines: 100,
      },
    },
  },
})
