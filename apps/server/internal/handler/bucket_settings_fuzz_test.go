package handler

// bucket_settings_fuzz_test.go —— 桶策略 JSON 边界的原生 fuzz（stdlib `testing.F`，无新依赖）。
//
// 为什么在 handler：`putBucketPolicy` 是用户策略字符串抵达 S3 的**唯一入口**，其校验只有
// 一处 `json.Valid`。本 fuzz 用真实 handler + 假 S3 端到端跑：任意策略串的响应码必须与
// `json.Valid` 判定严格一致，且任何输入都不得出现 5xx（400 是唯一的非法输入出口）。
//
// 运行（有界；CI 见 .github/workflows/fuzz.yml）：
//
//	cd apps/server && go test ./internal/handler/ -run '^$' -fuzz '^FuzzPutBucketPolicyBody$' -fuzztime=10s

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// fuzzPolicyS3 是只接受 PutBucketPolicy / DeleteBucketPolicy 的假 S3。
func fuzzPolicyS3(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut, http.MethodDelete:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// fuzzPolicyEnv 组装一套「真实 store + 真实路由 + 指向假 S3 的账号」环境。
func fuzzPolicyEnv(f *testing.F) (*Handler, string) {
	f.Helper()
	dir, err := os.MkdirTemp("", "handler-fuzz-policy-")
	if err != nil {
		f.Fatalf("创建临时目录: %v", err)
	}
	f.Cleanup(func() { os.RemoveAll(dir) })

	st, err := store.New(filepath.Join(dir, "accounts.json"))
	if err != nil {
		f.Fatalf("store: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(fuzzPolicyS3))
	f.Cleanup(srv.Close)
	acc, err := st.Create(&model.Account{
		Name: "fuzz", Endpoint: srv.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "fuzz-bucket", PathStyle: true,
	})
	if err != nil {
		f.Fatalf("create account: %v", err)
	}
	h := New(st, slog.New(slog.NewTextHandler(io.Discard, nil)), dir, nil, "", "test", false, false)
	return h, acc.ID
}

// FuzzPutBucketPolicyBody 验证策略 JSON 校验的 fail-closed 与响应码契约。
func FuzzPutBucketPolicyBody(f *testing.F) {
	h, accID := fuzzPolicyEnv(f)

	f.Add("")
	f.Add("{}")
	f.Add("[]")
	f.Add("null")
	f.Add("not json")
	f.Add(`{"Version":"2012-10-17","Statement":[]}`)
	f.Add(`{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*","Resource":"*"}]}`)
	f.Add(`{"Statement":[`)

	f.Fuzz(func(t *testing.T, policy string) {
		// JSON 编解码会把非法 UTF-8 替换为 U+FFFD，原始串与 handler 实际看到的串不再一致，
		// 无法做差分断言——这类输入直接跳过（不影响下面的等价类覆盖）。
		if !utf8.ValidString(policy) {
			t.Skip()
		}
		body, err := json.Marshal(map[string]string{"bucket": "fuzz-bucket", "policy": policy})
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/buckets/policy", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", accID)
		rr := httptest.NewRecorder()

		h.putBucketPolicy(rr, req)

		wantBad := policy != "" && !json.Valid([]byte(policy))
		switch {
		case rr.Code >= http.StatusInternalServerError:
			t.Fatalf("policy=%q -> %d（5xx 不是合法出口）body=%s", policy, rr.Code, rr.Body.String())
		case wantBad && rr.Code != http.StatusBadRequest:
			t.Fatalf("非法 JSON policy=%q -> %d，期望 400", policy, rr.Code)
		case !wantBad && rr.Code != http.StatusOK:
			t.Fatalf("策略为空（清除）或 JSON 合法 policy=%q -> %d，期望 200；body=%s",
				policy, rr.Code, rr.Body.String())
		}
	})
}
