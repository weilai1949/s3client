package model

import (
	"net/url"
	"strings"
	"time"
)

// Account 描述一个 S3 兼容存储的账号配置。
// 支持 AWS、MinIO、阿里云 OSS、腾讯云 COS、华为云 OBS 等任意 S3 兼容服务。
type Account struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Endpoint       string    `json:"endpoint"`       // 服务端访问地址，例如 http://minio:9000
	PublicEndpoint string    `json:"publicEndpoint"` // 浏览器直传/下载用的公网地址，例如 http://127.0.0.1:9000；留空则与 endpoint 相同
	Region         string    `json:"region"`         // 例如 us-east-1
	AccessKey      string    `json:"accessKey"`
	SecretKey      string    `json:"secretKey"`
	Bucket         string    `json:"bucket"`    // 默认桶
	PathStyle      bool      `json:"pathStyle"` // 是否强制 path-style（MinIO 等第三方通常需要）
	UseSSL         bool      `json:"useSSL"`    // 是否启用 TLS
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// MaskedSecret 是脱敏后用于对外展示的 SecretKey 占位值。
const MaskedSecret = "******"

// IsMaskedSecret 判断 SecretKey 是否已是脱敏占位（避免脱敏值回写）。
func IsMaskedSecret(s string) bool { return s == MaskedSecret }

// Sanitized 返回一份脱敏后的副本（隐藏 SecretKey），供存储层对 List/Create/Update
// 的返回值做防御性脱敏；不再直接序列化到 HTTP 响应（见 View）。
func (a *Account) Sanitized() *Account {
	if a == nil {
		return nil
	}
	c := *a
	if c.SecretKey != "" {
		c.SecretKey = MaskedSecret
	}
	return &c
}

// RedactEndpointUserinfo 抹掉端点 URL 里的 userinfo（`http://key:secret@host:9000` →
// `http://host:9000`），供日志与对外视图使用。
//
// 端点内嵌凭据是**用户误填**：SigV4 凭据来自账号的 accessKey / secretKey 独立字段，userinfo
// 没有任何正当用途，只会经日志、400 响应体、`AccountView` 与预签名 URL 泄漏出去
// （2026-10-15 渗透测试 P-3 / KNOWN_ISSUES #84）。
//
// 这是脱敏的**唯一实现**（`s3wrap.NormalizeEndpoint` 侧负责在建 client / 预签名之前剥离），
// 仅用标准库 net/url：无 userinfo 或解析失败 → 原样返回，不猜测也不重写。裸端点（无 scheme，
// 如 `key:secret@minio:9000`）补 `//` 按协议相对形式解析——否则 url.Parse 会把 `key` 当成
// scheme、把 userinfo 藏进 Opaque，漏掉这条通道。
func RedactEndpointUserinfo(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	schemeRelative := !strings.Contains(trimmed, "://")
	target := trimmed
	if schemeRelative {
		target = "//" + target
	}
	u, err := url.Parse(target)
	if err != nil || u.User == nil {
		return endpoint
	}
	u.User = nil
	out := u.String()
	if schemeRelative {
		// 保持调用方原本的「无 scheme」形态，只去掉 userinfo。
		out = strings.TrimPrefix(out, "//")
	}
	return out
}

// AccountView 是账号的对外展示视图（HTTP 响应 DTO）：不包含 secretKey，
// 用 secretSet 布尔表示是否已设置密钥，避免向客户端暴露任何密钥占位。
type AccountView struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Endpoint       string    `json:"endpoint"`
	PublicEndpoint string    `json:"publicEndpoint"`
	Region         string    `json:"region"`
	AccessKey      string    `json:"accessKey"`
	SecretSet      bool      `json:"secretSet"`
	Bucket         string    `json:"bucket"`
	PathStyle      bool      `json:"pathStyle"`
	UseSSL         bool      `json:"useSSL"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// View 返回账号的对外视图。secretSet 表示当前是否持有密钥
// （真实密钥或脱敏占位都算已设置；空串才算未设置）。Endpoint / PublicEndpoint 回传
// **脱敏值**（P-3）：契约上线前落盘的历史数据可能带 userinfo，任意已认证 token 都不该读到它。
func (a *Account) View() *AccountView {
	if a == nil {
		return nil
	}
	return &AccountView{
		ID:             a.ID,
		Name:           a.Name,
		Endpoint:       RedactEndpointUserinfo(a.Endpoint),
		PublicEndpoint: RedactEndpointUserinfo(a.PublicEndpoint),
		Region:         a.Region,
		AccessKey:      a.AccessKey,
		SecretSet:      a.SecretKey != "",
		Bucket:         a.Bucket,
		PathStyle:      a.PathStyle,
		UseSSL:         a.UseSSL,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}
