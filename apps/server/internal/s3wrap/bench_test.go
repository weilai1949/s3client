package s3wrap

// bench_test.go —— s3wrap 本地热路径的性能基线（结果与解读见 docs/PERFORMANCE.md）。
//
// 为什么只挑这两个：
//   - `NormalizeEndpoint`：每次构造 client（几乎每个 API 请求）都会走一遍；
//   - `PresignPut`：每次「浏览器直传 / 生成分享链接」都要做一次完整 SigV4 签名计算。
// 两者都是**纯本地计算**，结果不受网络与对端抖动影响，因此适合当作可回归的基线——
// 这也是本仓库不把需要真实 RustFS 的路径纳入基准的原因（那类验证属 E2E）。
//
// 运行：
//
//	cd apps/server && go test ./internal/s3wrap/ -run '^$' -bench 'Benchmark(NormalizeEndpoint|PresignPut)$' -benchmem
//
// 注意：本文件不含断言，不参与覆盖率统计（测试文件不参与 instrumentation）。

import (
	"context"
	"testing"
	"time"
)

// benchEndpoints 覆盖真实出现过的几种端点写法（含需要归一化的：带尾斜杠、
// 已被补成 https、大小写混合 scheme）——只测「已经是规范形态」的输入会低估成本。
var benchEndpoints = []string{
	"127.0.0.1:9000",
	"http://127.0.0.1:9000",
	"http://127.0.0.1:9000/",
	"HTTPS://S3.Example.COM",
	"s3.example.com/",
	"http://minio.internal:9000",
}

// BenchmarkNormalizeEndpoint 测端点归一化的单次成本。
func BenchmarkNormalizeEndpoint(b *testing.B) {
	for _, ep := range benchEndpoints {
		b.Run(ep, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = NormalizeEndpoint(ep, false)
			}
		})
	}
}

// BenchmarkPresignPut 测 SigV4 预签名 PUT 的单次成本（本地计算，不发起请求）。
func BenchmarkPresignPut(b *testing.B) {
	c, err := New(fakeAccount("http://127.0.0.1:9000"))
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	const (
		bucket = "bench-bucket"
		key    = "some/prefix/object-with-a-realistic-name.bin"
	)
	// 先跑一次确认签名确实能产出（避免基准在测一条恒定报错的路径）。
	if _, err := c.PresignPut(ctx, bucket, key, time.Hour); err != nil {
		b.Fatalf("PresignPut 前置校验失败: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.PresignPut(ctx, bucket, key, time.Hour); err != nil {
			b.Fatalf("PresignPut: %v", err)
		}
	}
}
