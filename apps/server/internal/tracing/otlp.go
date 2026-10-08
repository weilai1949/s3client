package tracing

import (
	"encoding/hex"
	"strconv"
)

// OTLP/HTTP JSON 报文结构：resourceSpans → scopeSpans → spans。
//
// 编码依据 OTLP/JSON 约定：traceId / spanId 为小写十六进制字符串；64 位时间戳与
// int64 属性值编码为十进制字符串；枚举（kind）编码为整数。仅覆盖本仓库实际产出的
// 字段子集（最小自研，不追求 OTLP 全量），字段名与 OTLP proto JSON 映射逐字一致。
type otlpPayload struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource   otlpResource     `json:"resource"`
	ScopeSpans []otlpScopeSpans `json:"scopeSpans"`
}

type otlpResource struct {
	Attributes []otlpAttribute `json:"attributes"`
}

type otlpScopeSpans struct {
	Scope otlpScope  `json:"scope"`
	Spans []otlpSpan `json:"spans"`
}

type otlpScope struct {
	Name string `json:"name"`
}

type otlpSpan struct {
	TraceID           string          `json:"traceId"`
	SpanID            string          `json:"spanId"`
	ParentSpanID      string          `json:"parentSpanId,omitempty"`
	Name              string          `json:"name"`
	Kind              int             `json:"kind"`
	StartTimeUnixNano string          `json:"startTimeUnixNano"`
	EndTimeUnixNano   string          `json:"endTimeUnixNano"`
	Attributes        []otlpAttribute `json:"attributes,omitempty"`
}

type otlpAttribute struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// scopeName 是 instrumentation scope 名（OtlpScope.name）。
const scopeName = "s3client/tracing"

// buildPayload 把一批 span 组装为 OTLP JSON 报文；resource attribute 含 service.name。
func buildPayload(service string, spans []*span) otlpPayload {
	out := otlpPayload{ResourceSpans: []otlpResourceSpans{{
		Resource: otlpResource{Attributes: []otlpAttribute{{
			Key:   "service.name",
			Value: map[string]any{"stringValue": service},
		}}},
		ScopeSpans: []otlpScopeSpans{{
			Scope: otlpScope{Name: scopeName},
			Spans: make([]otlpSpan, 0, len(spans)),
		}},
	}}}
	dst := &out.ResourceSpans[0].ScopeSpans[0].Spans
	for _, sp := range spans {
		encoded := otlpSpan{
			TraceID:           hex.EncodeToString(sp.traceID[:]),
			SpanID:            hex.EncodeToString(sp.spanID[:]),
			Name:              sp.name,
			Kind:              sp.kind,
			StartTimeUnixNano: strconv.FormatInt(sp.start.UnixNano(), 10),
			EndTimeUnixNano:   strconv.FormatInt(sp.end.UnixNano(), 10),
		}
		if sp.parentID != ([8]byte{}) {
			encoded.ParentSpanID = hex.EncodeToString(sp.parentID[:])
		}
		for _, a := range sp.attrs {
			encoded.Attributes = append(encoded.Attributes, otlpAttribute{Key: a.key, Value: a.value})
		}
		*dst = append(*dst, encoded)
	}
	return out
}
