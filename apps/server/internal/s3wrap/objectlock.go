package s3wrap

// objectlock.go —— Object Lock（WORM 保留）防腐层：桶级配置、对象保留期、法定保留。
//
// 降级口径（厂商支持度差异，ROADMAP §三 #5 / compatibility.md §6.1 E8）：
//   - 桶未启用 Object Lock / 厂商不实现该 API → 读操作返回「空配置」而非错误；
//   - 对象没有保留期 / 未设法定保留 → 返回 nil / "OFF" 而非错误；
//   - 写操作失败（不支持、保留期违规、对象被合规锁定）按错误码原样上抛，
//     由 errors.go 的映射表给出稳定 HTTP 状态与用户文案。

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectLockConfig 是桶 Object Lock 配置的展示型 DTO（默认保留 = 模式 + 天/年二选一）。
type ObjectLockConfig struct {
	Enabled               bool
	DefaultRetentionMode  string // "" | GOVERNANCE | COMPLIANCE
	DefaultRetentionDays  int32  // 与 Years 互斥（S3 侧二选一），零值表示未设置
	DefaultRetentionYears int32
}

// ObjectRetention 是对象级保留期 DTO；nil 表示未设置保留期（降级，不是错误）。
type ObjectRetention struct {
	Mode        string // GOVERNANCE | COMPLIANCE
	RetainUntil time.Time
}

// GetObjectLockConfiguration 读取桶 Object Lock 配置。
// 桶未启用（或厂商返回未实现）→ Enabled=false 且无错误；NoSuchBucket 等真实错误照常上抛。
func (c *Client) GetObjectLockConfiguration(ctx context.Context, bucket string) (*ObjectLockConfig, error) {
	out, err := c.s3.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if HasErrorCode(err, "ObjectLockConfigurationNotFoundError") {
			return &ObjectLockConfig{}, nil
		}
		return nil, err
	}
	cfg := &ObjectLockConfig{}
	if out.ObjectLockConfiguration == nil {
		return cfg, nil
	}
	cfg.Enabled = true
	if rule := out.ObjectLockConfiguration.Rule; rule != nil && rule.DefaultRetention != nil {
		dr := rule.DefaultRetention
		cfg.DefaultRetentionMode = string(dr.Mode)
		if dr.Days != nil {
			cfg.DefaultRetentionDays = *dr.Days
		}
		if dr.Years != nil {
			cfg.DefaultRetentionYears = *dr.Years
		}
	}
	return cfg, nil
}

// PutObjectLockConfiguration 设置桶默认保留策略（模式 + 天/年；桶须在创建时启用 Object Lock，
// 否则 S3 返回 NotImplemented / 配置不存在类错误，由上层映射为 400）。
func (c *Client) PutObjectLockConfiguration(ctx context.Context, bucket string, cfg ObjectLockConfig) error {
	dr := &types.DefaultRetention{Mode: types.ObjectLockRetentionMode(cfg.DefaultRetentionMode)}
	if cfg.DefaultRetentionDays > 0 {
		dr.Days = aws.Int32(cfg.DefaultRetentionDays)
	}
	if cfg.DefaultRetentionYears > 0 {
		dr.Years = aws.Int32(cfg.DefaultRetentionYears)
	}
	_, err := c.s3.PutObjectLockConfiguration(ctx, &s3.PutObjectLockConfigurationInput{
		Bucket: aws.String(bucket),
		ObjectLockConfiguration: &types.ObjectLockConfiguration{
			ObjectLockEnabled: types.ObjectLockEnabledEnabled,
			Rule:              &types.ObjectLockRule{DefaultRetention: dr},
		},
	})
	return err
}

// GetObjectRetention 读取对象保留期；无保留期 / 未启用 Object Lock → (nil, nil)。
//
// 降级码包含 RustFS 实测的 InvalidRequest（"Bucket is missing ObjectLockConfiguration"——
// 非锁定桶上读保留期返回该码；对象详情面板逐对象读，不降级会让普通桶全线 400）与
// AWS 口径的 ObjectLockConfigurationNotFoundError。NoSuchBucket / NoSuchKey 等
// 「对象真不存在」的错误仍原样上抛（404 语义不得吞）。
func (c *Client) GetObjectRetention(ctx context.Context, bucket, key, versionID string) (*ObjectRetention, error) {
	in := &s3.GetObjectRetentionInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	if versionID != "" {
		in.VersionId = aws.String(versionID)
	}
	out, err := c.s3.GetObjectRetention(ctx, in)
	if err != nil {
		if HasErrorCode(err, "ObjectLockConfigurationNotFoundError", "InvalidRequest") {
			return nil, nil
		}
		return nil, err
	}
	// 归一化：RustFS 对「锁桶上的无保留对象」返回**空 ObjectRetention 元素**（非 nil、
	// 字段全空），与「响应体为空」一样都表示未设置——零值整体判空，避免 configured=true 却无字段。
	ret := ObjectRetention{}
	if out.Retention != nil {
		ret = ObjectRetention{
			Mode:        string(out.Retention.Mode),
			RetainUntil: aws.ToTime(out.Retention.RetainUntilDate).UTC(),
		}
	}
	if ret == (ObjectRetention{}) {
		return nil, nil
	}
	return &ret, nil
}

// PutObjectRetention 设置对象保留期（mode ∈ GOVERNANCE|COMPLIANCE，until 为 UTC 到期时刻）。
func (c *Client) PutObjectRetention(ctx context.Context, bucket, key, versionID, mode string, until time.Time) error {
	in := &s3.PutObjectRetentionInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Retention: &types.ObjectLockRetention{
			Mode:            types.ObjectLockRetentionMode(mode),
			RetainUntilDate: &until,
		},
	}
	if versionID != "" {
		in.VersionId = aws.String(versionID)
	}
	_, err := c.s3.PutObjectRetention(ctx, in)
	return err
}

// GetObjectLegalHold 读取法定保留状态："ON" | "OFF"；未设置 / 未启用 → "OFF"（形状稳定）。
func (c *Client) GetObjectLegalHold(ctx context.Context, bucket, key, versionID string) (string, error) {
	in := &s3.GetObjectLegalHoldInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	if versionID != "" {
		in.VersionId = aws.String(versionID)
	}
	out, err := c.s3.GetObjectLegalHold(ctx, in)
	if err != nil {
		// InvalidRequest：RustFS 对非锁定桶的同一降级码（同 GetObjectRetention）。
		if HasErrorCode(err, "LegalHoldNotFoundError", "InvalidRequest") {
			return "OFF", nil
		}
		return "", err
	}
	status := "OFF"
	if out.LegalHold != nil && out.LegalHold.Status != "" {
		status = string(out.LegalHold.Status)
	}
	return status, nil
}

// PutObjectLegalHold 设置法定保留（status ∈ ON|OFF）。
func (c *Client) PutObjectLegalHold(ctx context.Context, bucket, key, versionID, status string) error {
	in := &s3.PutObjectLegalHoldInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		LegalHold: &types.ObjectLockLegalHold{
			Status: types.ObjectLockLegalHoldStatus(status),
		},
	}
	if versionID != "" {
		in.VersionId = aws.String(versionID)
	}
	_, err := c.s3.PutObjectLegalHold(ctx, in)
	return err
}
