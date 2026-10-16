package config

// scopes_test.go —— S3C_TOKEN_SCOPES（ROADMAP §三 #13）配置面测试。
//
// 断言走**导出面**（FromEnv / Validate / ScopeFor / TokenScope），不触碰解析私有函数：
// 非法配置必须在 Validate 阶段 fail-closed（启动失败），合法配置必须可查询到作用域。

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// scopeTestToken 是够长（>= MinTokenLength）的测试 token，避免撞上短 token 闸。
const (
	scopeTokA = "tok-aaaaaaaaaaaaaaaa"
	scopeTokB = "tok-bbbbbbbbbbbbbbbb"
)

// scopeFromEnv 在隔离环境变量后构建配置（S3C_TOKEN_TENV 只影响本测试）。
func scopeFromEnv(t *testing.T, token, raw string) Config {
	t.Helper()
	t.Setenv("S3C_TOKEN", token)
	t.Setenv("S3C_TOKEN_SCOPES", raw)
	t.Setenv("S3C_ADDR", "127.0.0.1:5000")
	t.Setenv("S3C_STORE_DRIVER", "json")
	t.Setenv("S3C_STORE_KEY", "")
	// 明文闸与本测试无关，显式 opt-in 隔离宿主环境（如 CI 注入的 S3C_*）。
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	return FromEnv()
}

// TestTokenScopesValidParsesAllFields 断言合法 JSON 的全部字段被解析，
// 且未给出的 expiresAt 保持零值（= 不过期）。
func TestTokenScopesValidParsesAllFields(t *testing.T) {
	raw := `{"` + scopeTokA + `":{"readonly":true,"prefixes":["bucket-a/","bucket-b/logs/"],"accounts":["acc-1"],"expiresAt":"2027-01-01T00:00:00Z"},` +
		`"` + scopeTokB + `":{}}`
	cfg := scopeFromEnv(t, scopeTokA+","+scopeTokB, raw)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	a, ok := cfg.ScopeFor(scopeTokA)
	if !ok {
		t.Fatal("ScopeFor(scopeTokA) 未命中")
	}
	if !a.Readonly {
		t.Error("Readonly 应为 true")
	}
	if len(a.Prefixes) != 2 || a.Prefixes[0] != "bucket-a/" || a.Prefixes[1] != "bucket-b/logs/" {
		t.Errorf("Prefixes = %v, want [bucket-a/ bucket-b/logs/]", a.Prefixes)
	}
	if len(a.Accounts) != 1 || a.Accounts[0] != "acc-1" {
		t.Errorf("Accounts = %v, want [acc-1]", a.Accounts)
	}
	want := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !a.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", a.ExpiresAt, want)
	}
	b, ok := cfg.ScopeFor(scopeTokB)
	if !ok {
		t.Fatal("ScopeFor(scopeTokB) 未命中")
	}
	if b.Readonly || len(b.Prefixes) != 0 || len(b.Accounts) != 0 || !b.ExpiresAt.IsZero() {
		t.Errorf("空对象作用域应全为缺省（不限制），got %+v", b)
	}
	if _, ok := cfg.ScopeFor("未登记的-token"); ok {
		t.Error("ScopeFor 对未登记 token 应返回 ok=false")
	}
}

// TestTokenScopesUnsetIsNilAndInert 断言未设置 S3C_TOKEN_SCOPES 时不产生任何作用域，
// 且 Validate 不受影响（向后兼容：所有 token 保持全权）。
func TestTokenScopesUnsetIsNilAndInert(t *testing.T) {
	for _, raw := range []string{"", "   ", "{}"} {
		cfg := scopeFromEnv(t, scopeTokA, raw)
		if err := cfg.Validate(); err != nil {
			t.Fatalf("raw=%q Validate() = %v, want nil", raw, err)
		}
		if len(cfg.TokenScopes) != 0 {
			t.Errorf("raw=%q TokenScopes = %v, want empty", raw, cfg.TokenScopes)
		}
		if _, ok := cfg.ScopeFor(scopeTokA); ok {
			t.Errorf("raw=%q 不应有任何 token 作用域", raw)
		}
	}
}

