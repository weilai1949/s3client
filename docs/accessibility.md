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
- **统计时点：2026-10-10**（数字随功能演进会变，回填时按下列命令重跑并更新本行日期）。
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
| `aria-hidden` | 44 | 纯装饰元素：emoji、内联 SVG 图标、虚拟滚动的上下垫片行（`v-spacer`） |
| `aria-label` | 30 | 无可见文字元素的名称：图标按钮（`✕` / `⋯` / `↻`）、语言与主题按钮、`<nav>` 与设置页 tab 行、批量改元数据三个控件（**2026-09-30 起与可见 `<label>` 同键**，见 §1.6）、列表与网格的行选择框、加载态容器（配合 `aria-busy`）、计划任务面板（`SchedulesSection`）的图标按钮 |
| `aria-expanded` | 3 | 「⋯」更多操作触发器的展开态（`ObjectList.vue` 两处模板 + 1 处注释文字；配合 `ctxEntryKey` 反映当前打开的右键菜单条目） |
| `aria-haspopup` | 2 | 同一「⋯」触发器声明弹出菜单语义（`aria-haspopup="menu"`） |
| `aria-selected` | 4 | **2026-09-30 起（#17③）**：四张有行选中的表，其数据行随选中集合变化（**刻意不用 `aria-multiselectable`**——axe 的 `aria-allowed-attr` 判定它在原生 `<table>` 上非法，见 §1.6） |
| `aria-modal` | 4 | 四个模态容器，值均为 `"true"` |
| `aria-labelledby` | 4 | **2026-09-30 起（#17③）**：`BucketTags` / `LifecycleDialog` 表格单元格里的输入框——一个 `<th>` 无法 `for` 到 N 行输入，故指向**可见列头**（见 §1.6） |
| `aria-sort` | 3 | 列表视图三个可排序表头（名称 / 大小 / 时间） |
| `aria-busy` | 3 | 列举中 / 加载中的容器（`ObjectList`、`MigratePanel`、`LifecycleDialog`） |
| `aria-valuenow` / `aria-valuemin` / `aria-valuemax` | 2 / 2 / 2 | 两处 `role="progressbar"`（`UploadPanel` 总进度、`MigratePanel` 任务进度） |
| `aria-live` | 2 | 见 §1.4 |
| `aria-current` | 1 | 左侧主导航当前页（`aria-current="page"`，`App.vue`） |

### 1.3 `role` 分布

| role | 次数 | 位置 |
|---|---:|---|
| `alert` | 19 | **2026-09-30 起（#17②）统一口径**：面板 / 对话框的错误横幅与校验失败（源码门禁钉住，见 §1.4）。逐文件：`ObjectDetailDialog`（4：校验和 / 保留 / 法定保留 / 保护状态的 `.badge` 错误）、`SchedulesSection`（3）、`StorageReportPanel`（2）、`AccountsPanel`（2）、`ObjectsPanel` / `BucketsPanel` / `CompareDialog` / `MigratePanel` / `RecycleBinPanel` / `ServerPanel` / `PromptDialog` / `BucketObjectLock`（各 1） |
| `menuitem` | 17 | `ObjectContextMenu.vue`（右键菜单的每个按钮） |
| `status` | 3 | `ObjectList.vue`（网格截断提示）、`UploadQueue.vue`、`Toasts.vue`（单条 toast） |
| `dialog` | 3 | `ModalDialog.vue`、`PromptDialog.vue`、`PreviewOverlay.vue` |
| `columnheader` | 3 | `ObjectList.vue` 三个可排序表头（配 `tabindex="0"` 与 `aria-sort`） |
| `progressbar` | 2 | `UploadPanel.vue`、`MigratePanel.vue` |
| `button` | 2 | `UploadPanel.vue` 拖放区、`ObjectList.vue` 网格单元格（自定义可点区域） |
| `menu` | 1 | `ObjectContextMenu.vue` 容器（`tabindex="-1"`） |
| `alertdialog` | 1 | `ConfirmDialog.vue`（破坏性操作确认） |

说明：**数据表没有显式 ARIA grid 角色**——它用的是原生 `<table>` / `<thead>` / `<th>` / `<tr>`，
语义由浏览器给出；`role="columnheader"` 只冗余地写在**那三个可排序表头**上（与 `tabindex="0"`、
`aria-sort` 配套，让「可排序」这件事对辅助技术可见），其余单元格没有任何 ARIA role。
复选框类控件用原生 `<input type="checkbox">` + `aria-label`，未自造 switch 角色。

### 1.4 `aria-live` 区域

