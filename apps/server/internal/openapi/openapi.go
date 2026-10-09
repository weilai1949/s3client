// Package openapi 提供轻量级的 OpenAPI 3.0 规范生成器。
//
// 设计目标：
//   - 不引入额外依赖（无 swag / 无反射），全部用显式 builder。
//   - 注册中心：路由旁登记 OpenAPI Operation，与 handler 同包，零注解散落。
//   - 输出 /api/openapi.json 作为 API 契约的单一来源（SSOT），便于客户端生成、契约测试。
//
// 使用：
//
//	api := openapi.New("s3client API", "1.0.0")
//	api.Operation("GET", "/api/health", openapi.Op{Summary: "...", ...}).
//	    Response("200", openapi.Res{JSON: openapi.Object()})
//	spec, _ := api.MarshalJSON()
package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Info 描述 API 元信息。
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Server 描述一个 server URL。
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Param 描述单个路径 / 查询 / 头参数。
// Ref 非空时输出为 {"$ref": "#/components/parameters/<name>"}（复用共享参数）。
type Param struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // path | query | header
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
	Ref         string  `json:"-"`
}

// Schema JSON Schema 子集（足够描述我们 69 个端点的形态）。
// 字段用指针 omitempty 表达「字段缺省时省略」，避免输出噪声。
type Schema struct {
	Ref         string `json:"$ref,omitempty"`
	Type        string `json:"type,omitempty"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
	Default     any    `json:"default,omitempty"`
	// Example 是 schema 级示例（`example`）。json.RawMessage 使示例以原始 JSON 原样嵌入，
	// 不必先解码成 any；调用方用 ex("...") 构造（见 handler/openapi_examples.go）。
	Example    json.RawMessage    `json:"example,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	// Nullable OAS 3.0 的 `nullable: true`：值域为「object 或 null」
	// （如 head 的 checksums——厂商不支持 / 对象无校验和时响应为 null）。
	Nullable bool `json:"nullable,omitempty"`
}

// MediaType 一个请求 / 响应体的描述。
type MediaType struct {
	Schema *Schema `json:"schema"`
	// Example 是媒体类型级示例（`example`），OpenAPI 3.0 允许任意 JSON 值。
	Example json.RawMessage `json:"example,omitempty"`
}

// Request 描述请求体。
type Request struct {
	Required bool      `json:"required,omitempty"`
	Content  MediaType `json:"content"`
}

// Response 描述单个响应。
// Ref 非空时输出为 {"$ref": "#/components/responses/<name>"}（复用共享响应，如 NotFound）。
// JSON / Example / Ref / ContentType 带 `json:"-"`：结构体默认序列化会把它们静默丢弃，让
// components.responses 里的共享响应变成无 schema 空壳（review §R15b），因此必须自定义
// MarshalJSON，与端点级渲染同形。
type Response struct {
	Description string  `json:"description,omitempty"`
	JSON        *Schema `json:"-"`
	// Example 是响应级示例（写在 content.<mediaType>.example 下）。
	Example json.RawMessage `json:"-"`
	// ContentType 覆盖响应的媒体类型（默认 application/json）。二进制 / SSE / 文本端点的
	// 2xx 响应没有 JSON schema，用它可以给出可读示例（如 application/zip）。
	ContentType string `json:"-"`
	Ref         string `json:"-"`
}

// MarshalJSON 输出 OpenAPI Response Object（$ref 或 description+content），
// 形状由 renderResponse 统一负责（与端点级 responses 共用，防止两处漂移）。
func (r Response) MarshalJSON() ([]byte, error) {
	return json.Marshal(renderResponse(r))
}

// Tag 是顶层 tags 的一个分组声明（name 必须与 Op.Tags 的取值一致）。
// Description 供人类与代码生成器识别分组语义，通常指向 docs/api.md 的对应章节。
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Op 单个操作的元数据。
type Op struct {
	Tags        []string
	Summary     string
	Description string
	OperationID string
	Deprecated  bool
	// NoAuth 显式声明该 operation **无需鉴权**，渲染为 `security: []`（覆盖文档级默认）。
	// 只有真实豁免鉴权的端点才可置 true（见 handler 的 withAuth 豁免名单）。
	NoAuth    bool
	Params    []Param
	Request   *Request
	Responses map[string]Response
}

