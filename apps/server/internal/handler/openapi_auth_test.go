package handler

// openapi_auth_test.go —— 契约的**鉴权与分组表达**门禁（WS1）。
//
// 背景：docs/api.md 规定除 `/api/health` 与 `/api/metrics` 外，所有 `/api/*` 需携带
// `Authorization: Bearer <token>`（真值来源是 middleware.go 的 withAuth）。但此前
// 机器可读契约既没有文档级 `security`，也没有任何 operation 声明 `security` /
// `security: []`，顶层 `tags` 也缺失 —— 代码生成器与 AI 代理从契约里读不出鉴权要求，
// 只能去翻 Go 源码。
//
// 断言对象是**运行时 HTTP 端点吐出的规范**（外部可见行为），不触碰 openapi 包的私有函数。

import (
	"sort"
	"strings"
	"testing"
)

// bearerExemptPaths 是 withAuth 显式豁免鉴权的端点。
// 真值与 middleware.go 的 `p == "/api/health" || p == "/api/metrics"` 一致；名单变更须同步此处。
var bearerExemptPaths = map[string]bool{
	"/api/health":  true,
	"/api/metrics": true,
}

// declaredSecurity 提取某层 `security` 的 scheme 名集合。
// present=false 表示该层**没有** security 字段（≠ 显式 `security: []`），
// 用于区分「继承文档级默认」与「显式豁免」两种语义。
func declaredSecurity(v any) (schemes map[string]bool, present bool) {
	reqs, ok := v.([]any)
	if !ok {
		return nil, false
	}
	schemes = map[string]bool{}
	for _, req := range reqs {
		obj, ok := req.(map[string]any)
		if !ok {
			continue
		}
		for name := range obj {
			schemes[name] = true
		}
	}
	return schemes, true
}

// effectiveSecurity 解析 operation 的实际鉴权要求：operation 级优先，否则继承文档级。
func effectiveSecurity(t *testing.T, doc, op map[string]any, id string) map[string]bool {
	t.Helper()
	if schemes, present := declaredSecurity(op["security"]); present {
		return schemes
	}
	schemes, present := declaredSecurity(doc["security"])
	if !present {
		t.Fatalf("%s：operation 级与文档级均未声明 security，鉴权要求对机器不可读", id)
	}
	return schemes
}

// TestOpenAPISpecDeclaresBearerAuthAndGuardsOperations 断言 bearerAuth 方案存在，
// 且所有**非豁免** operation 的有效鉴权要求包含 bearerAuth。
func TestOpenAPISpecDeclaresBearerAuthAndGuardsOperations(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)

	comps, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("components 类型 = %T, want object", doc["components"])
	}
	schemes, ok := comps["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatalf("components.securitySchemes 类型 = %T, want object", comps["securitySchemes"])
	}
	bearer, ok := schemes["bearerAuth"].(map[string]any)
	if !ok {
		t.Fatalf("components.securitySchemes.bearerAuth 缺失或类型 = %T", schemes["bearerAuth"])
	}
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Errorf("bearerAuth = %v, want type=http scheme=bearer", bearer)
	}

	global, present := declaredSecurity(doc["security"])
	if !present {
		t.Fatal("文档级 security 缺失：非豁免 operation 未从契约表达鉴权要求")
	}
	if !global["bearerAuth"] {
		t.Errorf("文档级 security 未要求 bearerAuth：%v", global)
	}

	paths := openAPIPaths(t, doc)
	for p, methods := range paths {
		if bearerExemptPaths[p] {
			continue
		}
		for m, op := range methods {
			id := strings.ToUpper(m) + " " + p
			if got := effectiveSecurity(t, doc, op, id); !got["bearerAuth"] {
				t.Errorf("%s 未要求 bearerAuth（有效 security = %v）", id, got)
			}
		}
	}
}

// TestOpenAPISpecExemptOperationsOptOutOfSecurity 断言豁免端点在 operation 上**显式**
// 声明 `security: []`；同时反向拦截「非豁免端点被意外豁免」。
func TestOpenAPISpecExemptOperationsOptOutOfSecurity(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	for p := range bearerExemptPaths {
		if _, ok := paths[p]; !ok {
			t.Errorf("豁免端点 %s 不在契约中（豁免名单与 routes.go 漂移）", p)
		}
	}

	for p, methods := range paths {
		exempt := bearerExemptPaths[p]
		for m, op := range methods {
			id := strings.ToUpper(m) + " " + p
			schemes, present := declaredSecurity(op["security"])
			switch {
			case exempt && !present:
				t.Errorf("%s 是鉴权豁免端点，必须显式声明 `security: []`（缺字段会继承文档级 bearerAuth）", id)
			case exempt && len(schemes) != 0:
				t.Errorf("%s 是鉴权豁免端点，security 必须为空数组，got %v", id, schemes)
			case !exempt && present && len(schemes) == 0:
				t.Errorf("%s 非豁免端点却显式豁免了鉴权（security: []）", id)
			}
		}
	}
}

// TestOpenAPISpecDeclaresTagsAndAssignsEveryOperation 断言顶层 tags 非空、有说明、
// 无重复无孤儿，且每个 operation 至少归入一个**已声明**的 tag。
func TestOpenAPISpecDeclaresTagsAndAssignsEveryOperation(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)

	rawTags, ok := doc["tags"].([]any)
	if !ok || len(rawTags) == 0 {
		t.Fatalf("顶层 tags 缺失或为空（%T）：分组信息对代码生成器不可读", doc["tags"])
	}
	declared := map[string]bool{}
	for i, raw := range rawTags {
		tag, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("tags[%d] 类型 = %T, want object", i, raw)
		}
		name, _ := tag["name"].(string)
		if name == "" {
			t.Fatalf("tags[%d] 缺 name", i)
		}
		if declared[name] {
			t.Errorf("顶层 tags 重复声明 %q", name)
		}
		declared[name] = true
		if desc, _ := tag["description"].(string); desc == "" {
			t.Errorf("tag %q 缺 description（应指向 docs/api.md 的对应章节）", name)
		}
	}

	used := map[string]bool{}
	paths := openAPIPaths(t, doc)
	for p, methods := range paths {
		for m, op := range methods {
			id := strings.ToUpper(m) + " " + p
			names, _ := op["tags"].([]any)
			if len(names) == 0 {
				t.Errorf("%s 未归入任何 tag", id)
				continue
			}
			for _, raw := range names {
				name, _ := raw.(string)
				if !declared[name] {
					t.Errorf("%s 引用了未在顶层 tags 声明的分组 %q", id, name)
				}
				used[name] = true
			}
		}
	}

	var orphan []string
	for name := range declared {
		if !used[name] {
			orphan = append(orphan, name)
		}
	}
	sort.Strings(orphan)
	if len(orphan) > 0 {
		t.Errorf("顶层 tags 声明了无人引用的分组（死声明）：%v", orphan)
	}
}
