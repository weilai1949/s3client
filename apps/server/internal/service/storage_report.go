package service

import (
	"sort"
	"strings"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// storage_report.go —— ROADMAP §三 #7「FinOps：存储分析与成本洞察」的聚合内核。
//
// 分层口径：service 只做**纯聚合**（给定已列举的对象元数据 + 前缀 + 基准时刻，
// 产出按存储类 / 顶层前缀的用量与成本估算、低频/归档建议）；列举、分页与 HTTP
// 校验留在 handler（页数 / 对象数硬上限见 handler 的 storageReportMax*）。
//
// 成本口径（便于对外解释、也便于测试钉住）：
//   - 体积按 GiB 计费（1 GiB = 1<<30 字节），与 S3 定价单位一致；
//   - 定价表见 storageClassPrices（USD / GiB / 月，取自 AWS S3 us-east-1 公开价，
//     供**量级估算**用，不是账单真值）；未知存储类回退 STANDARD 价，避免出现 0 成本；
//   - 建议的「预计月省」= 体积 ×（源类价 − 目标类价），恒为非负。

// 定价与阈值常量。
const (
	// reportStandardPrice 是 STANDARD 的 GiB/月单价，未知存储类的回退价。
	reportStandardPrice = 0.023
	// reportGib 是计费体积单位（1 GiB）。
	reportGib = int64(1) << 30
	// 建议阈值：30 天以上未修改视为低频候选，90 天以上视为归档候选。
	reportInfrequentMinAge = 30 * 24 * time.Hour
	reportArchiveMinAge    = 90 * 24 * time.Hour
)

// storageClassPrices 是各存储类的 GiB/月单价（USD）。枚举成员即使当前无对象引用
// 也保留（契约的一部分，见 AGENTS.md 枚举豁免）。
var storageClassPrices = map[string]float64{
	"STANDARD":            reportStandardPrice,
	"REDUCED_REDUNDANCY":  0.024,
	"STANDARD_IA":         0.0125,
	"ONEZONE_IA":          0.01,
	"INTELLIGENT_TIERING": reportStandardPrice,
	"GLACIER":             0.0036,
	"GLACIER_IR":          0.004,
	"DEEP_ARCHIVE":        0.00099,
	"EXPRESS_ONEZONE":     0.016,
}

// 建议映射：存储类 → 目标类。不在表中的类不产生对应建议（无法可靠推断）。
// 低频候选：仍属「热」层且高于 STANDARD_IA 单价者；归档候选：比 GLACIER_IR 更热者。
var (
	reportInfrequentTargets = map[string]string{
		"STANDARD":            "STANDARD_IA",
		"REDUCED_REDUNDANCY":  "STANDARD_IA",
		"INTELLIGENT_TIERING": "STANDARD_IA",
	}
	reportArchiveTargets = map[string]string{
		"STANDARD":            "GLACIER_IR",
		"REDUCED_REDUNDANCY":  "GLACIER_IR",
		"STANDARD_IA":         "GLACIER_IR",
		"ONEZONE_IA":          "GLACIER_IR",
		"INTELLIGENT_TIERING": "GLACIER_IR",
	}
)

// StorageClassUsage 是按存储类聚合的用量与月成本估算。
type StorageClassUsage struct {
	StorageClass string  `json:"storageClass"`
	Count        int64   `json:"count"`
	Size         int64   `json:"size"`
	MonthlyCost  float64 `json:"monthlyCost"`
}

// PrefixUsage 是按「首层前缀」聚合的用量。
type PrefixUsage struct {
	Prefix string `json:"prefix"`
	Count  int64  `json:"count"`
	Size   int64  `json:"size"`
}

// StorageRecommendation 是一条成本优化建议（按 kind+源类+目标类聚合）。
type StorageRecommendation struct {
	Kind                   string  `json:"kind"`
	FromStorageClass       string  `json:"fromStorageClass"`
	ToStorageClass         string  `json:"toStorageClass"`
	Count                  int64   `json:"count"`
	Size                   int64   `json:"size"`
	EstimatedMonthlySaving float64 `json:"estimatedMonthlySaving"`
}

// StorageReport 是一次成本报告的聚合结果。
type StorageReport struct {
	ObjectCount     int64
	TotalSize       int64
	MonthlyCost     float64
	ByStorageClass  []StorageClassUsage
	ByPrefix        []PrefixUsage
	Recommendations []StorageRecommendation
}

// classAgg 是聚合循环内部的累加器。
type classAgg struct {
	count int64
	size  int64
}

// AggregateStorageReport 把一次列举得到的对象元数据聚合为成本报告。
// prefix 为本次列举使用的前缀（用于把 byPrefix 的标签还原为完整 key 前缀）；
// now 为年龄判定基准（注入以便确定性测试）。返回的切片恒非 nil（JSON 序列化为 []）。
func AggregateStorageReport(items []s3wrap.ObjectItem, prefix string, now time.Time) StorageReport {
	report := StorageReport{
		ByStorageClass:  make([]StorageClassUsage, 0),
		ByPrefix:        make([]PrefixUsage, 0),
		Recommendations: make([]StorageRecommendation, 0),
	}
	classAggs := map[string]*classAgg{}
	prefixAggs := map[string]*PrefixUsage{}
	recAggs := map[string]*StorageRecommendation{}

	for _, o := range items {
		class := normalizeStorageClass(o.StorageClass)
		report.ObjectCount++
		report.TotalSize += o.Size

		ca := classAggs[class]
		if ca == nil {
			ca = &classAgg{}
			classAggs[class] = ca
		}
		ca.count++
		ca.size += o.Size

		label := reportPrefixLabel(prefix, o.Key)
		pu := prefixAggs[label]
		if pu == nil {
			pu = &PrefixUsage{Prefix: label}
			prefixAggs[label] = pu
		}
		pu.Count++
		pu.Size += o.Size

		if rec := reportRecommendation(o, class, now); rec != nil {
			key := rec.Kind + "\x00" + rec.FromStorageClass + "\x00" + rec.ToStorageClass
			ra := recAggs[key]
			if ra == nil {
				ra = &StorageRecommendation{
					Kind: rec.Kind, FromStorageClass: rec.FromStorageClass, ToStorageClass: rec.ToStorageClass,
				}
				recAggs[key] = ra
			}
			ra.Count++
			ra.Size += rec.Size
			ra.EstimatedMonthlySaving += rec.EstimatedMonthlySaving
		}
	}

	for class, ca := range classAggs {
		cost := float64(ca.size) / float64(reportGib) * storageClassPrice(class)
		report.MonthlyCost += cost
		report.ByStorageClass = append(report.ByStorageClass, StorageClassUsage{
			StorageClass: class, Count: ca.count, Size: ca.size, MonthlyCost: cost,
		})
	}
	sort.Slice(report.ByStorageClass, func(i, j int) bool {
		if report.ByStorageClass[i].Size != report.ByStorageClass[j].Size {
			return report.ByStorageClass[i].Size > report.ByStorageClass[j].Size
		}
		return report.ByStorageClass[i].StorageClass < report.ByStorageClass[j].StorageClass
	})

	for _, pu := range prefixAggs {
		report.ByPrefix = append(report.ByPrefix, *pu)
	}
	sort.Slice(report.ByPrefix, func(i, j int) bool {
		if report.ByPrefix[i].Size != report.ByPrefix[j].Size {
			return report.ByPrefix[i].Size > report.ByPrefix[j].Size
		}
		return report.ByPrefix[i].Prefix < report.ByPrefix[j].Prefix
	})

	for _, ra := range recAggs {
		report.Recommendations = append(report.Recommendations, *ra)
	}
	sort.Slice(report.Recommendations, func(i, j int) bool {
		if report.Recommendations[i].Kind != report.Recommendations[j].Kind {
			// infrequent 恒排在 archive 之前（"archive" < "infrequent"，故显式判序而非字典序）。
			return report.Recommendations[i].Kind == "infrequent"
		}
		if report.Recommendations[i].Size != report.Recommendations[j].Size {
			return report.Recommendations[i].Size > report.Recommendations[j].Size
		}
		return report.Recommendations[i].FromStorageClass < report.Recommendations[j].FromStorageClass
	})

	return report
}

// normalizeStorageClass 把空存储类视为 STANDARD（ListObjectsV2 对标准层常省略该字段）。
func normalizeStorageClass(class string) string {
	if class == "" {
		return "STANDARD"
	}
	return class
}

// storageClassPrice 返回存储类单价；未知类回退 STANDARD。
func storageClassPrice(class string) float64 {
	if p, ok := storageClassPrices[class]; ok {
		return p
	}
	return reportStandardPrice
}

// reportPrefixLabel 把 key 归到「本次列举前缀之下的首层」标签：取相对路径的首个 '/'
// （含），无 '/' 的对象归到前缀本身（基准层）。label 为完整前缀（含请求前缀）。
func reportPrefixLabel(prefix, key string) string {
	rel := strings.TrimPrefix(key, prefix)
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		return prefix + rel[:i+1]
	}
	return prefix
}

