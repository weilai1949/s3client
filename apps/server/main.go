package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/config"
	"github.com/weilai1949/s3client/apps/server/internal/handler"
	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/service"
	"github.com/weilai1949/s3client/apps/server/internal/store"
	"github.com/weilai1949/s3client/apps/server/internal/tracing"
)

// version 由构建时注入（ldflags -X main.version=...）；缺省与发版号对齐，便于本地 go run/build。
var version = "v1.0.0"

// healthPath 是 -healthcheck 子命令探测的服务端健康端点，容器 HEALTHCHECK 走该子命令。
// 必须与 handler 注册的健康路由字面量保持一致：两处是各自硬编码（无共享常量），
// 一旦漂移，健康检查会永远失败并触发容器无谓重启循环。
const healthPath = "/api/health"

func main() {
	// 容器健康检查子命令：探测自身 /api/health，成功返回 0。
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runHealthcheck())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(runServer(ctx))
}

// runServer 组装并运行服务；返回进程退出码：0 = 正常优雅关停（信号 / 主动 Stop），
// 1 = 启动失败或运行期服务错误（端口占用等）。错误终止必须非零，否则 systemd/Docker
// 的 on-failure 重启策略不会生效，与自述退出码契约矛盾（review §R15a）。
// 信号/上下文经 ctx 注入，便于测试直接驱动启停。
func runServer(ctx context.Context) int {
	cfg := config.FromEnv()

	level := parseLevel(cfg.LogLevel)
	var logHandler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	if cfg.LogJSON {
		logHandler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// 配置安全校验：短 S3C_TOKEN 拒绝启动；非回环监听必须开启鉴权。
	if err := cfg.Validate(); err != nil {
		logger.Error("配置校验失败", "err", err)
		return 1
	}

	// 明文落盘告警：json / sqlite 未设 S3C_STORE_KEY 时 secretKey 明文写入 DataDir（roadmap §5.1 R3）。
	if w := cfg.StorePlaintextWarning(); w != "" {
		logger.Warn(w)
	}

	// 可选加固：拒绝私网 / 回环 S3 端点（默认放行，见 ADR-003；roadmap §5.1 R8）。
	s3wrap.SetDenyPrivateNetworks(cfg.SSRFDenyPrivate)

	// 单写者锁：文件型 store + 内存 JobRegistry 只支持单副本，第二个进程必须被挡住而不是
	// 静默互相覆盖写入（roadmap §5.1 R4）。
	releaseLock, err := store.AcquireDataDirLock(cfg.DataDir)
	if err != nil {
		logger.Error("data dir lock", "err", err)
		return 1
	}
	defer releaseLock()

	// 账号存储（json / sqlite / encrypted）
	st, err := store.Open(cfg.DataDir, cfg.StoreDriver, cfg.StoreKey)
	if err != nil {
		logger.Error("init store", "err", err)
		return 1
	}
	defer func() { _ = st.Close() }()

	h := handler.New(st, logger, cfg.StaticDir, cfg.CORSOrigins, cfg.Token, version, cfg.ExposeMetrics, cfg.ExposeOpenAPI)
	h.SetCSPConnectSrc(cfg.CSPConnectSrc)
	// Token 作用域（S3C_TOKEN_SCOPES）：按 token 限定只读 / 桶前缀 / 账号 / 过期；
	// 未登记的 token 保持全权（向后兼容）。必须在 Routes() 前设置。
	h.SetTokenScopes(cfg.ScopeFor)
	// 仅信任显式配置的反向代理 IP 的 X-Forwarded-For（默认不信任，防直连伪造绕过限速）。
	h.SetTrustedProxies(cfg.TrustedProxies)
	// 异步任务清单落盘：重启后未完成任务标记为 interrupted，便于对账
	// 「复制成功但源未删除」的移动任务（KNOWN_ISSUES #19）。
	h.SetJobPersister(service.NewFileJobPersister(filepath.Join(cfg.DataDir, "jobs.json")))
	// 计划任务清单落盘（0600 原子写）：重启自动恢复计划与排期（ROADMAP #6）。
	h.SetSchedulePersister(service.NewFileSchedulePersister(filepath.Join(cfg.DataDir, "schedules.json")))
	// 数据目录用于 /api/metrics 的卷容量（statfs）与关停耗时落盘；
	// 上一次优雅关停的耗时在启动时载入，使该指标跨进程可读（ROADMAP #18）。
	h.SetDataDir(cfg.DataDir)
	h.LoadLastShutdown()

	// OTel tracing（可选）：S3C_OTEL_ENDPOINT 为空即禁用，Middleware 原样透传、零开销。
	// Endpoint / 采样比例非法时 fail-closed 拒绝启动（与其余配置同口径）。
	tracer, err := tracing.New(tracing.Config{
		Endpoint:    cfg.OTelEndpoint,
		ServiceName: cfg.OTelServiceName,
		SampleRatio: cfg.OTelSampleRatio,
	})
	if err != nil {
		logger.Error("init tracing", "err", err)
		return 1
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           tracer.Middleware(h.Routes()),
		ReadHeaderTimeout: 15 * time.Second,
		// 仅限制读取请求体阶段，抵御慢速请求体攻击；不设 WriteTimeout，
		// 避免截断大文件（proxy / download-zip）的流式输出。
		ReadTimeout: 60 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	// 基于 注入 ctx 派生可取消上下文：ListenAndServe 失败时也能进入优雅关闭流程。
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// serveErrCh 容量 1：ListenAndServe 的失败先入 channel 再 cancel()，主流程关停收尾后
	// 非阻塞读取即可区分「信号优雅关停（0）」与「服务自身出错（非 0）」；channel 通信天然
	// 建立 happens-before，无需额外同步。
	serveErrCh := make(chan error, 1)
	go func() {
		logger.Info("s3client server",
			"version", version,
			"addr", cfg.Addr,
			"dataDir", cfg.DataDir,
			"store", cfg.StoreDriver,
			"staticDir", cfg.StaticDir,
			"auth", cfg.Token != "",
			"cors", corsSummary(cfg.CORSOrigins),
			"region", cfg.Region,
			"ssrfDenyPrivate", cfg.SSRFDenyPrivate,
			"tracing", tracer.Enabled(),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			serveErrCh <- err
			cancel()
		}
	}()

	<-ctx.Done()
	// 关停耗时从收到信号起算，覆盖「取消在册任务 + 等待在途请求」两段
	// （与日志 "shutting down..." → "shutdown complete" 的时间差同口径）。
	shutdownStart := time.Now()
	logger.Info("shutting down...")
	h.Shutdown()
	shutdownTimeout := time.Duration(cfg.ShutdownTimeoutSec) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx) // ctx 超时在生产不可达；shutting down 流程已写 INFO。
	// 落盘后进程即退出，/api/metrics 无法再 scrape 本次值 —— 指标由下次启动载入暴露。
	h.RecordShutdown(time.Since(shutdownStart))
	// tracing 收尾：关闭后台导出并刷出剩余 span（有界等待）。导出失败已在包内记日志，
	// 此处丢弃 Close 错误不会吞掉对外可见的失败信息。
	traceCtx, cancelTrace := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTrace()
	_ = tracer.Close(traceCtx)
	logger.Info("shutdown complete")
	// goroutine 已记录过 "server error"；此处只决定退出码，不重复刷日志。
	select {
	case <-serveErrCh:
		return 1
	default:
		return 0
	}
}
func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// runHealthcheck 探测自身健康端点，用于容器 HEALTHCHECK（-healthcheck 子命令）：
// GET http://<host>:<port>/healthPath，3 秒超时；200 → 0，连接失败 / 非 200 → 1。
//
// 地址取 S3C_ADDR：通配 host（如 ":8080"，服务绑全接口）时探测回退 127.0.0.1——回环上
// 同一端口必可达，避免把探测打到外部网卡地址。SplitHostPort 失败（地址缺端口，如
// "no-port-in-here"）时
// "no-port-in-here"）时 host 与 port 均为空串：host 置 127.0.0.1、port 为空拼出
// "http://127.0.0.1:/api/health"，按 http 方案默认 80 端口连接，几乎必然失败返回 1。
// 这是刻意的 fail-closed：同一非法地址下服务端自身也无法监听（missing port），
// 错误配置必须由健康检查暴露，而不是假装健康。
func runHealthcheck() int {
	addr := config.FromEnv().Addr
	host, port, _ := net.SplitHostPort(addr)
	if host == "0.0.0.0" || host == "::" || host == "" {
		host = "127.0.0.1"
	}
	// IPv6 字面量必须经 JoinHostPort 加方括号：直接拼 "::1:8080" 会产出
	// "http://::1:8080/api/health"，url.Parse 报 invalid port → client.Get 失败 → 恒返回 1，
	// 让健康的容器被 HEALTHCHECK 判死并反复重启（[::1]:port 是 IsLoopbackAddr 认可的合法
	// 回环监听地址、允许不设 token，属于会真实用到的一类配置）。
	base := fmt.Sprintf("http://%s%s", net.JoinHostPort(host, port), healthPath)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck error:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck status: %d\n", resp.StatusCode)
		return 1
	}
	return 0
}

func corsSummary(origins []string) string {
	if len(origins) == 0 {
		return "safe-default(localhost/tauri)"
	}
	return strings.Join(origins, ",")
}
