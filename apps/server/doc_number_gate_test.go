package main

// doc_number_gate_test.go —— **文档叙述性数字**的源码门禁。
//
// 背景（docs/archive/review-2026-09-19.md §7.4）：`routes.go` ↔ 注册表 ↔ `docs/api.md` 已有三重门禁，
// 但**写在 md 正文里的数字**（「N 个端点」这类叙述）没有任何机械校验——改代码时忘了改文档，
// 门禁全绿而文档失真。审查点名这是矩阵里唯一标「否」的一行。
//
// 本门禁把「可机械推导的叙述性数字」钉在源码上：
//   - 端点总数：以 `apps/server/internal/handler/routes.go` 的 `mux.HandleFunc` 注册数为准，
//     校验 README / docs 中「N 个 `/api/*` 端点」的 N。
//   - 配置项总数：以 `internal/config` 生产代码抽取的 `S3C_*` 变量数为准（复用
//     config_doc_gate_test.go 的 `configEnvVarsFromSource`），校验 README 中
//     「N 个 `S3C_*` 变量」的 N——2026-10-10 实测该数字停在 18，而真值为 22。
//   - i18n 模块清单（2026-10-10 交接快照 §5 未做第 9 项）：`docs/i18n.md` §2 的模块表
//     （文件 / key 数）与「拆成 N 个模块」「共 **M** 个 key」两句叙述，必须与
//     `apps/web/src/i18n/messages/*.ts`（除 `types.ts`）实测一致——此前新增消息模块后
//     文档可以停在旧数字而全绿（前端 `coverage.test.ts` 只校验**键集合**，不校验模块清单）。
//
// 断言范围（刻意不做的事）：
//   - 只校验**登记在 docNumberClaims 里**的数字声明。md 里其它叙述性数字（测试用例数、覆盖率、
//     文件行数等）依赖运行环境或时刻，不适合做静态门禁——它们属「历史记录」而非「当前契约」。
//     新增一条可机械推导的数字声明时，在此登记即可获得保护。
//   - 只匹配**精确的措辞**（正则），改文案可能让门禁失效；因此 docNumberClaims 的每条都要求
//     在目标文件里**至少命中一次**，命中 0 次即红灯（防止文案漂移后门禁静默失效）。
//   - `TestI18nModuleInventoryMatchesDoc` 只认「模块**文件**」粒度：命名空间示例列与覆盖范围
//     列的文字不校验（自然语言）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// docNumberClaim 描述一条「文档里的数字必须等于源码推导值」的声明。
type docNumberClaim struct {
	file string // 相对仓库根的 md 路径
	// re 必须捕获第 1 组为数字；用精确措辞避免误伤历史记录。
	re  *regexp.Regexp
	got func(t *testing.T) int // 从源码推导的期望值
}

// apiRouteCountFromSource 解析 routes.go 的 mux.HandleFunc 注册数。
func apiRouteCountFromSource(t *testing.T) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(serverRoot(t), "internal", "handler", "routes.go"))
	if err != nil {
		t.Fatalf("读取 routes.go: %v", err)
	}
	n := len(regexp.MustCompile(`(?m)^\s*mux\.HandleFunc\(`).FindAll(b, -1))
	if n == 0 {
		t.Fatal("routes.go 未解析到任何 mux.HandleFunc，疑似解析口径失效")
	}
	return n
}

// gateFileCountFromSource 统计包根（`apps/server/`，非递归）的 `*_gate_test.go` 文件数——
// AGENT_EVALS.md §一 的基线句会引用这个数，加/删门禁文件必须同步该句。
func gateFileCountFromSource(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(repoRoot(t) + "/apps/server")
	if err != nil {
		t.Fatalf("读取 apps/server: %v", err)
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_gate_test.go") {
			continue
		}
		n++
	}
	if n == 0 {
		t.Fatal("apps/server 下未找到任何 *_gate_test.go，疑似目录口径失效")
	}
	return n
}

// frontendMethodCountFromSource 解析前端 endpoints.ts 的 `s3api` 对象顶层方法数
// （行首两空格 + `name:` 的条目，与 ops 生成物 operations.ts 同量纲）。
func frontendMethodCountFromSource(t *testing.T) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "apps", "web", "src", "api", "endpoints.ts"))
	if err != nil {
		t.Fatalf("读取 endpoints.ts: %v", err)
	}
	n := len(regexp.MustCompile(`(?m)^  [a-zA-Z_]+:`).FindAll(b, -1))
	if n == 0 {
		t.Fatal("endpoints.ts 未解析到任何 s3api 方法，疑似解析口径失效")
	}
	return n
}

