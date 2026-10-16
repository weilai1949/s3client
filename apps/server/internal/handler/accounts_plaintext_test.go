package handler

// accounts_plaintext_test.go —— A5（2026-10-10，威胁模型 §6.2 的「明文 http:// 载荷完整性」残留）：
//
// 残留风险本体只能靠 TLS 消除（`useSSL=true` 或反向代理终止 TLS），本测试**不要求**服务端拦截
// ——自托管刚需下拦截属行为变更。它钉的是另一半：**不能让风险无感知**。此前
// `S3C_ALLOW_PLAINTEXT_STORE=1` 有启动 WARN（`cfg.StorePlaintextWarning()`），
// 而新建 / 更新一个 `http://` 账号端点时**零提示**，运维要到读威胁模型才知道。
//
// 断言的外部可见行为：HTTP 201 / 200 之外，服务端日志必须出现一条 WARN，且
// **不含任何密钥**（根 AGENTS.md「SecretKey 不落日志」）；HTTPS 端点则不得出现该 WARN
// （否则告警噪声会让人忽略真正的明文告警）。
//
// P-3（2026-10-15 渗透测试，2026-10-16 收口）在同一文件里补齐「端点 userinfo 凭据」的
// 外部可见行为：建号 / 更新带 userinfo 的 endpoint → **400**（fail-closed，不落库）；日志、
// 400 响应体、`AccountView` 都不得出现凭据；历史落盘数据由 `AccountView` 回传脱敏值。
// 原 `TestPlaintextWarnDoesNotLeakEndpointCredentials` 的期望随之下调为 400 并**改名为**
// `TestUserinfoEndpointNeverLeaksCredentials`（原为 201 + 脱敏 WARN，路径已不再产生 WARN）；
// 明文 WARN 路径由 `TestPlaintextWarnStaysCredentialFreeForLegacyAccount` 明确保活。

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

// 端点 userinfo 凭据哨兵（P-3）：任何对外可见的日志 / 响应里出现它们即为泄漏。
const (
	endpointUserinfoUser = "AKIAUSER-SENTINEL"
	endpointUserinfoPass = "URLPASS-SENTINEL"
)

// assertNoEndpointCredential 断言文本里没有端点凭据哨兵。
func assertNoEndpointCredential(t *testing.T, s string) {
	t.Helper()
	if strings.Contains(s, endpointUserinfoUser) || strings.Contains(s, endpointUserinfoPass) {
		t.Fatalf("端点 userinfo 凭据泄漏: %s", s)
	}
}

