package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/atomicfile"
)

// shutdownFileName 是优雅关停耗时的落盘文件名（位于 S3C_DATA_DIR 下）。
//
// 为什么需要落盘：关停发生在进程退出**之前**，`/api/metrics` 的 scrape 永远赶不上
// 关停后的值（监听已关闭），进程内计数器对「优雅关停耗时」这个指标天然不可读
// （ROADMAP #18）。因此把最近一次的耗时写进数据目录，由**下一次启动**载入后暴露，
// 与 `jobs.json` 跨重启对账是同一取舍。
const shutdownFileName = "shutdown.json"

// shutdownRecord 是 shutdown.json 的载荷：耗时以**微秒**记录（毫秒会在空闲关停时
// 截断成 0，与「无记录」混淆），保持最小面。
type shutdownRecord struct {
	DurationUs int64 `json:"durationUs"`
}

// LoadLastShutdown 在启动时读入上一次优雅关停的耗时，作为
// `s3c_last_shutdown_duration_seconds` 的当前值。
//
// 文件缺失 / 不可读 / 格式错误一律回退 0（= 无记录）：关停耗时是诊断信息，
// 不参与任何启动决策，缺失绝不能阻断服务启动（同 jobs.json 清单的降级口径）。
func (h *Handler) LoadLastShutdown() {
	if h.dataDir == "" {
		return
	}
	raw, err := os.ReadFile(filepath.Join(h.dataDir, shutdownFileName))
	if err != nil {
		return
	}
	var rec shutdownRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return
	}
	h.lastShutdown = time.Duration(rec.DurationUs) * time.Microsecond
}

// RecordShutdown 记录一次已完成的优雅关停：更新内存值并落盘，供下一次启动暴露。
// 未设置数据目录时只更新内存值（测试 / 嵌入场景），不向当前工作目录写文件。
func (h *Handler) RecordShutdown(d time.Duration) {
	h.lastShutdown = d
	if h.dataDir == "" {
		return
	}
	// shutdownRecord 只含 int64 字段，Marshal 不可能失败；写失败只影响下一次启动的
	// 诊断值（读到 0），关停本身不得被它拖累 —— 与 jobs.json 落盘的降级口径一致。
	raw, _ := json.Marshal(shutdownRecord{DurationUs: d.Microseconds()})
	_ = atomicfile.WriteFile(filepath.Join(h.dataDir, shutdownFileName), raw)
}
