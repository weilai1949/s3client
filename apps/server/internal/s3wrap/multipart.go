package s3wrap

import (
	"context"
	"fmt"
	"io"
	"net/url"
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
	return c.CreateMultipartUploadWithMeta(ctx, bucket, key, contentType, nil)
}

// CreateMultipartUploadWithMeta 初始化分段上传并携带用户元数据（x-amz-meta-*）。
// 分段复制超大对象时必须走这里，否则元数据在完成分段上传后会丢失。
func (c *Client) CreateMultipartUploadWithMeta(ctx context.Context, bucket, key, contentType string, metadata map[string]string) (string, error) {
	in := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if len(metadata) > 0 {
		in.Metadata = metadata
	}
	out, err := c.s3.CreateMultipartUpload(ctx, in)
	if err != nil {
		return "", err
	}
	return derefString(out.UploadId), nil
}

// UploadPart 上传单个分段（跨 endpoint 大文件迁移），返回 ETag。
//
// size 必须是 body 的确切字节数，并显式写入 Content-Length：流式 body 不可 seek，
// SDK 无法自行计算长度（缺失时请求失败，ADR-014 实测）。
func (c *Client) UploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int32, body io.Reader, size int64) (string, error) {
	out, err := c.s3.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		UploadId:      aws.String(uploadID),
		PartNumber:    aws.Int32(partNumber),
		Body:          body,
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", err
	}
	return strings.Trim(derefString(out.ETag), `"`), nil
}

// UploadPartCopy 服务端分段复制：把源对象 [offset, offset+length) 直接拷为一个分段，
// 数据不经过本进程。返回 ETag。源与目标的凭证都需要相应权限（与 CopyObject 同口径）。
func (c *Client) UploadPartCopy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey, uploadID string, partNumber int32, offset, length int64) (string, error) {
	out, err := c.s3.UploadPartCopy(ctx, &s3.UploadPartCopyInput{
		Bucket:          aws.String(dstBucket),
		Key:             aws.String(dstKey),
		UploadId:        aws.String(uploadID),
		PartNumber:      aws.Int32(partNumber),
		CopySource:      aws.String(url.PathEscape(srcBucket + "/" + srcKey)),
		CopySourceRange: aws.String(fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)),
	})
	if err != nil {
		return "", err
	}
	if out.CopyPartResult == nil {
		return "", nil
	}
	return strings.Trim(derefString(out.CopyPartResult.ETag), `"`), nil
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

// listPartsMaxPages 是 ListParts 翻页上限：1000 段/页 × 100 页 = 10 万段，十倍于协议 1 万段上限。
// 与「游标不前进即停」共同保证循环**有界**（口径同 service/sync_list_gate_test.go）——
// 对端始终返回 truncated 或游标原地打转时，没有守卫会让断点续传清单查询永久挂住。
const listPartsMaxPages = 100

// ListParts 列出某次分段上传已上传的分段（断点续传对齐服务端真实清单用）。
// 自动按 NextPartNumberMarker 翻页，保证超过 1000 段的大文件清单完整；
// uploadId 失效（NoSuchUpload）时错误透传，由调用方决定是否重新 init。
func (c *Client) ListParts(ctx context.Context, bucket, key, uploadID string) ([]MultipartPart, error) {
	parts := make([]MultipartPart, 0)
	var marker *string
	for page := 0; ; page++ {
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
		// 有界守卫（页上限 + 游标不前进即停）：对端异常不得让查询永久挂住。
		if page+1 >= listPartsMaxPages || (marker != nil && *out.NextPartNumberMarker <= *marker) {
			return parts, nil
		}
		marker = out.NextPartNumberMarker
	}
}
