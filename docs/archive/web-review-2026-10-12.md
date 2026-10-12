# apps/web 页面 UI 与交互优化快照（2026-10-12）

> **评审时点**：2026-10-12 ｜ **评审对象**：[`apps/web`](../../apps/web/)（Vue 3 单页控制台，零运行时依赖：`dependencies` 只有 `vue`，见 [ADR-004](../decisions/0004-minimal-frontend-deps.md)）
> **结论**：按**高 / 中 / 低**三档列出的优化点**全部实施**（9 批），另外在实施过程中由新增的渲染态扫描**实测抓到 1 处真实无障碍缺陷（D1）并同批修复**。门禁全绿：`pnpm test` **87 文件 1360 例** / `pnpm test:coverage` **四指标 100%** / `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` exit 0 / `pnpm build` OK / `pnpm e2e`（a11y 7 例含新移动视口态）passed。
> **性质**：时点性**子树评审快照**——命名先例 [desktop-review-2026-10-12.md](desktop-review-2026-10-12.md) / [server-review-2026-10-12.md](server-review-2026-10-12.md)（同一评审日）。条目全部闭环后按 [archive/index.md](index.md) 的「归档四步」`git mv` 进 `docs/archive/` 冻结。
> **证据口径**：本文件所有 `file:line` 均为**改动落地后**实读所得（行号对应本次改动做完的工作区，不是评审前的旧行号）；§8 的命令与输出为实跑原文，含**未跑 / 跑不动**的项，不代偿、不美化。
> **本文与其它文档的分工（本批次的一次性约定）**：按用户当轮指示，本批次的**用户可见行为说明（§6）、台账（§7）、发版记录草稿（§8）**集中写在本文件内，**未**同步写入 [`user-guide.md`](../user-guide.md) / [`FEATURES.md`](../FEATURES.md) / [`CHANGELOG.md`](../../CHANGELOG.md)——这是**对根 [`AGENTS.md`](../../AGENTS.md) 硬约束 2「改完代码必须同步文档」的显式偏离**，归档前若要发版，应先把 §6 / §7 / §8 的内容搬进对应活跃文档（搬的时候按 [DEVELOPMENT.md](../DEVELOPMENT.md) §4 的对照表逐项过一遍）。
> **归档冻结（2026-10-12）**：本快照条目（H1–H3 / M1–M3 / L1–L2 / D1）全部闭环，已按「归档四步」冻结入 `docs/archive/`。文首「未同步写入 user-guide / FEATURES / CHANGELOG」的一次性偏离**已解除**——§6 用户可见行为已搬进 [`user-guide.md`](../user-guide.md)、§7 台账已搬进 [`FEATURES.md`](../FEATURES.md) §CL、§8 发版草稿已搬进 [`CHANGELOG.md`](../../CHANGELOG.md) 的 `[Unreleased]`，并复核了 [`ROADMAP.md`](../ROADMAP.md) §四 与 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 页头的前端基线数字。正文按归档纪律**不改写**。

## 1. 结论摘要