**2026-09-30 起（ROADMAP §三 #17②）**：「操作成功 / 失败统一播报」已收口——
**成功**统一走 `toast()` → `Toasts` 容器；**失败**若是面板内联横幅则自带 `role="alert"`，
并由 [`apps/web/src/a11y_gate.test.ts`](../apps/web/src/a11y_gate.test.ts) 的源码门禁钉住
「每个 `msg err` / `modal-err` 开标签必须声明 `role="alert"`」。

带 `aria-live` 属性的区域仍是 **2 处**（都是 `aria-live="polite"`，没有 `assertive`）：

| 位置 | 播报内容 | 测试断言 |
|---|---|---|
| `Toasts.vue` | 全局 toast 容器（每条 toast 另带 `role="status"`）——**操作成功 / 失败的统一播报通道** | `Toasts.test.ts`「操作成功 / 失败统一经容器的 aria-live 播报」 |
| `BatchMetadataDialog.vue` | 批量改元数据的执行状态 / 结果区（`v-if="running \|\| result"`） | 同名组件测试「执行状态经 aria-live 播报区呈现（running 与 done 都在区内）」 |

**内联失败横幅**（不经 toast 的那部分）用 `role="alert"`（隐式 assertive）而非 `aria-live` 属性：
`ObjectsPanel` / `AccountsPanel` / `BucketsPanel` / `CompareDialog` / `MigratePanel` /
`RecycleBinPanel` / `ServerPanel` / `SchedulesSection`（3 处）/ `StorageReportPanel`（2 处）/
`BucketObjectLock` 的 `class="msg err"`，`PromptDialog` 的 `class="modal-err"`（校验失败），
以及 `ObjectDetailDialog` 的 4 个 `.badge` 错误（校验和 / 保留 / 法定保留 / 保护状态）。
前一类（`msg err` / `modal-err`）都是 `v-if` 条件渲染——`role="alert"` 只在元素**插入 DOM 时**播报一次，
不会因为文案更新而反复打断。行为断言见 `PromptDialog.test.ts`
「校验失败文案带 role="alert"」；**漏加 / 回退由 `a11y_gate` 源码门禁红灯点名**（已做变异验证：
去掉 `ServerPanel` 的 `role` → 门禁列出该标签 → 还原绿灯）。

`ObjectsPanel.vue` 的错误横幅原本就是 `role="alert"`，本批是把它变成**全仓统一口径**而非孤例。

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

### 1.6 表格可访问名称与表单可见标签（2026-09-30 起，ROADMAP §三 #17③）

**表格 `caption`**：`src/components` 下 **19 张 `<table>`（分布在 17 个组件文件）全部带 `<caption class="sr-only">`**
（统计时点 2026-10-10；`StorageReportPanel` 一个文件 3 张表。`.sr-only` 为 `src/styles.css` 新增的全局工具类——裁剪到 1px 但**不** `display:none`，
否则会一并从无障碍树里消失）。用 sr-only 而非可见 caption：这些表上方都已有可见标题 / 面板标题，
再来一行可见 caption 只会视觉重复；WCAG 要的是「表有可访问名称」，不要求它可见。
由 [`apps/web/src/a11y_gate.test.ts`](../apps/web/src/a11y_gate.test.ts) 的
「每张数据表都有 `<caption class="sr-only">`」源码门禁钉住（`<caption>` 必须紧跟开标签，
否则读取后续 120 字符匹配不到即红灯）。

**行选中语义**：四张有行选中的表，其数据行声明 `:aria-selected`（随选中集合变化）：

| 组件 | 选择控件 | 依据 |
|---|---|---|
| `ObjectList.vue` | 复选框 | 列表视图多选 / Shift 连选 |
| `MigratePanel.vue` | 复选框 | 迁移对象多选 |
| `AccountsPanel.vue` | 单选按钮组（`name="acc"`） | 当前账号 |
| `ServerPanel.vue` | 单选按钮组（`name="server"`） | 当前生效服务器 |

> **为什么不写 `aria-multiselectable`**：初版把它写在了 `<table>` 上，**组件级 axe 扫描
> （§5.2）当场报 `aria-allowed-attr`（critical）**——该属性只属于 `grid` / `listbox` /
> `select` 等选择型角色，原生 `table` 不支持。为一条属性把整张表升级成 `role="grid"`
> （连带 `gridcell` 与方向键焦点管理）不划算，故**移除该属性**、只保留合法的 `aria-selected`；
> 「多选还是单选」由复选框 vs 单选按钮本身表达。这正是组件级扫描存在的价值：它把
> 「看起来对、实则非法」的 ARIA 当场点名。

