package s3wrap

import (
	"context"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ListObjectsPage 使用 ListObjectsV2 分页列出对象（防腐层 DTO）。
func (c *Client) ListObjectsPage(ctx context.Context, bucket, prefix, delimiter, continuationToken, startAfter string, maxKeys int32) (*ListPage, error) {
	out, err := c.listObjectsV2(ctx, bucket, prefix, delimiter, continuationToken, startAfter, maxKeys)
	if err != nil {
		return nil, err
	}
	return listPageFrom(out), nil
}

func (c *Client) listObjectsV2(ctx context.Context, bucket, prefix, delimiter, continuationToken, startAfter string, maxKeys int32) (*s3.ListObjectsV2Output, error) {
	in := &s3.ListObjectsV2Input{
		Bucket:            aws.String(bucket),
		Prefix:            aws.String(prefix),
		ContinuationToken: optionalString(continuationToken),
		StartAfter:        optionalString(startAfter),
	}
	if delimiter != "" {
		in.Delimiter = aws.String(delimiter)
	}
	if maxKeys > 0 {
		in.MaxKeys = aws.Int32(maxKeys)
	}
	return c.s3.ListObjectsV2(ctx, in)
}

// DeleteObject 删除单个对象。
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	return c.DeleteObjectCond(ctx, bucket, key, Conditions{})
}

// DeleteObjectCond 条件删除：仅当对象当前 ETag 匹配 IfMatch 才删除（防误删已变更对象）。
func (c *Client) DeleteObjectCond(ctx context.Context, bucket, key string, cond Conditions) error {
	in := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if cond.IfMatch != "" {
		in.IfMatch = aws.String(cond.IfMatch)
	}
	_, err := c.s3.DeleteObject(ctx, in)
	return err
}

// DeleteFailure 描述批量删除中被服务端逐 key 拒绝的条目。
//
// S3 对「部分 key 删不掉」（桶策略、保留期、MFA Delete）仍返回 200，只在响应体内列 <Error>；
// 调用方必须据此把失败 key 从「已删除」中剔除，否则会向用户误报删除成功。
type DeleteFailure struct {
	Key     string
	Code    string
	Message string
}

// DeleteObjects 批量删除（SDK 单次上限 1000 个）。分批处理。
//
// 返回的 error 仅表示传输/协议层失败（该批 key 的结果未知）；逐 key 的服务端拒绝通过
// 返回的 []DeleteFailure 表达。两者可同时非空：前面的批次已确定失败，后面的批次传输中断。
func (c *Client) DeleteObjects(ctx context.Context, bucket string, keys []string) ([]DeleteFailure, error) {
	const batchSize = 1000
	var failures []DeleteFailure
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		objs := make([]types.ObjectIdentifier, 0, end-i)
		for _, k := range keys[i:end] {
			objs = append(objs, types.ObjectIdentifier{Key: aws.String(k)})
		}
		out, err := c.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{Objects: objs},
		})
		if err != nil {
			return failures, err
		}
		for _, e := range out.Errors {
			failures = append(failures, DeleteFailure{
				Key:     aws.ToString(e.Key),
				Code:    aws.ToString(e.Code),
				Message: aws.ToString(e.Message),
			})
		}
	}
	return failures, nil
}

// CopyObject 在服务端复制对象（同 endpoint 使用 CopyObject 接口）。
func (c *Client) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	return c.CopyObjectCond(ctx, srcBucket, srcKey, dstBucket, dstKey, CopyOptions{})
}

// CopyObjectCond 复制对象并带**目标端**条件写谓词与可选的校验和物化：
// 条件不满足时 S3 返回 412/409（源对象不受影响）；ChecksumAlgorithm 非空时服务端
// 计算并存储全对象校验和（RustFS 实测支持，供 verify-checksum 端到端比对）。
func (c *Client) CopyObjectCond(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string, opts CopyOptions) error {
	in := &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(url.PathEscape(srcBucket + "/" + srcKey)),
	}
	if opts.IfMatch != "" {
		in.IfMatch = aws.String(opts.IfMatch)
	}
	if opts.IfNoneMatch != "" {
		in.IfNoneMatch = aws.String(opts.IfNoneMatch)
	}
	if opts.ChecksumAlgorithm != "" {
		in.ChecksumAlgorithm = types.ChecksumAlgorithm(opts.ChecksumAlgorithm)
	}
	_, err := c.s3.CopyObject(ctx, in)
	return wrapObjectTooLarge(err)
}