| # | 项 | 档位 | 类型 | 证据位置 | 状态 |
|---|---|---|---|---|---|
| H1 | 切 Tab 中断上传：`onDeactivated(abortAll)` 让「切走」等于「取消」；且切走后任务进度在界面上不可见 | 高 | 缺陷修复 + 新能力（全局任务中心） | [`useUploadQueue.ts`](../../apps/web/src/composables/useUploadQueue.ts) 第 202–206 行 · [`store.ts`](../../apps/web/src/store.ts) 第 73–102 行 · [`ActivityCenter.vue`](../../apps/web/src/components/ActivityCenter.vue)（新，93 行）· [`App.vue`](../../apps/web/src/App.vue) 第 174 行 | ✅ 已闭环 |
| H2 | 对象面板只能点按钮选文件；同类控制台（COS / OSS）支持把文件甚至**整个目录**拖进来 | 高 | 新能力（拖拽上传，含目录递归） | [`dropFiles.ts`](../../apps/web/src/dropFiles.ts)（新，87 行）· [`ObjectsPanel.vue`](../../apps/web/src/components/ObjectsPanel.vue) 第 188–209 / 221–230 行 | ✅ 已闭环 |
| H3 | 切换账号 / 服务器后 `window.location.reload()` **整页刷新**（丢滚动位置、闪白、重连健康检查） | 高 | 缺陷修复（软刷新） | [`ServerPanel.vue`](../../apps/web/src/components/ServerPanel.vue) 第 78 / 94 / 103 行 · [`App.vue`](../../apps/web/src/App.vue) 第 90–92 / 239 行 | ✅ 已闭环 |
| M1 | 过滤逐字符触发重算且**只筛不标注**：用户看不出某行为何还在，也不知道过滤只作用于已加载条目 | 中 | 体验优化（防抖 + 命中高亮 + 范围提示） | [`ObjectToolbar.vue`](../../apps/web/src/components/ObjectToolbar.vue) 第 37–55 行 · [`highlight.ts`](../../apps/web/src/highlight.ts)（新）· [`HighlightText.vue`](../../apps/web/src/components/HighlightText.vue)（新）· [`ObjectList.vue`](../../apps/web/src/components/ObjectList.vue) 第 307 / 383 行 | ✅ 已闭环 |
| M2 | 列表行不可聚焦：键盘用户无法移动「当前条目」，Enter / F2 / Delete 只能作用于「首个已勾选文件」 | 中 | 无障碍 + 可用性（roving tabindex） | [`useObjectBrowser.ts`](../../apps/web/src/composables/useObjectBrowser.ts) 第 52–73 / 152–155 / 375–397 行 · [`ObjectList.vue`](../../apps/web/src/components/ObjectList.vue) 第 60–90 / 163 / 236 行 | ✅ 已闭环 |
| M3 | Toast：同屏可堆成墙、长文案读不完就消失、鼠标正要点击时先消失 | 中 | 体验优化（栈上限 4 / 长文案加时 / 悬停·聚焦暂停） | [`store.ts`](../../apps/web/src/store.ts) 第 116–160 行 · [`Toasts.vue`](../../apps/web/src/components/Toasts.vue) 第 15–18 行 | ✅ 已闭环 |
| L1 | 视图（列表 / 网格）与排序方向**每次刷新都回默认** | 低 | 体验优化（偏好落 localStorage） | [`useObjectBrowser.ts`](../../apps/web/src/composables/useObjectBrowser.ts) 第 82–105 / 218 / 568–573 行 | ✅ 已闭环 |
| L2 | 窄屏（≤900px）的响应式规则**从未被任何自动化扫过** | 低 | 门禁扩面（移动视口 axe 扫描） | [`a11y.spec.ts`](../../apps/web/e2e/a11y.spec.ts) 第 191–203 行 · [accessibility.md](../accessibility.md) §4 第 13 条 / §5.2 | ✅ 已闭环（并抓到 D1） |
| L3 | 列表图标是 emoji（平台字形不一、无法跟随主题着色）→ **刻意不改** | 低 | 建议（不做） | [`ObjectList.vue`](../../apps/web/src/components/ObjectList.vue) 第 293 / 320 行等 | ➖ 维持现状（理由见 §4.3） |
| D1 | **实测发现**：对象列表选中行内的存储类型徽标对比度 **4.39:1 < 4.5:1**（AA） | — | 缺陷修复（真实缺陷） | [`styles.css`](../../apps/web/src/styles.css) 第 31 / 106 行 · [`ObjectList.vue`](../../apps/web/src/components/ObjectList.vue) 第 383 行 · [accessibility.md](../accessibility.md) §5.5 第 14 行 | ✅ 已闭环（由 L2 的扫描抓到） |

## 2. 高优先级

### H1 切 Tab 不再中断上传 + 全局任务中心

- **现象（两侧各一半）**：
  1. [`useUploadQueue.ts`](../../apps/web/src/composables/useUploadQueue.ts) 挂了 `onDeactivated(abortAll)`——用户切到别的页面（组件被 `KeepAlive` 缓存）时，**正在进行的分段上传被主动中止**，且没有任何确认提示；
  2. 状态机随面板一起被缓存，切走后**界面上看不到任何进度**：用户无法判断「切走以后任务是还在跑还是停了」。
- **改动**：
  - 删掉 `onDeactivated(abortAll)`，只在 `onBeforeUnmount` 中止（[`useUploadQueue.ts`](../../apps/web/src/composables/useUploadQueue.ts) 第 202–206 行）——面板被缓存不等于任务取消，卸载才是。
  - 活动快照提到 store（[`store.ts`](../../apps/web/src/store.ts) 第 73–102 行：`Activity` / `activities` / `reportActivity` / `clearActivity`），队列内 watcher 在「活跃时报进度、空闲 / 卸载时清除」（第 179–197 行）。
  - 新增顶栏角标 [`ActivityCenter.vue`](../../apps/web/src/components/ActivityCenter.vue)：整体百分比 + 展开明细（每条任务的 name / 进度条 / `done/total`）、`Esc` 关闭（走 `useKeydownStack`，与其它浮层同一套键栈）、点击外部关闭；**无活动时整体不渲染**，不占顶栏。
  - 三条上报路径：面板上传（`upload:panel`）、对象页多选上传（`upload:objects`，[`useObjectActions.ts`](../../apps/web/src/composables/useObjectActions.ts) 第 373 行）、迁移（`migrate`，`MigratePanel` watch `[busy, progressPct]`）。
  - `App.vue` 的 `<KeepAlive>` 去掉 `:exclude`，改为 `:max="8" :key="serverEpoch"`（[`App.vue`](../../apps/web/src/App.vue) 第 239 行）——`:exclude` 会**卸载**被排除的面板，等于把「切走即中断」写死在模板里。
