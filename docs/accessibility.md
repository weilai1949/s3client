# 可访问性（Accessibility）

> 本文件**盘点现状**：仓库里实际做了哪些可访问性工作、键盘与焦点如何运作、主题如何跟随系统，
> 以及**明确没做**的部分。所有数字都由文中的命令在 `apps/web/src` 上跑出来，统计口径写在各处；
> 结论只依据**代码与测试的实际存在**，不做任何合规性断言。
> 相关：[`glossary.md`](glossary.md)（术语）· [`i18n.md`](i18n.md)（文案体系）· [`FEATURES.md`](FEATURES.md) §一.12

> ⚠️ **本仓库没有做过任何正式的可访问性审计**，也**没有**任何 WCAG（A / AA / AAA）合规声明、
> 部分合规声明或第三方审计报告。本文件描述的是「已实现的机制」与「已知缺口」，
> **不构成**「符合 WCAG」或任何等级的合规结论。凡涉及「是否达标」的判断，请以一次真实的
> 人工审计 + 辅助技术实测为准，而不要引用本文件。

---

## 1. 现状盘点（统计口径与命令）

### 1.1 统计口径

- 范围：`apps/web/src/` 下的 **`*.vue` 与 `*.ts`**。
- **生产源码**＝排除 `*.test.ts`（测试文件里的 `aria-*` 是断言用字面量，不是界面属性；
  不排除会把数字抬高，本文所有数字都取排除后的值）。
- **重复出现即重复计数**：一个属性在模板里写 N 次就计 N 次（同一组件内多行同理）；
  换行写法（`:aria-\n label`）不会被正则统计到。
- 复现命令：

  ```bash
  cd apps/web/src
  # aria-* 属性出现次数（排除测试文件）
  for f in $(grep -rl --include=*.vue --include=*.ts 'aria-' . | grep -v '\.test\.ts$'); do grep -ho 'aria-[a-z]*' "$f"; done | sort | uniq -c | sort -rn
  # role 值分布（排除测试文件）
  for f in $(grep -rl --include=*.vue 'role="' . | grep -v '\.test\.ts$'); do grep -ho 'role="[^"]*"' "$f"; done | sort | uniq -c | sort -rn
  ```

### 1.2 ARIA 属性分布

| 属性 | 次数 | 主要用途（本仓库） |
|---|---:|---|
| `aria-hidden` | 43 | 纯装饰元素：emoji、内联 SVG 图标、虚拟滚动的上下垫片行（`v-spacer`） |
| `aria-label` | 28 | 无可见文字元素的名称：图标按钮（`✕` / `⋯` / `↻`）、语言与主题按钮、`<nav>` 与设置页 tab 行、`<select>` / `<input>`（批量改元数据、标签输入行）、列表与网格的行选择框、加载态容器（配合 `aria-busy`） |
| `aria-modal` | 4 | 四个模态容器，值均为 `"true"` |
| `aria-sort` | 3 | 列表视图三个可排序表头（名称 / 大小 / 时间） |
| `aria-busy` | 3 | 列举中 / 加载中的容器（`ObjectList`、`MigratePanel`、`LifecycleDialog`） |
| `aria-valuenow` / `aria-valuemin` / `aria-valuemax` | 2 / 2 / 2 | 两处 `role="progressbar"`（`UploadPanel` 总进度、`MigratePanel` 任务进度） |
| `aria-live` | 2 | 见 §1.4 |
| `aria-current` | 1 | 左侧主导航当前页（`aria-current="page"`，`App.vue`） |

### 1.3 `role` 分布

| role | 次数 | 位置 |
|---|---:|---|
| `menuitem` | 17 | `ObjectContextMenu.vue`（右键菜单的每个按钮） |
| `columnheader` | 3 | `ObjectList.vue` 三个可排序表头（配 `tabindex="0"` 与 `aria-sort`） |
| `status` | 3 | `ObjectList.vue`（网格截断提示）、`UploadQueue.vue`、`Toasts.vue`（单条 toast） |
| `dialog` | 3 | `ModalDialog.vue`、`PromptDialog.vue`、`PreviewOverlay.vue` |
| `progressbar` | 2 | `UploadPanel.vue`、`MigratePanel.vue` |
| `button` | 2 | `UploadPanel.vue` 拖放区、`ObjectList.vue` 网格单元格（自定义可点区域） |
| `menu` | 1 | `ObjectContextMenu.vue` 容器（`tabindex="-1"`） |
| `alertdialog` | 1 | `ConfirmDialog.vue`（破坏性操作确认） |
| `alert` | 1 | `ObjectsPanel.vue` 错误横幅 |