// CopyObjectWithMeta 复制对象并替换元数据。
func (c *Client) CopyObjectWithMeta(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey, contentType string, metadata map[string]string) error {
	in := &s3.CopyObjectInput{
		Bucket:            aws.String(dstBucket),
		Key:               aws.String(dstKey),
		CopySource:        aws.String(url.PathEscape(srcBucket + "/" + srcKey)),
		MetadataDirective: types.MetadataDirectiveReplace,
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if len(metadata) > 0 {
		in.Metadata = metadata
	}
	_, err := c.s3.CopyObject(ctx, in)
	return err
}

func (c *Client) getObjectIn(ctx context.Context, bucket, key, versionID, rangeHeader string) (*s3.GetObjectOutput, error) {
	in := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		in.VersionId = aws.String(versionID)
	}
	if rangeHeader != "" {
		in.Range = aws.String(rangeHeader)
	}
	return c.s3.GetObject(ctx, in)
}

// headObject 获取对象元数据（仅包内使用；对外走 HeadObjectMeta DTO）。
func (c *Client) headObject(ctx context.Context, bucket, key, versionID string) (*s3.HeadObjectOutput, error) {
	in := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		// ChecksumMode=ENABLED 让服务端在响应中带回存储的校验和；
		// 不支持的厂商忽略该请求头（读侧降级，见 checksum.go）。
		ChecksumMode: types.ChecksumModeEnabled,
	}
	if versionID != "" {
		in.VersionId = aws.String(versionID)
	}
	return c.s3.HeadObject(ctx, in)
}

// PutObject 上传对象内容（ContentType/Metadata 用纯值传递，防腐层不暴露 SDK 输入类型）。
func (c *Client) PutObject(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string) error {
	return c.PutObjectCond(ctx, bucket, key, body, contentType, metadata, Conditions{})
}

// PutObjectCond 带条件写谓词的上传：If-None-Match:"*" 只创建不覆盖，If-Match 按 ETag 乐观锁。
func (c *Client) PutObjectCond(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string, cond Conditions) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if len(metadata) > 0 {
		in.Metadata = metadata
	}
	if cond.IfMatch != "" {
		in.IfMatch = aws.String(cond.IfMatch)
	}
	if cond.IfNoneMatch != "" {
		in.IfNoneMatch = aws.String(cond.IfNoneMatch)
	}
	_, err := c.s3.PutObject(ctx, in)
	return wrapObjectTooLarge(err)
}

// GetObjectAcl 读取对象 ACL，返回展示型 DTO（owner/是否公开/授权行）。
func (c *Client) GetObjectAcl(ctx context.Context, bucket, key string) (owner string, public bool, rows []GrantRow, err error) {
	out, err := c.s3.GetObjectAcl(ctx, &s3.GetObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", false, nil, err
	}
	owner, public, rows = DescribeACL(out)
	return owner, public, rows, nil
}

// PutObjectAcl 设置对象 canned ACL。
func (c *Client) PutObjectAcl(ctx context.Context, bucket, key, acl string) error {
	in := &s3.PutObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		ACL:    toObjectCannedACL(acl),
	}
	_, err := c.s3.PutObjectAcl(ctx, in)
	return err
}

func toObjectCannedACL(acl string) types.ObjectCannedACL {
	switch acl {
	case "public-read":
		return types.ObjectCannedACLPublicRead
	case "public-read-write":
		return types.ObjectCannedACLPublicReadWrite
	case "authenticated-read":
		return types.ObjectCannedACLAuthenticatedRead
	case "aws-exec-read":
		return types.ObjectCannedACLAwsExecRead
	default:
		return types.ObjectCannedACLPrivate
	}
}

// GetObjectTags 读取对象标签（防腐层 DTO）。
func (c *Client) GetObjectTags(ctx context.Context, bucket, key string) (*TaggingSet, error) {
	out, err := c.s3.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return taggingSetFrom(out), nil
}

// PutObjectTags 覆盖写入对象标签。
func (c *Client) PutObjectTags(ctx context.Context, bucket, key string, tags map[string]string) error {
	tagSet := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		tagSet = append(tagSet, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	_, err := c.s3.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
		Bucket:  aws.String(bucket),
		Key:     aws.String(key),
		Tagging: &types.Tagging{TagSet: tagSet},
	})
	return err
}