// TestTokenScopesRejectUnregisteredToken 断言登记了不在 S3C_TOKEN 中的 token 即拒绝启动，
// 且错误文案**不得回显 token 明文**（用户输入即凭证，不能落日志）。
func TestTokenScopesRejectUnregisteredToken(t *testing.T) {
	cfg := scopeFromEnv(t, scopeTokA, `{"`+scopeTokB+`":{"readonly":true}}`)
	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidTokenScopes) {
		t.Fatalf("Validate() = %v, want wraps ErrInvalidTokenScopes", err)
	}
	if strings.Contains(err.Error(), scopeTokB) {
		t.Errorf("错误文案回显了 token 明文：%v", err)
	}
}

// TestTokenScopesRejectInvalidElements 断言 prefixes / accounts 元素非法即拒绝启动。
func TestTokenScopesRejectInvalidElements(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"empty prefix", `{"` + scopeTokA + `":{"prefixes":[""]}}`},
		{"blank prefix", `{"` + scopeTokA + `":{"prefixes":["  "]}}`},
		{"leading slash", `{"` + scopeTokA + `":{"prefixes":["/key"]}}`},
		{"empty account", `{"` + scopeTokA + `":{"accounts":[""]}}`},
		{"blank account", `{"` + scopeTokA + `":{"accounts":[" "]}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := scopeFromEnv(t, scopeTokA, c.raw).Validate()
			if !errors.Is(err, ErrInvalidTokenScopes) {
				t.Fatalf("Validate() = %v, want wraps ErrInvalidTokenScopes", err)
			}
		})
	}
}

// TestTokenScopesRejectBadJSON 断言 JSON 语法错误 / 未知字段 / 末尾多余数据 / 坏时间
// 一律 fail-closed（沿用 envErr 机制，在 Validate 首查上抛）。
func TestTokenScopesRejectBadJSON(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"not json", `{not json`},
		{"wrong type", `{"` + scopeTokA + `":[]}`},
		{"unknown field", `{"` + scopeTokA + `":{"readonly":true,"bogus":1}}`},
		{"unknown nested field", `{"` + scopeTokA + `":{"prefixes":["b/"],"extra":true}}`},
		{"trailing data", `{"` + scopeTokA + `":{}} {"x":1}`},
		{"bad time", `{"` + scopeTokA + `":{"expiresAt":"not-a-time"}}`},
		{"bad time offset", `{"` + scopeTokA + `":{"expiresAt":"2027-13-01T00:00:00Z"}}`},
		{"empty token key", `{"":{"readonly":true}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := scopeFromEnv(t, scopeTokA, c.raw).Validate()
			if !errors.Is(err, ErrInvalidTokenScopes) {
				t.Fatalf("Validate() = %v, want wraps ErrInvalidTokenScopes", err)
			}
		})
	}
}

// TestTokenScopesExpiresAtIsRFC3339 断言带小数的 RFC3339 时间也可解析（Go time.Parse 允许），
// 且非 UTC 偏移被正确换算。
func TestTokenScopesExpiresAtIsRFC3339(t *testing.T) {
	raw := `{"` + scopeTokA + `":{"expiresAt":"2027-01-01T08:00:00.500+08:00"}}`
	cfg := scopeFromEnv(t, scopeTokA, raw)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	s, _ := cfg.ScopeFor(scopeTokA)
	want := time.Date(2027, 1, 1, 0, 0, 0, 500000000, time.UTC)
	if !s.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", s.ExpiresAt, want)
	}
}

// TestTokenScopesRequiresTokenWhenScopesSet 断言只设 S3C_TOKEN_SCOPES 而不设 S3C_TOKEN
// 时拒绝启动：没有任何 token 能匹配这张表，静默接受等于把作用域配置丢掉。
func TestTokenScopesRequiresTokenWhenScopesSet(t *testing.T) {
	err := scopeFromEnv(t, "", `{"`+scopeTokA+`":{"readonly":true}}`).Validate()
	if !errors.Is(err, ErrInvalidTokenScopes) {
		t.Fatalf("Validate() = %v, want wraps ErrInvalidTokenScopes", err)
	}
}