行为断言见 `ObjectList.test.ts`「行 aria-selected 随选中集合变化」；名单与属性存在性由
a11y_gate 源码门禁钉住（含**自检**：全仓声明 `:aria-selected` 的组件**恰好**是这 4 张，
防扫描面塌缩）。

**表单可见标签**：`src` 下 **94 个可见表单控件**（`<input>` / `<select>` / `<textarea>` 共 96 个，
其中 2 个为 `display:none` 的 `<input type="file">`，见下）**全部**有可关联的标签来源，三种形态按场景取用——

1. **包裹 `<label class="field">`**（既有主流形态，约 40 处）；
2. **行内可见 `<label for>`**（紧凑行编辑器）：`TagsDialog` / `HeadersDialog` 的键值行、
   `ObjectsPanel` / `BucketsPanel` / `RecycleBinPanel` 紧邻下拉的**可见徽标**改成
   `<label for>`（文案不变、零视觉改动）、`ObjectToolbar` 的路径编辑框与过滤框、
   `BatchMetadataDialog` 的 ACL / 标签模式 / 存储类型三个控件（原硬编码英文 `aria-label`
   改为与可见 `<label>` **同一条 i18n 键**，顺带修掉「切换英文才对、中文界面却朗读英文」）；
3. **`aria-labelledby` 指向可见列头**（表格单元格）：`BucketTags` / `LifecycleDialog` 的输入框
   ——一个 `<th>` 无法 `for` 到 N 行输入，改指向**可见**列头，文本仍是可见的、且程序可关联，
   又不必在每行重复一遍列头文字。

仅剩 2 个 `display:none` 的 `<input type="file">` 无标签：它们不可见也不可聚焦，
由已带标签的拖放区 / 按钮触发，属正常形态。断言见 `TagsDialog.test.ts`
「每行键 / 值输入都有可见 `<label for>`」与 `BatchMetadataDialog.test.ts`
「三个控件都被可见 `<label>` 包裹」。

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

**2026-09-30 起（ROADMAP §三 #17①）**：「记住 / 恢复焦点 + 初始移入 + `Tab` 首尾回卷」已抽成
[`apps/web/src/composables/useFocusTrap.ts`](../apps/web/src/composables/useFocusTrap.ts)，**四个模态共用**
（`ModalDialog` / `ConfirmDialog` / `PromptDialog` / `PreviewOverlay`），不再是「只有 `ModalDialog` 有陷阱」。

1. **记住并恢复焦点**：打开时把 `document.activeElement` 存进 `previousFocus`，
   关闭时若该元素**仍在文档里**（`document.contains`）则 `focus()` 回去；已从文档移除则不做恢复
   （`ModalDialog.test.ts` 有对应用例，防「恢复崩溃」）。
2. **焦点移入对话框**：打开后 `await nextTick()`，聚焦第一个可聚焦元素
   （选择器 `a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])`，
   并用 `offsetParent !== null` 过滤不可见元素）。四个模态的首个可聚焦元素恰好都是「主操作」：
   `ModalDialog` 的 ✕、`ConfirmDialog` 的确认按钮、`PromptDialog` 的输入框、`PreviewOverlay` 的下载按钮。
3. **焦点陷阱**：`Tab` 在最后一个元素上回卷到第一个、`Shift+Tab` 在第一个元素上回卷到最后一个，
   中间位置不干预（不 `preventDefault`）。用例：`Tab focus trap cycles first/last and ignores mid elements`、
   `Tab trap with only the close button wraps to itself`，以及另外三个模态各自的同名用例。
4. **`Tab` 回卷由各模态的 keydown 栈 handler 调用 `trapTab(e)`**——组合式**不**自行监听 window，
   否则会破坏「一次按键只影响最上层模态」的 LIFO 语义（见 §2.1）。
5. **锁背景滚动**仍是 `ModalDialog` 自己的事（只它一个锁），打开置 `hidden`、关闭还原原值。

> **为什么初始聚焦要 `await nextTick()` 而不能用 `flush: 'post'`**：`v-model` 是**运行时指令**，
> 它的 `mounted` 钩子同样排在 post 队列里且晚于本 watcher——先跑会拿到尚未写入 DOM 的输入值，
> `PromptDialog` 的全选会落空。代价是「打开的同一 tick 内关闭 / 卸载」会让挂起的续体看到已被置空的
> 模板 ref，故续体开头有一道空容器守卫（由 `ConfirmDialog.test.ts` 的
> 「打开后同 tick 内卸载」用例覆盖）。

`Escape` 由键栈处理并 `preventDefault` 后 `emit('close')`（用例 `Escape emits close with preventDefault`）。

