package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/openapi"
)

func registerOpenAPI(r *openapi.Registry, version string) {
	r.SetInfo(openapi.Info{
		Title:       "s3clinet API",
		Version:     version,
		Description: "s3clinet 是面向 S3 兼容对象存储的多账号 Web 控制台。本文档为 /api/* 端点的 OpenAPI 3.0 契约，所有响应均 JSON（除 /api/health 等纯状态端点）。鉴权：Bearer Token（环境变量 S3C_TOKEN，多值逗号分隔）。",
	})
	r.AddServer(openapi.Server{URL: "/", Description: "同源（前端 Vite 代理或后端 SPA fallback）"})

	registerAccounts(r)
	registerBuckets(r)
	registerBucketSettings(r)
	registerObjects(r)
	registerObjectMeta(r)
	registerMultipart(r)
	registerVersions(r)
	registerTrash(r)
	registerMigrate(r)
	registerSystem(r)
}

// ---- Accounts ----

// refParam 构造指向 components.parameters 的 $ref 参数（如 "AccountID"）。
func refParam(component string) openapi.Param {
	return openapi.Param{Ref: "#/components/parameters/" + component}
}

// refResp 构造指向 components.responses 的 $ref 响应（如 "NotFound"）。
func refResp(component string) openapi.Response {
	return openapi.Response{Ref: "#/components/responses/" + component}
}

// refSchema 构造指向 components.schemas 的 $ref schema（如 "Account"），用于响应体。
func refSchema(component string) *openapi.Schema {
	return openapi.Ref("#/components/schemas/" + component)
}

// acctIDParam 账号 UUID 路径参数（复用共享 AccountID 参数）。
func acctIDParam() openapi.Param {
	return refParam("AccountID")
}

// desc 给已有 schema 补一段描述，保持 BuildObj 一行式声明不被长串 Description 撑爆。

func desc(s *openapi.Schema, d string) *openapi.Schema {
	if s == nil {
		return nil
	}
	s.Description = d
	return s
}
