package store

// bench_test.go —— 账号存储读写路径的性能基线（结果与解读见 docs/PERFORMANCE.md）。
//
// 为什么挑读写而不是别的：这两条是**唯一有真实 I/O 成本**的路径，且都在请求链路上——
//   - 读：`List()` 在每个需要账号列表的 API 请求里都会走（对象浏览、桶列举、迁移选择…）；
//   - 写：`Create/Update` 走「编码 → 临时文件 → fsync → rename → fsync 父目录」的原子写，
//     每加一个账号一次；`encrypted` 驱动还要多一次 Argon2id 派生。
//
// 三种驱动分别测（json / json+key / encrypted），因为它们的成本量级完全不同：
// 加密路径的 Argon2id 是**有意设计成慢**的（抗暴力破解），基线的作用是「钉住它没有被
// 无意改快或改慢」，而不是追求更快。
//
// 运行：
//
//	cd apps/server && go test ./internal/store/ -run '^$' -bench 'BenchmarkStore' -benchmem

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

// benchAccount 构造第 i 个基准账号（字段长度贴近真实值，避免测出「空结构体特别快」的假象）。
func benchAccount(i int) *model.Account {
	now := time.Now()
	return &model.Account{
		ID:        fmt.Sprintf("bench-account-%06d", i),
		Name:      fmt.Sprintf("bench-account-%06d", i),
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

// benchStores 返回三种驱动各自的构造函数（storeKey 为空即明文 json）。
func benchStores(tb testing.TB) map[string]func(tb testing.TB) AccountStore {
	tb.Helper()
	const key = "bench-store-key-0123456789"
	return map[string]func(tb testing.TB) AccountStore{
		"json": func(tb testing.TB) AccountStore {
			tb.Helper()
			st, err := New(filepath.Join(tb.TempDir(), "accounts.json"))
			if err != nil {
				tb.Fatalf("store.New: %v", err)
			}
			return st
		},
		"json+key": func(tb testing.TB) AccountStore {
			tb.Helper()
			st, err := newStore(filepath.Join(tb.TempDir(), "accounts.json"), key, false)
			if err != nil {
				tb.Fatalf("newStore(json+key): %v", err)
			}
			return st
		},
		"encrypted": func(tb testing.TB) AccountStore {
			tb.Helper()
			st, err := NewEncrypted(filepath.Join(tb.TempDir(), "accounts.json.enc"), key)
			if err != nil {
				tb.Fatalf("NewEncrypted: %v", err)
			}
			return st
		},
	}
}

// benchSeed 预置 n 个账号（不计入计时）。
func benchSeed(tb testing.TB, st AccountStore, n int) {
	tb.Helper()
	for i := 0; i < n; i++ {
		if _, err := st.Create(benchAccount(i)); err != nil {
			tb.Fatalf("seed Create(%d): %v", i, err)
		}
	}
}

// BenchmarkStoreList 测账号列表读取（含未加密 / 加密驱动的解码差异）。
func BenchmarkStoreList(b *testing.B) {
	for name, mk := range benchStores(b) {
		b.Run(name, func(b *testing.B) {
			st := mk(b)
			benchSeed(b, st, 32)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				got, err := st.List()
				if err != nil {
					b.Fatalf("List: %v", err)
				}
				if len(got) != 32 {
					b.Fatalf("List 返回 %d 条，期望 32", len(got))
				}
			}
		})
	}
}

// BenchmarkStoreCreate 测单次账号创建（原子写：temp → fsync → rename → fsync 父目录）。
//
// 每轮用不同的 ID，避免命中「更新已有条目」的短路分支而低估写入成本。
func BenchmarkStoreCreate(b *testing.B) {
	for name, mk := range benchStores(b) {
		b.Run(name, func(b *testing.B) {
			st := mk(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := st.Create(benchAccount(i)); err != nil {
					b.Fatalf("Create: %v", err)
				}
			}
		})
	}
}
