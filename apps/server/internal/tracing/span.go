package tracing

import (
	"crypto/rand"
	"strconv"
	"time"
)

// span kind（OTLP SpanKind 枚举；OTLP/JSON 要求枚举编码为整数）。
const (
	spanKindInternal = 1
	spanKindServer   = 2
)

// attribute 是 span 属性。value 直接持有 OTLP/JSON 的 value 对象形状
// （如 `{"stringValue":"GET"}` / `{"intValue":"200"}`），属性在产生处定型，
// 导出侧不再做类型分支。
type attribute struct {
	key   string
	value any
}

// span 是一条 trace 中的一个 span；times 用零值表示未结束（ended 判定见 otlp.go）。
type span struct {
	traceID  [16]byte
	spanID   [8]byte
	parentID [8]byte
	name     string
	kind     int
	start    time.Time
	end      time.Time
	sampled  bool
	attrs    []attribute
}

// setStr 追加字符串属性。
func (s *span) setStr(key, value string) {
	s.attrs = append(s.attrs, attribute{key: key, value: map[string]any{"stringValue": value}})
}

// setInt 追加整数属性（OTLP/JSON 的 int64 按十进制字符串编码）。
func (s *span) setInt(key string, value int) {
	s.attrs = append(s.attrs, attribute{key: key, value: map[string]any{"intValue": strconv.Itoa(value)}})
}

// newTraceID 生成非全 0 的 trace id：读出 16 字节后强制最低位置 1。
// crypto/rand 读取失败的概率可忽略（读失败时缓冲区保持全 0，置位后仍是合法 id）。
func newTraceID() [16]byte {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[15] |= 1
	return b
}

// newSpanID 生成非全 0 的 span id（口径同 newTraceID）。
func newSpanID() [8]byte {
	var b [8]byte
	_, _ = rand.Read(b[:])
	b[7] |= 1
	return b
}
