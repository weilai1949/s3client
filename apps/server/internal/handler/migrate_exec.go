package handler

import "github.com/weilai1949/s3clinet/apps/server/internal/service"

type migrateRequest struct {
	SourceAccountID string   `json:"sourceAccountId"`
	SourceBucket    string   `json:"sourceBucket"`
	SourceKeys      []string `json:"sourceKeys"`
	TargetAccountID string   `json:"targetAccountId"`
	TargetBucket    string   `json:"targetBucket"`
	TargetPrefix    string   `json:"targetPrefix"`
}

type migrateResult struct {
	Migrated  int
	Failed    int
	LastError string
	FailKeys  []string
}

func migrateResultJSON(out migrateResult) map[string]any {
	resp := map[string]any{"migrated": out.Migrated, "failed": out.Failed}
	if out.LastError != "" {
		resp["lastError"] = out.LastError
	}
	if keys := capFailKeys(out.FailKeys); len(keys) > 0 {
		resp["failedKeys"] = keys
	}
	return resp
}

func migrateBatchJSON(out service.BatchResult) map[string]any {
	return migrateResultJSON(migrateResult{
		Migrated: out.OK, Failed: out.Failed, LastError: out.LastError, FailKeys: out.FailKeys,
	})
}

// jobResultFromBatch 把批量结果转为异步任务结果，并在此处兑现「failedKeys ≤ 200」承诺。
//
// 异步结果会落盘到 jobs.json，因此裁剪必须发生在 Finish 之前：10 万个 key 全失败时
// 未裁剪的 FailKeys 约 10 MB，会同时撑大内存、SSE 帧与持久化文件（review §7.3 D4）。
func jobResultFromBatch(out service.BatchResult) service.JobResult {
	res := service.ResultFromBatch(out)
	res.FailKeys = capFailKeys(res.FailKeys)
	return res
}