// OpExample 是一个 operation 的示例集合：请求体示例 + 各状态码的响应示例。
//
// 与 Op 分开登记（handler 侧 applyExamples 在全部注册完成后统一附加），原因：示例是纯文档数据，
// 内联进每条 Operation(...) 调用会把注册语句撑成巨型字面量、淹没结构信息。字段为原始 JSON，
// 因此示例可以逐字节 review，也不会因 struct 往返而改变数字/转义形式。
type OpExample struct {
	Request   json.RawMessage
	Responses map[string]json.RawMessage
	// ContentTypes 覆盖指定状态码的响应媒体类型（默认 application/json）。
	// 二进制 / SSE / 文本端点的 2xx 没有 JSON schema，用它把示例挂到真实媒体类型下
	// （如 application/zip、text/event-stream），而不是错误地标成 JSON。
	ContentTypes map[string]string
}

// Registry 维护所有 path -> method -> Op 的映射。
type Registry struct {
	mu    sync.RWMutex
	paths map[string]map[string]Op
	info  Info
	srvs  []Server
	tags  []Tag
	// spec 是 MarshalJSON 的结果缓存（nil = 未缓存/已失效）。
	//
	// 背景（docs/archive/review-2026-09-19.md §6.2 P3）：注册表在启动时构建完成后不再变化，但
	// HTTPHandler 此前每次请求都全量重新 marshal 70 个 operation。这里按「变更即失效」
	// 缓存：任何注册/覆盖（Operation / SetInfo / AddServer）都置 nil，
	// 因此语义与「每次都重新 marshal」完全一致，只是消除了重复计算。
	spec []byte
}

// New 构造一个空 Registry。
func New(title, version string) *Registry {
	return &Registry{
		paths: map[string]map[string]Op{},
		info:  Info{Title: title, Version: version},
	}
}

// SetInfo 覆盖 Info。
func (r *Registry) SetInfo(info Info) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.info = info
	r.spec = nil
}

// AddServer 注册一个 server URL。
func (r *Registry) AddServer(s Server) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.srvs = append(r.srvs, s)
	r.spec = nil
}

// AddTag 声明一个顶层 tag（分组 + 说明）；声明顺序即输出顺序。
func (r *Registry) AddTag(tag Tag) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tags = append(r.tags, tag)
	r.spec = nil
}

// Operation 注册 / 覆盖一个 operation（param / response 直接放在 Op 结构里）。
// 早期的 *OpBuilder 链式 API（Param / Respond）生产零引用、仅测试使用，
// 已按死代码纪律删除。
func (r *Registry) Operation(method, path string, op Op) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.paths[path] == nil {
		r.paths[path] = map[string]Op{}
	}
	r.paths[path][strings.ToUpper(method)] = op
	r.spec = nil
}

// SetExamples 为已注册的 operation 附加请求 / 响应示例。
//
// 返回 error 而非静默忽略：示例 key（method+path）或状态码拼错时，示例会「凭空消失」且
// 生成的规范仍然合法——这正是最难发现的一类文档腐烂。调用方（handler.applyExamples）在
// 注册期即失败，门禁测试再对最终规范兜底。
func (r *Registry) SetExamples(method, path string, ex OpExample) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	m := strings.ToUpper(method)
	ops := r.paths[path]
	if ops == nil {
		return fmt.Errorf("openapi: 未注册的 path %q", path)
	}
	op, ok := ops[m]
	if !ok {
		return fmt.Errorf("openapi: 未注册的 operation %s %s", m, path)
	}
	if ex.Request != nil {
		if op.Request == nil {
			return fmt.Errorf("openapi: %s %s 无 requestBody，不能登记请求示例", m, path)
		}
		op.Request.Content.Example = ex.Request
	}
	for status, raw := range ex.Responses {
		resp, ok := op.Responses[status]
		if !ok {
			return fmt.Errorf("openapi: %s %s 未注册响应 %s", m, path, status)
		}
		resp.Example = raw
		if ct, ok := ex.ContentTypes[status]; ok {
			resp.ContentType = ct
		}
		op.Responses[status] = resp
	}
	// ContentTypes 指向未登记示例的状态码是配置错误（媒体类型会被静默忽略）。
	for status := range ex.ContentTypes {
		if _, ok := ex.Responses[status]; !ok {
			return fmt.Errorf("openapi: %s %s 的 ContentTypes 指向了未提供示例的状态码 %s", m, path, status)
		}
	}
	ops[m] = op
	r.spec = nil
	return nil
}

