package service

import "testing"

func TestStreamCopyConstants(t *testing.T) {
	// S3 单次 PUT 上限 5GB：这是协议契约，改小会退化为分段、改大会被 S3 拒绝。
	if MaxSinglePutBytes != 5_000_000_000 {
		t.Fatalf("MaxSinglePutBytes = %d", MaxSinglePutBytes)
	}
}
