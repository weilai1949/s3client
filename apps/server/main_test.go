package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/config"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// TestParseLevel 日志级别解析：全部分支表驱动。
func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"DEBUG", slog.LevelInfo}, // 大小写敏感，未识别回退 info
	}
	for _, c := range cases {
		if got := parseLevel(c.in); got != c.want {
			t.Fatalf("parseLevel(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestIsLoopbackAddr 回环判定：显式回环与无法解析按回环处理；
// 无 host（":8080"，net.Listen 绑 [::] 全接口）必须按**非**回环处理（C2）。
func TestIsLoopbackAddr(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"127.0.0.1:", true},
		{"[::1]:8080", true},
		{"::1:8080", true},
		{":8080", false},      // 未指定 host = 绑定全部网卡，等价 0.0.0.0（C2）
		{"not an addr", true}, // 解析失败本就无法监听，按回环避免误报
		{"0.0.0.0:8080", false},
		{"192.168.1.5:8080", false},
	}
	for _, c := range cases {
		if got := config.IsLoopbackAddr(c.in); got != c.want {
			t.Fatalf("config.IsLoopbackAddr(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestCorsSummary CORS 汇总：空缺省安全提示，非空逗号拼接。
func TestCorsSummary(t *testing.T) {
	if got := corsSummary(nil); got != "safe-default(localhost/tauri)" {
		t.Fatalf("corsSummary(nil) = %q", got)
	}
	if got := corsSummary([]string{"http://a", "http://b"}); got != "http://a,http://b" {
		t.Fatalf("corsSummary = %q", got)
	}
}

// TestRunHealthcheck 健康检查四态：200→0、非 200→1、连接失败→1；
// 通配 host（":port"，服务绑全接口）回退 127.0.0.1 探测→0。
func TestRunHealthcheck(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(okSrv.Close)
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(badSrv.Close)
	_, okPort, err := net.SplitHostPort(okSrv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split okSrv addr: %v", err)
	}

	// IPv6 回环字面量：[::1]:port 是合法且被 IsLoopbackAddr 认可的回环监听地址
	// （因此允许不设 token），但旧实现拼出 "http://::1:8080/api/health"——url.Parse 直接
	// 报 invalid port，健康检查恒为 1，容器 HEALTHCHECK 会把健康服务判死并反复重启。
	var v6Addr string
	if ln, err := net.Listen("tcp", "[::1]:0"); err != nil {
		t.Logf("IPv6 回环不可用，跳过该用例: %v", err)
	} else {
		v6Srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		v6Srv.Listener = ln
		v6Srv.Start()
		t.Cleanup(v6Srv.Close)
		v6Addr = ln.Addr().String()
	}

	cases := []struct {
		name string
		addr string
		want int
	}{
		{"ok", okSrv.Listener.Addr().String(), 0},
		{"non-200", badSrv.Listener.Addr().String(), 1},
		{"refused", "127.0.0.1:1", 1},
		{"wildcard host falls back to loopback", ":" + okPort, 0},
	}
	if v6Addr != "" {
		cases = append(cases, struct {
			name string
			addr string
			want int
		}{"ipv6 loopback literal", v6Addr, 0})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("S3C_ADDR", c.addr)
			if got := runHealthcheck(); got != c.want {
				t.Fatalf("runHealthcheck(addr=%q) = %d, want %d", c.addr, got, c.want)
			}
		})
	}
}

// reserveLoopbackPort 申请一个回环空闲端口（先绑再放，缓解竞态）。
func reserveLoopbackPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close probe listener: %v", err)
	}
	return addr
}

// pollHealth 轮询 /api/health 直到 200 或超时。
func pollHealth(t *testing.T, url string) bool {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestRunServerStartsAndShutsDownGracefully runServer 直接驱动：健康端点可访问、
// ctx 取消后优雅退出返回 0、端口释放。
func TestRunServerStartsAndShutsDownGracefully(t *testing.T) {
	addr := reserveLoopbackPort(t)
	t.Setenv("S3C_ADDR", addr)
	t.Setenv("S3C_DATA_DIR", t.TempDir())
	t.Setenv("S3C_STORE_DRIVER", "json")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1") // 安全默认：json + 空 key 需显式 opt-in
	t.Setenv("S3C_TOKEN", "unit-test-token-0123456789")
	t.Setenv("S3C_SHUTDOWN_TIMEOUT", "5")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() {
		done <- runServer(ctx)
	}()

	base := fmt.Sprintf("http://%s/api/health", addr)
	if !pollHealth(t, base) {
		cancel()
		<-done
		t.Fatalf("server did not become healthy at %s", base)
	}
	cancel()
	if code := <-done; code != 0 {
		t.Fatalf("runServer return = %d, want 0", code)
	}
	// 端口应已释放：立刻重绑成功即视为关闭完成。
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("port %s should be released after shutdown: %v", addr, err)
	}
	_ = l.Close()
}

// TestRunServerRejectsNonLoopbackWithoutToken 非回环监听 + 无 token → 拒绝启动返回 1。
func TestRunServerRejectsNonLoopbackWithoutToken(t *testing.T) {
	t.Setenv("S3C_ADDR", "0.0.0.0:0")
	t.Setenv("S3C_TOKEN", "")
	t.Setenv("S3C_DATA_DIR", t.TempDir())
	if code := runServer(context.Background()); code != 1 {
		t.Fatalf("runServer(non-loopback, no token) = %d, want 1", code)
	}
}

// TestRunServerStoreInitFails 存储初始化失败（encrypted 缺 key）→ 返回 1。
func TestRunServerStoreInitFails(t *testing.T) {
	addr := reserveLoopbackPort(t)
	t.Setenv("S3C_ADDR", addr)
	t.Setenv("S3C_TOKEN", "unit-test-token-0123456789")
	t.Setenv("S3C_DATA_DIR", t.TempDir())
	t.Setenv("S3C_STORE_DRIVER", "encrypted")
	t.Setenv("S3C_STORE_KEY", "")
	if code := runServer(context.Background()); code != 1 {
		t.Fatalf("runServer(store init failure) = %d, want 1", code)
	}
}

// TestRunServerListenBindFailure 端口被占用 → ListenServe败 → 走完清理流程后必须返回
// 非零（review §R15a：曾返回 0，与自述退出码契约矛盾，systemd/Docker on-failure 不重启）。
// 正常优雅关停仍返回 0，由 TestRunServerStartsAndShutsDownGracefully 锚定。
func TestRunServerListenBindFailure(t *testing.T) {
	// 先占用端口，使 ListenAndServe 绑定失败。
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind blocker: %v", err)
	}
	defer l.Close()
	t.Setenv("S3C_ADDR", l.Addr().String())
	t.Setenv("S3C_DATA_DIR", t.TempDir())
	t.Setenv("S3C_STORE_DRIVER", "json")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	t.Setenv("S3C_TOKEN", "unit-test-token-0123456789")
	code := runServer(context.Background())
	if code == 0 {
		t.Fatal("runServer(bind failure) = 0, want non-zero（服务因错误终止必须以非零码退出）")
	}
}

