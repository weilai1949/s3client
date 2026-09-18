package handler

// acc_helper_test.go —— 账号/桶/基建类测试的共享辅助（仅本代理新增，函数名以 acc 前缀避免冲突）。
// 约定：不改生产代码与既有测试；桩与假服务全部并发安全（-race）。

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/aws/smithy-go"
	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// ---- 错误码桩：实现 smithy.APIError，用于白盒直测 writeInternalErr 的映射分支 ----

// accCodeErr 携带固定 S3 错误码的最小 APIError 实现。
type accCodeErr struct{ code string }

func (e accCodeErr) Error() string                 { return "s3 error: " + e.code }
func (e accCodeErr) ErrorCode() string             { return e.code }
func (e accCodeErr) ErrorMessage() string          { return e.code }
func (e accCodeErr) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

// ---- 可注入错误的账号存储桩 ----

// accStubStore 可注入错误的账号存储桩：各方法返回可配置的错误/结果。
type accStubStore struct {
	mu        sync.Mutex
	listErr   error
	getErr    error // 为 nil 且 getAcc 为 nil 时返回 store.ErrNotFound
	getAcc    *model.Account
	createErr error
	updateErr error
	deleteErr error
}

func (s *accStubStore) List() ([]*model.Account, error) { return nil, s.listErr }

func (s *accStubStore) Get(id string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getAcc != nil {
		cp := *s.getAcc
		return &cp, nil
	}
	return nil, store.ErrNotFound
}

func (s *accStubStore) Create(a *model.Account) (*model.Account, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	a.ID = "acc-stub-id"
	return a.Sanitized(), nil
}

func (s *accStubStore) Update(id string, a *model.Account) (*model.Account, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	cp := *a
	cp.ID = id
	return cp.Sanitized(), nil
}

func (s *accStubStore) Delete(string) error { return s.deleteErr }
func (s *accStubStore) Ping() error         { return nil }
func (s *accStubStore) Close() error        { return nil }

// ---- 假 S3：桶操作（list/location/versioning/bucket） ----

// accListBucketsXML ListBuckets 响应体。
const accListBucketsXML = `<?xml version="1.0" encoding="UTF-8"?><ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Owner><ID>o</ID></Owner><Buckets><Bucket><Name>b</Name><CreationDate>2026-09-01T00:00:00.000Z</CreationDate></Bucket></Buckets></ListAllMyBucketsResult>`

// accBucketsOp 从请求推断桶操作键（list/location/versioning/bucket）。
func accBucketsOp(r *http.Request) string {
	q := r.URL.Query()
	switch {
	case q.Has("location"):
		return "location"
	case q.Has("versioning"):
		return "versioning"
	case r.URL.Path == "/" || r.URL.Path == "":
		return "list" // ListBuckets
	default:
		return "bucket" // PUT/DELETE/HEAD /{name}
	}
}

// accBucketsFake 通用桶操作假 S3：列桶/建桶/删桶/位置/版本控制/HeadBucket。
func accBucketsFake() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := accBucketsOp(r)
		switch {
		case op == "list" && r.Method == http.MethodGet:
			io.WriteString(w, accListBucketsXML)
		case op == "location" && r.Method == http.MethodGet:
			io.WriteString(w, `<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-west-2</LocationConstraint>`)
		case op == "versioning" && r.Method == http.MethodGet:
			io.WriteString(w, `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`)
		case op == "versioning" && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case op == "bucket" && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case op == "bucket" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case op == "bucket" && r.Method == http.MethodHead:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

// ---- 测试环境组装 ----

// accEnv 一套被测 Handler 环境（store + 完整路由 + 裸 Handler + 账号）。
type accEnv struct {
	t   *testing.T
	st  store.AccountStore
	hnd *Handler     // 裸 Handler（白盒调用内部方法）
	h   http.Handler // 完整路由栈（黑盒请求）
	acc *model.Account
}

// accDoRec 以 application/json 发一次请求并返回 recorder。
func (e *accEnv) accDoRec(method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	return doJSON(e.t, e.h, method, path, body)
}

// accNewEnv 基于真实 store 与指向 fakeURL 的账号构建环境；bucket 为默认桶（可为空）。
func accNewEnv(t *testing.T, fakeURL, bucket string) *accEnv {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "acc-fake", Endpoint: fakeURL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: bucket, PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	hnd := accNewHandler(t, st, nil, "")
	return &accEnv{t: t, st: st, hnd: hnd, h: hnd.Routes(), acc: acc}
}

// accNewHandler 用给定 store/cors/token 构造裸 Handler（静态目录为空临时目录）。
func accNewHandler(t *testing.T, st store.AccountStore, cors []string, token string) *Handler {
	t.Helper()
	return New(st, accDiscardLogger(), t.TempDir(), cors, token, "test", false, false)
}

// accDiscardLogger 丢弃输出的 slog（避免测试刷屏）。
func accDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// accStartFake 启动假 S3 并在测试结束时自动关闭。
func accStartFake(t *testing.T, hf http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(hf)
	t.Cleanup(srv.Close)
	return srv
}
