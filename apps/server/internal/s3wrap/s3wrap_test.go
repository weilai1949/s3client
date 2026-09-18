package s3wrap

import (
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

func TestNormalizeEndpoint(t *testing.T) {
	cases := []struct {
		in     string
		useSSL bool
		want   string
	}{
		{"localhost:9000", false, "http://localhost:9000"},
		{"localhost:9000/", false, "http://localhost:9000"},
		{"http://localhost:9000", false, "http://localhost:9000"},
		{"https://s3.amazonaws.com/", true, "https://s3.amazonaws.com"},
		{"minio.example.com", true, "https://minio.example.com"},
		{"", false, ""},
		// 大小写 scheme：此前只做大小写敏感的前缀判断，会把 "HTTP://Host" 当成
		// 无 scheme 的裸主机，产出损坏的 "http://HTTP://Host"（features.md §K，原 todolist #10）。
		{"HTTP://MinIO:9000", false, "http://minio:9000"},
		{"HTTPS://S3.AmazonAWS.com/", false, "https://s3.amazonaws.com"},
		// 首尾空白：账号 endpoint 由用户手填，创建时只校验非空、不 trim。
		{"  minio:9000  ", false, "http://minio:9000"},
		{" http://MinIO:9000/ ", false, "http://minio:9000"},
	}
	for _, c := range cases {
		if got := NormalizeEndpoint(c.in, c.useSSL); got != c.want {
			t.Errorf("NormalizeEndpoint(%q,%v) = %q, want %q", c.in, c.useSSL, got, c.want)
		}
	}
}

// TestNormalizeEndpointNeverDoubleScheme 归一化结果不得含多个 scheme。
// SameEndpoint 的比较语义必须与建 client 时的归一化一致：若比较说「同一端点」，
// 用它建出的 BaseEndpoint 就不能是损坏的 URL。两份实现此前不一致
// （service 侧正确、s3wrap 侧损坏），此测试锁定统一后的行为。
func TestNormalizeEndpointNeverDoubleScheme(t *testing.T) {
	for _, in := range []string{"HTTP://MinIO:9000", " http://A.com/ ", "minio:9000", ""} {
		got := NormalizeEndpoint(in, false)
		if strings.Count(got, "://") > 1 {
			t.Errorf("NormalizeEndpoint(%q) = %q contains more than one scheme", in, got)
		}
		if in != "" && got == "" {
			t.Errorf("NormalizeEndpoint(%q) = empty", in)
		}
	}
}

func TestEscapeKeyPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"a/b/c.txt", "a/b/c.txt"},
		{"folder/hello world.txt", "folder/hello%20world.txt"},
		{"", ""},
	}
	for _, c := range cases {
		if got := escapeKeyPath(c.in); got != c.want {
			t.Errorf("escapeKeyPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPublicURL(t *testing.T) {
	cases := []struct {
		name              string
		acc               *model.Account
		bucket, key, want string
	}{
		{
			name:   "path-style http",
			acc:    &model.Account{Endpoint: "http://127.0.0.1:9000", PathStyle: true},
			bucket: "mybucket", key: "a/b.txt",
			want: "http://127.0.0.1:9000/mybucket/a/b.txt",
		},
		{
			name:   "virtual-host https",
			acc:    &model.Account{Endpoint: "https://s3.amazonaws.com", UseSSL: true, PathStyle: false},
			bucket: "mybucket", key: "file.txt",
			want: "https://mybucket.s3.amazonaws.com/file.txt",
		},
		{
			name:   "public endpoint override",
			acc:    &model.Account{Endpoint: "http://minio:9000", PublicEndpoint: "http://127.0.0.1:9000", PathStyle: true},
			bucket: "b", key: "k",
			want: "http://127.0.0.1:9000/b/k",
		},
		{
			// 用户手填的 endpoint 可能带首尾空白或大小写 scheme；PublicURL 必须
			// 与 NormalizeEndpoint 用同一套规则，否则会产出带空格/双重 scheme 的链接。
			name:   "trim and lowercase scheme",
			acc:    &model.Account{Endpoint: " HTTP://MinIO:9000/ ", PathStyle: true},
			bucket: "b", key: "k",
			want: "http://minio:9000/b/k",
		},
		{
			// endpoint 自带的路径前缀（反向代理）必须保留，不能被归一化吃掉。
			name:   "keeps endpoint path prefix",
			acc:    &model.Account{Endpoint: "https://gw.example.com/s3/", PathStyle: true},
			bucket: "b", key: "k",
			want: "https://gw.example.com/s3/b/k",
		},
		{
			// 未配置端点 → 无公开地址；必须返回空串而不是拼出损坏的相对链接。
			name:   "unset endpoint yields empty url",
			acc:    &model.Account{PathStyle: true},
			bucket: "b", key: "k",
			want: "",
		},
		{
			// 退化为 scheme-only 的端点同样不可用（NormalizeEndpoint 归一化为空）。
			name:   "degenerate endpoint yields empty url",
			acc:    &model.Account{Endpoint: "http://", PathStyle: true},
			bucket: "b", key: "k",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := &Client{acc: c.acc}
			if got := client.PublicURL(c.bucket, c.key); got != c.want {
				t.Errorf("PublicURL() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestPresignEndpointFallback(t *testing.T) {
	// PublicEndpoint 为空时应回退到 Endpoint（New 内逻辑；此处验证 normalize 一致性）。
	acc := &model.Account{
		Endpoint:  "http://127.0.0.1:9000",
		AccessKey: "ak", SecretKey: "sk", PathStyle: true,
	}
	ep := acc.PublicEndpoint
	if ep == "" {
		ep = acc.Endpoint
	}
	if got := NormalizeEndpoint(ep, acc.UseSSL); got != "http://127.0.0.1:9000" {
		t.Fatalf("presign endpoint = %q", got)
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("expected error for nil account")
	}
	if _, err := New(&model.Account{AccessKey: "ak"}); err == nil {
		t.Fatal("expected error for missing secret key")
	}
}