说明：**数据表没有显式 ARIA grid 角色**——它用的是原生 `<table>` / `<thead>` / `<th>` / `<tr>`，
语义由浏览器给出；`role="columnheader"` 只冗余地写在**那三个可排序表头**上（与 `tabindex="0"`、
`aria-sort` 配套，让「可排序」这件事对辅助技术可见），其余单元格没有任何 ARIA role。
复选框类控件用原生 `<input type="checkbox">` + `aria-label`，未自造 switch 角色。

### 1.4 `aria-live` 区域

全仓只有 **2 处**，且都是 `aria-live="polite"`（没有 `assertive`）：

| 位置 | 播报内容 |
|---|---|
| `Toasts.vue` | 全局 toast 容器（每条 toast 另带 `role="status"`） |
| `BatchMetadataDialog.vue` | 批量改元数据的执行状态 / 结果区（`v-if="running \|\| result"`） |

`ObjectsPanel.vue` 的错误横幅是 `role="alert"`（隐式 assertive），但它不经过 `aria-live` 属性。
**这两个 `aria-live` 区域都没有专门的测试断言**（见 §5.2）。

### 1.5 对话框的 `aria-modal`

四个模态容器都写了 `role` + `aria-modal="true"` + `aria-label`：

| 组件 | role | 名称来源 |
|---|---|---|
| `ModalDialog.vue`（通用弹窗，被各设置类对话框复用） | `dialog` | `:aria-label="title"` |
| `ConfirmDialog.vue` | `alertdialog` | `confirmState.title` |
| `PromptDialog.vue` | `dialog` | `promptState.title` |
| `PreviewOverlay.vue` | `dialog` | `tf('preview.aria', { key })` |

`ModalDialog.test.ts` 有对属性本身的断言（`renders footer slot, custom width and dialog attributes`
断言 `aria-modal === 'true'`）；其余三个组件没有对 `aria-modal` 的专项断言。

---

## 2. 键盘可达性

### 2.1 单一键栈：`useKeydownStack` 的 LIFO 语义

`apps/web/src/composables/useKeydownStack.ts` 维护**全进程唯一**的 handler 栈：

```ts
/** LIFO stack: only the topmost registered handler receives keydown. */
const stack: KeydownHandler[] = []
```

- `pushKeydown(handler)` 入栈并返回一个「出栈函数」；栈从空变为非空时才 `window.addEventListener('keydown', …)`，
  栈空后立即 `removeEventListener`（不会常驻监听）。
- 派发时**只调用栈顶**：`const top = stack[stack.length - 1]; top?.(e)`。
- `useKeydownStack(handler, active?)` 是 Vue 封装：传 `active`（`Ref` / `WatchSource`）时按它 watch
  开关自动入栈 / 出栈，并在 `onBeforeUnmount` 收口；不传则 `onMounted` 入栈。

**为什么用 LIFO**：多个模态可以叠放（确认框压在设置弹窗上、预览浮层压在列表上）。
若所有 handler 都监听 `window`，一次 `Escape` 会被每个打开的对话框各处理一遍——层层关闭或误触发破坏性操作。
栈顶独占派发保证「一次按键只影响最上面那个模态」。这条语义由
`useKeydownStack.test.ts` 的 `dispatch calls only the topmost handler` 与
`real window keydown dispatches to the topmost handler` 断言。

> 上游一致性提醒：`ConfirmDialog.vue` 的注释记录了历史写法——
> 它曾额外用 `&& isTopKeydown(onKey)` 自保，后判定为**恒真守卫（不可达分支）**而删除。
> 该注释也是「为什么不需要在 handler 内再判一次栈顶」的现场说明。

使用方共 4 个：`ModalDialog`、`ConfirmDialog`、`PromptDialog`、`PreviewOverlay`。

### 2.2 模态下的焦点处理

`ModalDialog.vue` 是最完整的一处，做四件事：

