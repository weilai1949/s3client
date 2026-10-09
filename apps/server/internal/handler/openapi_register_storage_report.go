package handler

import (
	"github.com/weilai1949/s3client/apps/server/internal/openapi"
)

// reportNumber 是「number」型 schema（金额字段用）。openapi 包未提供 number builder，
// 局部构造以免为单个端点向基础包添加无独立测试面覆盖的导出符号。
func reportNumber() *openapi.Schema { return &openapi.Schema{Type: "number"} }

// openapi_register_storage_report.go —— GET /api/accounts/{id}/storage-report 的契约登记
// （ROADMAP §三 #7 FinOps 成本看板）。响应形状与 handler 的 storageReportResponse 逐字段一致。

func registerStorageReport(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/storage-report", openapi.Op{
		Tags: []string{"objects"}, Summary: "存储分析与成本洞察（按存储类 / 前缀聚合）", OperationID: "storageReport",
		Params: []openapi.Param{acctIDParam(), refParam("Bucket"), refParam("Prefix")},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"bucket":           openapi.Str(),
			"prefix":           openapi.Str(),
			"objectCount":      openapi.Int64(),
			"totalSize":        openapi.Int64(),
			"truncated":        openapi.Bool(),
			"monthlyCost":      reportNumber(),
			"prefixGroupCount": openapi.Int(),
			"byStorageClass":   openapi.Arr(storageClassUsageSchema()),
			"byPrefix":         openapi.Arr(prefixUsageSchema()),
			"recommendations":  openapi.Arr(storageRecommendationSchema()),
		}, "bucket", "prefix", "objectCount", "totalSize", "truncated", "monthlyCost",
			"prefixGroupCount", "byStorageClass", "byPrefix", "recommendations")}},
	})
}

// storageClassUsageSchema 是按存储类聚合的用量条目。
func storageClassUsageSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"storageClass": openapi.Str(),
		"count":        openapi.Int64(),
		"size":         openapi.Int64(),
		"monthlyCost":  reportNumber(),
	}, "storageClass", "count", "size", "monthlyCost")
}

// prefixUsageSchema 是按前缀聚合的用量条目。
func prefixUsageSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"prefix": openapi.Str(),
		"count":  openapi.Int64(),
		"size":   openapi.Int64(),
	}, "prefix", "count", "size")
}

// storageRecommendationSchema 是成本优化建议条目。
func storageRecommendationSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"kind":                   openapi.EnumStr("infrequent", "archive"),
		"fromStorageClass":       openapi.Str(),
		"toStorageClass":         openapi.Str(),
		"count":                  openapi.Int64(),
		"size":                   openapi.Int64(),
		"estimatedMonthlySaving": reportNumber(),
	}, "kind", "fromStorageClass", "toStorageClass", "count", "size", "estimatedMonthlySaving")
}
