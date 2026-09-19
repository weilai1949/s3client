package handler

import (
	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

// s3UserMessage 委托防腐层。
func s3UserMessage(err error) string { return s3wrap.UserMessage(err) }

// s3UserMessageForCode 委托防腐层：把 S3 错误码映射为用户可见短消息。
// 用于「失败只出现在 200 响应体内、拿不到 error 值」的场景（DeleteObjects 的逐 key <Error>）。
func s3UserMessageForCode(code string) string { return s3wrap.UserMessageForCode(code) }

// s3HTTPStatus 委托防腐层。
func s3HTTPStatus(err error) int { return s3wrap.HTTPStatus(err) }