// MarshalJSON 输出 OpenAPI 3.0 JSON。
//
// 结果按「注册表未变更」缓存（P3）：并发调用只会计算一次，返回的字节切片是缓存副本，
// 调用方改写不会污染后续请求。任何注册动作都会置空缓存，故与「每次重新 marshal」等价。
func (r *Registry) MarshalJSON() ([]byte, error) {
	r.mu.RLock()
	if r.spec != nil {
		out := make([]byte, len(r.spec))
		copy(out, r.spec)
		r.mu.RUnlock()
		return out, nil
	}
	r.mu.RUnlock()

	b, err := r.buildSpec()
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	// 双检：并发构建时以先写入者为准（内容确定，二者字节相同）。
	if r.spec == nil {
		r.spec = b
	}
	out := make([]byte, len(r.spec))
	copy(out, r.spec)
	r.mu.Unlock()
	return out, nil
}

// buildSpec 实际渲染 OpenAPI 文档（不触碰缓存字段）。
func (r *Registry) buildSpec() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make(map[string]any, len(r.paths))
	pathKeys := make([]string, 0, len(r.paths))
	for p := range r.paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)
	for _, p := range pathKeys {
		ops := r.paths[p]
		entry := map[string]any{}
		methodKeys := make([]string, 0, len(ops))
		for m := range ops {
			methodKeys = append(methodKeys, m)
		}
		sort.Strings(methodKeys)
		for _, m := range methodKeys {
			op := ops[m]
			entry[strings.ToLower(m)] = renderOp(op)
		}
		paths[p] = entry
	}

	doc := map[string]any{
		"openapi":  "3.0.3",
		"info":     r.info,
		"security": defaultSecurityRequirement(),
		"paths":    paths,
		"components": map[string]any{
			"schemas":         sharedSchemas(),
			"securitySchemes": defaultSecurity(),
			"parameters":      sharedParams(),
			"responses":       sharedResponses(),
		},
	}
	if len(r.srvs) > 0 {
		doc["servers"] = r.srvs
	}
	if len(r.tags) > 0 {
		doc["tags"] = r.tags
	}
	return json.Marshal(doc)
}

func renderOp(op Op) map[string]any {
	out := map[string]any{}
	if op.Summary != "" {
		out["summary"] = op.Summary
	}
	if op.Description != "" {
		out["description"] = op.Description
	}
	if op.OperationID != "" {
		out["operationId"] = op.OperationID
	}
	if len(op.Tags) > 0 {
		out["tags"] = op.Tags
	}
	if op.Deprecated {
		out["deprecated"] = true
	}
	// 显式豁免：必须渲染空数组，覆盖文档级默认（缺字段会被读成「继承 bearerAuth」）。
	if op.NoAuth {
		out["security"] = []any{}
	}
	if len(op.Params) > 0 {
		out["parameters"] = renderParams(op.Params)
	}
	if op.Request != nil {
		req := map[string]any{
			"required": op.Request.Required,
			"content":  renderMedia(op.Request.Content),
		}
		out["requestBody"] = req
	}
	if len(op.Responses) > 0 {
		out["responses"] = renderResponses(op.Responses)
	}
	return out
}

func renderParams(ps []Param) []map[string]any {
	out := make([]map[string]any, 0, len(ps))
	for _, p := range ps {
		if p.Ref != "" {
			out = append(out, map[string]any{"$ref": p.Ref})
			continue
		}
		m := map[string]any{
			"name":        p.Name,
			"in":          p.In,
			"description": p.Description,
		}
		if p.Required {
			m["required"] = true
		}
		if p.Schema != nil {
			m["schema"] = p.Schema
		}
		out = append(out, m)
	}
	return out
}

func renderMedia(mt MediaType) map[string]any {
	if mt.Schema == nil {
		return map[string]any{}
	}
	media := map[string]any{"schema": mt.Schema}
	if mt.Example != nil {
		media["example"] = mt.Example
	}
	return map[string]any{
		"application/json": media,
	}
}

func renderResponses(rs map[string]Response) map[string]any {
	out := map[string]any{}
	keys := make([]string, 0, len(rs))
	for k := range rs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = renderResponse(rs[k])
	}
	return out
}

