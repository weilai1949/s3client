package main

// bench_budget_test.go —— 后端**热路径性能回归门禁**（结果与解读见 docs/PERFORMANCE.md §4）。
//
// 背景（2026-09-30）：`internal/{store,s3wrap,service}/bench_test.go` 提供了基线，但
// `docs/PERFORMANCE.md` §4 明说「基准不设阈值门禁（有意）」——于是「把 `RelKey` 从 0 分配
// 改成 2 次分配」这类回归只能靠人记得去跑基准，CI 完全看不见。本文件把**可机械判定的部分**
// 变成红灯，同时刻意不把机器敏感的绝对值做成 flaky 断言。
//
// 为什么这样切分（**flakiness 判定，务必读**）：
//
//   - **分配数 / 分配字节**在固定 Go 版本 + 固定输入下是**确定性**的，跨机器基本不变。
//     `PresignPut` 实测 362 allocs/op / 27.5 KB/op，`encrypted` 写入实测 212 allocs/op /
//     67.3 MB/op（Argon2id 的 64 MiB 内存硬开销）。因此分配类断言取 ~3 倍余量——
//     足以拦住「多一次拷贝 / 多一层编码」的量级回归，又几乎不可能被调度噪声触发。
//   - **单次耗时**对机器、cgroup 限额与磁盘 fsync 极敏感，同一份代码在不同 runner 上可以
//     差数倍。故耗时只设**极宽**的兜底上限（约 70–80 倍实测值），用于拦住「算法级崩塌」
//     （例如 O(n) 变 O(n²)、Argon2id 参数被误调大一个数量级），而不是拦住 2–3 倍的小回归。
//     → 结论：本门禁**不承诺**捕捉 10 倍以内的耗时回归；那类趋势交给
//     `.github/workflows/perf.yml` 的定时基准与人工比对（`make bench`）。
//   - **账号持久化 O(n)** 用「小库 vs 大库的单次写入耗时比」做**结构性**断言：线性实现实测
//     约 25 倍（远小于 64 倍阈值），二次实现会是三个数量级以上；同时给 10ms 绝对地板 +
//     250ms 绝对兜底，避免小库耗时趋零时比值被噪声放大。
//
// 运行（与 `make bench` 同命令）：
//
//	cd apps/server && go test . -run 'TestBench' -count=1 -v
//
// 变异验证（复核步骤，2026-09-30 实测）：把 `PresignPut` 的分配上限改成 1 →
// 红灯并打印实测 362 allocs/op → 还原后绿灯；把 `perf.yml` 的 `schedule:` 删掉 →
// `TestPerfGateAndDocsAreWired` 红灯点名。

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

// 预算常量。注释里的「实测」取自 2026-09-30 的本仓库开发机（go1.26.6 linux/amd64，
// i5-13400F / GOMAXPROCS=16），修订时请连同 docs/PERFORMANCE.md §4 的表格一起改。
const (
	// PresignPut：实测 ~26 µs / 362 allocs / 27.5 KB。耗时上限 ~78 倍余量（兜底），
	// 分配上限 ~3.3 倍余量（确定性，主力断言）。
	benchPresignPutCeiling      = 2 * time.Millisecond
	benchPresignPutAllocCeiling = 1200
	benchPresignPutBytesCeiling = 96 << 10

	// encrypted 写入：实测 ~29 ms / 212 allocs / 67.3 MB（Argon2id 64 MiB 是有意设计）。
	// 耗时上限 ~70 倍余量；分配字节上限 ~2.4 倍余量。
	benchEncryptedWriteCeiling      = 2 * time.Second
	benchEncryptedWriteAllocCeiling = 1200
	benchEncryptedWriteBytesCeiling = 160 << 20

	// 账号持久化 O(n)：小库 4 个账号 vs 大库 256 个账号，各取 5 次单写的最小值比。
	// 线性实现实测比值约 25（远小于 64）；二次实现会到三个数量级。绝对地板 10ms 防小库
	// 耗时趋零时比值被噪声放大；另有 250ms 绝对兜底上限。
	benchPersistSmallSeed     = 4
	benchPersistLargeSeed     = 256
	benchPersistIterations    = 5
	benchPersistGrowthCeiling = 64
	benchPersistGrowthFloor   = 10 * time.Millisecond
	benchPersistLargeCeiling  = 250 * time.Millisecond
	benchStoreKey             = "bench-store-key-0123456789"
	benchObjectKey            = "some/prefix/object-with-a-realistic-name.bin"
	benchPersistJSONFile      = "accounts.json"
	benchEncryptedFileName    = "accounts.json.enc"
)

