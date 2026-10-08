package tracing

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// traceparent 是 W3C Trace Context `traceparent` 头的解析结果。
type traceparent struct {
	traceID [16]byte
	spanID  [8]byte
	sampled bool
}

// parseTraceparent 解析 `00-<32hex trace-id>-<16hex span-id>-<2hex flags>`。
//
// 版本非 `00`、字段数 / 长度不符、非十六进制、trace id 或 span id 全 0，一律判定为
// 「不可继承」（ok=false）：调用方据此新建 trace，而不是把畸形值塞进导出报文。
// 采样位取 flags 最低位（`01` = 采样）；其余 flag 位当前保留、不影响判定。
func parseTraceparent(h string) (traceparent, bool) {
	var tp traceparent
	parts := strings.Split(h, "-")
	if len(parts) != 4 || parts[0] != "00" {
		return tp, false
	}
	if len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return tp, false
	}
	traceID, err := hex.DecodeString(parts[1])
	if err != nil {
		return tp, false
	}
	spanID, err := hex.DecodeString(parts[2])
	if err != nil {
		return tp, false
	}
	flags, err := hex.DecodeString(parts[3])
	if err != nil {
		return tp, false
	}
	copy(tp.traceID[:], traceID)
	copy(tp.spanID[:], spanID)
	if tp.traceID == ([16]byte{}) || tp.spanID == ([8]byte{}) {
		return tp, false
	}
	tp.sampled = flags[0]&0x01 == 1
	return tp, true
}

// formatTraceparent 生成小写十六进制的 W3C traceparent 头。
func formatTraceparent(traceID [16]byte, spanID [8]byte, sampled bool) string {
	flags := "00"
	if sampled {
		flags = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", hex.EncodeToString(traceID[:]), hex.EncodeToString(spanID[:]), flags)
}
