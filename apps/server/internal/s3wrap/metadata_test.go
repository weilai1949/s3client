package s3wrap

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestValidateUserMetadata(t *testing.T) {
	mk := func(n int) string { return strings.Repeat("a", n) }
	cases := []struct {
		name    string
		in      map[string]string
		wantErr bool
	}{
		{"nil", nil, false},
		{"empty", map[string]string{}, false},
		{"ok", map[string]string{"env": "prod", "owner": "weilai"}, false},
		{"empty key", map[string]string{"": "x"}, true},
		{"key too long", map[string]string{mk(MaxUserMetaKeyLen + 1): "v"}, true},
		{"value too long", map[string]string{"k": mk(MaxUserMetaValueLen + 1)}, true},
		{"non-ascii key", map[string]string{"环境": "prod"}, true},
		{"non-printable key", map[string]string{"a\x01b": "v"}, true},
		// 值侧此前只查 UTF-8 合法性与长度，控制字符一路走到 Go transport 才被
		// `invalid header field value` 拒发 ⇒ 边界该给的 400 变成传输期 500，
		// 恰好违背本文件「提前在 API 边界校验」的自述目标。
		{"value with CRLF", map[string]string{"note": "line1\r\nline2"}, true},
		{"value with lone LF", map[string]string{"note": "line1\nline2"}, true},
		{"value with control char", map[string]string{"k": "a\x01b"}, true},
		{"value with DEL", map[string]string{"k": "a\x7fb"}, true},
		// HTAB 与空格是 Go httpguts.ValidHeaderFieldValue 放行的（isLWS），
		// 拒绝它们会误伤正常的多行备注写法。
		{"value with tab allowed", map[string]string{"k": "a\tb"}, false},
		{"value with space allowed", map[string]string{"k": "a b"}, false},
		// 非 ASCII 但合法 UTF-8 的值必须继续放行（值侧契约是 UTF-8，不是 US-ASCII）。
		{"value with utf8 text allowed", map[string]string{"k": "环境 prod-中文"}, false},
		{"invalid utf8 value", map[string]string{"k": string([]byte{0xff, 0xfe, 0xfd})}, true},
		{"total too big", map[string]string{"k1": mk(1000), "k2": mk(1000), "k3": mk(200)}, true},
		// Each pair is individually valid (value ≤ 256), but the sum of all
		// key+value lengths exceeds MaxUserMetaTotalLen, exercising the total-size
		// branch (the prior "total too big" case trips the per-value length limit first).
		{"total too big via many pairs", func() map[string]string {
			m := map[string]string{}
			for i := 0; i < 30; i++ {
				m[fmt.Sprintf("key%d", i)] = mk(100)
			}
			return m
		}(), true},
		{"total under limit", map[string]string{"k1": mk(100), "k2": mk(100)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateUserMetadata(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("want err, got nil")
				}
				if !errors.Is(err, ErrUserMetadataInvalid) {
					t.Fatalf("err = %v, want wraps ErrUserMetadataInvalid", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}
