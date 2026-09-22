// Package openapi 提供轻量级的 OpenAPI 3.0 规范生成器。
//
// 设计目标：
//   - 不引入额外依赖（无 swag / 无反射），全部用显式 builder。
//   - 注册中心：路由旁登记 OpenAPI Operation，与 handler 同包，零注解散落。
//   - 输出 /api/openapi.json 作为 API 契约的单一来源（SSOT），便于客户端生成、契约测试。
//
// 使用：
//
//	api := openapi.New("s3clinet API", "1.0.0")
//	api.Operation("GET", "/api/health", openapi.Op{Summary: "...", ...}).
//	    Response("200", openapi.Res{JSON: openapi.Object()})
//	spec, _ := api.MarshalJSON()
package openapi

import (
	"encoding/json"
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
	Ref         string             `json:"$ref,omitempty"`
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Enum        []any              `json:"enum,omitempty"`
	Default     any                `json:"default,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
}

// MediaType 一个请求 / 响应体的描述。
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Request 描述请求体。
type Request struct {
	Required bool      `json:"required,omitempty"`
	Content  MediaType `json:"content"`
}

// Response 描述单个响应。
// Ref 非空时输出为 {"$ref": "#/components/responses/<name>"}（复用共享响应，如 NotFound）。
type Response struct {
	Description string  `json:"description,omitempty"`
	JSON        *Schema `json:"-"`
	Ref         string  `json:"-"`
}

// Op 单个操作的元数据。
type Op struct {
	Tags        []string
	Summary     string
	Description string
	OperationID string
	Deprecated  bool
	Params      []Param
	Request     *Request
	Responses   map[string]Response
}

// Registry 维护所有 path -> method -> Op 的映射。
type Registry struct {
	mu    sync.RWMutex
	paths map[string]map[string]Op
	info  Info
	srvs  []Server
	// spec 是 MarshalJSON 的结果缓存（nil = 未缓存/已失效）。
	//
	// 背景（review-2026-09-19.md §6.2 P3）：注册表在启动时构建完成后不再变化，但
	// HTTPHandler 此前每次请求都全量重新 marshal 70 个 operation。这里按「变更即失效」
	// 缓存：任何注册/覆盖（Operation / Param / Respond / SetInfo / AddServer）都置 nil，
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

// Operation 注册 / 覆盖一个 operation。返回 *OpBuilder 以便链式添加 param/response。
func (r *Registry) Operation(method, path string, op Op) *OpBuilder {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.paths[path] == nil {
		r.paths[path] = map[string]Op{}
	}
	r.paths[path][strings.ToUpper(method)] = op
	r.spec = nil
	return &OpBuilder{r: r, method: method, path: path}
}

// OpBuilder 链式追加 param / response。
type OpBuilder struct {
	r      *Registry
	method string
	path   string
}

// Param 追加一个参数。
func (b *OpBuilder) Param(p Param) *OpBuilder {
	b.r.mu.Lock()
	defer b.r.mu.Unlock()
	m := b.r.paths[b.path][strings.ToUpper(b.method)]
	m.Params = append(m.Params, p)
	b.r.paths[b.path][strings.ToUpper(b.method)] = m
	b.r.spec = nil
	return b
}

// Respond 追加一个 response。
func (b *OpBuilder) Respond(status string, resp Response) *OpBuilder {
	b.r.mu.Lock()
	defer b.r.mu.Unlock()
	m := b.r.paths[b.path][strings.ToUpper(b.method)]
	if m.Responses == nil {
		m.Responses = map[string]Response{}
	}
	m.Responses[status] = resp
	b.r.paths[b.path][strings.ToUpper(b.method)] = m
	b.r.spec = nil
	return b
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
		"openapi": "3.0.3",
		"info":    r.info,
		"paths":   paths,
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
	return map[string]any{
		"application/json": map[string]any{
			"schema": mt.Schema,
		},
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
		r := rs[k]
		if r.Ref != "" {
			out[k] = map[string]any{"$ref": r.Ref}
			continue
		}
		entry := map[string]any{"description": r.Description}
		if r.JSON != nil {
			entry["content"] = map[string]any{
				"application/json": map[string]any{"schema": r.JSON},
			}
		}
		out[k] = entry
	}
	return out
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

func defaultSecurity() map[string]any {
	return map[string]any{
		"bearerAuth": map[string]any{
			"type":   "http",
			"scheme": "bearer",
		},
	}
}

// sharedSchemas / sharedParams / sharedResponses 输出 components 下可复用的片段，
// 并被各端点以 $ref 接线（见 openapi_register_*.go）。片段保持与真实响应形状一致。
func sharedSchemas() map[string]*Schema {
	return map[string]*Schema{
		"Error": BuildObj(map[string]*Schema{
			"error": Str(),
		}, "error"),
		"Account": BuildObj(map[string]*Schema{
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
		}, "id", "name", "endpoint", "accessKey"),
		"Bucket": BuildObj(map[string]*Schema{
			"name":         Str(),
			"creationDate": Str("date-time"),
		}, "name", "creationDate"),
		"ObjectItem": BuildObj(map[string]*Schema{
			"key":          Str(),
			"size":         Int64(),
			"lastModified": Str("date-time"),
			"etag":         Str(),
			"contentType":  Str(),
			"storageClass": Str(),
			"isDir":        Bool(),
		}, "key", "size", "lastModified", "isDir"),
		"ListObjectsResp": BuildObj(map[string]*Schema{
			"objects":        Arr(Ref("#/components/schemas/ObjectItem")),
			"commonPrefixes": Arr(Str()),
			"isTruncated":    Bool(),
			"nextToken":      Str(),
		}, "objects", "commonPrefixes", "isTruncated"),
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