// TestTokenScopesTokenWithSpacesMustMatchTrimmedList 断言 S3C_TOKEN 列表元素两端的空白
// 被裁剪后再比对（与既有 splitTokens 口径一致），未登记仍拒绝。
func TestTokenScopesTokenWithSpacesMustMatchTrimmedList(t *testing.T) {
	cfg := scopeFromEnv(t, " "+scopeTokA+" , "+scopeTokB, `{"`+scopeTokA+`":{"readonly":true}}`)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	if _, ok := cfg.ScopeFor(scopeTokA); !ok {
		t.Fatal("裁剪后应命中 scopeTokA")
	}
}

// TestTokenScopesRejectPrefixesWithoutAccounts：`prefixes` 只约束**桶/键**、不含**账号维度**。
// 只声明 prefixes 而不声明 accounts 时，该 token 可对**任意账号**执行界内桶/键的操作，并以
// 该账号的凭据签发预签名 URL（2026-10-16 实测：`POST /api/accounts/{victim}/presign`
// 返回 200，且 URL 的 `X-Amz-Credential` 是受害账号的 AccessKey）——运维写 prefixes 的本意
// 是「这个 token 只能碰这个桶前缀」，实际却是「任意账号的这个桶前缀」。
//
// 账号维度在 prefixes-only 下**不可判定**（攻击者只需挑一个恰好有该桶名的账号），运行时无法
// 补救，因此在配置期 fail-closed：拒绝启动，并在文案里给出修法。这条规则同时让运行时保持
// 单一判据——accounts 非空后，`{id}` 闸与账号引用闸本来就在生效。
func TestTokenScopesRejectPrefixesWithoutAccounts(t *testing.T) {
	cases := []struct {
		name, raw string
	}{
		{"prefixes-only 整桶", `{"` + scopeTokA + `":{"prefixes":["bucket-a"]}}`},
		{"prefixes-only 键前缀", `{"` + scopeTokA + `":{"prefixes":["bucket-a/logs/"]}}`},
		{"readonly + prefixes-only", `{"` + scopeTokA + `":{"readonly":true,"prefixes":["bucket-a/logs/"]}}`},
		{"expiresAt + prefixes-only", `{"` + scopeTokA + `":{"expiresAt":"2027-01-01T00:00:00Z","prefixes":["bucket-a"]}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := scopeFromEnv(t, scopeTokA, c.raw).Validate()
			if !errors.Is(err, ErrInvalidTokenScopes) {
				t.Fatalf("Validate() = %v, want wraps ErrInvalidTokenScopes", err)
			}
			// 报错必须点出修法（同时声明 accounts），否则运维只能靠猜。
			if !strings.Contains(err.Error(), "accounts") {
				t.Errorf("错误文案未提示补 accounts：%v", err)
			}
			// 与其它作用域错误同口径：不得回显 token 明文。
			if strings.Contains(err.Error(), scopeTokA) {
				t.Errorf("错误文案回显了 token 明文：%v", err)
			}
		})
	}
}

// TestTokenScopesWithoutPrefixesStillValid：新规则**只**针对 prefixes-only 这一条轴。
// `accounts`-only 是**有界**作用域（爆炸半径 = 该账号的桶），安全，不要求反配 prefixes；
// `readonly` / `expiresAt` 是能力与时间限制而非资源维度；空对象 = 不限制——四者均须保持可启动。
func TestTokenScopesWithoutPrefixesStillValid(t *testing.T) {
	cases := []struct {
		name, raw string
	}{
		{"accounts-only", `{"` + scopeTokA + `":{"accounts":["acc-1"]}}`},
		{"readonly-only", `{"` + scopeTokA + `":{"readonly":true}}`},
		{"expiresAt-only", `{"` + scopeTokA + `":{"expiresAt":"2027-01-01T00:00:00Z"}}`},
		{"empty scope", `{"` + scopeTokA + `":{}}`},
		{"prefixes + accounts", `{"` + scopeTokA + `":{"prefixes":["bucket-a/logs/"],"accounts":["acc-1"]}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := scopeFromEnv(t, scopeTokA, c.raw).Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
		})
	}
}
