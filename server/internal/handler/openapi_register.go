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

func acctIDParam() openapi.Param {
	return openapi.Param{
		Name: "id", In: "path", Required: true,
		Description: "账号 UUID",
		Schema:      openapi.Str(),
	}
}

// desc 给已有 schema 补一段描述，保持 BuildObj 一行式声明不被长串 Description 撑爆。

func desc(s *openapi.Schema, d string) *openapi.Schema {
	if s == nil {
		return nil
	}
	s.Description = d
	return s
}
