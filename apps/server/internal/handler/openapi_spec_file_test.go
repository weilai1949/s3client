package handler

// openapi_spec_file_test.go —— 提交版 OpenAPI 规范文件（docs/api/openapi.json）的一致性门禁。
//
// 背景：本仓库的 API 契约 SSOT 一直是**运行时**的 `GET /api/openapi.json`（默认 404，
// 需 S3C_EXPOSE_OPENAPI=1 才暴露）。这带来一个真实的可用性缺口：任何**外部**消费方
// ——Swagger UI、客户端代码生成、契约 diff 工具，以及 AI 编码代理——都必须先把服务跑起来
// 才能读到契约。故把规范落盘为仓库内文件，并加这道门禁防止「改了 API 忘了重新生成」。
//
// 断言范围与残留：
//   - 只断言「文件与运行时规范**语义一致**」。文件为便于 review / diff 做了缩进，比对前先
//     `json.Compact` 归一化；`buildSpec` 走 `map[string]any` + `json.Marshal`（key 有序且
//     确定，见 openapi_shape_test.go 的确定性用例），故归一化后逐字节相等成立。
//   - **不**断言缩进宽度等格式细节。
//   - 版本号取自 `apps/server/main.go` 的 `var version`（源码真值，口径同
//     doc_number_gate_test.go 的 `apiRouteCountFromSource`）——改版本号必须重新生成规范，
//     否则红灯。这也是发版脚本 `scripts/release-version.sh` 需要同步本文件的原因。
//
// 重新生成（改完 handler / 注册表后）：
//
//	go test ./internal/handler/ -run TestCommittedOpenAPISpecMatchesRuntime -update-openapi-spec
//
// 该文件格式为 JSON（非 YAML）：`internal/openapi` 明确「不引入额外依赖」，而 JSON 是
// YAML 1.2 的子集，YAML 工具链同样可消费。

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// updateOpenAPISpec 控制是否用运行时规范重写提交版文件（黄金文件模式）。
var updateOpenAPISpec = flag.Bool("update-openapi-spec", false,
	"用运行时 OpenAPI 规范重写 docs/api/openapi.json")

// mainVersionRe 匹配 `apps/server/main.go` 里经 ldflags 注入的版本变量声明。
var mainVersionRe = regexp.MustCompile(`(?m)^var version = "([^"]+)"`)

// committedSpecPath 返回提交版规范的绝对路径（handler → internal → server → apps → 仓库根）。
func committedSpecPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "docs", "api", "openapi.json")
}

// serverMainVersion 从 `apps/server/main.go` 读出当前版本号（源码真值）。
func serverMainVersion(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "main.go"))
	if err != nil {
		t.Fatalf("读取 main.go: %v", err)
	}
	m := mainVersionRe.FindStringSubmatch(string(b))
	if m == nil {
		t.Fatal("main.go 未找到 `var version = \"...\"`（解析口径需同步）")
	}
	return m[1]
}

// runtimeSpecWithVersion 用给定版本号构建规范并返回紧凑 JSON 字节。
func runtimeSpecWithVersion(t *testing.T, version string) []byte {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	h := New(st, quietLogger(), t.TempDir(), nil, "", version, false, true)
	t.Cleanup(h.Shutdown)
	b, err := h.openapi.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	return b
}

// canonicalJSON 去掉 JSON 中的非语义空白，用于「语义相等」比对。
func canonicalJSON(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		t.Fatalf("json.Compact: %v", err)
	}
	return buf.Bytes()
}

// TestCommittedOpenAPISpecMatchesRuntime 断言提交版规范与运行时生成的规范语义一致。
func TestCommittedOpenAPISpecMatchesRuntime(t *testing.T) {
	want := runtimeSpecWithVersion(t, serverMainVersion(t))
	path := committedSpecPath(t)

	if *updateOpenAPISpec {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, want, "", "  "); err != nil {
			t.Fatalf("json.Indent: %v", err)
		}
		pretty.WriteByte('\n')
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("创建目录 %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, pretty.Bytes(), 0o644); err != nil {
			t.Fatalf("写入 %s: %v", path, err)
		}
		t.Logf("已重新生成 %s（%d 字节）", path, pretty.Len())
		return
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v\n提示：用 `go test ./internal/handler/ -run %s -update-openapi-spec` 生成",
			path, err, t.Name())
	}
	if !bytes.Equal(canonicalJSON(t, got), canonicalJSON(t, want)) {
		t.Errorf("提交版 OpenAPI 规范与运行时不一致：%s（%d 字节）vs 运行时（%d 字节）\n"+
			"改了 handler / openapi_register_*.go 之后必须重新生成：\n"+
			"  go test ./internal/handler/ -run %s -update-openapi-spec",
			path, len(got), len(want), t.Name())
	}
}

// TestCommittedOpenAPISpecIsDiscoverable 断言文件落在约定位置且可被外部工具解析。
//
// 存在意义：`docs/api/openapi.json` 的价值在于「不跑服务也能读到契约」。若它被误移、
// 被 gitignore、或写成非法 JSON，上面那道门禁可能因路径漂移而报「文件不存在」——
// 本用例把「位置与可解析性」也钉住，并顺带断言 openapi 版本字段存在（外部工具据此分派）。
func TestCommittedOpenAPISpecIsDiscoverable(t *testing.T) {
	b, err := os.ReadFile(committedSpecPath(t))
	if err != nil {
		t.Fatalf("读取提交版规范: %v", err)
	}
	var doc struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Version string `json:"version"`
		} `json:"info"`
		Paths map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("提交版规范不是合法 JSON: %v", err)
	}
	if doc.OpenAPI == "" {
		t.Error("提交版规范缺 `openapi` 字段（外部工具据此分派版本）")
	}
	if doc.Info.Version != serverMainVersion(t) {
		t.Errorf("提交版规范 info.version=%q，main.go 为 %q——未随版本号重新生成",
			doc.Info.Version, serverMainVersion(t))
	}
	if len(doc.Paths) == 0 {
		t.Error("提交版规范 paths 为空")
	}
}