// TestMainServerExitsNonZeroOnListenFailure 进程级退出码锚定（review §R15a）：
// 端口占用导致 ListenAndServe 失败时，main() 必须以非零码退出并留下 server error 日志，
// 部署侧的 on-failure 重启策略据此生效；-healthcheck 子命令的退出码互不影响。
func TestMainServerExitsNonZeroOnListenFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("依赖类 unix 端口与进程退出码语义")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind blocker: %v", err)
	}
	defer l.Close()
	cmd := childCmd(t, "server", map[string]string{
		"S3C_ADDR":                  l.Addr().String(),
		"S3C_DATA_DIR":              t.TempDir(),
		"S3C_TOKEN":                 "unit-test-token-0123456789",
		"S3C_STORE_DRIVER":          "json",
		"S3C_ALLOW_PLAINTEXT_STORE": "1",
		"S3C_SHUTDOWN_TIMEOUT":      "5",
	})
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("listen failure child must exit non-zero, out=%s", out)
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() == 0 {
		t.Fatalf("want non-zero exit, got err=%v out=%s", err, out)
	}
	if !strings.Contains(string(out), "server error") {
		t.Fatalf("exit非零但缺少 server error 日志（退出原因必须可见）, out=%s", out)
	}
}

// TestMainChild 子进程入口：按 S3C_MAIN_CHILD 模式直接调用 main()（内部 os.Exit）。
func TestMainChild(t *testing.T) {
	mode := os.Getenv("S3C_MAIN_CHILD")
	if mode == "" {
		t.Skip("child only")
	}
	switch mode {
	case "healthcheck":
		os.Args = []string{os.Args[0], "-healthcheck"}
	}
	main()
}

// childCmd 构造子进程（继承环境 + 注入变量），超时强杀兜底。
func childCmd(t *testing.T, mode string, env map[string]string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainChild$", "-test.count=1")
	cmd.Env = append(os.Environ(), "S3C_MAIN_CHILD="+mode)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd
}