1. **记住并恢复焦点**：打开时把 `document.activeElement` 存进 `previousFocus`，
   关闭时若该元素**仍在文档里**（`document.contains`）则 `focus()` 回去；
   已从文档移除则不做恢复（`ModalDialog.test.ts` 有对应用例，防「恢复崩溃」）。
2. **焦点移入对话框**：打开后 `await nextTick()`，聚焦第一个可聚焦元素
   （选择器 `a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])`，
   并用 `offsetParent !== null` 过滤不可见元素；因为关闭按钮恒存在，所以取 `els[0]` 是安全的）。
3. **焦点陷阱**：`Tab` 在最后一个元素上回卷到第一个、`Shift+Tab` 在第一个元素上回卷到最后一个，
   中间位置不干预（不 `preventDefault`）。用例：`Tab focus trap cycles first/last and ignores mid elements`、
   `Tab trap with only the close button wraps to itself`。
4. **锁背景滚动**：打开时把 `document.body.style.overflow` 置 `hidden`，关闭时还原原值。

`Escape` 由键栈处理并 `preventDefault` 后 `emit('close')`（用例 `Escape emits close with preventDefault`）。

`ConfirmDialog` / `PromptDialog` / `PreviewOverlay` **不是**通过 `ModalDialog` 实现的
（各自有 `.modal-card` / `.pv-card` 模板），它们只做**初始聚焦**：
`ConfirmDialog` 打开后 `nextTick` 聚焦确认按钮，`PromptDialog` 打开后 `nextTick` 聚焦输入框。
也就是说：**焦点陷阱只有 `ModalDialog` 这一份实现**，其余模态依赖「栈顶独占 + 初始聚焦」。

### 2.3 组件内的键盘操作

| 位置 | 键 | 行为 |
|---|---|---|
| `ObjectContextMenu.vue` | 打开时自动聚焦首个按钮；`ArrowDown` / `ArrowUp` / `Home` / `End` | 菜单项之间循环 / 跳到首尾（源码注释：「无障碍键盘操作」） |
| `ObjectList.vue` 网格视图 | `Enter` / `Space` | 双击语义 / 单击语义（`role="button"` + `tabindex="0"`） |
| `ObjectList.vue` 列表视图 | `Enter` / `Space` | 行打开 / 选中；表头 `Enter` / `Space` 触发排序（配合 `aria-sort`） |
| `UploadPanel.vue` 拖放区 | `Enter` / `Space` | 触发隐藏的 `<input type="file">`（拖放区本身是可聚焦的 `role="button"`） |
| `ObjectToolbar.vue` 路径输入 | `Enter` 提交 / `Esc` 取消编辑 | — |
| `CreateBucketDialog` / `DestDialog` / `PromptDialog` 的输入框 | `Enter` | 提交当前表单 |

**文件管理器式全局快捷键**（`composables/useObjectBrowser.ts` 的 `onGlobalKey`，
挂在 `window` 上）——三条守卫保证不误触：

1. **面板激活守卫**：`if (!panelActive.value) return`。面板用 KeepAlive 缓存，切走后监听仍存活，
   否则会用旧选中集触发删除 / 预览 / 全选（源码注释的原始缺陷描述）。
2. **输入控件守卫**：焦点在 `INPUT` / `TEXTAREA` / `SELECT` / `BUTTON` 或 `contentEditable` 元素上时直接返回。
3. **上下文守卫**：未选账号 / 未选桶时返回。

键位：`Enter` = 预览（未知类型转下载）、`F2` = 重命名、`Delete` / `Backspace` = 删除选中、
`Ctrl/Cmd+A` = 全选 / 取消全选。首次进入会出现**可关闭的快捷键提示条**
（`localStorage['s3c.hintsHidden']` 记忆，存储不可用时按「未隐藏」处理）。

### 2.4 焦点可见性

`apps/web/src/styles.css` 有全局 `:focus-visible` 规则，覆盖
`button` / `input` / `select` / `textarea` / `a` / `[tabindex]`，样式为
`outline: 2px solid var(--primary); outline-offset: 2px;`（注释标题「===== 无障碍焦点 =====」）。

