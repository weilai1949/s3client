package handler

// schedules_test.go —— 计划任务5 端点（ROADMAP §三 #6）的行为测试。
//
// 覆盖面：CRUD 校验矩阵（400/404）、手动触发（202 + 异步真实假 S3 跑通 +
// 409 不叠加 + 404 账号已删 + 503 在册满）、调度器到点触发端到端、
// 落盘恢复、作用域（readonly 拦写 / prefixes 按计划桶校验）。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/config"
	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/service"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

// schedAccounts 起两个指向假 S3 的账号；返回 (handler, store, srcID, dstID)。
// fake 返回其 URL，两个账号共用同一假 S3（同 endpoint，走 CopyObject 分支也可，
// 假 S3 对无 X-Amz-Copy-Source 的 PUT 直接 200，对 list-type=2 返回 XML）。
func schedAccounts(t *testing.T) (*Handler, *store.Store, string, string) {
	t.Helper()
	fake := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>false</IsTruncated></ListBucketResult>`)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	src, err := st.Create(&model.Account{
		Name: "src", Endpoint: fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "src-bucket", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create src: %v", err)
	}
	dst, err := st.Create(&model.Account{
		Name: "dst", Endpoint: fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "dst-bucket", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create dst: %v", err)
	}
	h := New(st, gapLogger(), t.TempDir(), nil, "", "test", false, false)
	t.Cleanup(h.Shutdown)
	return h, st, src.ID, dst.ID
}

// schedBody 构造合法计划请求体（可被 mutator 覆盖个别字段做反例）。
func schedBody(srcID, dstID string, mutate func(m map[string]any)) string {
	m := map[string]any{
		"sourceAccountId": srcID,
		"sourceBucket":    "src-bucket",
		"sourcePrefix":    "data/",
		"targetAccountId": dstID,
		"targetBucket":    "dst-bucket",
		"targetPrefix":    "backup/",
		"mode":            "etag",
		"cron":            "0 2 * * *",
	}
	if mutate != nil {
		mutate(m)
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// ---- 列表 ----

func TestSchedulesListEmptyIsArray(t *testing.T) {
	h, _, _, _ := schedAccounts(t)
	rr := doJSON(t, h.Routes(), http.MethodGet, "/api/schedules", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body)
	}
	// 空清单必须是 [] 而非 null（前端直接 .length/map）。
	if !strings.Contains(rr.Body.String(), `"schedules":[]`) {
		t.Fatalf("body = %s, want schedules:[]", rr.Body)
	}
}

// ---- 创建 ----

func TestSchedulesCreateValid(t *testing.T) {
	h, _, srcID, dstID := schedAccounts(t)
	h.SetSchedulePersister(service.NewFileSchedulePersister(filepath.Join(t.TempDir(), "schedules.json")))

	rr := doJSON(t, h.Routes(), http.MethodPost, "/api/schedules", schedBody(srcID, dstID, nil))
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rr.Code, rr.Body)
	}
	var resp struct {
		Schedule service.Schedule `json:"schedule"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body)
	}
	s := resp.Schedule
	if s.ID == "" || s.Cron != "0 2 * * *" || !s.Enabled {
		t.Errorf("schedule = %+v", s)
	}
	// 空 mode 归一为 etag（API 回显明确值，而非空串）。
	if s.Mode != service.CompareETag {
		t.Errorf("mode = %q, want etag", s.Mode)
	}
	// 下一次触发必须已排期（cron 0 2 * * * → 未来某天 02:00）。
	if s.NextRunAt.IsZero() || !s.NextRunAt.After(time.Now()) {
		t.Errorf("nextRunAt = %v, want future", s.NextRunAt)
	}
	if !s.LastRunAt.IsZero() {
		t.Errorf("lastRunAt 应为零值（从未运行）: %v", s.LastRunAt)
	}

	// 列表可见 + 落盘文件存在。
	listRR := doJSON(t, h.Routes(), http.MethodGet, "/api/schedules", "")
	if !strings.Contains(listRR.Body.String(), s.ID) {
		t.Errorf("list 缺新建计划: %s", listRR.Body)
	}

	// 空 mode 归一为 etag（API 回显明确值，而非空串）。
	rr = doJSON(t, h.Routes(), http.MethodPost, "/api/schedules",
		schedBody(srcID, dstID, func(m map[string]any) { m["mode"] = "" }))
	if rr.Code != http.StatusCreated {
		t.Fatalf("空 mode status = %d; body=%s", rr.Code, rr.Body)
	}
	var emptyModeResp struct {
		Schedule service.Schedule `json:"schedule"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &emptyModeResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if emptyModeResp.Schedule.Mode != service.CompareETag {
		t.Errorf("空 mode 归一 = %q, want etag", emptyModeResp.Schedule.Mode)
	}
}

func TestSchedulesCreateValidation(t *testing.T) {
	h, st, srcID, dstID := schedAccounts(t)

	// 无默认桶账号（建号时指定空 Bucket）→ 桶校验 400。
	noBucket, err := st.Create(&model.Account{
		Name: "nb", Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create noBucket: %v", err)
	}
	// 缺 SecretKey 的账号 → 客户端构建失败 → 400 invalid config。
	badCfg, err := st.Create(&model.Account{
		Name: "bad", Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		AccessKey: "ak", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create badCfg: %v", err)
	}

	cases := []struct {
		name string
		body string
		want int
		msg  string
	}{
		{"bad json", `{`, 400, "invalid request body"},
		{"缺源账号", schedBody(srcID, dstID, func(m map[string]any) { m["sourceAccountId"] = "" }), 400, "sourceAccountId"},
		{"缺目标账号", schedBody(srcID, dstID, func(m map[string]any) { m["targetAccountId"] = "" }), 400, "targetAccountId"},
		{"源账号不存在", schedBody("missing", dstID, nil), 404, "source account not found"},
		{"目标账号不存在", schedBody(srcID, "missing", nil), 404, "target account not found"},
		{"源账号配置无效", schedBody(badCfg.ID, dstID, func(m map[string]any) { m["sourceBucket"] = "b" }), 400, "invalid source account configuration"},
		{"无桶可解析", schedBody(noBucket.ID, dstID, func(m map[string]any) { m["sourceBucket"] = "" }), 400, "bucket is required"},
		{"cron 非法", schedBody(srcID, dstID, func(m map[string]any) { m["cron"] = "61 * * * *" }), 400, "cron"},
		{"cron 永不触发", schedBody(srcID, dstID, func(m map[string]any) { m["cron"] = "0 0 30 2 *" }), 400, "never"},
		{"mode 非法", schedBody(srcID, dstID, func(m map[string]any) { m["mode"] = "weird" }), 400, "mode must be"},
		{"缺 cron", schedBody(srcID, dstID, func(m map[string]any) { m["cron"] = "" }), 400, "cron"},
		{"目标账号配置无效", schedBody(srcID, badCfg.ID, func(m map[string]any) { m["targetBucket"] = "b" }), 400, "invalid target account configuration"},
		{"目标桶无法解析", schedBody(srcID, noBucket.ID, func(m map[string]any) { m["targetBucket"] = "" }), 400, "bucket is required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := doJSON(t, h.Routes(), http.MethodPost, "/api/schedules", c.body)
			if rr.Code != c.want {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, c.want, rr.Body)
			}
			if c.msg != "" && !strings.Contains(rr.Body.String(), c.msg) {
				t.Errorf("body = %s, want substring %q", rr.Body, c.msg)
			}
		})
	}
}

// TestSchedulesValidationDoesNotEchoInput 固定 400 文案不回显用户提交的 cron
// 原文（KNOWN_ISSUES #80）：服务层详细原因只落日志。
func TestSchedulesValidationDoesNotEchoInput(t *testing.T) {
	h, _, srcID, dstID := schedAccounts(t)
	const marker = "NOT_A_CRON_TOKEN"
	body := schedBody(srcID, dstID, func(m map[string]any) { m["cron"] = marker + " * * * *" })
	rr := doJSON(t, h.Routes(), http.MethodPost, "/api/schedules", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body)
	}
	if strings.Contains(rr.Body.String(), marker) {
		t.Errorf("响应回显了用户 cron 原文: %s", rr.Body)
	}
	if !strings.Contains(rr.Body.String(), "cron is invalid") {
		t.Errorf("body = %s, want 固定文案 cron is invalid", rr.Body)
	}
}

// ---- 更新 ----

func TestSchedulesUpdate(t *testing.T) {
	h, _, srcID, dstID := schedAccounts(t)
	routes := h.Routes()

	created := createSched(t, routes, srcID, dstID, nil)

	t.Run("合法更新", func(t *testing.T) {
		body := schedBody(srcID, dstID, func(m map[string]any) {
			m["cron"] = "30 3 * * *"
			m["targetPrefix"] = "new-backup/"
			m["mode"] = "size_mtime"
			m["enabled"] = false
		})
		rr := doJSON(t, routes, http.MethodPut, "/api/schedules/"+created.ID, body)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d; body=%s", rr.Code, rr.Body)
		}
		var resp struct {
			Schedule service.Schedule `json:"schedule"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		got := resp.Schedule
		if got.Cron != "30 3 * * *" || got.TargetPrefix != "new-backup/" || got.Enabled {
			t.Errorf("update 未生效: %+v", got)
		}
		if got.Mode != service.CompareSizeTime {
			t.Errorf("mode = %q", got.Mode)
		}
		// ID 与 CreatedAt 由服务端保留。
		if got.ID != created.ID || !got.CreatedAt.Equal(created.CreatedAt) {
			t.Errorf("ID/CreatedAt 被改写: %+v vs %+v", got, created)
		}
		// cron 变更 → 排期重算（次日 03:30 而非原 02:00 逻辑）。
		if got.NextRunAt.Equal(created.NextRunAt) {
			t.Errorf("cron 变更后 nextRunAt 未重算: %v", got.NextRunAt)
		}
	})

	t.Run("未知 id 404", func(t *testing.T) {
		rr := doJSON(t, routes, http.MethodPut, "/api/schedules/nope", schedBody(srcID, dstID, nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rr.Code)
		}
	})

	t.Run("非法 cron 400", func(t *testing.T) {
		rr := doJSON(t, routes, http.MethodPut, "/api/schedules/"+created.ID,
			schedBody(srcID, dstID, func(m map[string]any) { m["cron"] = "bad" }))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("请求体非法 400", func(t *testing.T) {
		rr := doJSON(t, routes, http.MethodPut, "/api/schedules/"+created.ID, `{`)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400; body=%s", rr.Code, rr.Body)
		}
	})
}

