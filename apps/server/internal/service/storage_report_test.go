package service

// storage_report_test.go —— ROADMAP §三 #7 聚合内核的行为测试：
// 按存储类 / 顶层前缀聚合、月成本、低频与归档建议、空类归一、未知类回退。
// 端到端（HTTP）行为由 handler 侧 storage_report_test.go 覆盖，此处补聚合细节与排序稳定性。

import (
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// srAggGib 1 GiB。
const srAggGib = int64(1) << 30

func TestAggregateStorageReportTiesAndAggregation(t *testing.T) {
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	items := []s3wrap.ObjectItem{
		// 同一 (kind,from,to) 两条 → 验证建议聚合与累加。
		{Key: "a/1", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now.Add(-120 * 24 * time.Hour)},
		{Key: "a/2", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now.Add(-120 * 24 * time.Hour)},
		// 与 STANDARD 归档合计同体积，制造「同类同体积不同源」的排序 tie。
		{Key: "b/1", Size: 2 * srAggGib, StorageClass: "ONEZONE_IA", LastModified: now.Add(-120 * 24 * time.Hour)},
		{Key: "c/1", Size: 2 * srAggGib, StorageClass: "STANDARD_IA", LastModified: now.Add(-120 * 24 * time.Hour)},
	}
	report := AggregateStorageReport(items, "", now)

	if report.ObjectCount != 4 || report.TotalSize != 6*srAggGib {
		t.Fatalf("objectCount/totalSize = %d/%d, want 4/%d", report.ObjectCount, report.TotalSize, 6*srAggGib)
	}
	// 三个类体积相同（2 GiB）→ 按类名升序：ONEZONE_IA < STANDARD < STANDARD_IA。
	wantClasses := []string{"ONEZONE_IA", "STANDARD", "STANDARD_IA"}
	if len(report.ByStorageClass) != 3 {
		t.Fatalf("byStorageClass len = %d, want 3", len(report.ByStorageClass))
	}
	for i, w := range wantClasses {
		if got := report.ByStorageClass[i].StorageClass; got != w {
			t.Errorf("byStorageClass[%d] = %s, want %s", i, got, w)
		}
	}
	// 三个前缀体积相同 → 按前缀升序：a/ < b/ < c/。
	wantPrefixes := []string{"a/", "b/", "c/"}
	if len(report.ByPrefix) != 3 {
		t.Fatalf("byPrefix len = %d, want 3", len(report.ByPrefix))
	}
	for i, w := range wantPrefixes {
		if got := report.ByPrefix[i].Prefix; got != w {
			t.Errorf("byPrefix[%d] = %s, want %s", i, got, w)
		}
	}
	// STANDARD 两条归档建议聚合为 count=2、size=2 GiB。
	if len(report.Recommendations) != 3 {
		t.Fatalf("recommendations len = %d, want 3: %#v", len(report.Recommendations), report.Recommendations)
	}
	std := report.Recommendations[1] // 三类归档同体积：ONEZONE_IA < STANDARD < STANDARD_IA。
	if std.Kind != "archive" || std.FromStorageClass != "STANDARD" || std.Count != 2 || std.Size != 2*srAggGib {
		t.Errorf("STANDARD archive = %#v, want archive count=2 size=%d", std, 2*srAggGib)
	}
}

func TestAggregateStorageReportRecommendationGating(t *testing.T) {
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	items := []s3wrap.ObjectItem{
		// 空类归一为 STANDARD，40 天 → 低频 STANDARD→STANDARD_IA。
		{Key: "x", Size: 1 * srAggGib, StorageClass: "", LastModified: now.Add(-40 * 24 * time.Hour)},
		// STANDARD_IA 40 天：不在低频目标表 → 无建议。
		{Key: "y", Size: 1 * srAggGib, StorageClass: "STANDARD_IA", LastModified: now.Add(-40 * 24 * time.Hour)},
		// GLACIER_IR 200 天：已归档 → 无建议。
		{Key: "z", Size: 1 * srAggGib, StorageClass: "GLACIER_IR", LastModified: now.Add(-200 * 24 * time.Hour)},
		// 10 天：太新 → 无建议。
		{Key: "w", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now.Add(-10 * 24 * time.Hour)},
	}
	report := AggregateStorageReport(items, "", now)

	if len(report.Recommendations) != 1 {
		t.Fatalf("recommendations = %#v, want 仅 1 条低频", report.Recommendations)
	}
	rec := report.Recommendations[0]
	if rec.Kind != "infrequent" || rec.FromStorageClass != "STANDARD" || rec.ToStorageClass != "STANDARD_IA" {
		t.Errorf("rec = %#v, want infrequent STANDARD→STANDARD_IA", rec)
	}
	if rec.Count != 1 || rec.Size != 1*srAggGib {
		t.Errorf("rec count/size = %d/%d, want 1/%d", rec.Count, rec.Size, 1*srAggGib)
	}
	// 空类按 STANDARD 计入成本（1 GiB × 0.023），总成本 = 4 类对象各 1 GiB。
	want := 1*0.023 + 1*0.0125 + 1*0.004 + 1*0.023
	if diff := report.MonthlyCost - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("monthlyCost = %v, want %v", report.MonthlyCost, want)
	}
}

