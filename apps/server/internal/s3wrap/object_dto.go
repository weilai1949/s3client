package s3wrap

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ObjectStream 对象内容流（防腐层 DTO，避免 handler 依赖 *s3.GetObjectOutput）。
type ObjectStream struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength *int64
	ContentRange  string
	Metadata      map[string]string
}

// ObjectMeta 对象头信息（防腐层 DTO）。
type ObjectMeta struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
	StorageClass string
	Metadata     map[string]string
	// Checksums 服务端存储的校验和（Head 以 ChecksumMode=ENABLED 请求；
	// 厂商不支持或对象无校验和 → nil）。
	Checksums *ObjectChecksums
}

// GetObjectStream 读取对象内容流。
func (c *Client) GetObjectStream(ctx context.Context, bucket, key, versionID, rangeHeader string) (*ObjectStream, error) {
	out, err := c.getObjectIn(ctx, bucket, key, versionID, rangeHeader)
	if err != nil {
		return nil, err
	}
	return objectStreamFrom(out), nil
}

func objectStreamFrom(out *s3.GetObjectOutput) *ObjectStream {
	s := &ObjectStream{
		Body:          out.Body,
		ContentType:   aws.ToString(out.ContentType),
		ContentLength: out.ContentLength,
		ContentRange:  aws.ToString(out.ContentRange),
		Metadata:      out.Metadata,
	}
	return s
}

// HeadObjectMeta 获取对象元数据 DTO。
func (c *Client) HeadObjectMeta(ctx context.Context, bucket, key, versionID string) (*ObjectMeta, error) {
	out, err := c.headObject(ctx, bucket, key, versionID)
	if err != nil {
		return nil, err
	}
	m := &ObjectMeta{
		Size:         aws.ToInt64(out.ContentLength),
		ETag:         aws.ToString(out.ETag),
		ContentType:  aws.ToString(out.ContentType),
		StorageClass: string(out.StorageClass),
		Metadata:     out.Metadata,
		Checksums:    checksumsFrom(out),
	}
	if out.LastModified != nil {
		m.LastModified = *out.LastModified
	}
	return m, nil
}

// checksumsFrom 从 Head 响应提取校验和；四个算法值全空（哪怕带了 checksum-type）→ nil。
func checksumsFrom(out *s3.HeadObjectOutput) *ObjectChecksums {
	cs := &ObjectChecksums{
		CRC64NVME: aws.ToString(out.ChecksumCRC64NVME),
		CRC32C:    aws.ToString(out.ChecksumCRC32C),
		SHA256:    aws.ToString(out.ChecksumSHA256),
		SHA1:      aws.ToString(out.ChecksumSHA1),
		Type:      string(out.ChecksumType),
	}
	for _, v := range []string{cs.CRC64NVME, cs.CRC32C, cs.SHA256, cs.SHA1} {
		if v != "" {
			return cs
		}
	}
	return nil
}
