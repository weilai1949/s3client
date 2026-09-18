package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// 任务状态。running 之外均为终态；interrupted 表示进程在任务运行中退出，
// 由重启后的恢复流程标记（ASSESSMENT S1 / todolist #19）。
const (
	JobStatusRunning     = "running"
	JobStatusDone        = "done"
	JobStatusCancelled   = "cancelled"
	JobStatusInterrupted = "interrupted"
)

// IsTerminalJobStatus 判断任务是否已进入终态（含进程重启导致的 interrupted）。
func IsTerminalJobStatus(status string) bool {
	switch status {
	case JobStatusDone, JobStatusCancelled, JobStatusInterrupted:
		return true
	default:
		return false
	}
}

// JobRecord 是 Job 的可序列化快照，用于跨重启保留任务清单：
// 「复制成功但源未删」这类移动任务若在运行中被打断，重启后可据此对账。
type JobRecord struct {
	ID       string      `json:"id"`
	Created  time.Time   `json:"created"`
	Total    int         `json:"total"`
	Status   string      `json:"status"`
	Progress JobProgress `json:"progress"`
	Result   JobResult   `json:"result"`
}

// JobPersister 是任务清单的落盘抽象（Load 无既有数据时返回 (nil, nil)）。
// 由调用方注入；不注入则为纯内存行为，与历史一致。
type JobPersister interface {
	Load() ([]JobRecord, error)
	Save(recs []JobRecord) error
}

// FileJobPersister 把任务清单原子写入单个 JSON 文件。
//
// 为什么自带原子写而不复用 store/atomic.go：`service` 反向依赖 `store` 会造成
// 分层倒置（docs/architecture.md：`handler → service → s3wrap`）。两处实现都遵循
// 「临时文件 → rename → 0600」，且同样以包级变量暴露写入调用以便测试注入故障。
type FileJobPersister struct {
	path string
	mu   sync.Mutex // 串行化同文件并发写，避免 rename 相互覆盖
}

// NewFileJobPersister 创建文件任务清单持久化器（父目录需已存在）。
func NewFileJobPersister(path string) *FileJobPersister {
	return &FileJobPersister{path: path}
}

// Load 读取任务清单；文件不存在或为空时返回 (nil, nil)。
func (p *FileJobPersister) Load() ([]JobRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := os.ReadFile(p.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read job file: %w", err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	var recs []JobRecord
	if err := json.Unmarshal(b, &recs); err != nil {
		return nil, fmt.Errorf("parse job file: %w", err)
	}
	return recs, nil
}

// Save 原子写回整个清单（任务数有界，整体重写最简单且天然一致）。
func (p *FileJobPersister) Save(recs []JobRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := jobMarshal(recs)
	if err != nil {
		return fmt.Errorf("marshal job file: %w", err)
	}
	return atomicWriteFile(p.path, b)
}

// 写入调用暴露为包级变量，供测试注入故障覆盖错误分支（生产保持纯 stdlib）。
// json.Marshal 对 JobRecord 这类纯值结构不会失败，故同样注入以便覆盖。
var (
	jobMarshal   = json.Marshal
	jobWriteFile = os.WriteFile
	jobRename    = os.Rename
	jobRemove    = os.Remove
)

// atomicWriteFile 原子写：同目录临时文件写满后 rename 覆盖，最后收紧 0600。
// rename 失败时清理临时文件；成功后临时文件已不存在，无需清理。
func atomicWriteFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := jobWriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := jobRename(tmp, path); err != nil {
		_ = jobRemove(tmp)
		return err
	}
	return nil
}
