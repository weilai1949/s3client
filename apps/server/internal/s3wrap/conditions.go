package s3wrap

// conditions.go —— 条件写（S3 If-Match / If-None-Match）的防腐层表达。
//
// S3 语义（2025 起 GA 的 conditional writes，另 DeleteObject 的 If-Match 为条件删除）：
//   - If-None-Match: "*"  仅当目标对象不存在时写入（创建即失败，防并发覆盖）；
//   - If-Match: <etag>    仅当目标对象当前 ETag 匹配时写入（乐观锁，防丢更新）。
//
// 条件由服务端在写入时求值，与签名无关；预签名直传路径由客户端显式回传这些请求头
// （见 PresignPut 的 Headers 回显）。

// Conditions 描述一次写操作的条件谓词；零值 = 无条件（与历史行为完全一致）。
type Conditions struct {
	// IfMatch 目标对象当前 ETag 须匹配该值（含引号的 ETag 字面量）；空 = 不校验。
	IfMatch string
	// IfNoneMatch 仅当目标不存在时写入；S3 只接受 "*"。
	IfNoneMatch string
}

// CopyOptions 是复制操作的可选项：目标端条件写谓词 + 服务端物化全对象校验和的算法。
// 与 Conditions 分开是因为它只作用于 CopyObject（S3 的条件写与 ChecksumAlgorithm
// 仅在复制时可同时指定；单 PUT 写侧的校验和由服务端 / 客户端另行决定）。
type CopyOptions struct {
	Conditions
	// ChecksumAlgorithm 非空时要求服务端以该算法计算并存储全对象校验和
	//（取值 CRC64NVME / SHA256 / CRC32C / SHA1，由 handler 边界校验）；
	// 空 = 服务端默认，不携带该头（历史行为不变）。
	ChecksumAlgorithm string
}

// headers 返回客户端随请求必须携带的条件头（预签名直传回显用）。
func (c Conditions) headers() map[string]string {
	h := map[string]string{}
	if c.IfMatch != "" {
		h["If-Match"] = c.IfMatch
	}
	if c.IfNoneMatch != "" {
		h["If-None-Match"] = c.IfNoneMatch
	}
	return h
}
