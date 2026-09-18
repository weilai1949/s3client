package s3wrap

import (
	"errors"

	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// PresignPut 生成 v4 签名的上传 URL（单文件 PUT，≤5GB）。
// errInvalidExpiry 预签名过期时长必须为正。
var errInvalidExpiry = errors.New("presign: expiry must be positive")

func (c *Client) PresignPut(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		return "", errInvalidExpiry
	}
	res, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(o *s3.PresignOptions) { o.Expires = expires })
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

// PresignPost 生成 v4 签名的 POST 表单（multipart/form-data），用于浏览器大文件/表单直传。
func (c *Client) PresignPost(ctx context.Context, bucket, key string, expires time.Duration) (*PresignPostResult, error) {
	if expires <= 0 {
		return nil, errInvalidExpiry
	}
	res, err := c.presign.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(o *s3.PresignPostOptions) { o.Expires = expires })
	if err != nil {
		return nil, err
	}
	fields := make(map[string]string, len(res.Values))
	for k, v := range res.Values {
		fields[k] = v
	}
	return &PresignPostResult{URL: res.URL, Fields: fields}, nil
}

// PresignGetVersion 生成 v4 签名的、指向指定版本（versionId；空=当前）的下载 URL。
func (c *Client) PresignGetVersion(ctx context.Context, bucket, key, versionID string, expires time.Duration) (string, error) {
	if expires <= 0 {
		return "", errInvalidExpiry
	}
	in := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		in.VersionId = aws.String(versionID)
	}
	res, err := c.presign.PresignGetObject(ctx, in, func(o *s3.PresignOptions) { o.Expires = expires })
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

// PresignUploadPart 预签名单个分段的 PUT URL（浏览器直传每段，无需经过本服务）。
func (c *Client) PresignUploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int32, expires time.Duration) (string, error) {
	if expires <= 0 {
		return "", errInvalidExpiry
	}
	res, err := c.presign.PresignUploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		PartNumber: aws.Int32(partNumber),
		UploadId:   aws.String(uploadID),
	}, func(o *s3.PresignOptions) { o.Expires = expires })
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

// PublicURL 构造对象的公开访问 URL（对象设为 public-read 后才有意义）。
//
// 复用 NormalizeEndpoint 解析端点，保证与建 client / 端点比较用同一套规则
// （去空白、scheme 大小写不敏感、去尾斜杠）；否则用户手填的
// " HTTP://MinIO:9000/ " 会产出带空格与双重 scheme 的链接。
func (c *Client) PublicURL(bucket, key string) string {
	ep := c.acc.PublicEndpoint
	if ep == "" {
		ep = c.acc.Endpoint
	}
	base := NormalizeEndpoint(ep, c.acc.UseSSL)
	if base == "" {
		return ""
	}
	// 拆分 scheme 与剩余部分：NormalizeEndpoint 已保证形如 scheme://host[/path]。
	scheme, rest := base, ""
	if i := strings.Index(base, "://"); i >= 0 {
		scheme, rest = base[:i], base[i+3:]
	}
	// 保留 endpoint 自带的路径前缀（反向代理场景），与建 client 的 BaseEndpoint 一致。
	host, path := rest, ""
	if i := strings.Index(rest, "/"); i >= 0 {
		host, path = rest[:i], rest[i:]
	}
	escKey := escapeKeyPath(key)
	if c.acc.PathStyle {
		return fmt.Sprintf("%s://%s%s/%s/%s", scheme, host, path, bucket, escKey)
	}
	return fmt.Sprintf("%s://%s.%s%s/%s", scheme, bucket, host, path, escKey)
}

func escapeKeyPath(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
