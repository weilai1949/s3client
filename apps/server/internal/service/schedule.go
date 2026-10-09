package service

// schedule.go —— 计划任务模型与校验（ROADMAP §三 #6「计划任务与持续同步」）。
//
// 计划把一次性的 migrate/sync 升级为「桶 → 桶定时备份」：Schedule 描述
// 「谁 → 谁、按什么 cron、用哪种比对模式」，调度循环（scheduler.go）负责按
// NextRunAt 到点触发，触发后复用既有 JobRegistry 的落盘 / SSE / 重启恢复。

import (
	"fmt"
	"time"
)

// Schedule 一条定时同步计划。JSON 字段名是落盘（schedules.json）与 API 的契约面。
type Schedule struct {
	ID              string `json:"id"`
	SourceAccountID string `json:"sourceAccountId"`
	SourceBucket    string `json:"sourceBucket"`
	SourcePrefix    string `json:"sourcePrefix,omitempty"`
	TargetAccountID string `json:"targetAccountId"`
	TargetBucket    string `json:"targetBucket"`
	TargetPrefix    string `json:"targetPrefix,omitempty"`
	// Mode：etag（默认）/ size_mtime / always；空串在 Validate/运行时按 etag 处理。
	Mode CompareMode `json:"mode"`
	// Cron 是 5 字段表达式（见 schedule_cron.go），保存原串用于回显。
	Cron string `json:"cron"`
	// Enabled=false 只冻结调度，不删除计划；手动 run 不受影响。
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	// NextRunAt 下一次自动触发时刻（服务端计算）。零值 = 尚未排期（未启用或永不触发）。
	NextRunAt time.Time `json:"nextRunAt"`
	// LastRunAt 最近一次触发时刻（手动 + 自动）；从未运行时省略。
	LastRunAt time.Time `json:"lastRunAt,omitzero"`
	// LastJobID 最近一次触发产生的异步任务 id（前端跳转进度用）；从未运行时省略。
	LastJobID string `json:"lastJobId,omitempty"`
	// LastError 最近一次触发失败的原因（成功后清除）；正常时省略。
	LastError string `json:"lastError,omitempty"`
}

// Validate 校验计划的结构合法性；now 用于判定 cron 是否可触发。
//
// 错误信息点名字段（API 直接回传 400 正文），必须包含：
//   - 必填：sourceAccountId / sourceBucket / targetAccountId / targetBucket / cron；
//   - mode ∈ {空, etag, size_mtime, always}（空 = etag，与 syncHandler 同口径）；
//   - cron 可解析，且在视界内存在下一次触发（拒绝 `0 0 30 2 *` 这类永不可达计划）。
func (s *Schedule) Validate(now time.Time) error {
	switch {
	case s.SourceAccountID == "":
		return fmt.Errorf("sourceAccountId is required")
	case s.SourceBucket == "":
		return fmt.Errorf("sourceBucket is required")
	case s.TargetAccountID == "":
		return fmt.Errorf("targetAccountId is required")
	case s.TargetBucket == "":
		return fmt.Errorf("targetBucket is required")
	case s.Cron == "":
		return fmt.Errorf("cron is required")
	}
	if s.Mode != "" && s.Mode != CompareETag && s.Mode != CompareSizeTime && s.Mode != CompareAlways {
		return fmt.Errorf("mode must be etag, size_mtime or always")
	}
	c, err := ParseCron(s.Cron)
	if err != nil {
		return fmt.Errorf("cron: %w", err)
	}
	if c.Next(now).IsZero() {
		return fmt.Errorf("cron %q never matches within the search horizon", s.Cron)
	}
	return nil
}

// ComputeNext 计算自 now 起的下一次触发时刻；cron 非法或永不触发返回零值。
func (s *Schedule) ComputeNext(now time.Time) time.Time {
	c, err := ParseCron(s.Cron)
	if err != nil {
		return time.Time{}
	}
	return c.Next(now)
}