`ConfirmDialog` / `PromptDialog` / `PreviewOverlay` **不是**通过 `ModalDialog` 实现的
（各自有 `.modal-card` / `.pv-card` 模板），但它们与 `ModalDialog` 一样经 `useFocusTrap` 获得
**初始聚焦 + 焦点陷阱**；`ConfirmDialog` / `PromptDialog` 的 `Enter` 确认语义仍由各自的键栈 handler 保留。

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
2. ~~**没有引入任何自动化可访问性检测工具**~~ ✅ **已于 2026-09-30 部分闭环**：
   现引入 `@axe-core/playwright`（**devDependency**，只进 E2E，不动 `dependencies`——不违反
   前端「运行时仅 `vue`」的 [ADR-004](decisions/0004-minimal-frontend-deps.md)）并新增
   [`apps/web/e2e/a11y.spec.ts`](../apps/web/e2e/a11y.spec.ts)：在**真实 Chromium + 真实构建产物**
   下对 5 个界面状态（浅色初始态 / 新增登录对话框 / 服务器设置面板 / 对象网格视图 / 深色主题初始态）跑
   WCAG 2.0 + 2.1 的 A / AA 规则集，**serious / critical 违规即红灯**；另有一条
   「注入已知违规必须被报出」的自检用例，防止扫描器失效后静默全绿。CI 由既有
   [`.github/workflows/e2e-playwright.yml`](../.github/workflows/e2e-playwright.yml) 的
   `pnpm e2e:ci` 覆盖（`testDir: ./e2e` 自动纳入），**无需新增 workflow**。
   **2026-09-30 再补组件级一侧（ROADMAP §三 #17④）**：新增 devDependency `vitest-axe`
   （同样只进 devDependencies，`dependencies` 仍只有 `vue`，**不违反 ADR-004**）与
   [`apps/web/src/a11y_axe.test.ts`](../apps/web/src/a11y_axe.test.ts)——对**挂载后的组件 DOM**
   跑同一套 WCAG A / AA 规则集，覆盖 E2E 到不了的边界态（对话框打开、toast 堆叠、列表带选中行），
   同样只阻塞 serious / critical，并带「注入 `image-alt` 必须被报出」的空跑自检。
   它在落地当天就抓到一条真问题：`aria-multiselectable` 在原生 `<table>` 上**非法**
   （`aria-allowed-attr` critical，见 §1.6）。happy-dom 不做 CSS 级联 / 布局，故该文件
   **显式关掉 `color-contrast` 规则**——对比度仍由 E2E 侧 axe 与 §5.5 静态 token 表负责。
   **仍未覆盖**：屏幕阅读器实测，以及 axe 规则集本身不判定的项——故这**不是审计**，
   也不构成任何合规声明（见 §4 第 1 条）。
3. **没有做屏幕阅读器实测**：无 NVDA / JAWS / VoiceOver / TalkBack 的实测记录或截图证据。
4. ~~**没有对比度自动检测**~~ ✅ **已于 2026-09-30 部分闭环**：渲染态的对比度现由
   `e2e/a11y.spec.ts` 的 axe `color-contrast` 规则自动检测（浅色 + 深色初始态等 5 个状态），
   并已据此修掉 2 组实测不达标的 token（`--ok` 2.55:1 → 4.95:1、`--danger` 4.41:1 → 5.91:1，
   token 公式值，见 §5.5）。**静态 token 表（§5.5）仍保留且仍须维护**：axe 不评估它无法解析的背景
   （`--brand` 渐变上的白字）与**未出现在被扫描状态里**的配对，两份互补而非替代；
   表内数值与计数由 [`apps/server/contrast_gate_test.go`](../apps/server/contrast_gate_test.go)
   从 `styles.css` 机械重算比对，改色不重算即红灯。
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
7. ~~**焦点陷阱只覆盖 `ModalDialog`**~~ ✅ **已于 2026-09-30 修复（ROADMAP §三 #17①）**：
   焦点陷阱抽成 [`useFocusTrap.ts`](../apps/web/src/composables/useFocusTrap.ts)，`ConfirmDialog` /
   `PromptDialog` / `PreviewOverlay` 同样具备 `Tab` 回卷 + 焦点恢复（详见 §2.2），
   由三个组件各自的「Tab 焦点陷阱」用例钉住。
8. ~~**`aria-live` 覆盖很窄**~~ ✅ **已于 2026-09-30 收口（ROADMAP §三 #17②）**：
   两处 `aria-live` 区域**都有了测试断言**；「操作成功 / 失败统一播报」= 成功统一走 `toast()` →
   `Toasts` 的 `aria-live` 容器，失败的内联横幅统一 `role="alert"`（源码门禁钉住，见 §1.4）。
   **仍不覆盖**：屏幕阅读器实测播报时机、`aria-live` 的实际朗读效果（需辅助技术人工验证）。