// ---- 删除 ----

func TestSchedulesDelete(t *testing.T) {
	h, _, srcID, dstID := schedAccounts(t)
	routes := h.Routes()
	created := createSched(t, routes, srcID, dstID, nil)

	rr := doJSON(t, routes, http.MethodDelete, "/api/schedules/"+created.ID, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rr.Code, rr.Body)
	}
	if !strings.Contains(rr.Body.String(), created.ID) {
		t.Errorf("deleted 响应应回显 id: %s", rr.Body)
	}
	listRR := doJSON(t, routes, http.MethodGet, "/api/schedules", "")
	if strings.Contains(listRR.Body.String(), created.ID) {
		t.Errorf("删除后列表仍含该计划: %s", listRR.Body)
	}

	rr = doJSON(t, routes, http.MethodDelete, "/api/schedules/"+created.ID, "")
	if rr.Code != http.StatusNotFound {
		t.Errorf("重复删除 status = %d, want 404", rr.Code)
	}
}

// createSched 经 API 创建一个计划并返回之。
func createSched(t *testing.T, h http.Handler, srcID, dstID string, mutate func(m map[string]any)) service.Schedule {
	t.Helper()
	rr := doJSON(t, h, http.MethodPost, "/api/schedules", schedBody(srcID, dstID, mutate))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", rr.Code, rr.Body)
	}
	var resp struct {
		Schedule service.Schedule `json:"schedule"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Schedule
}

// ---- 手动触发 ----

// blockingListS3 造一个「首次 list 阻塞至 release」的假 S3：
// 让首个运行中的任务停在同步中途，从而可确定性地测 409 不叠加。
// 返回 (服务 URL, list 进入信号, 释放函数)。
func blockingListS3(t *testing.T) (url string, entered <-chan struct{}, release func()) {
	t.Helper()
	enteredCh := make(chan struct{}, 1)
	releaseCh := make(chan struct{})
	var once sync.Once
	rel := func() { once.Do(func() { close(releaseCh) }) }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			select {
			case enteredCh <- struct{}{}:
			default:
			}
			<-releaseCh // release 关闭后此处立即返回，后续 list 不再阻塞
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>false</IsTruncated></ListBucketResult>`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(rel)
	return srv.URL, enteredCh, rel
}

