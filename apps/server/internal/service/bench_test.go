package service

// bench_test.go —— key 映射内核的性能基线（结果与解读见 docs/PERFORMANCE.md）。
//
// 为什么挑它：`RelKey` / `BaseKey` 在批量复制、跨桶迁移与增量同步里**逐 key 调用**，
// 一次迁移动辄上万 key；而 `stripPrefix` 是它们共同的段边界内核（review R7 收敛点），
// 曾被裸 `TrimPrefix` 替代而产出错位 key——它既是正确性热点也是性能热点。
//
// 运行：
//
//	cd apps/server && go test ./internal/service/ -run '^$' -bench 'Benchmark(RelKey|BaseKey)$' -benchmem

import (
	"testing"
)

// benchKeyCases 复刻真实形态：深前缀、段边界命中与不命中、含中文与空格的对象名。
var benchKeyCases = []struct {
	name      string
	key       string
	srcPrefix string
	dstPrefix string
}{
	{"段边界命中", "backups/2026/09/dump.sql", "backups/2026/", "restore/"},
	{"段边界不命中", "backups/2026x/dump.sql", "backups/2026/", "restore/"},
	{"深前缀", "a/b/c/d/e/f/g/h/object.bin", "a/b/c/d/", "z/"},
	{"含中文与空格", "归档/2026 年 09 月/报告 final.pdf", "归档/", "archive/"},
	{"无前缀", "top-level.txt", "", "moved/"},
}

// BenchmarkRelKey 测「相对 key 映射到目标前缀」的单次成本。
func BenchmarkRelKey(b *testing.B) {
	for _, c := range benchKeyCases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = RelKey(c.key, c.srcPrefix, c.dstPrefix)
			}
		})
	}
}

// BenchmarkBaseKey 测「取 basename 拼目标前缀」的单次成本（copyMany 用）。
func BenchmarkBaseKey(b *testing.B) {
	for _, c := range benchKeyCases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = BaseKey(c.key, c.dstPrefix)
			}
		})
	}
}