9. **表格语义**：~~数据表没有 `caption`，也没有为「可选中的行」声明 `aria-selected` /
   `aria-multiselectable`~~ ✅ **已于 2026-09-30 修复（ROADMAP §三 #17③）**——19 张表全部带
   sr-only `<caption>`，四张有行选中的表声明行 `aria-selected`（见 §1.6，源码门禁 + 行为断言 +
   组件级 axe 三重钉；`aria-multiselectable` 经 axe 判定在原生 `<table>` 上**非法**，刻意不用）。
   ~~**仍未做**：网格视图的单元格是 `role="button"`，其名称依赖单元格内文本节点，未加 `aria-label`~~
   ✅ **2026-10-10 经 axe 判定为非缺陷，且该态此后受 CI 约束**：`e2e/a11y.spec.ts` 新增第 5 个扫描态
   「对象网格视图」（`installObjectsStub` 桩出账号 + 对象 → 切网格），axe 的 `button-name` /
   `aria-allowed-attr` 等规则对单元格文本节点（文件名 + 大小）派生的可访问名**判定通过**，
   该态 **0 违规**（阻塞级与非阻塞级均无）——因此**刻意不加** `aria-label`：它会覆盖文本内容、
   丢掉大小这类上下文，并造出第二事实源。此前四个扫描态都不含网格视图，故这条「仍未做」
   一直没被机械判定过；现在判定完成，回归（如去掉 `tabindex`、改成无名称的纯图标格）即红灯点名。
10. ~~**表单标签依赖 `aria-label` 而非可见 `<label>`**~~ ✅ **已于 2026-09-30 修复
    （ROADMAP §三 #17③）**：可见控件全部有可关联标签，三种形态与剩余例外见 §1.6。
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
9. **动效偏好（回归项）**：在系统里开启「减少动态效果」，触发过渡 / 动画（如弹窗、toast、骨架屏）——
   **应立即停止播放**（#67② 已修：`styles.css` 的 `prefers-reduced-motion` 块关闭 `animation` / `transition`）；
   若仍会播放即为回退，先看 [`a11y_gate.test.ts`](../apps/web/src/a11y_gate.test.ts) 是否被绕过。

### 5.2 仓库里已有的自动化手段（部分覆盖，非审计）