// TestCreateAccountRejectsUserinfoEndpoint P-3：带 userinfo 的 endpoint 建号必须 fail-closed（400），
// 不得落库、不得把凭据写进日志或响应体。旧行为是 201 + 落库（凭据随 `AccountView` 回传）。
func TestCreateAccountRejectsUserinfoEndpoint(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(t, &buf)
	rr := doJSON(t, h, http.MethodPost, "/api/accounts",
		accountBody(t, "http://"+endpointUserinfoUser+":"+endpointUserinfoPass+"@127.0.0.1:9000", "", false))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400（endpoint userinfo 必须 fail-closed）:\n%s", rr.Code, rr.Body.String())
	}
	assertNoEndpointCredential(t, rr.Body.String())
	assertNoEndpointCredential(t, buf.String())
	if strings.Contains(buf.String(), "SECRET-SENTINEL") {
		t.Errorf("日志泄漏了 secretKey：\n%s", buf.String())
	}
	// 建号失败 ⇒ 账号不得落库。
	list := doJSON(t, h, http.MethodGet, "/api/accounts", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d", list.Code)
	}
	var got struct {
		Accounts []json.RawMessage `json:"accounts"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析账号列表失败: %v\n%s", err, list.Body.String())
	}
	if len(got.Accounts) != 0 {
		t.Fatalf("被拒账号仍落库：%s", list.Body.String())
	}
}

// TestUpdateAccountRejectsUserinfoEndpoint P-3：更新路径同样 fail-closed，且已有账号不被改动。
func TestUpdateAccountRejectsUserinfoEndpoint(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(t, &buf)
	create := doJSON(t, h, http.MethodPost, "/api/accounts",
		accountBody(t, "https://s3.example.com", "", false))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d：\n%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("解析创建响应 id 失败（err=%v）：\n%s", err, create.Body.String())
	}
	buf.Reset()

	rr := doJSON(t, h, http.MethodPut, "/api/accounts/"+created.ID,
		accountBody(t, "http://"+endpointUserinfoUser+":"+endpointUserinfoPass+"@127.0.0.1:9000", "", false))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("update status = %d, want 400：\n%s", rr.Code, rr.Body.String())
	}
	assertNoEndpointCredential(t, rr.Body.String())
	assertNoEndpointCredential(t, buf.String())

	// 账号未被改写：endpoint 仍是原值且列表响应不含任何凭据。
	after := doJSON(t, h, http.MethodGet, "/api/accounts/"+created.ID, "")
	if after.Code != http.StatusOK {
		t.Fatalf("get status = %d：\n%s", after.Code, after.Body.String())
	}
	var view struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(after.Body.Bytes(), &view); err != nil {
		t.Fatalf("解析账号视图失败: %v\n%s", err, after.Body.String())
	}
	if view.Endpoint != "https://s3.example.com" {
		t.Fatalf("被拒的更新改动了账号 endpoint：%q", view.Endpoint)
	}
	assertNoEndpointCredential(t, after.Body.String())
}

// TestAccountViewRedactsEndpointCredentials P-3 通道③：`AccountView.endpoint` / `publicEndpoint`
// 回传脱敏值，防御**历史落盘数据**（契约上线前已存下的带 userinfo 端点）。覆盖 5 种形态：
// 带 scheme 有/无 userinfo、裸端点有/无 userinfo、url.Parse 失败原样返回。
func TestAccountViewRedactsEndpointCredentials(t *testing.T) {
	cases := []struct{ name, endpoint, want string }{
		{"scheme 带 userinfo", "http://" + endpointUserinfoUser + ":" + endpointUserinfoPass + "@minio.local:9000", "http://minio.local:9000"},
		{"scheme 无 userinfo", "http://minio.local:9000", "http://minio.local:9000"},
		{"裸端点带 userinfo", endpointUserinfoUser + ":" + endpointUserinfoPass + "@minio.local:9000", "minio.local:9000"},
		{"裸端点无 userinfo", "minio.local:9000", "minio.local:9000"},
		{"解析失败原样返回", "http://%zz:9000", "http://%zz:9000"},
	}
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	for _, c := range cases {
		// 直接建库（绕过 handler 校验）模拟历史数据：这些端点在新契约下不可能再被写入。
		if _, err := st.Create(&model.Account{
			Name: c.name, Endpoint: c.endpoint, PublicEndpoint: c.endpoint,
			AccessKey: "AKIATEST", SecretKey: "SECRET-SENTINEL",
		}); err != nil {
			t.Fatalf("seed %s: %v", c.name, err)
		}
	}
	h := New(st, slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)), t.TempDir(), nil, "", "test", false, false).Routes()
	rr := doJSON(t, h, http.MethodGet, "/api/accounts", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d：\n%s", rr.Code, rr.Body.String())
	}
	assertNoEndpointCredential(t, rr.Body.String())
	var got struct {
		Accounts []struct {
			Name           string `json:"name"`
			Endpoint       string `json:"endpoint"`
			PublicEndpoint string `json:"publicEndpoint"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析账号列表失败: %v\n%s", err, rr.Body.String())
	}
	if len(got.Accounts) != len(cases) {
		t.Fatalf("账号数 = %d, want %d：\n%s", len(got.Accounts), len(cases), rr.Body.String())
	}
	for _, a := range got.Accounts {
		want := ""
		for _, c := range cases {
			if c.name == a.Name {
				want = c.want
			}
		}
		if a.Endpoint != want || a.PublicEndpoint != want {
			t.Errorf("%s: endpoint=%q publicEndpoint=%q, want %q（脱敏后的值）", a.Name, a.Endpoint, a.PublicEndpoint, want)
		}
	}
}

