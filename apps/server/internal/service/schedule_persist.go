package service

// schedule_persist.go —— 计划清单落盘（schedules.json），策略沿用 job_persist.go：
// 原子写（internal/atomicfile 叶子包，0600）、Load 失败降级不阻塞启动、
// Save 失败静默（下一次状态变更重试）。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/weilai1949/s3client/apps/server/internal/atomicfile"
)

// SchedulePersister 是计划清单的落盘抽象（Load 无既有数据时返回 (nil, nil)）。
type SchedulePersister interface {
	Load() ([]Schedule, error)
	Save(recs []Schedule) error
}

// FileSchedulePersister 把计划清单原子写入单个 JSON 文件。
type FileSchedulePersister struct {
	path string
	mu   sync.Mutex // 串行化同文件并发写，避免 rename 相互覆盖
}

// NewFileSchedulePersister 创建文件计划清单持久化器（父目录需已存在）。
func NewFileSchedulePersister(path string) *FileSchedulePersister {
	return &FileSchedulePersister{path: path}
}

// Load 读取计划清单；文件不存在或为空时返回 (nil, nil)。
//
// 解析失败时**不再静默丢弃**：把损坏文件改名为 `<path>.corrupt` 保留现场后返回错误
// （KNOWN_ISSUES #83）。原实现返回空清单，调度器下一次 Save 会用空列表覆盖损坏
// 但可能可恢复的内容——一旦发生就永久丢计划清单。
func (p *FileSchedulePersister) Load() ([]Schedule, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := os.ReadFile(p.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read schedule file: %w", err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	var recs []Schedule
	if err := json.Unmarshal(b, &recs); err != nil {
		// 尽力保留现场：改名失败也不改变「降级为空清单」的结果。
		_ = os.Rename(p.path, p.path+".corrupt")
		return nil, fmt.Errorf("parse schedule file: %w", err)
	}
	return recs, nil
}

// Save 原子写回整个清单（计划数量有界，整体重写最简单且天然一致）。
func (p *FileSchedulePersister) Save(recs []Schedule) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := scheduleMarshal(recs)
	if err != nil {
		return fmt.Errorf("marshal schedule file: %w", err)
	}
	return atomicfile.WriteFile(p.path, b)
}

// scheduleMarshal 暴露为包级变量，供测试注入故障覆盖 marshal 错误分支
// （json.Marshal 对 Schedule 纯值结构不会失败，同 jobMarshal 的注入理由）。
var scheduleMarshal = json.Marshal