- **风险与取舍**：`KeepAlive` 同时缓存 8 个面板，内存占用上升（每个面板持有自己的列表状态）；8 是「够用且可证明有界」的常数，且换账号时靠 `:key` 整体重建，不叠加。

### H2 对象页拖拽上传（含目录递归）

- **现象**：对象面板此前只能点「上传」开文件选择框；把文件夹拖进窗口**什么都不发生**（浏览器在无监听时默认跳转到该文件）。
- **改动**：
  - 新增 [`dropFiles.ts`](../../apps/web/src/dropFiles.ts)：把一次拖放展开为「文件 + 相对路径」列表。**优先走 `webkitGetAsEntry()` 递归**（`DataTransfer.files` 只给扁平的顶层文件，目录会变成一个 0 字节条目——静默丢结构比不支持更糟）；`readEntries()` **每次最多返回 100 条，必须循环到返回空数组为止**；单次上限 `MAX_DROPPED_FILES = 5000` 兜住病态目录树；目录读取失败按空目录处理，不中断整次拖放。
  - 队列新增 `enqueueItems(entries)`（[`useUploadQueue.ts`](../../apps/web/src/composables/useUploadQueue.ts) 第 58 / 89 行），把「文件 + 目标 key」直接入队，复用既有并发 / 分段 / 断点续传管线。
  - [`ObjectsPanel.vue`](../../apps/web/src/components/ObjectsPanel.vue) 第 188–209 行：`dragDepth` 计数判断「是否还在窗口内」（只在 `dragenter` / `dragleave` 成对加减，而不是靠 `dragleave` 单点判断，避免拖过子元素时覆盖层闪烁）；第 221–230 行是**固定定位、`pointer-events:none`** 的覆盖层——覆盖层若参与命中测试就会自己触发 `dragleave`，形成抖动的死循环。
  - 无账号 / 无桶（没地方可传）时不做任何拦截，交给浏览器默认行为。
- **路径语义**：保留拖入根的名字（拖 `photos/` 得到 `photos/a.jpg`），与浏览器上传行为一致。
- **测试**：新增 [`dropFiles.test.ts`](../../apps/web/src/dropFiles.test.ts)（11 例：扁平文件回退、目录递归、`readEntries` 分批、超限截断、读失败跳过、非 `file` 条目过滤、无 `items` 时回退 `files`）。

### H3 切换服务器 / 账号不再整页刷新

- **现象**：[`ServerPanel.vue`](../../apps/web/src/components/ServerPanel.vue) 三处（第 78 / 94 / 103 行）在切换活动账号后调用 `window.location.reload()`——整页重载会丢滚动位置、闪白、重跑一遍健康检查与桶列表。
- **改动**：组件改为 `emit('changed')`，由 [`App.vue`](../../apps/web/src/App.vue) 第 90–92 行统一处理：**先 `await loadAccounts()` 再 `bumpServerEpoch()`**（顺序是必需的：先让账号数据就位，再让面板重建，否则重建出的面板会拿到旧账号）。第 239 行 `<KeepAlive :key="serverEpoch">` 用代次切换**整体重建**面板树——比逐面板 `watch` 账号更省心，也避免「旧面板残留旧桶 / 旧目录」。
- **测试**：`ServerPanel.test.ts` 的 `reloadSpy` 断言改为 `emitted('changed')`；`App.test.ts` 新增软刷新用例；`store.test.ts` 新增 `serverEpoch` 用例。

## 3. 中优先级

### M1 过滤体验：防抖 + 命中高亮 + 范围提示

- **防抖**：[`ObjectToolbar.vue`](../../apps/web/src/components/ObjectToolbar.vue) 第 37–55 行——输入框绑 `filterDraft`，200ms 后才写回 `filter`；「清除」按钮**立即**生效（`clearFilter` 不走防抖），因为它表达的是「我确定不要这个条件了」。
- **命中高亮**：新增 [`highlight.ts`](../../apps/web/src/highlight.ts)（大小写不敏感匹配、输出保留原文大小写、空查询返回整段未命中）与 [`HighlightText.vue`](../../apps/web/src/components/HighlightText.vue)——**逐段渲染 `<mark>`，不使用 `v-html`**（禁 `v-html` 是仓库硬约束；外部数据一律不可信，拼接 HTML 会把对象名当代码执行）。
- **范围提示**：[`ObjectList.vue`](../../apps/web/src/components/ObjectList.vue) 第 307 行——过滤生效且结果被截断时提示「过滤只作用于已加载的 N 项，点『加载全部』可搜索整个目录」；否则用户会把「搜不到」误读成「不存在」。