// TestClientInitFailureLogHidesEndpointCredentials P-3 通道②：历史数据里的 userinfo 端点在
// `clients.get` 失败时由 handler.go 打 Debug 日志（"s3 client init"）；错误串必须已脱敏，
// 否则任意已认证 token 都能用「故意让客户端构建失败」把凭据写进日志。
func TestClientInitFailureLogHidesEndpointCredentials(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "legacy", Endpoint: "http://" + endpointUserinfoUser + ":" + endpointUserinfoPass + "@127.0.0.1:9000",
		AccessKey: "AKIATEST", SecretKey: "SECRET-SENTINEL",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()

	rr := doJSON(t, h, http.MethodPost, "/api/accounts/"+acc.ID+"/test", "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("test status = %d, want 400（userinfo 端点必须 fail-closed）：\n%s", rr.Code, rr.Body.String())
	}
	log := buf.String()
	if !strings.Contains(log, "s3 client init") {
		t.Fatalf("未捕获到 clients.get 失败日志（用例失效）：\n%s", log)
	}
	assertNoEndpointCredential(t, log)
	assertNoEndpointCredential(t, rr.Body.String())
	if strings.Contains(log, "SECRET-SENTINEL") {
		t.Errorf("日志泄漏了 secretKey：\n%s", log)
	}
}

// plaintextWarnMarker 是 WARN 消息里必含的稳定片段（改动文案需同步本测试）。
const plaintextWarnMarker = "载荷完整性"

// newCapturingHandler 构造带**可捕获日志**的被测 Handler 与请求体编码器。
func newCapturingHandler(t *testing.T, buf *bytes.Buffer) http.Handler {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	return New(st, slog.New(slog.NewJSONHandler(buf, nil)), t.TempDir(), nil, "", "test", false, false).Routes()
}

// accountBody 编码账号创建 / 更新请求体（`secretKey` 用可搜索的哨兵值）。
func accountBody(t *testing.T, endpoint, publicEndpoint string, useSSL bool) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"name":           "acc-plain",
		"endpoint":       endpoint,
		"publicEndpoint": publicEndpoint,
		"accessKey":      "AKIATEST",
		"secretKey":      "SECRET-SENTINEL",
		"region":         "us-east-1",
		"pathStyle":      true,
		"useSSL":         useSSL,
	})
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}
	return string(b)
}

// TestCreateAccountWarnsOnPlaintextEndpoint 断言创建明文端点账号时打 WARN、
// TLS 端点不打，且日志里绝不出现密钥。
func TestCreateAccountWarnsOnPlaintextEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		useSSL   bool
		wantWarn bool
	}{
		{"显式 http://", "http://127.0.0.1:9000", false, true},
		{"scheme 大小写不敏感", "HTTP://127.0.0.1:9000", false, true},
		{"裸端点 + useSSL=false（默认 http）", "127.0.0.1:9000", false, true},
		{"显式 https://", "https://s3.example.com", false, false},
		{"裸端点 + useSSL=true", "s3.example.com", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := newCapturingHandler(t, &buf)
			rr := doJSON(t, h, http.MethodPost, "/api/accounts",
				accountBody(t, tc.endpoint, "", tc.useSSL))
			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201（本用例只关心日志，不改行为）：\n%s", rr.Code, rr.Body.String())
			}
			got := strings.Contains(buf.String(), plaintextWarnMarker)
			if got != tc.wantWarn {
				t.Errorf("明文告警 = %v, want %v；日志：\n%s", got, tc.wantWarn, buf.String())
			}
			if strings.Contains(buf.String(), "SECRET-SENTINEL") {
				t.Errorf("日志泄漏了 secretKey：\n%s", buf.String())
			}
		})
	}
}

// TestUpdateAccountWarnsOnPlaintextEndpoint 断言**更新**路径同样有告知
// （编辑既有账号把 TLS 关掉，是明文端点最常见的进入方式）。
func TestUpdateAccountWarnsOnPlaintextEndpoint(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(t, &buf)

	create := doJSON(t, h, http.MethodPost, "/api/accounts",
		accountBody(t, "https://s3.example.com", "", false))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d：\n%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("解析创建响应 id 失败（err=%v）：\n%s", err, create.Body.String())
	}
	buf.Reset()

	rr := doJSON(t, h, http.MethodPut, "/api/accounts/"+created.ID,
		accountBody(t, "http://127.0.0.1:9000", "", false))
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200：\n%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(buf.String(), plaintextWarnMarker) {
		t.Errorf("更新为明文端点时未打 WARN：\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "SECRET-SENTINEL") {
		t.Errorf("日志泄漏了 secretKey：\n%s", buf.String())
	}
}