// TestMainHealthcheckSubprocess 用 fork 方式驱动 main() 的 -healthcheck 分支。
func TestMainHealthcheckSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fork 模式依赖类 unix 信号与进程语义")
	}
	t.Run("ok", func(t *testing.T) {
		okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(okSrv.Close)
		cmd := childCmd(t, "healthcheck", map[string]string{"S3C_ADDR": okSrv.Listener.Addr().String()})
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("healthcheck child should exit 0, got err=%v out=%s", err, out)
		}
	})
	t.Run("refused", func(t *testing.T) {
		cmd := childCmd(t, "healthcheck", map[string]string{"S3C_ADDR": "127.0.0.1:1"})
		if err := cmd.Run(); err == nil {
			t.Fatal("healthcheck against refused port should exit non-zero")
		}
	})
}

// TestMainServerSubprocess fork 子进程跑完整 main()：启动→健康→SIGTERM→优雅退出 0。
func TestMainServerSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("依赖 SIGTERM")
	}
	addr := reserveLoopbackPort(t)
	cmd := childCmd(t, "server", map[string]string{
		"S3C_ADDR":                  addr,
		"S3C_DATA_DIR":              t.TempDir(),
		"S3C_TOKEN":                 "unit-test-token-0123456789",
		"S3C_STORE_DRIVER":          "json",
		"S3C_ALLOW_PLAINTEXT_STORE": "1",
		"S3C_SHUTDOWN_TIMEOUT":      "5",
	})
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	base := fmt.Sprintf("http://%s/api/health", addr)
	if !pollHealth(t, base) {
		_ = cmd.Process.Signal(syscall.SIGKILL)
		t.Fatalf("child server not healthy at %s", base)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal child: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			var ee *exec.ExitError
			if !errors.As(err, &ee) || ee.ExitCode() != 0 {
				t.Fatalf("child should exit 0 after SIGTERM, got %v", err)
			}
		}
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("child did not exit after SIGTERM")
	}
}

// TestMainServerWarnsPlaintextStore 子进程跑 main()：sqlite + 空 S3C_STORE_KEY 且
// 显式 S3C_ALLOW_PLAINTEXT_STORE=1 时，启动日志必须出现明文落盘告警（roadmap §5.1 R3），
// 否则运维无从察觉生产误用明文驱动。安全默认下不 opt-in 会直接硬失败（见下个测试）。
func TestMainServerWarnsPlaintextStore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("依赖 SIGTERM")
	}
	addr := reserveLoopbackPort(t)
	cmd := childCmd(t, "server", map[string]string{
		"S3C_ADDR":                  addr,
		"S3C_DATA_DIR":              t.TempDir(),
		"S3C_TOKEN":                 "unit-test-token-0123456789",
		"S3C_STORE_DRIVER":          "sqlite",
		"S3C_STORE_KEY":             "",
		"S3C_ALLOW_PLAINTEXT_STORE": "1",
		"S3C_SHUTDOWN_TIMEOUT":      "5",
	})
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	if !pollHealth(t, fmt.Sprintf("http://%s/api/health", addr)) {
		_ = cmd.Process.Signal(syscall.SIGKILL)
		t.Fatalf("child server not healthy, out=%s", out.String())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal child: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("child did not exit after SIGTERM")
	}
	if !strings.Contains(out.String(), "明文落盘") {
		t.Fatalf("startup log missing plaintext-store warning, got: %s", out.String())
	}
}

// TestMainServerRejectsPlaintextStoreWithoutOptIn 安全默认（KNOWN_ISSUES #29/#31）：
// json / sqlite + 空 S3C_STORE_KEY 且未显式 S3C_ALLOW_PLAINTEXT_STORE=1 时，
// 进程必须在启动阶段硬失败（非 0 退出），且报错指明两条出路；不得泄露密钥值。
func TestMainServerRejectsPlaintextStoreWithoutOptIn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("依赖子进程退出码语义")
	}
	cmd := childCmd(t, "server", map[string]string{
		"S3C_ADDR":                  reserveLoopbackPort(t),
		"S3C_DATA_DIR":              t.TempDir(),
		"S3C_TOKEN":                 "unit-test-token-0123456789",
		"S3C_STORE_DRIVER":          "json",
		"S3C_STORE_KEY":             "",
		"S3C_ALLOW_PLAINTEXT_STORE": "",
	})
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("json + empty key without opt-in must exit non-zero, out=%s", out)
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() == 0 {
		t.Fatalf("want non-zero exit, got err=%v out=%s", err, out)
	}
	for _, want := range []string{"S3C_STORE_KEY", "S3C_ALLOW_PLAINTEXT_STORE"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("startup error missing %q, got: %s", want, out)
		}
	}
}

