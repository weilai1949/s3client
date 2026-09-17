package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/s3wrap"
)

// s3UserMessage 委托防腐层。
func s3UserMessage(err error) string { return s3wrap.UserMessage(err) }

// s3HTTPStatus 委托防腐层。
func s3HTTPStatus(err error) int { return s3wrap.HTTPStatus(err) }
