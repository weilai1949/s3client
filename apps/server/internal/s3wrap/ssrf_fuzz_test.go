package s3wrap

// ssrf_fuzz_test.go —— 端点归一化 / SSRF 校验的原生 fuzz（stdlib `testing.F`，无新依赖）。
//
// 为什么在这里：`NormalizeEndpoint` 是端点归一化的唯一实现（`service.SameEndpoint` 与
// 建 client 的 BaseEndpoint 共用），`ValidateEndpoint` 是 SSRF 主防线入口。两者都吃用户
// 手填的任意字符串，历史上修过「大小写 scheme 产出损坏 URL」「归一化为空却静默放行」
// 两类边界；本文件对输入空间做探索。
//
// 断言的性质（property）：
//   - 归一化结果要么为空、要么：无结尾斜杠、scheme 分隔符存在、host 已小写，
//     且对自身幂等（避免「比较判定同一端点、建出的 URL 却不同」）；
//   - 归一化后命中屏蔽主机名的端点，`ValidateEndpoint` 必须拒绝（且在 DNS 之前，故不受网络影响）；
//   - `ValidateEndpoint` 对任意输入不 panic，错误文案非空。
//
// 运行（有界；CI 见 .github/workflows/fuzz.yml）：
//
//	cd apps/server && go test ./internal/s3wrap/ -run '^$' -fuzz '^FuzzNormalizeEndpoint$' -fuzztime=10s
//	cd apps/server && go test ./internal/s3wrap/ -run '^$' -fuzz '^FuzzValidateEndpoint$' -fuzztime=10s

import (
	"strings"
	"testing"
)

// FuzzNormalizeEndpoint 探索端点归一化的不变量与幂等性。
func FuzzNormalizeEndpoint(f *testing.F) {
	for _, seed := range []struct {
		endpoint string
		useSSL   bool
	}{
		{"", false},
		{"   ", true},
		{"minio.local:9000", false},
		{"MINIO.LOCAL:9000", true},
		{"HTTP://MinIO.Local:9000/Bucket/Path", false},
		{"https://s3.amazonaws.com/", true},
		{"http://", false},
		{"://", false},
		{"/", false},
		{"http://169.254.169.254/latest/meta-data/", false},
		{"http://[fd00:ec2::254]/", false},
		{"http://100.100.100.200/", false},
		{"münchen.de", false},
		{"http://user:pass@Host:8080/a/B/", true},
		{"http://host/%zz", false},
		{"?x=1", false},
		{"#frag", false},
		// 回归种子（2026-09-30 fuzz 发现，KNOWN_ISSUES #70 已修复）：曾暴露
		// 「切分前去尾斜杠」把内部空白顶到结果末尾的非幂等缺陷，现应幂等。
		{"00  /", false},
		{"host/a  /", false},
	} {
		f.Add(seed.endpoint, seed.useSSL)
	}

	// KNOWN_ISSUES #70（2026-09-30 已修复）：`NormalizeEndpoint` 曾在切分 host/path **之前**
	// 对整个 rest 去尾斜杠，把尾斜杠之后的内部空白暴露到结果末尾 → 输出带尾随空白且
	// 不幂等（`"00  /"` → `"http://00  "` → 再归一 `"http://00"`）。修复改为**先切分
	// host/path**、host `TrimSpace`、path 去尾部斜杠与空白，输出保证
	// `got == strings.TrimSpace(got)`——原「带尾随空白就跳过幂等断言」的短路分支已不可达
	// 而删除（死代码零容忍）。`"00  /"` / `"host/a  /"` 两个原退化输入**保留**在 corpus
	// 作回归种子，另有确定性断言 `TestNormalizeEndpointIdempotent`。
	f.Fuzz(func(t *testing.T, endpoint string, useSSL bool) {
		got := NormalizeEndpoint(endpoint, useSSL)
		if got == "" {
			return
		}
		if strings.HasSuffix(got, "/") {
			t.Fatalf("归一化结果保留了结尾斜杠: %q", got)
		}
		if again := NormalizeEndpoint(got, useSSL); again != got {
			t.Fatalf("NormalizeEndpoint 不幂等: %q -> %q", got, again)
		}
		i := strings.Index(got, "://")
		if i < 0 {
			t.Fatalf("归一化结果缺 scheme 分隔符: %q", got)
		}
		rest := got[i+3:]
		host := rest
		if j := strings.IndexAny(rest, "/?#"); j >= 0 {
			host = rest[:j]
		}
		if host != strings.ToLower(host) {
			t.Fatalf("归一化后 host 未小写: %q (host=%q)", got, host)
		}
		// 说明：这里**不**用 url.Parse 交叉比对 host 形态——`NormalizeEndpoint` 允许
		// 「scheme 段含 `/`」的退化输入（如 `"//0://0"` 原样返回，其 scheme 非法），
		// url.Parse 会把它当 protocol-relative URL 从而与手写切分不一致。这类输入不构成
		// 崩溃/SSRF 绕过（ValidateEndpoint 仍走 hostname 校验 + DNS），仅登记为观察项。
		// SSRF 硬性质：屏蔽主机名必须在 DNS 解析之前被拒（故这里不触网）。
		if isBlockedHostname(host) {
			if err := ValidateEndpoint(got); err == nil {
				t.Fatalf("屏蔽主机名未被拒绝: %q (host=%q)", got, host)
			}
		}
	})
}

