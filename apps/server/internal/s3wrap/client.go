package s3wrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"

	"github.com/weilai1949/s3client/apps/server/internal/model"
)

var (
	sharedHTTP     *ssrfAwareClient
	sharedHTTPOnce sync.Once
)

// Client 对某个账号的 S3 操作做薄封装，方便复用 SDK 接口。
type Client struct {
	acc     *model.Account
	s3      *s3.Client
	presign *s3.PresignClient
}

// Endpoint 返回该客户端配置的 endpoint（含 scheme；空表示未配置）。
// 用于跨账号 endpoint 比对（service.SameEndpoint）。
func (c *Client) Endpoint() string {
	if c == nil || c.acc == nil {
		return ""
	}
	return c.acc.Endpoint
}

// Region 返回该客户端配置的 region。用于对「均使用默认端点」的账号做同/异端判定。
func (c *Client) Region() string {
	if c == nil || c.acc == nil {
		return ""
	}
	return c.acc.Region
}

// UseSSL 返回该客户端账号配置的 TLS 开关。与 Endpoint()/Region() 一起作为
// service.SameEndpoint 的同/异端判定输入：裸端点按它补全 http/https，
// 与建 client 时 NormalizeEndpoint(endpoint, acc.UseSSL) 用的是同一个值。
func (c *Client) UseSSL() bool {
	if c == nil || c.acc == nil {
		return false
	}
	return c.acc.UseSSL
}

// New 根据账号构建 S3 客户端与预签名客户端。
func New(acc *model.Account) (*Client, error) {
	if acc == nil {
		return nil, errors.New("empty account")
	}
	if acc.AccessKey == "" || acc.SecretKey == "" {
		return nil, errors.New("missing access key or secret key")
	}
	if err := ValidateEndpoint(acc.Endpoint); err != nil {
		return nil, fmt.Errorf("endpoint: %w", err)
	}
	if err := ValidateEndpoint(acc.PublicEndpoint); err != nil {
		return nil, fmt.Errorf("publicEndpoint: %w", err)
	}
	creds := credentials.NewStaticCredentialsProvider(acc.AccessKey, acc.SecretKey, "")
	region := acc.Region
	if region == "" {
		region = "us-east-1"
	}
	// 配置加载使用 10 秒超时，防止不可达 endpoint 永久阻塞 handler goroutine。
	cfgCtx, cfgCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cfgCancel()
	cfg, err := awsconfig.LoadDefaultConfig(cfgCtx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(creds),
		awsconfig.WithHTTPClient(sharedHTTPClient()),
	)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	svc := newS3FromConfig(cfg, acc, acc.Endpoint, registerMiddlewares)
	presignEndpoint := acc.PublicEndpoint
	if presignEndpoint == "" {
		presignEndpoint = acc.Endpoint
	}
	// presign client 走独立的中间件注册：签名所需的 UNSIGNED-PAYLOAD 必须保留（否则
	// 预签名 URL 校验失败），但 metricsMiddleware 必须去掉——预签名只构造 URL、不产生
	// 真实 S3 调用，计入 s3c_s3_calls_total 会凭空抬高调用数并把延迟 p99 拉向 0
	// （分段直传每段预签名一次 → N 段注入 N 个假调用，review §R16）。
	presignSvc := newS3FromConfig(cfg, acc, presignEndpoint, registerPresignMiddlewares)
	return &Client{acc: acc, s3: svc, presign: s3.NewPresignClient(presignSvc)}, nil
}

func newS3FromConfig(cfg aws.Config, acc *model.Account, endpoint string, apiMiddleware func(*middleware.Stack) error) *s3.Client {
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if ep := NormalizeEndpoint(endpoint, acc.UseSSL); ep != "" {
			o.BaseEndpoint = aws.String(ep)
		}
		o.UsePathStyle = acc.PathStyle
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.APIOptions = append(o.APIOptions, apiMiddleware)
	})
}

// registerMiddlewares 在 smithy 栈上注册本项目的中间件（数据面 client）：
//   - unsignedPayloadSetter 置于 ResolveEndpointV2 之后（注入 UNSIGNED-PAYLOAD）；
//   - metricsMiddleware 置于 Finalize 最外层（覆盖每次 S3 调用，含传输错误）。
func registerMiddlewares(stack *middleware.Stack) error {
	if err := stack.Finalize.Insert(&unsignedPayloadSetter{}, "ResolveEndpointV2", middleware.After); err != nil {
		return err
	}
	return stack.Finalize.Add(&metricsMiddleware{}, middleware.Before)
}