### M2 键盘行导航（roving tabindex）

- **现象**：`<tr>` 不可聚焦 ⇒ 键盘用户无法移动「当前条目」，`Enter` / `F2` / `Delete` 这些全局快捷键只能作用于「首个已勾选文件」，与鼠标右键菜单的能力不对等。
- **改动**：
  - 焦点落点由 [`useObjectBrowser.ts`](../../apps/web/src/composables/useObjectBrowser.ts) 第 52–73 行掌握（`focusKey` / `focusRow` / `moveFocus` / `focusEdge`）——只有它知道**完整的可见顺序**（虚拟窗口只渲染局部）；
  - `ObjectList` 只负责把焦点落到对应行元素上：[`ObjectList.vue`](../../apps/web/src/components/ObjectList.vue) 第 60–90 行的 `setRowEl` + `watch(focusKey)` 落焦，第 163 / 236 行 `:tabindex="e.key === focusKey ? 0 : -1"`，第 85–105 行 `onRowKeydown`（↑/↓ 移动、Home/End 跳首尾、**Enter / Space 行内自行处理并 `stopPropagation`**，否则 window 上的全局快捷键会对「已选中项」再执行一次）；
  - 越界**钳到首尾、不绕回**（文件管理器习惯）；方向键从无焦点状态起步时落到首行；
  - 过滤把焦点行筛掉时**清空焦点**（第 152–155 行），否则方向键会从第一行重启、`F2` / `Delete` 会落在看不见的目标上；
  - 全局快捷键改为**焦点行优先**（第 375–397 行）：`Enter` 预览、`F2` 改名、`Delete` 删除；`Delete` 对**未勾选**的焦点行先勾选再删（否则键盘用户按 Delete 毫无反应）。
- **踩坑（值得留档）**：`@vue/test-utils` 的 `trigger('keydown.enter')` 派发的 `event.key` 是**小写 `'enter'`**，而真实浏览器是 `'Enter'`；组件里必须 `e.key.toLowerCase()` 比较，否则「测试绿、真机失效」。
- **网格视图**同样参与（`ObjectList.vue` 第 163–168 行），保持两种视图的键盘行为一致。

### M3 Toast：栈上限 / 长文案加时 / 悬停与聚焦暂停

- **栈上限 4**（[`store.ts`](../../apps/web/src/store.ts) 第 116 / 145 行）：超出时挤掉**最旧**一条——批量删除 / 迁移会连发多条提示，不设上限时 toast 会糊满右下角。
- **长文案加时**（第 124 行）：基准 3600ms，文案每超 60 字的部分按 **+40ms/字** 加时、封顶 **+3s**（即最长 6.6s）——长句常含文件名与计数，读一半就消失会逼用户重试复现。
- **悬停 / 聚焦暂停**（第 155 行 + [`Toasts.vue`](../../apps/web/src/components/Toasts.vue) 第 15–18 行）：鼠标进入或焦点进入（`focusin`/`focusout`，键盘用户 Tab 到「查看对象 / 关闭」按钮上同理）时**冻结倒计时**，移开 / 失焦后按**剩余时长**续跑；重复进入 / 重复离开都是 no-op。

## 4. 低优先级

### 4.1 L1 视图 / 排序偏好持久化

- **改动**：[`useObjectBrowser.ts`](../../apps/web/src/composables/useObjectBrowser.ts) 第 82–105 行（`readPref` / `writePref` 各自吞掉存储异常、非法值回退默认）+ 第 218 行（视图）+ 第 568–573 行（变更即落盘的 watch）。三个键：`s3c.bucketView` / `s3c.sortKey` / `s3c.sortDir`。
- **刻意不落盘：过滤条件**。它不是「用户的设定」而是「这一次查找」——持久化会让下次打开看到一个被筛过的列表却不知为何，甚至以为对象丢了。这条判断写进 [accessibility.md](../accessibility.md) 同批的改动之外，也记在本文件 §6.7。

### 4.2 L2 移动视口渲染态扫描（并抓到 D1）