// renderResponse 渲染单个响应条目：$ref 优先（复用共享响应），否则 description + 可选
// content（schema / example）。content 的媒体类型默认 application/json，二进制 / SSE / 文本
// 端点可用 ContentType 覆盖。components.responses（经 Response.MarshalJSON）与端点级 responses
// 共用此函数——两处输出必须同形，否则契约 SSOT 会漂移（review §R15b）。
func renderResponse(r Response) map[string]any {
	if r.Ref != "" {
		return map[string]any{"$ref": r.Ref}
	}
	entry := map[string]any{"description": r.Description}
	if r.JSON != nil || r.Example != nil {
		media := map[string]any{}
		if r.JSON != nil {
			media["schema"] = r.JSON
		}
		if r.Example != nil {
			media["example"] = r.Example
		}
		ct := r.ContentType
		if ct == "" {
			ct = "application/json"
		}
		entry["content"] = map[string]any{ct: media}
	}
	return entry
}

// HTTPHandler 返回一个 http.Handler，吐出当前 Registry 的 JSON 快照。
// 输出经 MarshalJSON 缓存，重复请求不再重新渲染 70 个 operation（P3）。
func (r *Registry) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b, err := r.MarshalJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(b)
	})
}

// ---- 共享 Schema / Parameter / Response ----

// 常用类型便捷构造器。

func Str(format ...string) *Schema {
	s := &Schema{Type: "string"}
	if len(format) > 0 {
		s.Format = format[0]
	}
	return s
}
func Int() *Schema   { return &Schema{Type: "integer"} }
func Int64() *Schema { return &Schema{Type: "integer", Format: "int64"} }
func Bool() *Schema  { return &Schema{Type: "boolean"} }
func Arr(items *Schema) *Schema {
	return &Schema{Type: "array", Items: items}
}
func Obj() *Schema { return &Schema{Type: "object"} }

// Ref 构造一个指向 components 的 $ref 引用（如 "#/components/schemas/Account"）。
// 用于把共享 schema / parameter / response 片段接线到各端点，避免组件零引用。
func Ref(name string) *Schema { return &Schema{Ref: name} }

// BuildObj 把一组 (name, schema) 升格为 Object Schema，并按 optRequiredNames 标注 required。
// 设计取舍：不依赖运行时反射，所有字段显式登记，避免「struct tag 改了忘了同步文档」。
func BuildObj(props map[string]*Schema, required ...string) *Schema {
	requiredSet := map[string]bool{}
	for _, r := range required {
		requiredSet[r] = true
	}
	out := &Schema{Type: "object", Properties: map[string]*Schema{}, Required: nil}
	for k, v := range props {
		out.Properties[k] = v
		if requiredSet[k] {
			out.Required = append(out.Required, k)
		}
	}
	if len(out.Required) == 0 {
		out.Required = nil
	} else {
		sort.Strings(out.Required)
	}
	return out
}

// EnumStr 构造带枚举值的字符串 schema。
func EnumStr(values ...string) *Schema {
	e := make([]any, 0, len(values))
	for _, v := range values {
		e = append(e, v)
	}
	return &Schema{Type: "string", Enum: e}
}

// Ex 把一段 JSON 字面量转成示例值；非法 JSON 立即 panic。
// 示例是静态数据，写错应在本包第一次构造（注册表 / 规范生成）时就炸出来，而不是生成一份
// 「合法但示例字段早已改名」的契约。
func Ex(raw string) json.RawMessage {
	if !json.Valid([]byte(raw)) {
		panic("openapi: 示例不是合法 JSON: " + raw)
	}
	return json.RawMessage(raw)
}

// withExample 给 schema 附上示例，返回同一指针以支持 `"X": withExample(BuildObj(...), "...")`。
func withExample(s *Schema, raw string) *Schema {
	s.Example = Ex(raw)
	return s
}

func defaultSecurity() map[string]any {
	return map[string]any{
		"bearerAuth": map[string]any{
			"type":   "http",
			"scheme": "bearer",
			// 作用域语义（ROADMAP §三 #13）：S3C_TOKEN_SCOPES 可为单个 token 限定
			// 只读 / 桶前缀 / 账号 / 过期时间。越权返回 403，token 过期返回 401。
			"description": "S3C_TOKEN 中的 Bearer token。可通过 S3C_TOKEN_SCOPES 为单个 token 限定 readonly / prefixes（桶或桶内键前缀）/ accounts / expiresAt；越权请求返回 403，过期 token 返回 401。未在 S3C_TOKEN_SCOPES 中登记的 token 为全权。",
		},
	}
}

// defaultSecurityRequirement 是文档级默认安全要求：全部 /api/* 默认需 Bearer token。
// 真实豁免鉴权的端点由 Op.NoAuth 逐 operation 覆盖为 `security: []`
// （真值来源是 handler 的 withAuth）。
func defaultSecurityRequirement() []map[string][]string {
	return []map[string][]string{{"bearerAuth": {}}}
}