| 手段 | 覆盖什么 | 命令 |
|---|---|---|
| `components/ModalDialog.test.ts`（14 例） | 焦点移入 / 恢复、`document.contains` 保护、`Escape`、`Tab` 陷阱（首尾回卷、中间不干预）、仅关闭按钮时自回卷、footer 插槽与 `aria-modal` 属性 | `cd apps/web && pnpm test src/components/ModalDialog.test.ts` |
| `components/{ConfirmDialog,PromptDialog,PreviewOverlay}.test.ts` 的焦点用例（各 2–3 例） | **2026-09-30 起（#17①）**：初始焦点落点、关闭后焦点恢复到打开前元素、`Tab` 首尾回卷 / 中间不干预；`PromptDialog` 另断言全选（`selectionStart`/`selectionEnd`），`ConfirmDialog` 另断言「打开后同 tick 内卸载不抛错」 | `pnpm test src/components/ConfirmDialog.test.ts` 等三个文件 |
| **live region 断言**：`Toasts.test.ts`（1 例）、`BatchMetadataDialog.test.ts`（1 例）、`PromptDialog.test.ts`（1 例）、`a11y_gate.test.ts`（源码门禁） | **2026-09-30 起（#17②）**：toast 容器 `aria-live="polite"` 且成功 / 失败文案都在区内、批量改元数据状态区 `aria-live` 承载 running / done、校验失败横幅 `role="alert"`、**每个 `msg err` / `modal-err` 开标签必须带 `role="alert"`**（漏加即红灯点名） | `pnpm test src/a11y_gate.test.ts` |
| **表格与可见标签断言**：`a11y_gate.test.ts`（2 条源码门禁：sr-only `<caption>` + 行 `aria-selected` 名单自检恰好 4 张）、`ObjectList.test.ts` / `TagsDialog.test.ts` / `BatchMetadataDialog.test.ts` 各 1 例 | **2026-09-30 起（#17③）**：每张 `<table>` 紧跟 sr-only `<caption>`、有行选中的表其数据行声明 `aria-selected`（**刻意不用 `aria-multiselectable`**，axe 判定其在原生 `<table>` 上非法）、行 `aria-selected` 随选中集合变化、键值行 `label[for]` 能解析到目标输入框、批量元数据三控件被可见 `<label>` 包裹 | `pnpm test src/a11y_gate.test.ts` 等 |
| `composables/useKeydownStack.test.ts`（7 例） | 键栈 LIFO 语义：`dispatch` 只调栈顶、真实 `window` 事件也只到栈顶、重复 `pop` 是 no-op、空栈分支、`active` 开关的入栈/出栈与重复激活守卫 | `pnpm test src/composables/useKeydownStack.test.ts` |
| `components/ObjectList.test.ts` | 排序表头的 `aria-sort` 取值随排序变化（`none` / `ascending` / `descending`）与键盘触发排序 | `pnpm test src/components/ObjectList.test.ts` |
| Playwright E2E（`apps/web/e2e/*.spec.ts`） | 用例大量使用 `getByRole('button' \| 'dialog' \| 'alertdialog' \| 'row', { name })` 定位元素——**这等于顺带验证了这些角色与可访问名称确实存在**，但它不是可访问性审计（不检查朗读顺序、不跑 a11y 规则集） | `pnpm e2e`（或 `make e2e-real` 走真实后端） |
| **`e2e/a11y.spec.ts`（6 例，axe-core）** | **真实 Chromium + 真实构建产物**上的 WCAG 2.0 / 2.1 A + AA 规则集扫描：5 个界面状态（浅色初始态 / 新增登录对话框 / 服务器设置面板 / **对象网格视图（2026-10-10 新增）** / 深色主题初始态）的 **serious / critical 违规必须为 0**，非阻塞级违规打印供人工判断；另 1 例「axe 有效性自检」（注入 `image-alt` 违规必须被报出，防空跑）。**覆盖对比度、ARIA 角色 / 名称、表单标签、landmark 等渲染态规则** | `cd apps/web && pnpm build && pnpm exec playwright test e2e/a11y.spec.ts` |
| **`src/a11y_axe.test.ts`（6 例，vitest-axe + axe-core）** | **2026-09-30 起（#17④）组件级**：对**挂载后的组件 DOM** 跑同一套 WCAG 2.0 / 2.1 A + AA 规则集，覆盖 E2E 到不了的边界态（`ModalDialog` 带 footer / `ConfirmDialog` 危险态 / `PromptDialog` 带校验失败 / `Toasts` 成功+失败堆叠 / `ObjectList` 带 caption 与选中行）；**serious / critical 即红灯**，非阻塞项打印供人工判断；另 1 例「注入 `image-alt` 必须被报出」空跑自检。**不判对比度**（happy-dom 无 CSS 级联，显式关掉 `color-contrast`） | `cd apps/web && pnpm test src/a11y_axe.test.ts` |

> 以上命令均为 `apps/web/package.json` 的既有脚本（`test` = `vitest run`、`build` = `vue-tsc --noEmit && vite build`、
> `e2e` = `playwright test`）；行尾的路径参数是 vitest 的文件过滤，不是自定义脚本。

**自 2026-09-30 起仍没有任何自动化手段覆盖**：屏幕阅读器**实测**播报（`aria-live` 的存在、
内容落位与 `role="alert"` 由单测 + 源码门禁钉住，但**真正读没读出来、何时读**只能靠辅助技术人工验证）、
axe 无法解析的背景（渐变）上的对比度。
**已经覆盖的**：渲染态 WCAG A/AA 规则集（含对比度）→ `e2e/a11y.spec.ts`；
**组件挂载态的 WCAG A/AA 结构规则 → 2026-09-30 起已覆盖**（`src/a11y_axe.test.ts`，除对比度）；
**非 `ModalDialog` 模态的焦点陷阱 → 2026-09-30 起已覆盖**（`useFocusTrap` + 四个组件的 `Tab` 用例）；
**`aria-live` / `role="alert"` 的存在与内容 → 2026-09-30 起已覆盖**（见上表 live region 行）；
`prefers-reduced-motion` 与 `<html lang>` → `src/a11y_gate.test.ts` 与 `src/i18n/index.test.ts`
（源码形态 / 行为断言）。即便如此，**没有一条构成审计**——审计需要人工评审 + 辅助技术实测。

### 5.3 建议的（尚未加入的）机械化检查

以下都**还没做**，仅作为后续选项列出，不要误读为「已具备」：

- ~~在 Playwright E2E 里跑 `@axe-core/playwright`，覆盖真实浏览器下的对比度与结构规则~~ ✅
  **已于 2026-09-30 完成**：即 [`apps/web/e2e/a11y.spec.ts`](../apps/web/e2e/a11y.spec.ts)
  （5 个界面状态 + 1 条有效性自检，见 §5.2；CI 由既有 `e2e-playwright.yml` 覆盖）；