// benchBudgetAccount 构造第 i 个预算基准账号（字段长度贴近真实值，避免测出「空结构体特别快」）。
func benchBudgetAccount(i int) *model.Account {
	now := time.Now()
	return &model.Account{
		ID:        fmt.Sprintf("bench-budget-%06d", i),
		Name:      fmt.Sprintf("bench-budget-%06d", i),
		Endpoint:  fmt.Sprintf("http://minio-%02d.internal:9000", i%32),
		Region:    "us-east-1",
		AccessKey: fmt.Sprintf("AKIA%016d", i),
		SecretKey: fmt.Sprintf("secret-%032d", i),
		Bucket:    "bench-bucket",
		PathStyle: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TestBenchPresignPutStaysWithinBudget：SigV4 预签名 PUT（每次「直传 / 分享链接」一次）的
// 分配与耗时都必须在预算内。分配数是主力断言（确定性），耗时上限只兜底算法级崩塌。
func TestBenchPresignPutStaysWithinBudget(t *testing.T) {
	res := testing.Benchmark(func(b *testing.B) {
		c, err := s3wrap.New(&model.Account{
			Endpoint:  "http://127.0.0.1:9000",
			Region:    "us-east-1",
			AccessKey: "AKIAIOSFODNN7EXAMPLE",
			SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			PathStyle: true,
		})
		if err != nil {
			b.Fatalf("s3wrap.New: %v", err)
		}
		ctx := context.Background()
		// 先跑一次，确认基准确实在测一条能产出签名的路径（而不是恒定报错）。
		if _, err := c.PresignPut(ctx, "bench-bucket", benchObjectKey, time.Hour, s3wrap.Conditions{}); err != nil {
			b.Fatalf("PresignPut 前置校验失败: %v", err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := c.PresignPut(ctx, "bench-bucket", benchObjectKey, time.Hour, s3wrap.Conditions{}); err != nil {
				b.Fatalf("PresignPut: %v", err)
			}
		}
	})
	if res.N == 0 {
		t.Fatal("testing.Benchmark 未产生任何迭代（基准函数未执行）")
	}
	nsPerOp, allocsPerOp, bytesPerOp := res.NsPerOp(), res.AllocsPerOp(), res.AllocedBytesPerOp()
	t.Logf("PresignPut：%d ns/op，%d B/op，%d allocs/op（N=%d）", nsPerOp, bytesPerOp, allocsPerOp, res.N)

	if time.Duration(nsPerOp) > benchPresignPutCeiling {
		t.Errorf("PresignPut 单次 %s，超过兜底上限 %s（实测 2026-09-30 约 26µs）——"+
			"疑似算法级回归（如 HMAC 链 / URI 编码被改写、重复签名）；若确认是有意取舍，请连同 "+
			"docs/PERFORMANCE.md §4 一起更新预算", time.Duration(nsPerOp), benchPresignPutCeiling)
	}
	if allocsPerOp > benchPresignPutAllocCeiling {
		t.Errorf("PresignPut 每次 %d 次分配，超过预算 %d（实测 362）——分配数是确定性指标，"+
			"多一层拷贝 / 编码即会触发；这属于应解释的回归（docs/PERFORMANCE.md §3.3）",
			allocsPerOp, benchPresignPutAllocCeiling)
	}
	if bytesPerOp > benchPresignPutBytesCeiling {
		t.Errorf("PresignPut 每次 %d B，超过预算 %d B（实测 27.5 KB）——分配字节也应被解释",
			bytesPerOp, benchPresignPutBytesCeiling)
	}
}

// TestBenchEncryptedStoreWriteStaysWithinBudget：`encrypted` 驱动的账号写入（Argon2id +
// 原子写盘）分配与耗时预算。它**有意**慢（内存硬函数抗暴力破解），门禁只是钉住它没有被
// 无意改快 / 改慢（参数漂移）。
func TestBenchEncryptedStoreWriteStaysWithinBudget(t *testing.T) {
	res := testing.Benchmark(func(b *testing.B) {
		st, err := store.NewEncrypted(filepath.Join(b.TempDir(), benchEncryptedFileName), benchStoreKey)
		if err != nil {
			b.Fatalf("store.NewEncrypted: %v", err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := st.Create(benchBudgetAccount(i)); err != nil {
				b.Fatalf("Create(%d): %v", i, err)
			}
		}
	})
	if res.N == 0 {
		t.Fatal("testing.Benchmark 未产生任何迭代（基准函数未执行）")
	}
	nsPerOp, allocsPerOp, bytesPerOp := res.NsPerOp(), res.AllocsPerOp(), res.AllocedBytesPerOp()
	t.Logf("StoreCreate/encrypted：%d ns/op，%d B/op，%d allocs/op（N=%d）", nsPerOp, bytesPerOp, allocsPerOp, res.N)

	if time.Duration(nsPerOp) > benchEncryptedWriteCeiling {
		t.Errorf("encrypted 写入单次 %s，超过兜底上限 %s（实测 2026-09-30 约 29ms）——"+
			"疑似 Argon2id 参数被放大或原子写路径退化；若是有意调整，请同步 "+
			"docs/PERFORMANCE.md §3.1 与本节预算", time.Duration(nsPerOp), benchEncryptedWriteCeiling)
	}
	if allocsPerOp > benchEncryptedWriteAllocCeiling {
		t.Errorf("encrypted 写入每次 %d 次分配，超过预算 %d（实测 212）", allocsPerOp, benchEncryptedWriteAllocCeiling)
	}
	if bytesPerOp > benchEncryptedWriteBytesCeiling {
		t.Errorf("encrypted 写入每次 %d B，超过预算 %d B（实测 67.3 MB，主要是 Argon2id 的 64 MiB）"+
			"——超限通常是派生参数被放大，会直接顶爆容器内存上限", bytesPerOp, benchEncryptedWriteBytesCeiling)
	}
}

// TestBenchAccountPersistStaysLinear：账号写入是 **O(n)**（每次重写整文件，见
// docs/PERFORMANCE.md §3.2）。本用例用「小库 vs 大库的单次写入最小耗时比」做结构性断言，
// 逻辑上的二次实现会到三个数量级，而线性实现远低于阈值。
func TestBenchAccountPersistStaysLinear(t *testing.T) {
	// 固定为明文 json 驱动：避免测试环境里 S3C_STORE_KEY 让等价路径变成 Argon2id 慢路径。
	t.Setenv("S3C_STORE_KEY", "")

	small := benchMinCreateDuration(t, benchPersistSmallSeed)
	large := benchMinCreateDuration(t, benchPersistLargeSeed)
	limit := time.Duration(benchPersistGrowthCeiling) * small
	if limit < benchPersistGrowthFloor {
		limit = benchPersistGrowthFloor
	}
	t.Logf("账号写入单次最小耗时：%d 个账号时 %s，%d 个账号时 %s（比值 %.1fx，上限 %s）",
		benchPersistSmallSeed, small, benchPersistLargeSeed, large, float64(large)/float64(small), limit)

	if large > benchPersistLargeCeiling {
		t.Errorf("账号持久化大库单次写入 %s，超过绝对兜底上限 %s（实测约 0.7ms）——"+
			"账号写入是请求链路路径，量级崩塌会直接表现为接口超时", large, benchPersistLargeCeiling)
	}
	if large > limit {
		t.Errorf("账号持久化疑似超线性：%d 个账号的单次写入 %s，超过「%d 个账号时 %s × %d 倍」"+
			"（且不低于地板 %s）的上限 %s。fileStore.persistLocked 应是「序列化全部账号 + 原子写盘」的"+
			"O(n) 路径（docs/PERFORMANCE.md §3.2）；退化通常来自每写一条就重读整个文件。",
			benchPersistLargeSeed, large, benchPersistSmallSeed, small, benchPersistGrowthCeiling,
			benchPersistGrowthFloor, limit)
	}
}

// benchMinCreateDuration 在预置 seed 个账号的明文 json 库上做 iterations 次新建，
// 返回单次耗时的**最小值**（最小值对 fsync 尖峰与调度抖动最不敏感）。
func benchMinCreateDuration(t *testing.T, seed int) time.Duration {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), benchPersistJSONFile))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	for i := 0; i < seed; i++ {
		if _, err := st.Create(benchBudgetAccount(i)); err != nil {
			t.Fatalf("预置 Create(%d): %v", i, err)
		}
	}
	var best time.Duration
	for i := 0; i < benchPersistIterations; i++ {
		start := time.Now()
		if _, err := st.Create(benchBudgetAccount(seed + i)); err != nil {
			t.Fatalf("Create(%d): %v", seed+i, err)
		}
		if d := time.Since(start); best == 0 || d < best {
			best = d
		}
	}
	if best <= 0 {
		t.Fatalf("未能测得 %d 个账号库的单次写入耗时", seed)
	}
	return best
}

// TestPerfGateAndDocsAreWired：性能门的**结构/存在性**断言——Makefile 有 `bench` 目标、
// 定时 workflow 存在且同时支持 schedule 与 workflow_dispatch、文档登记了仪表盘与基准门禁。
//
// 为什么需要它：门禁本身（以及定时基准、文档）被删掉时，不会有任何别的红灯——这正是
// 「配置正确但断言缺席」的盲区（与 repo_infra_gate_test.go 同一动机）。
func TestPerfGateAndDocsAreWired(t *testing.T) {
	mk := readRepoFile(t, "Makefile")
	if !strings.Contains(mk, "\nbench:") {
		t.Error("Makefile 缺少 `bench` 目标——本地无法一键复现 perf workflow 的基准")
	}
	if !strings.Contains(mk, "-bench 'Benchmark'") {
		t.Error("Makefile 的 bench 目标必须真的跑 `-bench 'Benchmark'`")
	}
	// .PHONY 由 TestMakefileMirrorsCIGates 统一校验，这里不重复。

	perfRel := filepath.Join(".github", "workflows", "perf.yml")
	perf := readRepoFile(t, perfRel)
	for _, want := range []struct{ needle, why string }{
		{"schedule:", "定时跑基准是发现「无 PR 的缓慢退化」的唯一手段"},
		{"workflow_dispatch:", "必须支持手动触发，便于发布前复核与二分定位"},
		{"-bench", "workflow 没有真的跑基准"},
		{"actions/upload-artifact@", "基准结果必须作为 artifact 留存，供跨运行比对"},
		{"go-version-file: apps/server/go.mod", "基准的 Go 版本必须与 apps/server/go.mod 一致（否则数字不可比）"},
	} {
		if !strings.Contains(perf, want.needle) {
			t.Errorf("%s 缺少 %q——%s", perfRel, want.needle, want.why)
		}
	}

	// 文档同步（docs/DEVELOPMENT.md §4）：仪表盘路径必须能被运维找到；性能门禁的命令必须可复现。
	ops := readRepoFile(t, filepath.Join("docs", "OPERATIONS.md"))
	if !strings.Contains(ops, "deploy/grafana/s3client.dashboard.json") {
		t.Error("docs/OPERATIONS.md 未登记仪表盘路径 deploy/grafana/s3client.dashboard.json" +
			"——运维无法发现随仓库分发的 dashboard")
	}
	perfDoc := readRepoFile(t, filepath.Join("docs", "PERFORMANCE.md"))
	for _, want := range []string{"make bench", "bench_budget_test.go"} {
		if !strings.Contains(perfDoc, want) {
			t.Errorf("docs/PERFORMANCE.md 未登记 %q——性能门禁的用法与解读必须与代码同 PR", want)
		}
	}
}