func TestSchedulesRunNowEndToEnd(t *testing.T) {
	h, _, srcID, dstID := schedAccounts(t)
	routes := h.Routes()
	s := createSched(t, routes, srcID, dstID, nil)

	rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("run status = %d; body=%s", rr.Code, rr.Body)
	}
	var runResp struct {
		JobID      string `json:"jobId"`
		ScheduleID string `json:"scheduleId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body)
	}
	if runResp.JobID == "" || runResp.ScheduleID != s.ID {
		t.Fatalf("run resp = %+v", runResp)
	}

	// 计划回显 lastJobId / lastRunAt（运行态对 UI 可见）。
	listRR := doJSON(t, routes, http.MethodGet, "/api/schedules", "")
	if !strings.Contains(listRR.Body.String(), runResp.JobID) {
		t.Errorf("列表应回显 lastJobId: %s", listRR.Body)
	}

	// 异步任务必须真实跑完（假 S3 list 空源 → done）。
	waitJobDone(t, h, runResp.JobID)
}

func TestSchedulesRunNowNotRunning409(t *testing.T) {
	fakeURL, entered, release := blockingListS3(t)
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "src", Endpoint: fakeURL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "src-bucket", PathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	h := New(st, gapLogger(), t.TempDir(), nil, "", "test", false, false)
	t.Cleanup(h.Shutdown)
	routes := h.Routes()
	s := createSched(t, routes, acc.ID, acc.ID, func(m map[string]any) {
		m["targetBucket"] = "dst-bucket"
	})

	// 第一次触发：202，任务阻塞在 list。
	rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("first run status = %d; body=%s", rr.Code, rr.Body)
	}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("任务未进入阻塞 list")
	}

	// 第二次触发：上一轮未结束 → 409 不叠加。
	rr = doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
	if rr.Code != http.StatusConflict {
		t.Fatalf("second run status = %d, want 409; body=%s", rr.Code, rr.Body)
	}

	release()
}

func TestSchedulesRunNotFound(t *testing.T) {
	h, _, _, _ := schedAccounts(t)
	rr := doJSON(t, h.Routes(), http.MethodPost, "/api/schedules/nope/run", "")
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestSchedulesRunAccountGone(t *testing.T) {
	h, st, srcID, dstID := schedAccounts(t)
	routes := h.Routes()
	s := createSched(t, routes, srcID, dstID, nil)

	// 创建后账号被删 → 运行时解析失败 → 404（计划仍存在，账号没了）。
	if err := st.Delete(srcID); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body=%s", rr.Code, rr.Body)
	}
}

func TestSchedulesRunTooManyJobs503(t *testing.T) {
	h, _, srcID, dstID := schedAccounts(t)
	routes := h.Routes()
	s := createSched(t, routes, srcID, dstID, nil)

	// 填满在册任务（上限 defaultMaxJobs=256）。
	_, cancel := context.WithCancel(context.Background())
	filled := 0
	for {
		if _, err := h.migrateJobs.TryCreate(1, cancel); err != nil {
			break
		}
		filled++
		if filled > 1024 {
			t.Fatal("在册任务无上限")
		}
	}
	rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503; body=%s", rr.Code, rr.Body)
	}
}

// ---- 触发失败路径（配置损坏 / store 故障 / 列举失败 / 截断） ----

// TestSchedulesRunConfigBroken400 账号配置在创建计划后损坏（绕过 API 校验直接登记计划）：
// 手动 run 必须 400 点名配置问题，且源/目标两侧的 client 构建失败分支都被执行。
func TestSchedulesRunConfigBroken400(t *testing.T) {
	h, st, srcID, dstID := schedAccounts(t)
	routes := h.Routes()
	// badCfg 缺 SecretKey（store 层允许创建，client 构建时失败）。
	badCfg, err := st.Create(&model.Account{
		Name: "bad-run", Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		AccessKey: "ak", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	mk := func(src, dst string) service.Schedule {
		s, err := h.sched.Create(service.Schedule{
			SourceAccountID: src, SourceBucket: "src-bucket",
			TargetAccountID: dst, TargetBucket: "dst-bucket",
			Mode: service.CompareETag, Cron: "0 2 * * *", Enabled: true,
		})
		if err != nil {
			t.Fatalf("sched.Create: %v", err)
		}
		return s
	}

	t.Run("源配置损坏", func(t *testing.T) {
		s := mk(badCfg.ID, dstID)
		rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400; body=%s", rr.Code, rr.Body)
		}
	})
	t.Run("目标配置损坏", func(t *testing.T) {
		s := mk(srcID, badCfg.ID)
		rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400; body=%s", rr.Code, rr.Body)
		}
	})
}

// TestSchedulesRunStoreFault500 run 时 store 读取故障（非 ErrNotFound）→ 500 通用错误；
// 计划本身先经 API 创建成功（此时 store 正常），再注入故障。
func TestSchedulesRunStoreFault500(t *testing.T) {
	good := &model.Account{
		Name: "good", Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	}
	st := &accStubStore{getAcc: good}
	h := accNewHandler(t, st, nil, "")
	routes := h.Routes()

	rr := doJSON(t, routes, http.MethodPost, "/api/schedules",
		schedBody("src", "dst", nil))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create = %d; body=%s", rr.Code, rr.Body)
	}
	var created struct {
		Schedule service.Schedule `json:"schedule"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	st.getErrs = map[string]error{"src": io.EOF}
	rr = doJSON(t, routes, http.MethodPost, "/api/schedules/"+created.Schedule.ID+"/run", "")
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("run store 故障 = %d, want 500; body=%s", rr.Code, rr.Body)
	}
}

