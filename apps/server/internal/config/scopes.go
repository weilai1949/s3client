package config

// scopes.go —— S3C_TOKEN_SCOPES：按 token 声明最小权限（ROADMAP §三 #13）。
//
// 设计口径（用户拍板）：
//   - 新增独立 env `S3C_TOKEN_SCOPES`（JSON 映射 token → scopes），`S3C_TOKEN` 语义**不变**；
//   - 未在该 JSON 中登记的 token 仍是全权（向后兼容，零配置行为与历史完全一致）；
//   - 字段全部可选、缺省即不限制：`readonly` / `prefixes` / `accounts` / `expiresAt`；
//   - 严格解析（DisallowUnknownFields）；登记的 token 必须在 `S3C_TOKEN` 列表中、
//     prefixes / accounts 元素非空、**声明 prefixes 时必须同时声明 accounts**、
//     expiresAt 可解析为 RFC3339，否则拒绝启动（fail-closed）。
//
// 「prefixes 必须配 accounts」的由来：prefixes 只约束桶/键、不含账号维度，缺省 accounts
// 会让「只能碰某个桶前缀」的 token 实际变成「**任意账号的**那个桶前缀」（可对任意 {id}
// 执行界内操作、并以他人凭据签发预签名 URL）。账号维度在这种配置下不可判定，运行时无法
// 补救，因此把「账号说清楚」变成配置期的硬约束。accounts-only 是**有界**作用域（爆炸半径
// = 该账号），安全，不要求反配 prefixes。
//
// 安全：错误文案**不回显 token 明文**（凭证不落日志）。

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidTokenScopes 表示 S3C_TOKEN_SCOPES 无法解析或不满足约束（启动失败）。
var ErrInvalidTokenScopes = errors.New("invalid S3C_TOKEN_SCOPES")

// TokenScope 是单个 token 的最小权限声明。零值 = 不限制（等价于未登记，全权）。
type TokenScope struct {
	Readonly  bool      // true = 仅放行 GET/HEAD（含预签名 POST 在内的写请求一律 403）
	Prefixes  []string  // "<bucket>" 或 "<bucket>/<key前缀>"；空 = 不限桶/键
	Accounts  []string  // 允许的账号 id（路径 {id}）；空 = 不限账号
	ExpiresAt time.Time // 零值 = 永不过期；过期后 401
}

// tokenScopeJSON 是 JSON 线格式。字段名与文档一致；DisallowUnknownFields 保证拼错即拒绝
// （静默忽略会让人以为作用域已生效，实际是全权）。
type tokenScopeJSON struct {
	Readonly  bool     `json:"readonly"`
	Prefixes  []string `json:"prefixes"`
	Accounts  []string `json:"accounts"`
	ExpiresAt string   `json:"expiresAt"`
}

// parseTokenScopes 严格解析 S3C_TOKEN_SCOPES；空串 = 未配置（返回 nil）。
func parseTokenScopes(raw string) (map[string]TokenScope, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var wire map[string]tokenScopeJSON
	if err := dec.Decode(&wire); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTokenScopes, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: 末尾存在多余数据", ErrInvalidTokenScopes)
	}
	if len(wire) == 0 {
		return nil, nil
	}
	out := make(map[string]TokenScope, len(wire))
	for tok, w := range wire {
		if strings.TrimSpace(tok) == "" {
			return nil, fmt.Errorf("%w: token 键不得为空", ErrInvalidTokenScopes)
		}
		scope := TokenScope{Readonly: w.Readonly, Prefixes: w.Prefixes, Accounts: w.Accounts}
		if w.ExpiresAt != "" {
			ts, err := time.Parse(time.RFC3339, w.ExpiresAt)
			if err != nil {
				return nil, fmt.Errorf("%w: expiresAt=%q 不是合法的 RFC3339 时间", ErrInvalidTokenScopes, w.ExpiresAt)
			}
			scope.ExpiresAt = ts
		}
		out[tok] = scope
	}
	return out, nil
}

// validateTokenScopes 校验作用域表与 S3C_TOKEN 的交叉约束（fail-closed）。
// 错误文案不含 token 明文：失败的 token 是凭证，不得写入启动日志。
func validateTokenScopes(token string, scopes map[string]TokenScope) error {
	if len(scopes) == 0 {
		return nil
	}
	registered := map[string]bool{}
	for _, t := range strings.Split(token, ",") {
		if t = strings.TrimSpace(t); t != "" {
			registered[t] = true
		}
	}
	for tok, scope := range scopes {
		if !registered[tok] {
			return fmt.Errorf("%w: 存在未在 S3C_TOKEN 中登记的 token（明文已隐去）", ErrInvalidTokenScopes)
		}
		for _, p := range scope.Prefixes {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" || strings.HasPrefix(trimmed, "/") {
				return fmt.Errorf("%w: prefixes 元素必须形如 <bucket> 或 <bucket>/<key前缀>，不得为空或以 / 开头", ErrInvalidTokenScopes)
			}
		}
		for _, a := range scope.Accounts {
			if strings.TrimSpace(a) == "" {
				return fmt.Errorf("%w: accounts 元素不得为空", ErrInvalidTokenScopes)
			}
		}
		// prefixes 只约束**桶/键**，不含**账号维度**：未声明 accounts 时，该 token 可对
		// **任意账号**执行界内桶/键的操作，并以该账号的凭据签发预签名 URL（2026-10-16
		// 实测：prefixes-only token 调 `POST /api/accounts/{victim}/presign` 返回 200，
		// URL 的 X-Amz-Credential 是受害账号的 AccessKey）。账号维度在 prefixes-only 下
		// **不可判定**——攻击者只需挑一个恰好有该桶名的账号，运行时无法补救，故在此
		// fail-closed；accounts 非空后，运行时的 {id} 闸与账号引用闸本来就在生效，
		// 一条配置规则即可覆盖整类问题，运行时判据保持单一来源。
		if len(scope.Prefixes) > 0 && len(scope.Accounts) == 0 {
			return fmt.Errorf("%w: 声明 prefixes 的 token 必须同时声明 accounts"+
				"（prefixes 只约束桶/键、不含账号维度：缺省 accounts 会让该 token 对任意账号"+
				"执行界内操作并以他人凭据签发预签名 URL）；确需跨账号请显式列出账号 id",
				ErrInvalidTokenScopes)
		}
	}
	return nil
}

// ScopeFor 返回 token 对应的作用域；ok=false 表示该 token 未登记 = 全权（向后兼容）。
func (c Config) ScopeFor(token string) (TokenScope, bool) {
	scope, ok := c.TokenScopes[token]
	return scope, ok
}