// reportRecommendation 依据对象年龄与存储类给出一条建议候选（未聚合的单项），
// 不可建议时返回 nil。归档优先于低频（≥90 天不再给低频建议）。
func reportRecommendation(o s3wrap.ObjectItem, class string, now time.Time) *StorageRecommendation {
	age := now.Sub(o.LastModified)
	if age >= reportArchiveMinAge {
		to, ok := reportArchiveTargets[class]
		if !ok {
			return nil
		}
		return &StorageRecommendation{
			Kind: "archive", FromStorageClass: class, ToStorageClass: to,
			Size:                   o.Size,
			EstimatedMonthlySaving: reportSaving(o.Size, class, to),
		}
	}
	if age >= reportInfrequentMinAge {
		to, ok := reportInfrequentTargets[class]
		if !ok {
			return nil
		}
		return &StorageRecommendation{
			Kind: "infrequent", FromStorageClass: class, ToStorageClass: to,
			Size:                   o.Size,
			EstimatedMonthlySaving: reportSaving(o.Size, class, to),
		}
	}
	return nil
}

// reportSaving 估算把 size 字节从 from 类迁到 to 类每月可省的费用。
// 两个映射表保证 from 的单价严格高于 to，故结果恒为正。
func reportSaving(size int64, from, to string) float64 {
	return float64(size) / float64(reportGib) * (storageClassPrice(from) - storageClassPrice(to))
}