// TestCreateAccountWarnsOnPlaintextPublicEndpoint 断言浏览器直传用的
// `publicEndpoint` 明文同样有告知——预签名 PUT 走的是它，明文链路下直传载荷同样无完整性保护。
func TestCreateAccountWarnsOnPlaintextPublicEndpoint(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(t, &buf)
	rr := doJSON(t, h, http.MethodPost, "/api/accounts",
		accountBody(t, "https://s3.example.com", "http://127.0.0.1:9000", false))
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d：\n%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(buf.String(), plaintextWarnMarker) {
		t.Errorf("publicEndpoint 为明文时未打 WARN：\n%s", buf.String())
	}
}

// TestUserinfoEndpointNeverLeaksCredentials（原 TestPlaintextWarnDoesNotLeakEndpointCredentials，
// 评审 2026-10-10 N1 / KNOWN_ISSUES #84；2026-10-16 P-3 改名 + 更新期望）：
// 带 userinfo 的 endpoint 外部行为已由「201 + WARN 脱敏」变为 **fail-closed 400**（冻结合同：
// `ValidateEndpoint` 拒绝 → 建号/更新 400），这条路径已不再产生 WARN，用例名若再提「WARN」会误导。
// **原始意图不变**：endpoint 里的 userinfo 是用户输入、可能携带凭据，断言凭据绝不进日志与响应体
// （根 AGENTS.md「密钥不落日志」）。
func TestUserinfoEndpointNeverLeaksCredentials(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(t, &buf)
	rr := doJSON(t, h, http.MethodPost, "/api/accounts",
		accountBody(t, "http://"+endpointUserinfoUser+":"+endpointUserinfoPass+"@127.0.0.1:9000", "", false))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400（P-3：endpoint userinfo fail-closed）:\n%s", rr.Code, rr.Body.String())
	}
	assertNoEndpointCredential(t, buf.String())
	assertNoEndpointCredential(t, rr.Body.String())
}

// TestPlaintextWarnStaysCredentialFreeForLegacyAccount —— 明文 WARN 路径的**保活**用例。
//
// 为什么需要它：P-3 后 `redactEndpoint` 变成 `model.RedactEndpointUserinfo` 的一层薄委托，而
// `warnPlaintextEndpoints` 是 handler 侧唯一的调用点——若哪天这条日志路径被整段移除，覆盖率与
// 「日志无凭据」两道防线会一起静默失效。本用例从 **store 预置的历史账号**（模拟契约上线前
// 落盘的 `http://` 账号，不带 userinfo）出发，更新为明文端点，断言：① WARN 出现；
// ② 日志里既没有 userinfo 哨兵、也没有 secretKey；③ 脱敏没有把正常端点也吃掉。
func TestPlaintextWarnStaysCredentialFreeForLegacyAccount(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	legacy, err := st.Create(&model.Account{
		Name: "legacy-plain", Endpoint: "http://minio.local:9000",
		AccessKey: "AKIATEST", SecretKey: "SECRET-SENTINEL",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	var buf bytes.Buffer
	h := New(st, slog.New(slog.NewJSONHandler(&buf, nil)), t.TempDir(), nil, "", "test", false, false).Routes()

	rr := doJSON(t, h, http.MethodPut, "/api/accounts/"+legacy.ID,
		accountBody(t, "http://127.0.0.1:9000", "", false))
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200：\n%s", rr.Code, rr.Body.String())
	}
	log := buf.String()
	if !strings.Contains(log, plaintextWarnMarker) {
		t.Fatalf("明文 WARN 路径失去覆盖/被移除：\n%s", log)
	}
	if !strings.Contains(log, "http://127.0.0.1:9000") {
		t.Errorf("WARN 未记录明文端点（脱敏不应吃掉正常端点）：\n%s", log)
	}
	assertNoEndpointCredential(t, log)
	if strings.Contains(log, "SECRET-SENTINEL") {
		t.Errorf("日志泄漏了 secretKey：\n%s", log)
	}
}
