package s3wrap

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// UploadPartSpec 描述一个已完成的分段（ETag + 段号），用于 CompleteMultipartUpload。
type UploadPartSpec struct {
	PartNumber int32
	ETag       string
}

// CreateMultipartUpload 初始化分段上传，返回 UploadID。
func (c *Client) CreateMultipartUpload(ctx context.Context, bucket, key, contentType string) (string, error) {
	in := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	out, err := c.s3.CreateMultipartUpload(ctx, in)
	if err != nil {
		return "", err
	}
	return derefString(out.UploadId), nil
}

// UploadPart 服务端上传单个分段（跨 endpoint 大文件迁移），返回 ETag。
func (c *Client) UploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int32, body io.Reader) (string, error) {
	out, err := c.s3.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(partNumber),
		Body:       body,
	})
	if err != nil {
		return "", err
	}
	return strings.Trim(derefString(out.ETag), `"`), nil
}

// CompleteMultipartUpload 汇总已上传分段，完成分段上传。
func (c *Client) CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, parts []UploadPartSpec) error {
	completed := make([]types.CompletedPart, 0, len(parts))
	for _, p := range parts {
		completed = append(completed, types.CompletedPart{
			ETag:       aws.String(p.ETag),
			PartNumber: aws.Int32(p.PartNumber),
		})
	}
	_, err := c.s3.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(bucket),
		Key:             aws.String(key),
		UploadId:        aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completed},
	})
	return err
}

// AbortMultipartUpload 中止分段上传（失败清理用）。
func (c *Client) AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error {
	_, err := c.s3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	return err
}

// MultipartPart 是 ListParts 返回的单个已上传分段（自有 DTO，AWS SDK 类型不外泄）。
// ETag 已去掉 S3 返回的引号，与 UploadPart 的口径一致，可直接用于 CompleteMultipartUpload。
type MultipartPart struct {
	PartNumber   int32
	ETag         string
	Size         int64
	LastModified time.Time
}

// listPartsPageSize 单次 ListParts 请求的分段上限（S3 协议上限 1000）。
const listPartsPageSize = 1000

// ListParts 列出某次分段上传已上传的分段（断点续传对齐服务端真实清单用）。
// 自动按 NextPartNumberMarker 翻页，保证超过 1000 段的大文件清单完整；
// uploadId 失效（NoSuchUpload）时错误透传，由调用方决定是否重新 init。
func (c *Client) ListParts(ctx context.Context, bucket, key, uploadID string) ([]MultipartPart, error) {
	parts := make([]MultipartPart, 0)
	var marker *string
	for {
		out, err := c.s3.ListParts(ctx, &s3.ListPartsInput{
			Bucket:           aws.String(bucket),
			Key:              aws.String(key),
			UploadId:         aws.String(uploadID),
			MaxParts:         aws.Int32(listPartsPageSize),
			PartNumberMarker: marker,
		})
		if err != nil {
			return nil, err
		}
		for _, p := range out.Parts {
			parts = append(parts, MultipartPart{
				PartNumber:   derefInt32(p.PartNumber),
				ETag:         strings.Trim(derefString(p.ETag), `"`),
				Size:         derefInt64(p.Size),
				LastModified: timeOrZero(p.LastModified),
			})
		}
		// 截断但缺下一页标记属异常响应：就地结束，避免死循环。
		if !boolOrFalse(out.IsTruncated) || out.NextPartNumberMarker == nil {
			return parts, nil
		}
		marker = out.NextPartNumberMarker
	}
}