// sharedSchemas / sharedParams / sharedResponses 输出 components 下可复用的片段，
// 并被各端点以 $ref 接线（见 openapi_register_*.go）。片段保持与真实响应形状一致。
func sharedSchemas() map[string]*Schema {
	return map[string]*Schema{
		"Error": withExample(BuildObj(map[string]*Schema{
			"error": Str(),
		}, "error"), `{"error":"account not found"}`),
		"Account": withExample(BuildObj(map[string]*Schema{
			"id":             Str(),
			"name":           Str(),
			"endpoint":       Str(),
			"publicEndpoint": Str(),
			"region":         Str(),
			"accessKey":      Str(),
			"secretSet":      Bool(),
			"bucket":         Str(),
			"pathStyle":      Bool(),
			"useSSL":         Bool(),
			"createdAt":      Str("date-time"),
			"updatedAt":      Str("date-time"),
		}, "id", "name", "endpoint", "publicEndpoint", "region", "accessKey", "secretSet",
			"bucket", "pathStyle", "useSSL", "createdAt", "updatedAt"), `{"id":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","name":"minio","endpoint":"http://localhost:9000","publicEndpoint":"https://s3.example.com","region":"us-east-1","accessKey":"AKIAEXAMPLE","secretSet":true,"bucket":"my-bucket","pathStyle":true,"useSSL":false,"createdAt":"2026-09-30T05:00:00Z","updatedAt":"2026-09-30T05:00:00Z"}`),
		"Bucket": withExample(BuildObj(map[string]*Schema{
			"name":         Str(),
			"creationDate": Str("date-time"),
		}, "name", "creationDate"), `{"name":"my-bucket","creationDate":"2026-09-30T05:00:00Z"}`),
		"ObjectItem": withExample(BuildObj(map[string]*Schema{
			"key":          Str(),
			"size":         Int64(),
			"lastModified": Str("date-time"),
			"etag":         Str(),
			"storageClass": Str(),
			"isDir":        Bool(),
		}, "key", "size", "lastModified", "etag", "storageClass", "isDir"), `{"key":"docs/a.txt","size":17,"lastModified":"2026-09-30T05:00:00Z","etag":"\"9c1d2f3a4b5c6d7e\"","storageClass":"STANDARD","isDir":false}`),
		"ListObjectsResp": withExample(BuildObj(map[string]*Schema{
			"objects":        Arr(Ref("#/components/schemas/ObjectItem")),
			"commonPrefixes": Arr(Str()),
			"isTruncated":    Bool(),
			"nextToken":      Str(),
		}, "objects", "commonPrefixes", "isTruncated", "nextToken"), `{"objects":[{"key":"docs/a.txt","size":17,"lastModified":"2026-09-30T05:00:00Z","etag":"\"9c1d2f3a4b5c6d7e\"","storageClass":"STANDARD","isDir":false}],"commonPrefixes":["docs/"],"isTruncated":false,"nextToken":""}`),
	}
}

func sharedParams() map[string]any {
	return map[string]any{
		"AccountID": Param{
			Name:        "id",
			In:          "path",
			Required:    true,
			Description: "账号 UUID",
			Schema:      Str(),
		},
		"Bucket": Param{
			Name:        "bucket",
			In:          "query",
			Description: "桶名；账号有默认桶时可省略",
			Schema:      Str(),
		},
		"Prefix": Param{
			Name:        "prefix",
			In:          "query",
			Description: "前缀（目录路径）",
			Schema:      Str(),
		},
		"MaxKeys": Param{
			Name:        "maxKeys",
			In:          "query",
			Description: "单页对象数（1-1000）",
			Schema:      Int(),
		},
		"ContinuationToken": Param{
			Name:        "continuationToken",
			In:          "query",
			Description: "分页游标",
			Schema:      Str(),
		},
	}
}

func sharedResponses() map[string]any {
	return map[string]any{
		"BadRequest": Response{
			Description: "请求参数错误",
			JSON:        Obj(),
		},
		"NotFound": Response{
			Description: "资源不存在",
			JSON:        Ref("#/components/schemas/Error"),
		},
		"InternalError": Response{
			Description: "服务端内部错误",
			JSON:        Ref("#/components/schemas/Error"),
		},
		"Unauthorized": Response{
			Description: "未鉴权或鉴权失败",
		},
		"TooManyRequests": Response{
			Description: "请求频率超限",
		},
	}
}
