package handler

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

type fakeAPIError struct {
	code string
}

func (e fakeAPIError) Error() string                 { return e.code }
func (e fakeAPIError) ErrorCode() string             { return e.code }
func (e fakeAPIError) ErrorMessage() string          { return e.code }
func (e fakeAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

func TestS3UserMessage(t *testing.T) {
	if got := s3UserMessage(nil); got != "" {
		t.Fatalf("nil = %q", got)
	}
	if got := s3UserMessage(&types.NoSuchKey{}); got != "object not found" {
		t.Fatalf("NoSuchKey = %q", got)
	}
	if got := s3UserMessage(fakeAPIError{code: "AccessDenied"}); got != "access denied" {
		t.Fatalf("AccessDenied = %q", got)
	}
	// sentinel 匹配（errors.Is），不再依赖错误文本子串。
	if got := s3UserMessage(fmt.Errorf("copy x: %w", s3wrap.ErrObjectTooLarge)); got != "object exceeds 5GB single-put limit; use multipart upload" {
		t.Fatalf("5GB = %q", got)
	}
	if got := s3UserMessage(fmt.Errorf("move x: %w", s3wrap.ErrSourceDeleteFailed)); got != "copied but failed to delete source" {
		t.Fatalf("delete source = %q", got)
	}
	if got := s3UserMessage(errors.New("something weird")); got != "storage operation failed" {
		t.Fatalf("unknown = %q", got)
	}
}