- ~~引入 `vitest-axe` 对组件测试**挂载后的 DOM** 做规则集扫描（能与现有 happy-dom 测试并列跑）~~ ✅
  **已于 2026-09-30 完成**（ROADMAP §三 #17④）：即 [`apps/web/src/a11y_axe.test.ts`](../apps/web/src/a11y_axe.test.ts)
  （`vitest-axe` **仅 devDependency**，`dependencies` 仍只有 `vue`，不违 ADR-004）。与 E2E 版互补：
  组件级覆盖 E2E 到不了的边界态（对话框打开、toast 堆叠、列表带选中行）；happy-dom 不做级联 / 布局，
  故该文件**显式关掉 `color-contrast`**，只判结构规则——对比度仍由 `e2e/a11y.spec.ts` 与 §5.5 负责。
- ~~增加一条针对 `prefers-reduced-motion` 的样式断言~~ ✅ **已于 2026-09-29 完成**：
  即 `src/a11y_gate.test.ts`（同时钉住 `textarea:focus-visible`）。

> `vitest-axe` 与下节的实现类改进项曾于 **2026-09-30** 按两源分工登记 [`ROADMAP.md`](ROADMAP.md)
> §三 **#17**；**同日已全部落地并按 §六 第 1 条移出转空号**（证据 [`FEATURES.md`](FEATURES.md) **§BM**）
> ——本节只保留选项说明，不再作为待办清单。

### 5.4 改进项（与本文件 §4 一一对应）

| 优先级 | 改进项 |
|---|---|
| ~~高~~ | ~~处理 `prefers-reduced-motion`~~ ✅ 2026-09-29 完成（#67②） |
| ~~高~~ | ~~让 `<html lang>` 跟随界面语言~~ ✅ 2026-09-29 完成（#67①） |
| ~~低~~ | ~~把 `textarea` 纳入 `:focus-visible` 规则~~ ✅ 2026-09-29 完成（#67③） |
| ~~低~~ | ~~补对比度核查记录~~ ✅ 2026-09-30 完成（§5.5 静态记录，含如实标注的不达标组合） |
| ~~中~~ | ~~引入自动化 a11y 扫描~~ ✅ 2026-09-30 完成 **两侧**：Playwright + axe 整页侧（`e2e/a11y.spec.ts`）与 vitest-axe 组件侧（`src/a11y_axe.test.ts`），见 §5.2 |
| 中 | ~~焦点陷阱提取为可复用组合式函数、覆盖其余三个模态~~ ✅ 2026-09-30 完成（`useFocusTrap.ts`，见 §2.2）；~~`aria-live` 补测试断言并把「操作成功 / 失败」统一纳入 live region~~ ✅ 2026-09-30 完成（见 §1.4）；~~`vitest-axe` 组件级扫描~~ ✅ 2026-09-30 完成（`src/a11y_axe.test.ts`，见 §5.2 与 §4 第 2 条） |
| 低 | ~~对比度修色：`--ok` / `--danger`~~ ✅ 2026-09-30 完成（`--ok` 2.55→4.95、`--danger` 4.41→5.91，token 公式值；修色由 axe 扫描驱动，见 §5.5）；**剩余**浅色 `--muted` / `--primary` / `--placeholder` 与 `--brand` 渐变白字、深色 `--danger` / `--placeholder` / `--brand` → ✅ **2026-09-30 修色收尾完成**，§5.5 现为 **13 行、双主题各 0 行低于 AA**（含 `--brand` 拆出 `--brand-mark-*` 供 header logo，见 §5.5 结论）；~~数据表 `caption` / 选中态语义；主要表单控件可见 `<label>`~~ ✅ 2026-09-30 完成（见 §1.6）；RTL 支持；组织一次正式审计（含辅助技术实测）——**两项本就不在 #17 范围内**（原行文「按需另行立项」），仍列本节 §4 已知限制，需要时另行登记 |

> 在正式审计之前，所有文档都**不得**声称任何 WCAG 合规等级（见 §4 第 1 条）。

### 5.5 对比度核查记录（静态计算，2026-09-30）

