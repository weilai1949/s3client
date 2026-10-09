package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/service"
)

// storage_report.go —— GET /api/accounts/{id}/storage-report（ROADMAP §三 #7 FinOps 成本看板）。
//
// 列举桶（可限定前缀）后交给 service.AggregateStorageReport 聚合，返回按存储类 / 顶层前缀的
// 用量与月成本估算、低频与归档建议。三层硬上限与 sync / delete-prefix 同口径：页数上限、
// 100k 对象上限、token 缺失或不前进即停并标记 truncated，避免畸形对端把请求拖成无界循环。

const (
	// storageReportMaxPages 是列举页数硬上限（与 service.listMaxPages 同值）。
	storageReportMaxPages = 100
	// storageReportMaxObjects 是收录对象数硬上限（与 service.listMaxTotal 同值）。
	storageReportMaxObjects = 100_000
	// storageReportPageSize 是单页对象数（S3 ListObjectsV2 上限 1000）。
	storageReportPageSize = 1000
)

// storageReportResponse 是成本报告的对外响应体；字段与 OpenAPI 注册表逐一对应。
type storageReportResponse struct {
	Bucket           string                          `json:"bucket"`
	Prefix           string                          `json:"prefix"`
	ObjectCount      int64                           `json:"objectCount"`
	TotalSize        int64                           `json:"totalSize"`
	Truncated        bool                            `json:"truncated"`
	MonthlyCost      float64                         `json:"monthlyCost"`
	PrefixGroupCount int                             `json:"prefixGroupCount"`
	ByStorageClass   []service.StorageClassUsage     `json:"byStorageClass"`
	ByPrefix         []service.PrefixUsage           `json:"byPrefix"`
	Recommendations  []service.StorageRecommendation `json:"recommendations"`
}

// storageReport 生成桶（可限定前缀）的存储分析与成本报告。
func (h *Handler) storageReport(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	prefix := q.Get("prefix")
	bucket := q.Get("bucket")
	if bucket, ok = h.bucketOr(w, acc, bucket); !ok {
		return
	}
	items, truncated, err := collectStorageReportObjects(r.Context(), client, bucket, prefix)
	if err != nil {
		h.writeInternalErr(w, err, "storage report failed")
		return
	}
	report := service.AggregateStorageReport(items, prefix, time.Now())
	h.writeJSON(w, http.StatusOK, storageReportResponse{
		Bucket:           bucket,
		Prefix:           prefix,
		ObjectCount:      report.ObjectCount,
		TotalSize:        report.TotalSize,
		Truncated:        truncated,
		MonthlyCost:      report.MonthlyCost,
		PrefixGroupCount: len(report.ByPrefix),
		ByStorageClass:   report.ByStorageClass,
		ByPrefix:         report.ByPrefix,
		Recommendations:  report.Recommendations,
	})
}

// collectStorageReportObjects 分页列举 prefix 下的对象元数据（含存储类与修改时间）。
// 返回的 truncated 表示「有对象未被收录」，四种触发与 service.listAll / deletePrefix 一致：
// 本页装不下、收满 100k、对端声称未完但 token 缺失/不前进、页数上限耗尽。
func collectStorageReportObjects(ctx context.Context, client *s3wrap.Client, bucket, prefix string) ([]s3wrap.ObjectItem, bool, error) {
	out := make([]s3wrap.ObjectItem, 0, 256)
	token := ""
	for page := 0; page < storageReportMaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return out, false, err
		}
		p, err := client.ListObjectsPage(ctx, bucket, prefix, "", token, "", storageReportPageSize)
		if err != nil {
			return out, false, err
		}
		dropped := false
		for _, o := range p.Objects {
			if len(out) >= storageReportMaxObjects {
				dropped = true // 本页还有对象装不进上限：明确截断，不得静默丢弃
				break
			}
			out = append(out, o)
		}
		if dropped {
			return out, true, nil
		}
		if len(out) >= storageReportMaxObjects {
			// 正好收满且本页无丢弃：对端还声称有下一页才算截断（正好等于上限且对端
			// 声明列举完成 ⇒ 全部收录）。
			return out, p.IsTruncated, nil
		}
		if !p.IsTruncated {
			return out, false, nil
		}
		// token 缺失或不前进必须停并标记截断，否则空转。
		if p.NextToken == "" || p.NextToken == token {
			return out, true, nil
		}
		token = p.NextToken
	}
	return out, true, nil
}
