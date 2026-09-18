package model

import "time"

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
// （真实密钥或脱敏占位都算已设置；空串才算未设置）。
func (a *Account) View() *AccountView {
	if a == nil {
		return nil
	}
	return &AccountView{
		ID:             a.ID,
		Name:           a.Name,
		Endpoint:       a.Endpoint,
		PublicEndpoint: a.PublicEndpoint,
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

// BucketOrDefault 返回默认桶；为空时返回空串（由调用方决定如何处理）。
func (a *Account) BucketOrDefault() string {
	if a.Bucket == "" {
		return ""
	}
	return a.Bucket
}