> **方法**：按 WCAG 2.1 相对亮度公式（sRGB 通道线性化后 `0.2126R + 0.7152G + 0.0722B`）对
> [`styles.css`](../apps/web/src/styles.css) 的设计 token 做**前景 / 背景配对**计算；深色主题的
> `rgba()` 覆层先按 alpha 合成到 `--panel` 再算。**这是静态 token 级记录、不是审计**：不含渐变中间
> 色、图片、阴影与实机渲染，也不覆盖组件临时配色。判定按 WCAG AA **正文 4.5:1**（大字 / UI 图形 3:1）。
> **本表数值与计数由 [`apps/server/contrast_gate_test.go`](../apps/server/contrast_gate_test.go) 从
> `styles.css` 机械重算比对**（改色不重算即红灯）；渲染态另有自动检测（`e2e/a11y.spec.ts` 的 axe
> `color-contrast` 规则），但 axe 不评估渐变背景（`--brand` 上的白字）与**未出现在被扫描状态里**的
> 配对——两者互补，不能只看 axe 绿灯。数值口径统一取 **token 公式值**（渲染态读数允许 ±0.1 级差异，
> 不改判定）。

| 组合（前景 on 背景） | 浅色 | 深色 | AA 判定（4.5:1） |
|---|---|---|---|
| `--text` on `--bg` | 13.86 | 15.98 | 双主题达标 |
| `--text` on `--panel` | 14.71 | 14.62 | 双主题达标 |
| `--muted` on `--bg` | 4.57 | 7.21 | 双主题达标（浅色修色 `#64786f`→`#62766d`：4.44→4.57） |
| `--muted` on `--panel` | 4.85 | 6.60 | 双主题达标 |
| `--primary` on `--panel`（链接 / 描边按钮文字） | 4.85 | 7.80 | 双主题达标（浅色修色 `#10a37c`→`#0d8162`：3.21→4.85） |
| `--primary` on `--bg` | 4.56 | 8.52 | 双主题达标（同上：3.02→4.56） |
| `--placeholder` on `--input-bg` | 4.54 | 4.54 | 双主题达标（浅 `#8aa89a`→`#657b71`：2.58→4.54；深 `#5f7a6d`→`#6e867a`：3.81→4.54） |
| `--tag-text` on `--tag-bg` | 5.94 | 7.58 | 双主题达标 |
| `--danger` on `--danger-bg` | **5.91** ✅ | 4.54 | 双主题达标（浅色 2026-09-30 已修；深色 `#ef4444`→`#f05050`：4.25→4.54） |
| `--ok` on `--ok-bg` | **4.95** ✅ | 7.08 | 双主题达标（浅色 2026-09-30 已修） |
| `--brand-mark-from` on `--bg`（header logo 渐变文字 · 浅端） | 4.56 | 7.87 | 双主题达标（**深色底回亮值 `#2dbd98`**，见下方 token 说明） |
| `--brand-mark-to` on `--bg`（header logo 渐变文字 · 深端） | 4.52 | 4.54 | 双主题达标 |
| 白字 on `--brand` 渐变（深端 `--brand-to`） | 4.80 | 4.80 | 双主题达标；浅端 `--brand-from` 4.84（原 2.37 / 4.23，渐变两端已加深） |

> **结论（2026-09-30 修色收尾后）**：本表共 **13 行**；浅色主题 **0 行**低于 AA；
> 深色主题 **0 行**低于 AA。
> 本轮把 §5.4「剩余」清单里的 6 组**全部修到 ≥4.5**（token 公式值）：
> `--muted` 4.44→4.57、`--primary` 3.21 / 3.02→4.85 / 4.56、`--placeholder` 浅 2.58→4.54、
> `--placeholder` 深 3.81→4.54、`--danger` 深 4.25→4.54、白字 on `--brand` 2.37 / 4.23→4.84 / 4.80。
> **`--brand` 的拆分**：该渐变同时是「主按钮 / 激活 tab / 拖放区图标的**白字背景**」与
> 「header logo 的**渐变文字**」——两个方向的对比度要求相反（背景要暗、文字在深色底上要亮）。
> 故 `:root` 把 `--brand-from/to` 加深到白字达标，并新增 `--brand-mark-from/to` 专供 logo 文字：
> 浅色沿用加深值（在 `--bg` 上 4.56 / 4.52），**深色块里回亮值**（`#2dbd98` / `#138e69`，
> 在 `--bg` 上 7.87 / 4.54）——否则「为白字加深」会把深色主题 logo 反拖到 4.1:1。
> **为什么此前 axe 全绿而这些组合仍在**：`--brand` 是 `background-image` 渐变（axe 无法解析其上的白字），
> `--placeholder` 是 placeholder 文本（axe 不判该规则），`--muted` / `--primary` 在被扫描的四个状态里
> 没有以「小字正文」形态出现——**扫描通过 ≠ 全站达标**。本仓库**不声称 WCAG 合规**（§4 第 1 条）：
> 表内全绿只代表**这些 token 配对**达标，仍不含渐变中间色、图片、阴影与组件临时配色。


