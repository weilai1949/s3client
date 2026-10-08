package handler

// openapi_examples_test.go —— applyExamples 的注册错误分支。
//
// 正常路径由 New() 构造 handler 时覆盖；这里显式钉住「示例 key 拼错 / 指向不存在的 operation」
// 必须 panic（静态注册错误要在构造期立刻暴露），避免示例被静默丢弃。
// 本测试不并行：它会临时替换包级 apiExamples，必须在并行测试启动前恢复。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/weilai1949/s3client/apps/server/internal/openapi"
)

// TestApplyExamplesPanicsOnBadConfig 断言错误配置 panic 且带可定位信息。
func TestApplyExamplesPanicsOnBadConfig(t *testing.T) {
	original := apiExamples
	t.Cleanup(func() { apiExamples = original })

	cases := []struct {
		name     string
		examples map[string]openapi.OpExample
		wantSub  string
	}{
		{
			name:     "key 不是 METHOD /path",
			examples: map[string]openapi.OpExample{"nonsense": {}},
			wantSub:  "METHOD /path",
		},
		{
			name: "指向未注册的 operation",
			examples: map[string]openapi.OpExample{
				"GET /api/does-not-exist": {Responses: map[string]json.RawMessage{"200": ex(`{}`)}},
			},
			wantSub: "未注册",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiExamples = tc.examples
			defer func() {
				got := recover()
				if got == nil {
					t.Fatal("applyExamples 未 panic")
				}
				msg, ok := got.(string)
				if !ok {
					t.Fatalf("panic 值不是字符串：%v", got)
				}
				if !strings.Contains(msg, tc.wantSub) {
					t.Errorf("panic = %q，期望包含 %q", msg, tc.wantSub)
				}
			}()
			applyExamples(openapi.New("s3client API", "vtest"))
		})
	}
}
