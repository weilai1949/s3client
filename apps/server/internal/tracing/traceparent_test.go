package tracing

import (
	"bytes"
	"strings"
	"testing"
)

const (
	testTraceIDHex = "4bf92f3577b34da6a3ce929d0e0e4736"
	testSpanIDHex  = "00f067aa0ba902b7"
)

// TestParseTraceparent W3C traceparent 解析口径表：合法继承，非法/全 0/版本未知一律
// 判定为「不可继承」（调用方据此新建 trace）。
func TestParseTraceparent(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		wantOK     bool
		wantSample bool
	}{
		{"合法且采样", "00-" + testTraceIDHex + "-" + testSpanIDHex + "-01", true, true},
		{"合法未采样", "00-" + testTraceIDHex + "-" + testSpanIDHex + "-00", true, false},
		{"其它 flag 位不影响采样判定", "00-" + testTraceIDHex + "-" + testSpanIDHex + "-02", true, false},
		{"大写十六进制可接受", "00-" + strings.ToUpper(testTraceIDHex) + "-" + strings.ToUpper(testSpanIDHex) + "-01", true, true},
		{"空串", "", false, false},
		{"版本未知 01", "01-" + testTraceIDHex + "-" + testSpanIDHex + "-01", false, false},
		{"版本未知 ff", "ff-" + testTraceIDHex + "-" + testSpanIDHex + "-01", false, false},
		{"trace id 全 0", "00-00000000000000000000000000000000-" + testSpanIDHex + "-01", false, false},
		{"span id 全 0", "00-" + testTraceIDHex + "-0000000000000000-01", false, false},
		{"trace id 过短", "00-" + testTraceIDHex[:30] + "-" + testSpanIDHex + "-01", false, false},
		{"span id 过短", "00-" + testTraceIDHex + "-" + testSpanIDHex[:14] + "-01", false, false},
		{"flags 过短", "00-" + testTraceIDHex + "-" + testSpanIDHex + "-1", false, false},
		{"trace id 非十六进制", "00-" + strings.Repeat("z", 32) + "-" + testSpanIDHex + "-01", false, false},
		{"span id 非十六进制", "00-" + testTraceIDHex + "-" + strings.Repeat("z", 16) + "-01", false, false},
		{"flags 非十六进制", "00-" + testTraceIDHex + "-" + testSpanIDHex + "-zz", false, false},
		{"多余字段", "00-" + testTraceIDHex + "-" + testSpanIDHex + "-01-extra", false, false},
		{"字段不足", "00-" + testTraceIDHex + "-" + testSpanIDHex, false, false},
	}
	for _, c := range cases {
		got, ok := parseTraceparent(c.in)
		if ok != c.wantOK {
			t.Errorf("%s: parseTraceparent(%q) ok = %v, want %v", c.name, c.in, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if hexOf(got.traceID[:]) != testTraceIDHex {
			t.Errorf("%s: traceID = %s, want %s", c.name, hexOf(got.traceID[:]), testTraceIDHex)
		}
		if hexOf(got.spanID[:]) != testSpanIDHex {
			t.Errorf("%s: spanID = %s, want %s", c.name, hexOf(got.spanID[:]), testSpanIDHex)
		}
		if got.sampled != c.wantSample {
			t.Errorf("%s: sampled = %v, want %v", c.name, got.sampled, c.wantSample)
		}
	}
}

// TestFormatTraceparentRoundTrip 断言生成的头能被同一解析器读回，且采样位随参数变化。
func TestFormatTraceparentRoundTrip(t *testing.T) {
	var traceID [16]byte
	var spanID [8]byte
	for i := range traceID {
		traceID[i] = byte(i + 1)
	}
	for i := range spanID {
		spanID[i] = byte(0xa0 + i)
	}
	sampled := formatTraceparent(traceID, spanID, true)
	tp, ok := parseTraceparent(sampled)
	if !ok {
		t.Fatalf("formatTraceparent 产物无法解析: %q", sampled)
	}
	if !tp.sampled {
		t.Error("sampled=true 应写出 01")
	}
	if !bytes.Equal(tp.traceID[:], traceID[:]) || !bytes.Equal(tp.spanID[:], spanID[:]) {
		t.Errorf("round trip 不一致: %s / %s", hexOf(tp.traceID[:]), hexOf(tp.spanID[:]))
	}
	if got, want := sampled, "00-"+hexOf(traceID[:])+"-"+hexOf(spanID[:])+"-01"; got != want {
		t.Errorf("formatTraceparent = %q, want %q", got, want)
	}
	unsampled := formatTraceparent(traceID, spanID, false)
	tp2, ok := parseTraceparent(unsampled)
	if !ok || tp2.sampled {
		t.Errorf("sampled=false 应写出 00 且解析为未采样: %q", unsampled)
	}
}