> **`textarea` 曾缺席该列表（2026-09-29 修复，KNOWN_ISSUES #67③）**：表单基础规则写了
> `input, select, textarea { … outline: none; }`，把默认轮廓关掉了，而 `:focus-visible` 列表里
> 没有 `textarea`——于是全仓 3 个 `<textarea>`（都在 `BucketPolicyVisualEditor.vue`，
> 均不带 `tabindex`，因此也不落进 `[tabindex]:focus-visible`）聚焦时只有 `textarea:focus` 的
> 边框变色 + 外发光，**拿不到**这条统一的 2px 高对比度轮廓。
> 现已纳入列表，并由 [`apps/web/src/a11y_gate.test.ts`](../apps/web/src/a11y_gate.test.ts)
> 断言选择器列表必须含 `textarea:focus-visible`（回退即红灯）。

---

## 3. 主题与视觉

`apps/web/src/theme.ts`：

- **支持三种选择**：`auto`（跟随系统）/ `light` / `dark`（`export type Theme = 'auto' | 'light' | 'dark'`）。
- **默认 `auto`**：`readTheme()` 读不到合法值时返回 `'auto'`；`resolvedTheme()` 在 `auto` 下按
  `window.matchMedia('(prefers-color-scheme: dark)')` 取 `dark` / `light`。
- **持久化**：`localStorage['s3c.theme']`；存储不可用（隐私模式 / 配额异常）时降级而不是抛错——
  模块**在初始化阶段**就会调用 `applyTheme()`，抛出即整站启动失败（源码注释指向 review §F9④）。
- **跟随系统是实时的**：模块顶层注册 `mq.addEventListener('change', …)`，系统主题变化时自增
  `systemThemeTick`（供 UI 响应式刷新）并在 `auto` 模式下立即 `applyTheme('auto')`。
- **落地方式**：写到 `<html data-theme="light|dark">`，由 `styles.css` 的 `[data-theme=…]` 选择器接收。
- **切换入口**：顶栏主题按钮调 `cycleTheme()`，顺序 `auto → light → dark → auto`，
  按钮文案与 `aria-label` 是「当前主题（循环提示）」。

> 主题只改变配色变量，**不改变布局与语义**；`<html lang>` 的初始值由 `index.html` 静态给出
> （`lang="zh-CN"`），运行时由 `i18n/index.ts` 的 `applyDocumentLang()` 跟随界面语言同步
> （2026-09-29 修复 #67①，见 §4 第 6 条）。

---

## 4. 已知限制（**明确没做**的事）

> 以下每一条都在仓库里核实过「确实不存在」（含测试、CI 配置、依赖与脚本），不是推测。
> 按影响从大到小排列。
>
> **编号固定不重排**：已闭环的条目用 ~~删除线~~ + ✅ 标注并保留原位（便于按编号回溯），
> 闭环证据见 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) §四 与本文件 §5.4。

1. **没有做过正式的 WCAG 审计**：仓库内无任何审计报告、无合规声明、无 VPAT / ACR 类文件。
   本文件（以及任何其它文档）都**不能**被引用为「符合 WCAG」的证据。
2. **没有引入任何自动化可访问性检测工具**：全仓（排除 `node_modules` 与 CI 构建缓存）搜不到
   `axe-core` / `pa11y` / `lighthouse` / `eslint-plugin-jsx-a11y` 之类的依赖、配置或 CI job。
   复现：`grep -rniE "wcag|axe-core|pa11y|lighthouse" --include=*.md --include=*.ts --include=*.vue --include=*.json --include=*.yml --include=*.go .`
   （排除 `node_modules`、`.gitlab-ci-local/`、`coverage/`）→ 无命中。
3. **没有做屏幕阅读器实测**：无 NVDA / JAWS / VoiceOver / TalkBack 的实测记录或截图证据。
4. **没有对比度自动检测，也没有对比度专项核查记录**：颜色全部走 CSS 变量（`--text` / `--muted` / `--primary` …），
   但没有脚本或测试校验前景/背景对比度，`styles.css` 里也没有相关注释或断言。
5. ~~**没有处理 `prefers-reduced-motion`**~~ ✅ **已于 2026-09-29 修复（KNOWN_ISSUES #67②）**：
   此前全仓搜不到该媒体查询，`styles.css` 的 `rise` / `toast-in` / `skel` 关键帧与
   `ModalDialog` 的 `modal-fade`、`ObjectContextMenu` 的 `pop-in` 过渡在「减少动效」偏好下
   **照常播放**。现已在 `styles.css` **末尾**加 `@media (prefers-reduced-motion: reduce)`，
   对 `*` / `*::before` / `*::after` 关闭 `animation` 与 `transition`（`!important` 是必要的：
   动效分散在组件级选择器与内联过渡上，逐条提优先级既易漏又难维护），并由
   [`apps/web/src/a11y_gate.test.ts`](../apps/web/src/a11y_gate.test.ts) 断言该媒体查询存在
   且确实关闭两者（回退即红灯）。