// docNumberClaims 是全部受门禁保护的「叙述性数字」声明。
//
// 注意：这里**不**收录 FEATURES.md §B/§C 等历史台账里的数字——那些记录的是当时状态，
// 按「不追溯篡改」纪律应保持原样。
var docNumberClaims = []docNumberClaim{
	{
		file: "README.md",
		re:   regexp.MustCompile("(\\d+) 个 `/api/\\*` 端点"),
		got:  apiRouteCountFromSource,
	},
	{
		file: "docs/api.md",
		re:   regexp.MustCompile("作为 (\\d+) 个 `/api/\\*` 端点的契约单一来源"),
		got:  apiRouteCountFromSource,
	},
	{
		file: "docs/ROADMAP.md",
		re:   regexp.MustCompile("(\\d+) 个 `/api/\\*` 端点、OpenAPI"),
		got:  apiRouteCountFromSource,
	},
	{
		file: "docs/FEATURES.md",
		re:   regexp.MustCompile(`\| REST 端点 \| \*\*(\d+)\*\* 个`),
		got:  apiRouteCountFromSource,
	},
	{
		// docs/CONFIGURATION.md 的「全量」口径由 config_doc_gate_test.go 保证；这里只钉
		// README 转述的那个数字，防止新增配置项后 README 继续停在旧值。
		file: "README.md",
		re:   regexp.MustCompile("(\\d+) 个 `S3C_\\*` 变量"),
		got:  func(t *testing.T) int { return len(configEnvVarsFromSource(t)) },
	},
	{
		// 英文页同款（2026-10-10 实测 docs/en/index.md 停在 18，真值 22——英文页此前不在扫描面）。
		// 「18」与「`S3C_*`」在源文件里被换行分隔，故用 \s+ 跨空白匹配。
		file: "docs/en/index.md",
		re:   regexp.MustCompile("all (\\d+)\\s+`S3C_\\*` variables"),
		got:  func(t *testing.T) int { return len(configEnvVarsFromSource(t)) },
	},
	{
		file: "docs/en/index.md",
		re:   regexp.MustCompile("(\\d+) `/api/\\*` endpoints"),
		got:  apiRouteCountFromSource,
	},
	{
		// 英文架构页把端点数写成 72（与前端方法数混淆），必须钉在 routes.go 上。
		file: "docs/en/architecture.md",
		re:   regexp.MustCompile("(\\d+) `/api/\\*` endpoints"),
		got:  apiRouteCountFromSource,
	},
	{
		// 前端 s3api 方法数（apps/web/src/api/endpoints.ts 顶层 `name:` 条目），
		// 与后端端点数是两个量纲，各自钉住。
		file: "docs/architecture.md",
		re:   regexp.MustCompile(`s3api 对象，(\d+) 个方法`),
		got:  frontendMethodCountFromSource,
	},
	{
		file: "docs/en/architecture.md",
		re:   regexp.MustCompile("`s3api` object, (\\d+) methods"),
		got:  frontendMethodCountFromSource,
	},
	{
		// 「包根门禁道数」是**每次加门禁就会漂移**的数字：2026-10-10 本批连加两道
		// （archive_index_gate / adr_format_gate）后文档停在 27。钉住它，新增/删除门禁文件
		// 必须同步 AGENT_EVALS.md §一 的基线句。
		file: "docs/AGENT_EVALS.md",
		re:   regexp.MustCompile("`apps/server/\\*_gate_test.go` (\\d+) 道"),
		got:  gateFileCountFromSource,
	},
}

// TestDocNarrativeNumbersMatchSource 校验受登记保护的叙述性数字与源码一致。
func TestDocNarrativeNumbersMatchSource(t *testing.T) {
	t.Parallel()
	if len(docNumberClaims) == 0 {
		t.Fatal("docNumberClaims 为空，门禁形同虚设")
	}
	for _, c := range docNumberClaims {
		c := c
		t.Run(c.file, func(t *testing.T) {
			t.Parallel()
			b, err := os.ReadFile(filepath.Join(repoRoot(t), c.file))
			if err != nil {
				t.Fatalf("读取 %s: %v", c.file, err)
			}
			ms := c.re.FindAllStringSubmatch(string(b), -1)
			if len(ms) == 0 {
				t.Fatalf("%s 中未匹配到受门禁保护的措辞 %q；文案已漂移，"+
					"门禁已失效——请同步 docNumberClaims 的正则", c.file, c.re.String())
			}
			want := c.got(t)
			for _, m := range ms {
				n, err := strconv.Atoi(m[1])
				if err != nil {
					t.Fatalf("%s 捕获组 %q 不是整数: %v", c.file, m[1], err)
				}
				if n != want {
					t.Errorf("%s 声称 %d，但源码推导值为 %d（改代码时未同步文档）", c.file, n, want)
				}
			}
		})
	}
}