// TestSchedulesCreateStoreFault500 创建时 store 读取故障 → 500（仅 ErrNotFound 才是 404）。
func TestSchedulesCreateStoreFault500(t *testing.T) {
	good := &model.Account{
		Name: "good", Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	}
	st := &accStubStore{getAcc: good, getErrs: map[string]error{"src": io.EOF}}
	h := accNewHandler(t, st, nil, "")
	rr := doJSON(t, h.Routes(), http.MethodPost, "/api/schedules", schedBody("src", "dst", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("create store 故障 = %d, want 500; body=%s", rr.Code, rr.Body)
	}
}

// ---- 列举失败与截断必须透出（防定时备份静默漏拷） ----

// schedFailListS3 造一个「list 请求返回 500」的假 S3。
func schedFailListS3(t *testing.T) string {
	t.Helper()
	srv := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return srv.URL
}

// TestSchedulesRunListFailureRecordsError 列举失败（源端 5xx）→ 任务 result.lastError 非空，
// 不得静默报成「扫描 0 个，无事可做」（同 sync 端点 review §B5 口径）。
func TestSchedulesRunListFailureRecordsError(t *testing.T) {
	fakeURL := schedFailListS3(t)
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatal(err)
	}
	acc, err := st.Create(&model.Account{
		Name: "src", Endpoint: fakeURL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "src-bucket", PathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := New(st, gapLogger(), t.TempDir(), nil, "", "test", false, false)
	t.Cleanup(h.Shutdown)
	routes := h.Routes()
	s := createSched(t, routes, acc.ID, acc.ID, func(m map[string]any) {
		m["targetBucket"] = "dst-bucket"
	})

	rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("run = %d; body=%s", rr.Code, rr.Body)
	}
	var runResp struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &runResp); err != nil {
		t.Fatal(err)
	}
	job, ok := h.migrateJobs.Get(runResp.JobID)
	if !ok {
		t.Fatal("任务未创建")
	}
	_, result, done := waitJobDoneState(t, job)
	if !done || result.FirstError == "" {
		t.Errorf("列举失败必须记入 lastError: done=%v result=%+v", done, result)
	}
}

