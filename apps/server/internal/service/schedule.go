package service

// schedule.go —— 计划任务模型与校验（ROADMAP §三 #6「计划任务与持续同步」）。
//
// 计划把一次性的 migrate/sync 升级为「桶 → 桶定时备份」：Schedule 描述
// 「谁 → 谁、按什么 cron、用哪种比对模式」，调度循环（scheduler.go）负责按
// NextRunAt 到点触发，触发后复用既有 JobRegistry 的落盘 / SSE / 重启恢复。

import (
	"errors"
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

// ScheduleValidationError 表示计划参数校验失败（KNOWN_ISSUES #80）。
//
// Msg 是**固定**的、不含任何用户输入的客户端文案——handler 只允许把它写进
// 400 响应正文（`error_echo_gate` 机械守住 err.Error() 直接透传）。
// Cause 是供服务端日志与排查的详细原因（可能含用户提交的 cron 原文），
// **不得**用作 HTTP 响应正文。
type ScheduleValidationError struct {
	Msg   string
	Cause error
}

func (e *ScheduleValidationError) Error() string {
	if e.Cause != nil {
		return e.Msg + ": " + e.Cause.Error()
	}
	return e.Msg
}

// Unwrap 暴露底层原因，供 errors.Is/As 继续下钻。
func (e *ScheduleValidationError) Unwrap() error { return e.Cause }

// ScheduleValidationMessage 返回计划校验失败的固定客户端文案：校验错误取 Msg
// （不含用户输入），其余错误回退通用文案——绝不把任意 error 正文透传给客户端
// （KNOWN_ISSUES #80）。handler 因此无需再判断错误种类。
func ScheduleValidationMessage(err error) string {
	var ve *ScheduleValidationError
	if errors.As(err, &ve) {
		return ve.Msg
	}
	return "invalid schedule configuration"
}

// Validate 校验计划的结构合法性；now 用于判定 cron 是否可触发。
//
// 失败一律返回 *ScheduleValidationError：Msg 点名字段但不回显用户输入，
// Cause 保留解析细节（如非法 cron 原文）仅供日志。校验点：
//   - 必填：sourceAccountId / sourceBucket / targetAccountId / targetBucket / cron；
//   - mode ∈ {空, etag, size_mtime, always}（空 = etag，与 syncHandler 同口径）；
//   - cron 可解析，且在视界内存在下一次触发（拒绝 `0 0 30 2 *` 这类永不可达计划）。
func (s *Schedule) Validate(now time.Time) error {
	switch {
	case s.SourceAccountID == "":
		return &ScheduleValidationError{Msg: "sourceAccountId is required"}
	case s.SourceBucket == "":
		return &ScheduleValidationError{Msg: "sourceBucket is required"}
	case s.TargetAccountID == "":
		return &ScheduleValidationError{Msg: "targetAccountId is required"}
	case s.TargetBucket == "":
		return &ScheduleValidationError{Msg: "targetBucket is required"}
	case s.Cron == "":
		return &ScheduleValidationError{Msg: "cron is required"}
	}
	if s.Mode != "" && s.Mode != CompareETag && s.Mode != CompareSizeTime && s.Mode != CompareAlways {
		return &ScheduleValidationError{Msg: "mode must be etag, size_mtime or always"}
	}
	c, err := ParseCron(s.Cron)
	if err != nil {
		return &ScheduleValidationError{Msg: "cron is invalid", Cause: err}
	}
	if c.Next(now).IsZero() {
		return &ScheduleValidationError{Msg: "cron expression never matches within the search horizon"}
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
