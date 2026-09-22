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

	"github.com/weilai1949/s3clinet/apps/server/internal/config"
	"github.com/weilai1949/s3clinet/apps/server/internal/handler"
	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3clinet/apps/server/internal/service"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// version 由构建时注入（ldflags -X main.version=...）；缺省与发版号对齐，便于本地 go run/build。
var version = "v1.0.0"

// healthPath 供容器 HEALTHCHECK 使用。
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

// runServer 组装并运行服务；返回进程退出码（0 正常、1 启动失败）。
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
	// 仅信任显式配置的反向代理 IP 的 X-Forwarded-For（默认不信任，防直连伪造绕过限速）。
	h.SetTrustedProxies(cfg.TrustedProxies)
	// 异步任务清单落盘：重启后未完成任务标记为 interrupted，便于对账
	// 「复制成功但源未删除」的移动任务（todolist #19）。
	h.SetJobPersister(service.NewFileJobPersister(filepath.Join(cfg.DataDir, "jobs.json")))

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 15 * time.Second,
		// 仅限制读取请求体阶段，抵御慢速请求体攻击；不设 WriteTimeout，
		// 避免截断大文件（proxy / download-zip）的流式输出。
		ReadTimeout: 60 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	// 基于 注入 ctx 派生可取消上下文：ListenAndServe 失败时也能进入优雅关闭流程。
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		logger.Info("s3clinet server",
			"version", version,
			"addr", cfg.Addr,
			"dataDir", cfg.DataDir,
			"store", cfg.StoreDriver,
			"staticDir", cfg.StaticDir,
			"auth", cfg.Token != "",
			"cors", corsSummary(cfg.CORSOrigins),
			"region", cfg.Region,
			"ssrfDenyPrivate", cfg.SSRFDenyPrivate,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")
	h.Shutdown()
	shutdownTimeout := time.Duration(cfg.ShutdownTimeoutSec) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx) // ctx 超时在生产不可达；shutting down 流程已写 INFO。
	logger.Info("shutdown complete")
	return 0
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

// runHealthcheck 探测自身健康端点，用于容器 HEALTHCHECK。
func runHealthcheck() int {
	addr := config.FromEnv().Addr
	// net.SplitHostPort 对 "host:port" 格式失败时回退到 127.0.0.1:8080——主路径不会触发。
	host, port, _ := net.SplitHostPort(addr)
	if host == "0.0.0.0" || host == "::" || host == "" {
		host = "127.0.0.1"
	}
	base := fmt.Sprintf("http://%s:%s%s", host, port, healthPath)
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