// DeleteObjectTags 删除对象全部标签。
func (c *Client) DeleteObjectTags(ctx context.Context, bucket, key string) error {
	_, err := c.s3.DeleteObjectTagging(ctx, &s3.DeleteObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

// ListObjectVersions 列出对象各版本（含删除标记）与分页游标（防腐层 DTO）。
func (c *Client) ListObjectVersions(ctx context.Context, bucket, prefix, keyMarker, versionIDMarker string, maxKeys int32) (*VersionsPage, error) {
	out, err := c.listObjectVersionsIn(ctx, bucket, prefix, keyMarker, versionIDMarker, maxKeys)
	if err != nil {
		return nil, err
	}
	return versionsPageFrom(out), nil
}

func (c *Client) listObjectVersionsIn(ctx context.Context, bucket, prefix, keyMarker, versionIDMarker string, maxKeys int32) (*s3.ListObjectVersionsOutput, error) {
	in := &s3.ListObjectVersionsInput{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(maxKeys),
	}
	if prefix != "" {
		in.Prefix = aws.String(prefix)
	}
	if keyMarker != "" {
		in.KeyMarker = aws.String(keyMarker)
	}
	if versionIDMarker != "" {
		in.VersionIdMarker = aws.String(versionIDMarker)
	}
	return c.s3.ListObjectVersions(ctx, in)
}

// DeleteObjectVersion 删除对象指定版本。
func (c *Client) DeleteObjectVersion(ctx context.Context, bucket, key, versionID string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: aws.String(versionID),
	})
	return err
}

// RestoreObjectVersion 将指定历史版本复制回当前 key。
func (c *Client) RestoreObjectVersion(ctx context.Context, bucket, key, versionID string) (string, error) {
	out, err := c.s3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		CopySource: aws.String(url.PathEscape(bucket+"/"+key) + "?versionId=" + url.QueryEscape(versionID)),
	})
	if err != nil {
		return "", err
	}
	return derefString(out.VersionId), nil
}

// RestoreDeleteMarker 移除指定的删除标记版本。
func (c *Client) RestoreDeleteMarker(ctx context.Context, bucket, key, versionID string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: aws.String(versionID),
	})
	return err
}

// PurgeObject 彻底清除桶内某 key 的全部版本与删除标记。
func (c *Client) PurgeObject(ctx context.Context, bucket, key string) (int, error) {
	deleted := 0
	keyMarker, versionIDMarker := "", ""
	for {
		out, err := c.ListObjectVersions(ctx, bucket, key, keyMarker, versionIDMarker, 1000)
		if err != nil {
			return deleted, err
		}
		ids := make([]types.ObjectIdentifier, 0, 1000)
		for _, v := range out.Versions {
			if v.Key == key && v.VersionID != "" {
				ids = append(ids, types.ObjectIdentifier{Key: aws.String(key), VersionId: aws.String(v.VersionID)})
			}
		}
		for _, d := range out.DeleteMarkers {
			if d.Key == key && d.VersionID != "" {
				ids = append(ids, types.ObjectIdentifier{Key: aws.String(key), VersionId: aws.String(d.VersionID)})
			}
		}
		if len(ids) > 0 {
			const batch = 1000
			for i := 0; i < len(ids); i += batch {
				end := i + batch
				if end > len(ids) {
					end = len(ids)
				}
				out, err := c.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
					Bucket: aws.String(bucket),
					Delete: &types.Delete{Objects: ids[i:end]},
				})
				if err != nil {
					return deleted, err
				}
				// 200 响应体里的逐版本 <Error> 同样算失败：把被拒版本计入「已删除」会让
				// 回收站 UI 显示「已彻底清除」而版本仍然存在（review §B3 第 4 个受影响点）。
				deleted += end - i - len(out.Errors)
				if len(out.Errors) > 0 {
					first := out.Errors[0]
					// 结构化载体携带已删计数：UserMessage 据此透出「删了多少」（review §R17）。
					return deleted, &partialDeleteError{
						deleted:   deleted,
						firstKey:  aws.ToString(first.Key),
						firstCode: aws.ToString(first.Code),
					}
				}
			}
		}
		if !out.IsTruncated || out.NextKeyMarker == "" {
			break
		}
		// 分页续读：marker 写入后由下一轮 ListObjectVersions 消费（测试中不触发分页分支）。
		keyMarker = out.NextKeyMarker
		versionIDMarker = out.NextVersionIDMarker
	}
	if deleted == 0 {
		if err := c.DeleteObject(ctx, bucket, key); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

// ChangeObjectStorageClass 通过 CopyObject 切换存储类型。
func (c *Client) ChangeObjectStorageClass(ctx context.Context, bucket, key, versionID, storageClass string) (string, error) {
	copySource := url.PathEscape(bucket + "/" + key)
	if versionID != "" {
		copySource += "?versionId=" + url.QueryEscape(versionID)
	}
	out, err := c.s3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		CopySource:   aws.String(copySource),
		StorageClass: types.StorageClass(storageClass),
	})
	if err != nil {
		return "", err
	}
	return derefString(out.VersionId), nil
}