// registerPresignMiddlewares 是 presign client 的注册表：只挂签名必需的
// unsignedPayloadSetter，不挂 metricsMiddleware（原因见 New 中的注释，review §R16）。
// 不用 registerMiddlewares 是刻意分叉：数据面与预签名的计量语义不同，合并会再次引入
// 假调用计数。
func registerPresignMiddlewares(stack *middleware.Stack) error {
	return stack.Finalize.Insert(&unsignedPayloadSetter{}, "ResolveEndpointV2", middleware.After)
}

const unsignedPayload = "UNSIGNED-PAYLOAD"

type unsignedPayloadSetter struct{}

func (m *unsignedPayloadSetter) ID() string { return "s3client:unsigned-payload" }

func (m *unsignedPayloadSetter) HandleFinalize(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
	ctx = v4.SetPayloadHash(ctx, unsignedPayload)
	return next.HandleFinalize(ctx, in)
}

func sharedHTTPClient() *ssrfAwareClient {
	sharedHTTPOnce.Do(func() {
		sharedHTTP = newHTTPClient()
	})
	return sharedHTTP
}

// ssrfAwareClient：禁重定向 + Dial 时拦截链路本地/云元数据 IP。
type ssrfAwareClient struct {
	client *http.Client
}

func (c *ssrfAwareClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func newHTTPClient() *ssrfAwareClient {
	inner := awshttp.NewBuildableClient().
		WithTransportOptions(func(tr *http.Transport) {
			tr.DialContext = dialContextSSRF
			tr.TLSHandshakeTimeout = 10 * time.Second
			tr.ResponseHeaderTimeout = 30 * time.Second
			tr.ExpectContinueTimeout = 5 * time.Second
			tr.IdleConnTimeout = 90 * time.Second
			tr.MaxIdleConns = 128
			tr.MaxIdleConnsPerHost = 32
			// 每 host 并发连接上限：批量操作（多账号）不无限占用 fd。
			tr.MaxConnsPerHost = 32
			// 安全关键：禁用 HTTP(S)_PROXY 环境变量，避免 S3 出站被代理到任意主机
			// （绕过 dialContextSSRF 的 IP 黑名单）。SSRF 防护只在直连下成立。
			tr.Proxy = nil
		})
	return &ssrfAwareClient{
		client: &http.Client{
			Transport:     inner.GetTransport(),
			CheckRedirect: checkRedirectDenied,
		},
	}
}

// NormalizeEndpoint 归一化 S3 端点，用于建 client 与端点比较。
//
// 规则：去首尾空白 → 去结尾斜杠 → scheme 大小写不敏感；无 scheme 时按 useSSL
// 补全（http/https）。scheme 与 host 统一小写（DNS 不区分大小写），**路径部分保留
// 原样**（可能大小写敏感）。
//
// 此函数是端点归一化的唯一实现：`service.SameEndpoint` 与建 client 的 BaseEndpoint
// 必须用同一套规则，否则会出现「比较判定为同一端点、建出的 URL 却连不上」。
// 旧实现只做大小写敏感的前缀判断，把 "HTTP://Host" 当成裸主机，产出损坏的
// "http://HTTP://Host"（FEATURES.md §K，KNOWN_ISSUES #10）。
//
// 顺序有讲究（KNOWN_ISSUES #70）：**先切分 host/path，再分别归一**。若先对整个
// rest 去尾斜杠，会把尾斜杠之后的内部空白暴露到结果末尾，而入口 TrimSpace 只做
// 一次——输出带尾随空白且二次归一化不同（不幂等）。故 host 去首尾空白、path 去
// 尾部斜杠与空白，保证输出不以空白结尾、对自身幂等。
func NormalizeEndpoint(endpoint string, useSSL bool) string {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		return ""
	}
	var scheme, rest string
	if i := strings.Index(ep, "://"); i >= 0 {
		scheme = strings.ToLower(ep[:i])
		rest = ep[i+3:]
	} else {
		scheme = "http"
		if useSSL {
			scheme = "https"
		}
		rest = ep
	}
	// 只小写 host，保留路径大小写。
	host, path := rest, ""
	if i := strings.Index(rest, "/"); i >= 0 {
		host, path = rest[:i], rest[i:]
	}
	host = strings.TrimSpace(host)
	path = strings.TrimRightFunc(path, func(r rune) bool { return r == '/' || unicode.IsSpace(r) })
	if host == "" && path == "" {
		return ""
	}
	return scheme + "://" + strings.ToLower(host) + path
}