// TestMainServerAllowsPlaintextStoreWithOptIn 显式 opt-in 后 json + 空 key 可正常启动
// 并可被 SIGTERM 优雅停止（返回 0），证明 opt-in 是唯一的明文落盘放行开关。
func TestMainServerAllowsPlaintextStoreWithOptIn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("依赖 SIGTERM")
	}
	addr := reserveLoopbackPort(t)
	cmd := childCmd(t, "server", map[string]string{
		"S3C_ADDR":                  addr,
		"S3C_DATA_DIR":              t.TempDir(),
		"S3C_TOKEN":                 "unit-test-token-0123456789",
		"S3C_STORE_DRIVER":          "json",
		"S3C_STORE_KEY":             "",
		"S3C_ALLOW_PLAINTEXT_STORE": "1",
		"S3C_SHUTDOWN_TIMEOUT":      "5",
	})
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	if !pollHealth(t, fmt.Sprintf("http://%s/api/health", addr)) {
		_ = cmd.Process.Signal(syscall.SIGKILL)
		t.Fatal("child server with plaintext opt-in should become healthy")
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal child: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			var ee *exec.ExitError
			if !errors.As(err, &ee) || ee.ExitCode() != 0 {
				t.Fatalf("child should exit 0 after SIGTERM, got %v", err)
			}
		}
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("child did not exit after SIGTERM")
	}
}

// TestRunServerRejectsLockedDataDir 同一 DataDir 已被占用时拒绝启动并返回 1
// （roadmap §5.1 R4：文件型 store 必须单副本，否则写覆盖 / 任务重复）。
// 非 unix 平台的锁是 no-op，跳过。
func TestRunServerRejectsLockedDataDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("非 unix 平台无跨进程文件锁（lock_other.go 为 no-op）")
	}
	dir := t.TempDir()
	release, err := store.AcquireDataDirLock(dir)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	defer release()

	t.Setenv("S3C_ADDR", reserveLoopbackPort(t))
	t.Setenv("S3C_DATA_DIR", dir)
	t.Setenv("S3C_TOKEN", "unit-test-token-0123456789")
	t.Setenv("S3C_STORE_DRIVER", "json")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	if code := runServer(context.Background()); code != 1 {
		t.Fatalf("runServer(locked data dir) = %d, want 1", code)
	}
}

// TestRunServerJSONLog S3C_LOG_JSON 开关：JSON handler 分支正常启停。
func TestRunServerJSONLog(t *testing.T) {
	addr := reserveLoopbackPort(t)
	t.Setenv("S3C_ADDR", addr)
	t.Setenv("S3C_DATA_DIR", t.TempDir())
	t.Setenv("S3C_STORE_DRIVER", "json")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	t.Setenv("S3C_TOKEN", "unit-test-token-0123456789")
	t.Setenv("S3C_LOG_JSON", "1")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- runServer(ctx) }()
	if !pollHealth(t, fmt.Sprintf("http://%s/api/health", addr)) {
		cancel()
		<-done
		t.Fatal("server not healthy")
	}
	cancel()
	if code := <-done; code != 0 {
		t.Fatalf("runServer(json log) = %d, want 0", code)
	}
}

// TestRunHealthcheckBadAddr 地址缺端口：SplitHostPort 失败时 host/port 均为空串，探测 URL
// 退化为 http://127.0.0.1:/api/health——按 http 方案默认 80 端口连接，几乎必然失败返回 1。
// 这是 fail-closed：同一非法地址下服务端自身也无法监听（missing port），错误配置必须由
// 健康检查暴露，而不是假装健康。
func TestRunHealthcheckBadAddr(t *testing.T) {
	t.Setenv("S3C_ADDR", "no-port-in-here")
	if got := runHealthcheck(); got != 1 {
		t.Fatalf("runHealthcheck(bad addr) = %d, want 1", got)
	}
}

// TestRunServerShortTokenRejects 短 token 拒绝启动返回 1。
// 即便监听回环，短口令易被暴力猜测，配置校验须硬失败。
func TestRunServerShortTokenRejects(t *testing.T) {
	addr := reserveLoopbackPort(t)
	t.Setenv("S3C_ADDR", addr)
	t.Setenv("S3C_TOKEN", "short")
	t.Setenv("S3C_DATA_DIR", t.TempDir())
	t.Setenv("S3C_STORE_DRIVER", "json")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	t.Setenv("S3C_LOG_FORMAT", "text")
	if code := runServer(context.Background()); code != 1 {
		t.Fatalf("runServer(short token) = %d, want 1 (硬失败)", code)
	}
}