func TestAggregateStorageReportPrefixLabelStripsRequestedPrefix(t *testing.T) {
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	items := []s3wrap.ObjectItem{
		{Key: "logs/2024/a.txt", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now},
		{Key: "logs/2024/b.txt", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now},
		{Key: "logs/root.txt", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now},
	}
	report := AggregateStorageReport(items, "logs/", now)
	if len(report.ByPrefix) != 2 {
		t.Fatalf("byPrefix = %#v, want 2 组", report.ByPrefix)
	}
	// 同体积（2 GiB vs 1 GiB）→ logs/2024/ 在前，基准层 logs/ 在后。
	if report.ByPrefix[0].Prefix != "logs/2024/" || report.ByPrefix[0].Count != 2 {
		t.Errorf("byPrefix[0] = %#v, want logs/2024/ count=2", report.ByPrefix[0])
	}
	if report.ByPrefix[1].Prefix != "logs/" || report.ByPrefix[1].Count != 1 {
		t.Errorf("byPrefix[1] = %#v, want logs/ count=1", report.ByPrefix[1])
	}
}

func TestAggregateStorageReportEmpty(t *testing.T) {
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	report := AggregateStorageReport(nil, "", now)
	if report.ObjectCount != 0 || report.TotalSize != 0 || report.MonthlyCost != 0 {
		t.Errorf("空报告 = %#v, want 全零", report)
	}
	if len(report.ByStorageClass) != 0 || len(report.ByPrefix) != 0 || len(report.Recommendations) != 0 {
		t.Errorf("空报告数组 = %#v, want 空", report)
	}
}

// TestAggregateStorageReportKindAndSizeOrdering 覆盖建议排序的层级：
// 跨 kind（infrequent 恒先于 archive）、同 kind 体积降序、未知存储类回退 STANDARD 价。
func TestAggregateStorageReportKindAndSizeOrdering(t *testing.T) {
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	items := []s3wrap.ObjectItem{
		// archive：合计 2 GiB。
		{Key: "a1", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now.Add(-120 * 24 * time.Hour)},
		{Key: "a2", Size: 1 * srAggGib, StorageClass: "STANDARD", LastModified: now.Add(-120 * 24 * time.Hour)},
		// infrequent：2 GiB（STANDARD）与 1 GiB（REDUCED_REDUNDANCY）→ 同 kind 体积降序。
		{Key: "i1", Size: 2 * srAggGib, StorageClass: "STANDARD", LastModified: now.Add(-40 * 24 * time.Hour)},
		{Key: "i2", Size: 1 * srAggGib, StorageClass: "REDUCED_REDUNDANCY", LastModified: now.Add(-40 * 24 * time.Hour)},
		// 未知存储类：成本按 STANDARD 价回退。
		{Key: "u", Size: 1 * srAggGib, StorageClass: "VENDOR_X", LastModified: now.Add(-1 * time.Hour)},
	}
	report := AggregateStorageReport(items, "", now)

	if len(report.Recommendations) != 3 {
		t.Fatalf("recommendations = %#v, want 3", report.Recommendations)
	}
	got := []StorageRecommendation{report.Recommendations[0], report.Recommendations[1], report.Recommendations[2]}
	if got[0].Kind != "infrequent" || got[0].FromStorageClass != "STANDARD" || got[0].Size != 2*srAggGib {
		t.Errorf("rec[0] = %#v, want infrequent STANDARD size=%d", got[0], 2*srAggGib)
	}
	if got[1].Kind != "infrequent" || got[1].FromStorageClass != "REDUCED_REDUNDANCY" || got[1].Size != 1*srAggGib {
		t.Errorf("rec[1] = %#v, want infrequent REDUCED_REDUNDANCY size=%d", got[1], 1*srAggGib)
	}
	if got[2].Kind != "archive" || got[2].Size != 2*srAggGib {
		t.Errorf("rec[2] = %#v, want archive size=%d", got[2], 2*srAggGib)
	}
	// 成本：4 GiB STANDARD @0.023 + 1 GiB REDUCED_REDUNDANCY @0.024 + 1 GiB VENDOR_X（回退 0.023）。
	want := 4*0.023 + 1*0.024 + 1*0.023
	if diff := report.MonthlyCost - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("monthlyCost = %v, want %v（未知类回退 STANDARD 价）", report.MonthlyCost, want)
	}
}
