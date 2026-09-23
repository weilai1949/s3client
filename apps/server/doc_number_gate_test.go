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
//
// 断言范围（刻意不做的事）：
//   - 只校验**登记在 docNumberClaims 里**的数字声明。md 里其它叙述性数字（测试用例数、覆盖率、
//     文件行数等）依赖运行环境或时刻，不适合做静态门禁——它们属「历史记录」而非「当前契约」。
//     新增一条可机械推导的数字声明时，在此登记即可获得保护。
//   - 只匹配**精确的措辞**（正则），改文案可能让门禁失效；因此 docNumberClaims 的每条都要求
//     在目标文件里**至少命中一次**，命中 0 次即红灯（防止文案漂移后门禁静默失效）。

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// docNumberClaims 是全部受门禁保护的「叙述性数字」声明。
//
// 注意：这里**不**收录 features.md §B/§C 等历史台账里的数字——那些记录的是当时状态，
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
		file: "docs/roadmap.md",
		re:   regexp.MustCompile("(\\d+) 个 `/api/\\*` 端点、OpenAPI"),
		got:  apiRouteCountFromSource,
	},
	{
		file: "docs/features.md",
		re:   regexp.MustCompile(`\| REST 端点 \| \*\*(\d+)\*\* 个`),
		got:  apiRouteCountFromSource,
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