- **现象**：`@media (max-width: 900px)` 的响应式规则此前**从未被任何自动化手段扫过**，只靠人读 CSS 与手点。
- **改动**：[`a11y.spec.ts`](../../apps/web/e2e/a11y.spec.ts) 第 191–203 行新增扫描态——视口 390×844 + 复用既有对象桩，除 axe 外**另断言 `documentElement.scrollWidth` 不超出视口**（窄屏最典型的可用性缺陷是「表格把整页撑出横向滚动」，axe 断不出来）。
- **收获**：该扫描**上线即抓到 D1**（见 §5）——这正是 L2 值得做的理由：**扫描通过 ≠ 全站达标，而「没扫过的状态」等于没覆盖**。

### 4.3 L3 图标 SVG 化 —— 刻意不做

- **建议内容**：把列表 / 网格里的 emoji 图标（📁 📄 🖼️ 🎬 🎵 📕 📦 📎）换成内联 SVG，理由是 emoji 字形随平台漂移、无法跟随主题着色。
- **不做的理由**：
  1. **测试面**：现有用例直接断言 emoji 文本（如 `ObjectList.test.ts` 的图标断言、`screenshots.spec.ts` 的截图基线），改图标要连带改一批断言与基线，而收益是纯观感；
  2. **收益有限**：emoji 的「跨平台字形不一致」在本产品的实际影响面很小（图标仅作辅助，行内同时有文字与 `title`），而主题着色问题在浅 / 深两套 token 下也不构成可读性缺陷；
  3. **风险不对称**：图标是每个用户每屏都会看到的元素，改动回归面大。
- **若将来要做**：先补「图标与类型映射」的行为断言（用 `data-icon` 之类的稳定钩子），再换 SVG，最后才有意义地更新截图基线。

## 5. D1：本轮实测抓到的真实缺陷（存储类型徽标对比度）

- **怎么发现的**：L2 的移动视口扫描态首次覆盖了**对象列表视图**（此前 6 个扫描态里只有网格视图，而网格把存储类型画在同一行的 `.gi-meta` 里，没有徽标）——axe 报 `color-contrast`（serious）：`.sc-badge` 前景 `#62766d`（`--muted`）、背景 `#e6f8f0`（`--row-sel`）、字号 10px，实测 **4.39:1**，低于 WCAG AA 正文阈值 **4.5:1**。
- **为什么此前全绿**：不是「axe 漏判」，而是**该 DOM 组合从未进入任何扫描态**；`--muted` 在 `--panel` 上是 4.85:1（达标），只有在 `--row-sel` 上配小字才不达标。这条与本文件 §4.2 的结论互为证据。
- **修复（根因，不是把字号调大绕开）**：`--muted` 是**通用次要文本色**，不该为了一个徽标整体加深（会牵动全站），故新增专用 token：
  - [`styles.css`](../../apps/web/src/styles.css) 第 31 行（浅色 `--badge-text: #3f5a4c`）、第 106 行（深色 `--badge-text: #9fb8ac`）；
  - [`ObjectList.vue`](../../apps/web/src/components/ObjectList.vue) 第 383 行 `.sc-badge` 改用 `var(--badge-text)`；
  - 静态重算：浅色 **6.86:1**、深色 **6.67:1**（方法同 [accessibility.md](../accessibility.md) §5.5 表头）。
- **防复发**：该配对已登记进 [accessibility.md](../accessibility.md) §5.5（表第 14 行），由 [`apps/server/contrast_gate_test.go`](../../apps/server/contrast_gate_test.go) 从 `styles.css` **机械重算比对**——改色不重算即红灯；渲染态另有移动视口扫描兜底。

## 6. 用户可见行为变更（原拟写入 `user-guide.md`，本批次集中在此）

> 这一节是为了让「界面怎么用」有单一出处；归档 / 发版前应搬进 [`user-guide.md`](../user-guide.md)（见文首分工声明）。

### 6.1 键盘行导航

| 按键 | 作用 |
|---|---|
| `Tab` | 进入列表：焦点落到**当前条目**（记忆上一次的焦点行，首次进入为第一行） |
| `↑` / `↓` | 在可见条目间上 / 下移动焦点（到首 / 尾即停，不绕回） |
| `Home` / `End` | 跳到可见条目的第一项 / 最后一项 |
| `Enter` | 打开 / 预览**焦点行**（未勾选也可用；文件夹为进入目录） |
| `Space` | 勾选 / 取消勾选焦点行 |
| `F2` | 重命名焦点行 |
| `Delete` | 删除已勾选文件；若没有勾选项，则删除**焦点行**（会先自动勾选该行） |

- 焦点行优先于「首个已勾选文件」；行内按键处理后会阻断冒泡，不会与全局快捷键重复执行。
- 列表与网格两种视图行为一致。

### 6.2 拖拽上传