// ---- compatibility.md §6.2 证据口径：符号而非行号（交接快照 §5 未做第 7 项）----
//
// 背景：§6.2 客户端支持矩阵的「依据」列原先引用 `file:line`（如 `styles.css 186–194 行`、
// `package.json 18 行`）——行号是**会漂移的叙述性数字**：每次重构都要人工重数，2026-10-10
// 的上一批刚一次性订正过 12 处。根治口径：只引用**符号**（配置键 / 脚本名 / 函数名 / 选择器 /
// 包名），可直接 grep 定位。本断言把「§6.2 不再写行号」变成红灯，防口径回退。

const docNumberCompatibility = "docs/compatibility.md"

// compatibilityLineNumberRe 匹配行号引用：`21–26 行` / `3、14 行` / `18 行`。
var compatibilityLineNumberRe = regexp.MustCompile(`\d+\s*[–—-]\s*\d+\s*行|\d+\s*行`)

// TestCompatibilityMatrixCitesSymbolsNotLineNumbers 断言 §6.2 证据列不含行号引用。
func TestCompatibilityMatrixCitesSymbolsNotLineNumbers(t *testing.T) {
	t.Parallel()

	text := readRepoFile(t, docNumberCompatibility)
	start := strings.Index(text, "### 6.2")
	if start < 0 {
		t.Fatalf("%s 未找到 §6.2 小节标题——章节结构变更需同步本门禁", docNumberCompatibility)
	}
	rest := text[start:]
	end := strings.Index(rest, "\n## 7")
	if end < 0 {
		t.Fatalf("%s §6.2 之后未找到 `## 7` 边界——章节结构变更需同步本门禁", docNumberCompatibility)
	}
	section := rest[:end]
	if len(section) < 1500 {
		t.Fatalf("扫描面塌缩：§6.2 只截到 %d 字节（基线 ≥1500）——切分口径失效", len(section))
	}
	if !strings.Contains(section, "文件与符号") {
		t.Fatalf("%s §6.2 表头不再是「依据（文件与符号）」——口径变更需同步本门禁", docNumberCompatibility)
	}
	for _, line := range strings.Split(section, "\n") {
		if m := compatibilityLineNumberRe.FindString(line); m != "" {
			t.Errorf("%s §6.2 仍引用行号 %q——证据只写符号（配置键 / 脚本名 / 函数 / 选择器），"+
				"行号会随重构漂移：%s", docNumberCompatibility, m, strings.TrimSpace(line))
		}
	}
}

// ---- i18n 模块清单 ⇄ 源码（交接快照 §5 未做第 9 项）----
//
// 口径与 `docs/i18n.md` §2 尾注的「统计口径」一致：逐模块取 `'zh-CN': {` 到 2 空格缩进
// 收尾 `},` 的块，统计块内 `'键':` 行数——与前端 `coverage.test.ts` 的中英键集合检查互补
// （后者管「键集合一致 / 无死键」，本断言管「模块清单与文档一致」）。

const (
	docNumberI18nDoc        = "docs/i18n.md"
	docNumberI18nMsgDir     = "apps/web/src/i18n/messages"
	docNumberMinI18nModules = 6
	docNumberMinI18nKeys    = 600
)

var (
	// i18nModuleRowRe 匹配 §2 表行的首列模块文件名（``| `common.ts` |``）。
	i18nModuleRowRe = regexp.MustCompile("^\\|\\s*`([A-Za-z][A-Za-z0-9_]*\\.ts)`")
	// i18nModuleCountRe / i18nTotalKeyRe 是两句叙述性数字的精确措辞。
	i18nModuleCountRe = regexp.MustCompile(`拆成 (\d+) 个模块`)
	i18nTotalKeyRe    = regexp.MustCompile(`共 \*\*(\d+)\*\* 个 key`)
	// i18nZhStartRe / i18nZhEndRe 圈定单模块的 `'zh-CN': { … }` 块（收尾为 2 空格缩进的 `},`）。
	i18nZhStartRe = regexp.MustCompile(`^\s*['"]zh-CN['"]\s*:\s*\{`)
	i18nZhEndRe   = regexp.MustCompile(`^\s{2}\},?\s*$`)
	// i18nKeyLineRe 统计块内的键行。
	i18nKeyLineRe = regexp.MustCompile(`^\s*'[^']+':`)
)

// i18nSourceModuleCounts 返回 messages 目录下每个模块（除 types.ts）的 zh-CN 键数。
// 形状不符（找不到语言块 / 块内 0 键）即 Fatal——模块模板变更必须同步本门禁。
func i18nSourceModuleCounts(t *testing.T) map[string]int {
	t.Helper()
	dir := filepath.Join(repoRoot(t), filepath.FromSlash(docNumberI18nMsgDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 %s: %v", docNumberI18nMsgDir, err)
	}
	out := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".ts") || name == "types.ts" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("读取 %s: %v", name, err)
		}
		inZh, n := false, 0
		for _, line := range strings.Split(string(b), "\n") {
			if !inZh {
				if i18nZhStartRe.MatchString(line) {
					inZh = true
				}
				continue
			}
			if i18nZhEndRe.MatchString(line) {
				break
			}
			if i18nKeyLineRe.MatchString(line) {
				n++
			}
		}
		if !inZh {
			t.Fatalf("%s 未找到 `'zh-CN': {` 语言块——模块形状变更需同步本门禁", name)
		}
		if n == 0 {
			t.Fatalf("%s 的 zh-CN 块未解析到任何键行——统计口径失效", name)
		}
		out[name] = n
	}
	return out
}