6. ~~**`<html lang>` 不随界面语言更新**~~ ✅ **已于 2026-09-29 修复（KNOWN_ISSUES #67①）**：
   `index.html` 仍硬编码 `lang="zh-CN"`，但那是**渲染前的初始值**（与默认语言一致）；
   `i18n/index.ts` 新增 `applyDocumentLang()`——模块初始化时同步一次，`setLocale()` 时再同步，
   故切到英文后文档语言标记即为 `en-US`（屏幕阅读器发音选择、浏览器翻译提示都随之正确）。
   由 `i18n/index.test.ts` 的 2 条用例钉住（`setLocale` 与 `cycleLocale` 两条路径）。
7. **焦点陷阱只覆盖 `ModalDialog`**：`ConfirmDialog` / `PromptDialog` / `PreviewOverlay` 没有
   `Tab` 循环，只靠「键栈栈顶独占 + 初始聚焦」。键盘用户在这些模态里 `Tab` 可以走出对话框。
   → 改进项。
8. **`aria-live` 覆盖很窄**：只有 2 处（`Toasts`、`BatchMetadataDialog` 状态区），
   且**都没有测试断言**；异步操作完成（如列表刷新、桶设置保存）多数不经过 live region。
9. **表格语义未完整声明**：数据表用原生 `<table>`，但没有 `caption`，也没有为「可选中的行」
   声明 `aria-selected` / `aria-multiselectable`；网格视图的单元格是 `role="button"`，
   其名称依赖单元格内文本节点，未加 `aria-label`。
10. **表单标签依赖 `aria-label` 而非可见 `<label>`**：多数输入框用 `aria-label` 提供名称
    （可访问名称没问题），但**没有可见标签**；只有少数场景用了 `sr-only` 的 `<label>`
    （`BatchMetadataDialog` 的标签键/值、`BucketPolicyVisualEditor` 的 JSON 文本域）。
11. **没有 RTL / 从右到左布局支持**，也没有多语言之外的区域格式（日期 / 数字）本地化测试。
12. ~~**`textarea` 不在统一焦点样式内**~~ ✅ **已于 2026-09-29 修复（KNOWN_ISSUES #67③）**：
    全局 `:focus-visible` 规则的选择器列表已纳入 `textarea`（详见 §2.4），
    由 [`apps/web/src/a11y_gate.test.ts`](../apps/web/src/a11y_gate.test.ts) 断言防回退。

---

## 5. 如何验证

### 5.1 手动核对清单（改动 UI 后逐项过一遍）

1. **只按键盘走完全流程**：`Tab` 能到达所有可操作元素（按钮、输入、下拉、行、右键菜单的 ⋯ 按钮），
   焦点框可见；没有键盘陷阱（除模态内应保留的陷阱）。
2. **模态内**：打开后焦点落在对话框内；`Tab` / `Shift+Tab` 在 `ModalDialog` 里循环不逃逸；
   `Escape` 关闭；关闭后焦点回到打开它的按钮（先在浏览器 DevTools 里 `document.activeElement` 确认）。
3. **叠放模态**：同时打开两层（如设置弹窗 + 确认框），按一次 `Escape` **只**关掉最上面那层。
4. **右键菜单**：用键盘打开后，`ArrowUp` / `ArrowDown` / `Home` / `End` 在菜单项之间移动。
5. **排序表头**：`Tab` 到表头，`Enter` / `Space` 触发排序，同时 `aria-sort` 在
   `none` / `ascending` / `descending` 之间变化（`ObjectList.vue` 的 `ariaSort()`；
   非当前排序列恒为 `none`）。
6. **加载态**：列表加载 / 迁移列举中，容器带 `aria-busy="true"`；两处进度条 `aria-valuenow` 随进度变化。
7. **文案切换联动**：`aria-label` 与可见文案来自同一批 i18n key，切语言后**同步**变（不应出现「按钮是英文、
   朗读名称还是中文」）。核对方式：切到英文后检查 `aria-label` 内容的语言。