- 在对象面板（已进入某个桶）把**文件或文件夹**拖到窗口任意位置：出现覆盖层「松开即可上传 / 上传到 `<当前目录>`（保留拖入的目录结构）」。
- 松手即入队，复用既有并发 / 分段 / 断点续传规则；目录按 `webkitGetAsEntry` 递归展开，**保留目录结构**。
- 限制：单次最多 **5000** 个文件（超出部分被丢弃，属有意的病态输入兜底）；无账号 / 未进入桶时不响应拖放。
- 上传面板的拖放区沿用同一套展开逻辑（`upload.dropTitle` / `upload.dropSub` 文案同批更新）。

### 6.3 顶栏任务中心

- 有上传 / 迁移在跑时，顶栏出现**进度角标**（旋转指示 + 整体百分比）。点击展开明细：每条任务的名称（如「文件上传」「文件迁移」）、进度条、`已完成/总数 · 百分比`。
- `Esc` 或点击弹层外部关闭；任务全部结束后角标自动消失。
- **切页面不再中断任务**：切到别的标签页后，上传 / 迁移继续推进，进度看角标。

### 6.4 过滤

- 输入过滤词后 **200ms** 才生效（连续输入不会逐字符重算）；点「清除」立即恢复全量。
- 命中的片段在对象名里**高亮**（`<mark>`），列表与网格都是。
- 若过滤时列表被截断（未加载全部），列表底部提示「过滤只作用于已加载的 N 项，点『加载全部』可搜索整个目录」。

### 6.5 切换服务器 / 账号

- 在「服务器设置」里切换活动账号后，**页面不再整页刷新**：只重载账号数据并重建各面板，滚动位置与其它面板的展开状态不再丢失。

### 6.6 Toast 提示

- 同屏最多 **4** 条，新的提示会挤掉最旧的。
- 文案越长停留越久（基准 3.6s，最长约 6.6s）。
- **鼠标移上去、或键盘焦点停在提示内的按钮上时，倒计时冻结**；移开 / 失焦后从剩余时间继续。

### 6.7 偏好持久化

- **会记住**：视图（列表 / 网格）、排序字段与方向——刷新 / 重启后保持上次的选择。
- **不会记住**：过滤条件（每次打开都是全量列表，避免「列表莫名少了一半」）。
- 存储不可用时（隐私模式 / 配额异常）静默回退默认值，功能不受影响。

### 6.8 新增 / 更新的界面文案（i18n）

- 新增 8 个 key：`activity.title` / `activity.aria` / `activity.upload` / `activity.migrate` / `activity.hint`（`common.ts`，49 → **54**）、`objects.dropTitle` / `objects.dropTo` / `objects.filterScopeHint`（`objects.ts`，139 → **142**）。
- 改写 3 个 key（数量不变，`upload.ts` 仍 73）：`upload.dropAria` / `upload.dropTitle` / `upload.dropSub`（拖放区改为同时支持文件夹，文案写明「保留目录结构」）。
- 字典总量 **818 → 826**；已同步 [`i18n.md`](../i18n.md)，由 [`doc_number_gate_test.go`](../../apps/server/doc_number_gate_test.go) 的 `TestI18nModuleInventoryMatchesDoc` 机械校验（本批实测：8 模块 / 826 key）。

## 7. 台账（原拟写入 `FEATURES.md`，本批次集中在此）