// schedTruncListS3 造一个「IsTruncated 恒 true 且 token 不前进」的假 S3（触发硬上限截断）。
func schedTruncListS3(t *testing.T) string {
	t.Helper()
	srv := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>true</IsTruncated><NextContinuationToken></NextContinuationToken></ListBucketResult>`)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return srv.URL
}

// TestSchedulesRunTruncatedRecordsError 列举被硬上限截断（有对象未参与本次同步）→
// 任务 result.lastError 非空，不得把「只拷了一部分」报成全量完成。
func TestSchedulesRunTruncatedRecordsError(t *testing.T) {
	fakeURL := schedTruncListS3(t)
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatal(err)
	}
	acc, err := st.Create(&model.Account{
		Name: "src", Endpoint: fakeURL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "src-bucket", PathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := New(st, gapLogger(), t.TempDir(), nil, "", "test", false, false)
	t.Cleanup(h.Shutdown)
	routes := h.Routes()
	s := createSched(t, routes, acc.ID, acc.ID, func(m map[string]any) {
		m["targetBucket"] = "dst-bucket"
	})

	rr := doJSON(t, routes, http.MethodPost, "/api/schedules/"+s.ID+"/run", "")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("run = %d; body=%s", rr.Code, rr.Body)
	}
	var runResp struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &runResp); err != nil {
		t.Fatal(err)
	}
	job, ok := h.migrateJobs.Get(runResp.JobID)
	if !ok {
		t.Fatal("任务未创建")
	}
	_, result, done := waitJobDoneState(t, job)
	if !done || !strings.Contains(result.FirstError, "truncated") {
		t.Errorf("截断必须记入 lastError: done=%v lastError=%q", done, result.FirstError)
	}
}

// waitJobDoneState 轮询直到任务终态并返回快照（waitJobDone 只判 done，这里要 result）。
func waitJobDoneState(t *testing.T, job *service.Job) (service.JobProgress, service.JobResult, bool) {
	t.Helper()
	// 30s：列举失败路径要等 SDK 把 500 重试退避跑完（默认 3 次），-race 并行下更慢。
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		p, res, done := job.Snapshot()
		if done {
			return p, res, true
		}
		time.Sleep(5 * time.Millisecond)
	}
	p, res, _ := job.Snapshot()
	return p, res, false
}

// ---- 调度器到点触发（handler 集成） ----

func TestSchedulerTickFiresScheduleThroughHandler(t *testing.T) {
	h, _, srcID, dstID := schedAccounts(t)
	routes := h.Routes()
	s := createSched(t, routes, srcID, dstID, func(m map[string]any) {
		m["cron"] = "* * * * *" // 下一分钟边界即到期
	})

	// 未到期不触发。
	h.sched.Tick(s.NextRunAt.Add(-time.Second))
	if got := doJSON(t, routes, http.MethodGet, "/api/schedules", ""); strings.Contains(got.Body.String(), `"lastJobId"`) {
		t.Fatalf("未到期不得触发: %s", got.Body)
	}

	// 到点触发 → 任务创建并回显 lastJobId。
	h.sched.Tick(s.NextRunAt.Add(time.Second))
	listRR := doJSON(t, routes, http.MethodGet, "/api/schedules", "")
	var listResp struct {
		Schedules []service.Schedule `json:"schedules"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, listRR.Body)
	}
	if len(listResp.Schedules) != 1 {
		t.Fatalf("list = %+v", listResp.Schedules)
	}
	got := listResp.Schedules[0]
	if got.LastJobID == "" || got.LastRunAt.IsZero() || got.LastError != "" {
		t.Errorf("触发后运行态 = %+v", got)
	}
	// 排期必须已前移到未来（不会每 tick 重复触发）。
	if !got.NextRunAt.After(s.NextRunAt) {
		t.Errorf("nextRunAt 未前移: %v → %v", s.NextRunAt, got.NextRunAt)
	}
	waitJobDone(t, h, got.LastJobID)
}