// TestNormalizeEndpointIdempotent 钉住 KNOWN_ISSUES #70 的修复（2026-09-30 推翻 ➖ 转修复）：
// 归一化必须幂等，且结果不得以空白结尾——入口 TrimSpace 只做一次，若先去尾斜杠会把
// 尾斜杠之后的内部空白暴露到结果末尾，二次归一化就会把它 trim 掉（`f(x) != f(f(x))`）。
//
// 复核命令：cd apps/server && go test ./internal/s3wrap/ -run 'TestNormalizeEndpointIdempotent' -count=1 -v
// 变异验证：删掉 client.go 里 host 的 `strings.TrimSpace(host)` → 本测试红灯点名
// `"00  /"` / `"http://host  /"` → 还原后绿灯（2026-09-30 实跑）。
func TestNormalizeEndpointIdempotent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		endpoint string
		useSSL   bool
	}{
		{"斜杠前空格暴露为尾随空白", "00  /", false},
		{"路径尾斜杠前空格", "host/a  /", false},
		{"scheme 后 host 尾斜杠前空格", "http://host  /", false},
		{"host 边缘空白", "host /x", false},
		{"host 内部空白保留", "host/my dir/", false},
		{"入口首尾空白", "  minio.local:9000/  ", false},
		{"普通端点", "minio.local:9000", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeEndpoint(tc.endpoint, tc.useSSL)
			if got == "" {
				return
			}
			if got != strings.TrimSpace(got) {
				t.Fatalf("NormalizeEndpoint(%q) = %q 以空白结尾", tc.endpoint, got)
			}
			if again := NormalizeEndpoint(got, tc.useSSL); again != got {
				t.Fatalf("NormalizeEndpoint 不幂等: %q -> %q (再归一 %q)", tc.endpoint, got, again)
			}
		})
	}
}

// FuzzValidateEndpoint 探索 SSRF 校验入口：任意输入不 panic、错误文案非空。
func FuzzValidateEndpoint(f *testing.F) {
	for _, seed := range []string{
		"",
		"   ",
		"minio.local:9000",
		"http://127.0.0.1:9000",
		"https://s3.amazonaws.com",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/",
		"http://metadata/",
		"http://instance-data/",
		"http://kubernetes.default.svc/",
		"http://100.100.100.200/",
		"http://100.96.0.2/",
		"http://[fd00:ec2::254]/",
		"münchen.de",
		"http:///x",
		"http://:8080",
		"file:///etc/passwd",
		"http://user@metadata.google.internal",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, endpoint string) {
		err := ValidateEndpoint(endpoint)
		if err != nil {
			if err.Error() == "" {
				t.Fatalf("ValidateEndpoint(%q) 返回了空错误文案", endpoint)
			}
			return
		}
		// 放行时不得是「归一化为空/无主机」的退化地址（那必须 fail-closed）。
		if got := NormalizeEndpoint(endpoint, false); got == "" && strings.TrimSpace(endpoint) != "" {
			t.Fatalf("非空端点 %q 归一化为空却被放行", endpoint)
		}
	})
}