| 批次 | 改动面（生产代码） | 新增 / 扩充测试 |
|---|---|---|
| H1 任务中心 | `store.ts`（`Activity` / `activities` / `reportActivity` / `clearActivity`）、`ActivityCenter.vue`（新）、`App.vue`（挂载 + KeepAlive 改造）、`useUploadQueue.ts`（activity 上报、去掉 `onDeactivated`）、`useObjectActions.ts` / `UploadPanel.vue` / `MigratePanel.vue`（activity 接线 + `onActivated` 刷新） | `ActivityCenter.test.ts`（新）、`useUploadQueue.test.ts`（活动用例）、`MigratePanel.test.ts`（活动 + `onActivated`）、`App.test.ts` |
| H2 拖拽上传 | `dropFiles.ts`（新）、`useUploadQueue.ts`（`enqueueItems`）、`useObjectActions.ts`（`dropUpload`）、`ObjectsPanel.vue`（drag 处理 + 覆盖层）、`UploadPanel.vue`（改用 `filesFromDrop`） | `dropFiles.test.ts`（新，11 例）、`ObjectsPanel.test.ts`（drop 用例）、`UploadPanel.test.ts` |
| H3 软刷新 | `ServerPanel.vue`（三处 reload → `emit('changed')`）、`App.vue`（`onServerChanged` + `:key="serverEpoch"`）、`store.ts`（`serverEpoch` / `bumpServerEpoch`） | `ServerPanel.test.ts`、`App.test.ts`、`store.test.ts` |
| M1 过滤体验 | `highlight.ts`（新）、`HighlightText.vue`（新）、`ObjectToolbar.vue`（防抖）、`ObjectList.vue`（高亮 + 范围提示） | `highlight.test.ts`（新，8 例）、`HighlightText.test.ts`（新，4 例）、`ObjectToolbar.test.ts`、`ObjectList.test.ts` |
| M2 键盘行导航 | `useObjectBrowser.ts`（`focusKey` / `moveFocus` / `focusEdge` / 失效焦点清理 / 全局键焦点行优先）、`ObjectList.vue`（roving tabindex + `onRowKeydown`） | `ObjectList.test.ts`、`useObjectBrowser.branches.test.ts` |
| M3 Toast | `store.ts`（栈上限 / 时长 / 暂停）、`Toasts.vue`（hover + focus 事件） | `store.test.ts`、`Toasts.test.ts` |
| L1 偏好持久化 | `useObjectBrowser.ts`（`readPref` / `writePref` + 落盘 watch） | `useObjectBrowser.test.ts`（读取 / 非法值 / 写失败三类） |
| L2 + D1 | `e2e/a11y.spec.ts`（移动视口态）、`styles.css`（`--badge-text`）、`ObjectList.vue`（`.sc-badge` 换 token） | `e2e/a11y.spec.ts`（e2e，非 vitest） |

## 8. 发版记录草稿（原拟写入 `CHANGELOG.md` 的 `[Unreleased]`，本批次集中在此）

> 归档 / 发版前请把下面这段（连同 §6 的行为说明、§7 的台账）搬进 [`../CHANGELOG.md`](../../CHANGELOG.md) 的 `[Unreleased]`。

**变更（2026-10-12 apps/web 页面 UI 与交互优化批次）**

- **高**：切 Tab 不再中断上传并新增**顶栏全局任务中心**（角标 + 明细 + Esc / 点外关闭，`upload:panel` / `upload:objects` / `migrate` 三类活动统一上报）；对象面板支持**拖拽上传文件与整个目录**（`webkitGetAsEntry` 递归 + `readEntries` 循环 + 5000 文件上限）；切换服务器 / 账号改为**软刷新**（`emit('changed')` → 先 `loadAccounts()` 再 `bumpServerEpoch()`，`<KeepAlive :key>` 重建面板，去掉三处 `window.location.reload()`）。
- **中**：过滤 **200ms 防抖** + 命中片段 **`<mark>` 高亮**（`highlight.ts` + `HighlightText.vue`，不用 `v-html`）+ 截断时**过滤范围提示**；列表 / 网格新增 **roving tabindex 键盘行导航**（↑↓ / Home / End / Enter / Space / F2 / Delete，焦点行优先，过滤筛掉即清焦点）；Toast **同屏上限 4**、长文案**按字数加时**（封顶 +3s）、**悬停 / 聚焦冻结倒计时**。
- **低**：视图与排序偏好落 localStorage（**过滤条件刻意不落盘**）；`e2e/a11y.spec.ts` 新增**移动视口 390×844 扫描态**（另断言无横向溢出），并据此修复**存储类型徽标对比度 4.39:1 → 6.86:1**（新增 `--badge-text` token，`accessibility.md` §5.5 由 `contrast_gate_test.go` 机械重算）。
- **验证**：`pnpm test` 87 文件 1360 例全绿 / `pnpm test:coverage` 四指标 100%（5108 / 3395 / 1290 / 4407）/ `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` exit 0 / `pnpm build` OK / `pnpm e2e a11y.spec.ts` 7 passed（含新增移动视口态）。§9 列明未跑项。

## 9. 门禁实跑与复现

| 门禁 | 命令 | 实测（2026-10-12） |
|---|---|---|
| 前端单测 | `cd apps/web && pnpm test` | ✅ **87 文件 / 1360 例**全绿（基线 83 / 1292；本批 +4 文件 / +68 例） |
| 前端覆盖率 | `cd apps/web && pnpm test:coverage` | ✅ 四指标 **100%**（statements 5108 / branches 3395 / functions 1290 / lines 4407） |
| 前端 lint | `cd apps/web && pnpm lint` | ✅ 0 error / 0 warning |
| 前端类型 | `cd apps/web && pnpm typecheck` · `pnpm typecheck:e2e` | ✅ 均 exit 0 |
| 前端构建 | `cd apps/web && pnpm build` | ✅ OK（`vue-tsc` + `vite build`；js 427.06 kB / gzip 128.51 kB，css 35.78 kB / gzip 7.45 kB） |
| 浏览器 a11y | `cd apps/web && pnpm e2e a11y.spec.ts` | ✅ **7 passed**（6 个界面状态 + 1 条 axe 有效性自检）；其中移动视口态上线即抓 D1 → 修复后复跑绿 |
| 根包文档门禁 | `cd apps/server && go test . -count=1` | ✅ ok（含 `TestI18nModuleInventoryMatchesDoc`（818→**826** 已同步）/ `TestContrastRecordMatchesStylesTokens`（新增第 14 行）/ 命名 · 导航 · 链接 · 归档索引各路门禁） |
| Go 静态检查 | `cd apps/server && go vet ./...` | ✅ 0 告警 |