func TestSchedulerTickTriggerErrorRecordsLastError(t *testing.T) {
	h, st, srcID, dstID := schedAccounts(t)
	routes := h.Routes()
	s := createSched(t, routes, srcID, dstID, func(m map[string]any) { m["cron"] = "* * * * *" })

	// 账号在排期到点前被删 → 触发失败 → LastError 记录、LastJobID 为空。
	if err := st.Delete(dstID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	h.sched.Tick(s.NextRunAt.Add(time.Second))

	listRR := doJSON(t, routes, http.MethodGet, "/api/schedules", "")
	var listResp struct {
		Schedules []service.Schedule `json:"schedules"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := listResp.Schedules[0]
	if got.LastError == "" || got.LastJobID != "" {
		t.Errorf("触发失败运行态 = %+v, want LastError 非空、LastJobID 空", got)
	}
	if !got.NextRunAt.After(s.NextRunAt) {
		t.Errorf("失败也须前移排期（防每 tick 重试轰炸）: %v", got.NextRunAt)
	}
}

// ---- 落盘恢复 ----

func TestSchedulePersisterRestoresAcrossHandlers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedules.json")
	h1, _, srcID, dstID := schedAccounts(t)
	h1.SetSchedulePersister(service.NewFileSchedulePersister(path))
	created := createSched(t, h1.Routes(), srcID, dstID, nil)

	// 新进程（新 Handler）从同一文件恢复。
	h2, _, _, _ := schedAccounts(t)
	h2.SetSchedulePersister(service.NewFileSchedulePersister(path))
	t.Cleanup(h2.Shutdown)

	rr := doJSON(t, h2.Routes(), http.MethodGet, "/api/schedules", "")
	if !strings.Contains(rr.Body.String(), created.ID) {
		t.Fatalf("恢复失败: %s", rr.Body)
	}
	if !strings.Contains(rr.Body.String(), created.Cron) {
		t.Errorf("恢复丢失 cron: %s", rr.Body)
	}
}

// ---- 作用域 ----

func TestSchedulesScopeReadonlyBlocksWrites(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Readonly: true},
	})
	for _, c := range []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/schedules", `{}`},
		{http.MethodPut, "/api/schedules/x", `{}`},
		{http.MethodDelete, "/api/schedules/x", ""},
		{http.MethodPost, "/api/schedules/x/run", ""},
	} {
		if rr := e.request(scopeTokReadonly, c.method, c.path, c.body); rr.Code != http.StatusForbidden {
			t.Errorf("readonly %s %s = %d, want 403", c.method, c.path, rr.Code)
		}
	}
	// GET 列表对 readonly 放行（只读语义）。
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/schedules", ""); rr.Code != http.StatusOK {
		t.Errorf("readonly GET = %d, want 200", rr.Code)
	}
}

// TestSchedulesScopePrefixes 前缀作用域对计划的桶引用判定：
//   - 创建/更新：body 的 sourceBucket/targetBucket 走既有 requestScopeRefs 分支；
//   - 运行：请求无 body，桶引用必须从已存计划注入（否则限前缀 token 可触发越界同步）。
func TestSchedulesScopePrefixes(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"in-bucket/"}},
	})
	// 全权 token（未登记 = 全权）先建一个越界计划（out-bucket）。
	acc2, err := e.h.store.Create(&model.Account{
		Name: "acc2", Endpoint: e.acc.Endpoint, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "out-bucket", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create acc2: %v", err)
	}
	rr := e.request(scopeTokUnlisted, http.MethodPost, "/api/schedules",
		schedBody(e.acc.ID, acc2.ID, func(m map[string]any) {
			m["sourceBucket"] = "out-bucket"
			m["sourcePrefix"] = ""
			m["targetBucket"] = "out-bucket"
			m["targetPrefix"] = ""
		}))
	if rr.Code != http.StatusCreated {
		t.Fatalf("全权创建越界计划 = %d; body=%s", rr.Code, rr.Body)
	}
	var createdResp struct {
		Schedule service.Schedule `json:"schedule"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &createdResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	created := createdResp.Schedule

	// 越界桶创建 → 403（body refs）。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/schedules",
		schedBody(e.acc.ID, acc2.ID, func(m map[string]any) { m["sourceBucket"] = "out-bucket" })); rr.Code != http.StatusForbidden {
		t.Errorf("越界桶创建 = %d, want 403", rr.Code)
	}
	// 界内桶创建 → 放行。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/schedules",
		schedBody(e.acc.ID, e.acc.ID, func(m map[string]any) {
			m["targetAccountId"] = e.acc.ID
			m["sourceBucket"] = "in-bucket"
			m["targetBucket"] = "in-bucket"
		})); rr.Code != http.StatusCreated {
		t.Errorf("界内桶创建 = %d, want 201; body=%s", rr.Code, rr.Body)
	}
	// 运行越界计划 → 403（计划桶引用注入 scope 判定）。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/schedules/"+created.ID+"/run", ""); rr.Code != http.StatusForbidden {
		t.Errorf("运行越界计划 = %d, want 403; body=%s", rr.Code, rr.Body)
	}
	// 删除越界计划 → 403（同注入逻辑）。
	if rr := e.request(scopeTokReadonly, http.MethodDelete, "/api/schedules/"+created.ID, ""); rr.Code != http.StatusForbidden {
		t.Errorf("删除越界计划 = %d, want 403", rr.Code)
	}
	// GET 列表：无桶引用 → 放行（与桶列表同类残留，见 threat-model）。
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/schedules", ""); rr.Code != http.StatusOK {
		t.Errorf("GET 列表 = %d, want 200（残留已文档化）", rr.Code)
	}
}

// waitJobDone 轮询直到任务进入终态。
func waitJobDone(t *testing.T, h *Handler, jobID string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if job, ok := h.migrateJobs.Get(jobID); ok {
			if _, _, done := job.Snapshot(); done {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("任务 %s 未在时限内完成", jobID)
}