// TestI18nModuleInventoryMatchesDoc 校验 docs/i18n.md §2 模块表（文件 ⇄ 逐模块 key 数）
// 与两句叙述数字（模块数 / key 总数）同 `apps/web/src/i18n/messages/` 实测一致。
func TestI18nModuleInventoryMatchesDoc(t *testing.T) {
	t.Parallel()

	src := i18nSourceModuleCounts(t)
	if len(src) < docNumberMinI18nModules {
		t.Fatalf("扫描面塌缩：只解析到 %d 个消息模块（基线 ≥%d）——目录口径或排除清单被改坏",
			len(src), docNumberMinI18nModules)
	}
	total := 0
	for _, n := range src {
		total += n
	}
	if total < docNumberMinI18nKeys {
		t.Fatalf("扫描面塌缩：消息模块 key 总数只有 %d（基线 ≥%d）——键行统计口径失效",
			total, docNumberMinI18nKeys)
	}

	text := readRepoFile(t, docNumberI18nDoc)

	m := i18nModuleCountRe.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("%s 未匹配到叙述数字措辞「拆成 N 个模块」——文案漂移，门禁已失效", docNumberI18nDoc)
	}
	claimedModules, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("%s 模块数解析失败 %q: %v", docNumberI18nDoc, m[1], err)
	}
	if claimedModules != len(src) {
		t.Errorf("%s 声称「%d 个模块」，但 %s 下有 %d 个模块文件（除 types.ts）",
			docNumberI18nDoc, claimedModules, docNumberI18nMsgDir, len(src))
	}

	m = i18nTotalKeyRe.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("%s 未匹配到叙述数字措辞「共 **M** 个 key」——文案漂移，门禁已失效", docNumberI18nDoc)
	}
	claimedKeys, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("%s key 总数解析失败 %q: %v", docNumberI18nDoc, m[1], err)
	}
	if claimedKeys != total {
		t.Errorf("%s 声称「%d 个 key」，但源码逐模块 zh-CN 键数之和为 %d",
			docNumberI18nDoc, claimedKeys, total)
	}

	// §2 表行：首列模块文件名 + 末列 key 数；types.ts 行末列为「—」，只校验其存在。
	docRows := map[string]int{}
	sawTypes := false
	for _, line := range strings.Split(text, "\n") {
		mm := i18nModuleRowRe.FindStringSubmatch(line)
		if mm == nil {
			continue
		}
		if mm[1] == "types.ts" {
			sawTypes = true
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			t.Errorf("%s §2 表行解析失败：%q", docNumberI18nDoc, line)
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(cells[len(cells)-2]))
		if err != nil {
			t.Errorf("%s §2 表行 %q 末列不是 key 数（应为整数）: %v", docNumberI18nDoc, mm[1], err)
			continue
		}
		docRows[mm[1]] = n
	}
	if !sawTypes {
		t.Errorf("%s §2 表缺 `types.ts` 行——表形态变更需同步本门禁", docNumberI18nDoc)
	}
	if len(docRows) < docNumberMinI18nModules {
		t.Fatalf("扫描面塌缩：%s §2 表只解析到 %d 行（基线 ≥%d）——表格形态被改坏",
			docNumberI18nDoc, len(docRows), docNumberMinI18nModules)
	}
	for name, n := range src {
		want, ok := docRows[name]
		if !ok {
			t.Errorf("消息模块 %s 未登记进 %s §2 表——新增模块必须同 PR 补表（§2 尾注要求五处同步）",
				name, docNumberI18nDoc)
			continue
		}
		if want != n {
			t.Errorf("%s §2 表称 %s 有 %d 个 key，实测 %d（改字典后未同步文档）",
				docNumberI18nDoc, name, want, n)
		}
	}
	for name := range docRows {
		if _, ok := src[name]; !ok {
			t.Errorf("%s §2 表登记的 %s 在 %s 下不存在——幽灵模块或文件被误删",
				docNumberI18nDoc, name, docNumberI18nMsgDir)
		}
	}
	t.Logf("消息模块 %d 个、zh-CN 键共 %d（文档声称 %d 个模块 / %d 个 key）",
		len(src), total, claimedModules, claimedKeys)
}