**本轮未跑 / 跑不完（如实披露）**：

- **全量 `pnpm e2e`**（除 a11y 外的其余 spec）：本轮被中断，只跑完 `a11y.spec.ts`；涉及拖拽上传与任务中心的交互式用例未在真实浏览器跑（单测覆盖了这些分支）。
- **`make e2e-real`**（真实后端 + 真实 RustFS + 真实产物，不 mock `/api`）：**未跑**——本批次**未改任何后端接口 / 请求体 / 预签名链路**（改动面见 §7 的生产代码列，全部在前端），故按根 [`AGENTS.md`](../../AGENTS.md) 的门禁口径「涉及签名 / 预签名直传 / 分段 / 复制 / 标签 / 版本控制时追加」**不属于必跑项**；但拖拽上传最终走的是既有的预签名直传管线，**若要按最严口径收口，建议补跑**。
- **`apps/server` 全量 `go test ./...`**：本轮被中断（只跑完根包 `go test .`，即全部门禁所在包）；本批无 Go 生产代码改动。
- **`pnpm audit`**：镜像 `registry.npmmirror.com` 不提供 audit 端点（既有已知限制，口径同 [ROADMAP.md](../ROADMAP.md) §四）。
- **真机 / 触摸设备**：移动视口扫描是**真实 Chromium 的窄视口**，不是真机；触摸拖拽（移动端拖放本就不被浏览器支持）与屏幕阅读器实测均未覆盖。

## 10. 刻意不做 / 未覆盖（汇总）

| 项 | 决定 | 理由 |
|---|---|---|
| 过滤条件持久化 | 不做 | 见 §4.1 / §6.7：临时查询不该跨会话粘住 |
| 图标 emoji → SVG（L3） | 不做 | 见 §4.3：测试面改动大、收益为观感 |
| 网格单元格加 `aria-label` | 不做（沿既有决策） | 单元格文本节点派生的可访问名已被 axe 判定通过，`aria-label` 会覆盖文本、丢掉大小上下文并造出第二事实源（见 [accessibility.md](../accessibility.md) §4 第 9 条） |
| 极窄视口（<360px）/ 横屏 / 200% 缩放扫描 | 未做 | 见 [accessibility.md](../accessibility.md) §4 第 13 条「仍未做」 |
| 用户手册 / 台账 / CHANGELOG 的正式同步 | **本轮刻意集中在本文件** | 见文首分工声明——这是对根 `AGENTS.md` 硬约束 2 的**显式偏离**，归档 / 发版前必须搬回 |

## 11. 闭环与归档

- **闭环状态**：§1 表内 H1–H3 / M1–M3 / L1–L2 / D1 **全部已闭环**；L3 为「刻意不做」的决策项（非待办），§10 的未覆盖项已如实标注。
- **归档前置**：本文件是**时点性子树评审快照**——归档 = 冻结、不回写。归档时按 [archive/index.md](index.md)「归档四步」执行：① `git mv` 进 `docs/archive/`；② 修正全仓指向旧路径的引用（含本文件在 [README.md](../README.md) 导航、[DEVELOPMENT.md](../DEVELOPMENT.md) §4 登记表、[`../llms.txt`](../../llms.txt) 目录段的登记）；③ 在 `archive/index.md` 补一行；④ 在 `CHANGELOG.md` 的 `[Unreleased]` 记一条「变更」。
- **归档前必须先做的一件事**：把 §6 / §7 / §8 搬进 [`user-guide.md`](../user-guide.md) / [`FEATURES.md`](../FEATURES.md) / [`CHANGELOG.md`](../../CHANGELOG.md)，并复核 [ROADMAP.md](../ROADMAP.md) §四 与 [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) 页头里的前端基线数字（本批后应为 **87 文件 1360 例 / 四指标 5108 · 3395 · 1290 · 4407**，原文仍写着上一批的 83 / 1292 / 4829…）——否则归档会把「一次性快照」变成唯一的现状来源。
