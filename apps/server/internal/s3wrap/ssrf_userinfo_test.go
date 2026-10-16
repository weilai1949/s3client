package s3wrap

// ssrf_userinfo_test.go —— P-3（2026-10-15 渗透测试，中危）：endpoint URL 内嵌凭据（userinfo）
// 的闭环。此前 `ValidateEndpoint` / `NormalizeEndpoint` 都不剥离 userinfo，用户误填
// `http://key:secret@host` 后凭据会经「错误串 → 日志 / 400 响应」与「BaseEndpoint → 拨号 /
// 预签名 URL」两条通道泄漏；`model.AccountView.endpoint` 还会把原样端点回传给任意已认证 token。
//
// 断言的都是**外部可见行为**（不依赖私有实现细节）：
//   - `ValidateEndpoint` 对带 userinfo 的端点 fail-closed（建号 / 更新 → 400 的来源），
//     且错误串里搜不到用户名与口令（错误串会进日志与响应体）；
//   - `NormalizeEndpoint`（拨号 / 签名 / 预签名 URL 的唯一归一化入口）剥离 userinfo，
//     无 userinfo 的端点行为不变、对自身幂等。

import (
	"strings"
	"testing"
)

// 哨兵值：任何对外可见的错误串 / 日志里出现它们即为泄漏。
const (
	userinfoUserSentinel = "AKIAUSER-SENTINEL"
	userinfoPassSentinel = "URLPASS-SENTINEL"
)

// TestValidateEndpointRejectsUserinfo 覆盖带 scheme / 裸端点 / 仅用户名三种 userinfo 形态：
// 必须拒绝；且错误串不得回显原始端点的用户名与口令。
func TestValidateEndpointRejectsUserinfo(t *testing.T) {
	cases := []struct{ name, endpoint string }{
		{"http 带用户名口令", "http://" + userinfoUserSentinel + ":" + userinfoPassSentinel + "@127.0.0.1:9000"},
		{"https 带路径", "https://" + userinfoUserSentinel + ":" + userinfoPassSentinel + "@S3.Example.com/bucket"},
		{"仅用户名", "http://" + userinfoUserSentinel + "@127.0.0.1:9000"},
		{"裸端点（无 scheme）", userinfoUserSentinel + ":" + userinfoPassSentinel + "@127.0.0.1:9000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateEndpoint(c.endpoint)
			if err == nil {
				t.Fatalf("ValidateEndpoint(%q) = nil, want fail-closed：endpoint userinfo 无正当用途"+
					"（SigV4 凭据来自独立字段），只会泄漏到日志 / 响应 / 预签名 URL", c.endpoint)
			}
			assertNoUserinfoLeak(t, err.Error())
		})
	}
	// 对照组：同一个主机、不含 userinfo 的端点仍必须放行（不能误伤）。
	if err := ValidateEndpoint("http://127.0.0.1:9000"); err != nil {
		t.Fatalf("ValidateEndpoint(无 userinfo) = %v, want nil", err)
	}
}

// TestValidateEndpointUserinfoErrorsDoNotEchoEndpoint 覆盖「userinfo + 退化地址」形态：
// 归一化后为空 / 无主机 / 非法端口的错误串此前用 %q 回显原始 endpoint，必须一并脱敏。
func TestValidateEndpointUserinfoErrorsDoNotEchoEndpoint(t *testing.T) {
	for _, ep := range []string{
		"http://" + userinfoUserSentinel + ":" + userinfoPassSentinel + "@",      // 归一化后为空
		"http://" + userinfoUserSentinel + ":" + userinfoPassSentinel + "@:5000", // 无主机名
		"http://" + userinfoUserSentinel + ":" + userinfoPassSentinel + "@host:notaport",
	} {
		err := ValidateEndpoint(ep)
		if err == nil {
			t.Fatalf("ValidateEndpoint(%q) = nil, want error", ep)
		}
		assertNoUserinfoLeak(t, err.Error())
	}
}

// TestNormalizeEndpointStripsUserinfo 钉住归一化入口剥离 userinfo：BaseEndpoint（拨号 / 签名）、
// 预签名 URL、`service.SameEndpoint` 全部复用本函数，剥离后这些下游一律拿不到凭据。
func TestNormalizeEndpointStripsUserinfo(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		useSSL bool
		want   string
	}{
		{"http userinfo", "http://" + userinfoUserSentinel + ":" + userinfoPassSentinel + "@MinIO:9000", false, "http://minio:9000"},
		{"保留路径与路径大小写", "http://" + userinfoUserSentinel + ":" + userinfoPassSentinel + "@MinIO:9000/Bucket/", false, "http://minio:9000/Bucket"},
		{"裸端点 userinfo", userinfoUserSentinel + ":" + userinfoPassSentinel + "@minio:9000", false, "http://minio:9000"},
		{"裸端点 userinfo + useSSL", userinfoUserSentinel + ":" + userinfoPassSentinel + "@minio:9000", true, "https://minio:9000"},
		{"路径里的 @ 不受影响", "http://minio:9000/a@b", false, "http://minio:9000/a@b"},
		{"无 userinfo 行为不变", "http://minio:9000/prefix/", false, "http://minio:9000/prefix"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NormalizeEndpoint(c.in, c.useSSL)
			if got != c.want {
				t.Fatalf("NormalizeEndpoint(%q,%v) = %q, want %q", c.in, c.useSSL, got, c.want)
			}
			assertNoUserinfoLeak(t, got)
			// 幂等：剥离后的结果再归一化必须不变（下游多处重复归一化同一端点）。
			if again := NormalizeEndpoint(got, c.useSSL); again != got {
				t.Fatalf("NormalizeEndpoint 不幂等: %q -> %q", got, again)
			}
		})
	}
}

// assertNoUserinfoLeak 断言给定文本里没有端点凭据哨兵。
func assertNoUserinfoLeak(t *testing.T, s string) {
	t.Helper()
	if strings.Contains(s, userinfoUserSentinel) || strings.Contains(s, userinfoPassSentinel) {
		t.Fatalf("端点 userinfo 凭据泄漏: %q", s)
	}
}