8. **主题**：切到 `auto`，在操作系统里切换深浅色，界面应即时跟随；刷新后保持所选主题。
9. **动效偏好（当前为已知缺口）**：在系统里开启「减少动态效果」，观察过渡 / 动画——**现状是仍会播放**，
   这一条用于确认缺口仍在（修好后应作为回归项保留）。

### 5.2 仓库里已有的自动化手段（部分覆盖，非审计）

| 手段 | 覆盖什么 | 命令 |
|---|---|---|
| `components/ModalDialog.test.ts`（14 例） | 焦点移入 / 恢复、`document.contains` 保护、`Escape`、`Tab` 陷阱（首尾回卷、中间不干预）、仅关闭按钮时自回卷、footer 插槽与 `aria-modal` 属性 | `cd apps/web && pnpm test src/components/ModalDialog.test.ts` |
| `composables/useKeydownStack.test.ts`（9 例） | 键栈 LIFO 语义：`dispatch` 只调栈顶、真实 `window` 事件也只到栈顶、重复 `pop` 是 no-op、`active` 开关的入栈/出栈与重复激活守卫 | `pnpm test src/composables/useKeydownStack.test.ts` |
| `components/ObjectList.test.ts` | 排序表头的 `aria-sort` 取值随排序变化（`none` / `ascending` / `descending`）与键盘触发排序 | `pnpm test src/components/ObjectList.test.ts` |
| Playwright E2E（`apps/web/e2e/*.spec.ts`） | 用例大量使用 `getByRole('button' \| 'dialog' \| 'alertdialog' \| 'row', { name })` 定位元素——**这等于顺带验证了这些角色与可访问名称确实存在**，但它不是可访问性审计（不检查对比度、不检查朗读顺序、不跑 a11y 规则集） | `pnpm e2e`（或 `make e2e-real` 走真实后端） |

> 以上命令均为 `apps/web/package.json` 的既有脚本（`test` = `vitest run`、`build` = `vue-tsc --noEmit && vite build`、
> `e2e` = `playwright test`）；行尾的路径参数是 vitest 的文件过滤，不是自定义脚本。

**没有任何自动化手段覆盖**：对比度、屏幕阅读器播报、`aria-live` 播报时机、非 `ModalDialog`
模态的焦点陷阱。（`prefers-reduced-motion` 与 `<html lang>` 自 2026-09-29 起**已各有回归门禁**：
`src/a11y_gate.test.ts` 与 `src/i18n/index.test.ts`——但它们是**源码形态 / 行为**断言，
不是 a11y 规则集扫描，仍不构成审计。）

### 5.3 建议的（尚未加入的）机械化检查

以下都**还没做**，仅作为后续选项列出，不要误读为「已具备」：

- 引入 `vitest-axe` / `axe-core` 对组件测试挂载后的 DOM 做规则集扫描（能与现有 happy-dom 测试并列跑）；
- 在 Playwright E2E 里跑 `@axe-core/playwright`，覆盖真实浏览器下的对比度与结构规则；
- ~~增加一条针对 `prefers-reduced-motion` 的样式断言~~ ✅ **已于 2026-09-29 完成**：
  即 `src/a11y_gate.test.ts`（同时钉住 `textarea:focus-visible`）。

### 5.4 改进项（与本文件 §4 一一对应）

| 优先级 | 改进项 |
|---|---|
| ~~高~~ | ~~处理 `prefers-reduced-motion`~~ ✅ 2026-09-29 完成（#67②） |
| ~~高~~ | ~~让 `<html lang>` 跟随界面语言~~ ✅ 2026-09-29 完成（#67①） |
| 中 | 把焦点陷阱从 `ModalDialog` 提取为可复用的组合式函数，覆盖其余三个模态 |
| 中 | 为 `aria-live` 区域补测试断言；把「操作成功 / 失败」统一纳入 live region |
| 中 | 引入自动化 a11y 扫描（§5.3 前两条） |
| 低 | 补对比度核查记录；为数据表补 `caption` / 选中态语义；为主要表单控件补可见 `<label>`；~~把 `textarea` 纳入 `:focus-visible` 规则~~ ✅ 2026-09-29 完成（#67③） |
| 低 | 组织一次正式审计（含辅助技术实测）——在此之前，所有文档都不得声称任何 WCAG 合规等级 |
