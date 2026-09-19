package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
)

// waitForCondition polls cond until it returns true or timeout elapses.
func waitForCondition(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

// closeCountingReader tracks how many times Close was called.
type closeCountingReader struct {
	closed atomic.Int32
	r      io.Reader
}

func (c *closeCountingReader) Read(p []byte) (int, error) { return c.r.Read(p) }
func (c *closeCountingReader) Close() error {
	c.closed.Add(1)
	return nil
}
func (c *closeCountingReader) count() int32 { return c.closed.Load() }

// ---- SyncKeys: workers default + first progress callback (lines 43-45, 52-54) ----

func TestSync_WorkersDefaultAndProgress(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{
		"src-bucket": {"x.txt"},
		"dst-bucket": {},
	}
	s3FakeSize = map[string]int64{"src-bucket/x.txt": 5}
	s3FakeEtag = map[string]uint64{"src-bucket/x.txt": 0x1234}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	var progress []Progress
	// workers = 0 → should default to 4 (lines 43-45). onProgress non-nil → first callback (52-54).
	out, err := SyncKeys(context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 0, func(p Progress) {
		progress = append(progress, p)
	})
	if err != nil {
		t.Fatalf("SyncKeys: %v", err)
	}
	if out.Copied != 1 || out.Failed != 0 {
		t.Fatalf("result = %+v", out)
	}
	if len(progress) == 0 {
		t.Fatal("no progress emitted")
	}
	first := progress[0]
	if first.Total != 1 || first.Done != 0 || first.Failed != 0 {
		t.Fatalf("first progress = %+v, want Total=1 Done=0 Failed=0", first)
	}
}

// ---- SyncKeys: empty toCopy progress branch (lines 72-77) ----

func TestSync_EmptyToCopyProgress(t *testing.T) {
	s3FakeMu.Lock()
	// a.txt exists in both with equal ETag → skipped, nothing to copy.
	s3FakeStore = map[string][]string{"src-bucket": {"a.txt"}, "dst-bucket": {"a.txt"}}
	s3FakeSize = map[string]int64{"src-bucket/a.txt": 10, "dst-bucket/a.txt": 10}
	s3FakeEtag = map[string]uint64{"src-bucket/a.txt": 0xaa, "dst-bucket/a.txt": 0xaa}
	s3FakeMu.Unlock()

	src, dst, closer := makeFakePair(t)
	defer closer()

	var progress []Progress
	out, err := SyncKeys(context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2, func(p Progress) {
		progress = append(progress, p)
	})
	if err != nil {
		t.Fatalf("SyncKeys: %v", err)
	}
	if out.Skipped != 1 || out.Copied != 0 {
		t.Fatalf("result = %+v", out)
	}
	// second callback is the empty-toCopy terminal progress (line 73-75).
	if len(progress) != 2 {
		t.Fatalf("progress length = %d, want 2 (%v)", len(progress), progress)
	}
	last := progress[len(progress)-1]
	if last.Total != 1 || last.Done != 1 {
		t.Fatalf("terminal progress = %+v, want Total=1 Done=1", last)
	}
}

// ---- isEqual: all mode branches (lines 146-166) ----

func TestIsEqualAllModes(t *testing.T) {
	t1 := time.Unix(1_700_000_000, 0).UTC()
	t2 := time.Unix(1_700_000_001, 0).UTC()

	etag := s3wrap.ObjectItem{Key: "k", Size: 10, ETag: `"abc"`, LastModified: t1}
	meta := func(etag string, size int64, mt time.Time) *s3wrap.ObjectMeta {
		return &s3wrap.ObjectMeta{ETag: etag, Size: size, LastModified: mt}
	}

	cases := []struct {
		name string
		mode CompareMode
		src  s3wrap.ObjectItem
		dst  *s3wrap.ObjectMeta
		want bool
	}{
		{"dst nil", CompareETag, etag, nil, false},
		{"always", CompareAlways, etag, meta(`"abc"`, 10, t1), false},
		{"etag equal", CompareETag, etag, meta(`"abc"`, 10, t1), true},
		{"etag differ", CompareETag, etag, meta(`"zzz"`, 10, t1), false},
		{"etag empty src falls to size", CompareETag, s3wrap.ObjectItem{Size: 10, ETag: ""}, meta("", 10, t1), true},
		{"etag empty src size differ", CompareETag, s3wrap.ObjectItem{Size: 10, ETag: ""}, meta("", 11, t1), false},
		{"etag empty dst size equal", CompareETag, etag, meta("", 10, t1), true},
		{"etag both empty size equal", CompareETag, s3wrap.ObjectItem{Size: 7, ETag: ""}, meta("", 7, t1), true},
		{"size_mtime size differ", CompareSizeTime, etag, meta(`"abc"`, 11, t1), false},
		{"size_mtime equal", CompareSizeTime, etag, meta(`"abc"`, 10, t1), true},
		{"size_mtime mtime differ", CompareSizeTime, etag, meta(`"abc"`, 10, t2), false},
		{"unknown mode", CompareMode("bogus"), etag, meta(`"abc"`, 10, t1), false},
	}
	for _, c := range cases {
		if got := isEqual(c.mode, c.src, c.dst); got != c.want {
			t.Errorf("%s: isEqual = %v, want %v", c.name, got, c.want)
		}
	}
}

// ---- stripPrefix: non-matching / shorter-key prefixes (line 173-176) ----

func TestStripPrefixNoMatch(t *testing.T) {
	cases := []struct{ in, prefix, want string }{
		{"x.txt", "a/", "x.txt"},   // prefix present but key does not start with it
		{"a", "ab", "a"},           // prefix longer than key (len check fails)
		{"ba.txt", "a/", "ba.txt"}, // starts with different chars, prefix present
		{"a/b.txt", "a/", "b.txt"}, // sanity: matching case still works
		{"foo", "foo", ""},         // prefix == key entirely
		// 段边界：前缀 "foo" 未结束在 "/" 上，且 "foobar" 的下一段不是 "/"，
		// 因此不算命中，原样返回（旧实现会削成 "bar"，使目标侧出现永不复位的错位 key）。
		{"foobar", "foo", "foobar"},
		// 段边界命中：前缀无尾斜杠但紧随其后是 "/"，剥掉前缀与分隔符。
		{"foo/bar", "foo", "bar"},
		{"foo/bar", "foo/", "bar"},
	}
	for _, c := range cases {
		if got := stripPrefix(c.in, c.prefix); got != c.want {
			t.Errorf("stripPrefix(%q, %q) = %q, want %q", c.in, c.prefix, got, c.want)
		}
	}
}

// ---- listFake: configurable ListObjectsV2 fake for listAll/indexDst paths ----

const listFakeBucket = "bkt"

type listFake struct {
	srv      *httptest.Server
	mu       sync.Mutex
	keys     map[string][]string // bucket -> keys
	pageSize int                 // objects per response page
	failList bool                // return HTTP 400 for list-type=2
}

func newListFake(t *testing.T) *listFake {
	t.Helper()
	f := &listFake{keys: map[string][]string{}, pageSize: 1}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *listFake) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("list-type") != "2" {
		http.Error(w, "unexpected request: "+r.Method+" "+r.URL.String(), http.StatusBadRequest)
		return
	}
	if f.failList {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>AccessDenied</Code></Error>`))
		return
	}
	bucket := firstSeg(r.URL.Path)
	prefix := r.URL.Query().Get("prefix")
	rawToken := r.URL.Query().Get("continuation-token")

	f.mu.Lock()
	var all []string
	for _, k := range f.keys[bucket] {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			all = append(all, k)
		}
	}
	f.mu.Unlock()

	page := 0
	if rawToken != "" {
		page, _ = strconv.Atoi(rawToken)
	}
	ps := f.pageSize
	if ps < 1 {
		ps = 1
	}
	start := page * ps
	if start > len(all) {
		start = len(all)
	}
	end := start + ps
	if end > len(all) {
		end = len(all)
	}
	pageKeys := all[start:end]
	isTruncated := end < len(all)
	next := ""
	if isTruncated {
		next = strconv.Itoa(page + 1)
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	for _, k := range pageKeys {
		tag := simpleHash(k)
		sb.WriteString("<Contents>")
		sb.WriteString("<Key>" + k + "</Key>")
		sb.WriteString("<Size>" + strconv.FormatInt(int64(len(k)), 10) + "</Size>")
		sb.WriteString(`<ETag>"` + strconv.FormatUint(tag, 16) + `"</ETag>`)
		sb.WriteString("<LastModified>" + time.Unix(1_700_000_000, 0).UTC().Format(time.RFC3339) + "</LastModified>")
		sb.WriteString("<StorageClass>STANDARD</StorageClass>")
		sb.WriteString("</Contents>")
	}
	sb.WriteString("<IsTruncated>" + strconv.FormatBool(isTruncated) + "</IsTruncated>")
	if isTruncated {
		sb.WriteString("<NextContinuationToken>" + next + "</NextContinuationToken>")
	}
	sb.WriteString("</ListBucketResult>")

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(sb.String()))
}

func (f *listFake) client(t *testing.T) *s3wrap.Client {
	t.Helper()
	return newTestClient(t, f.srv.URL)
}

// ---- listAll: pagination/truncation (line 115), error (103-104), hard cap (108-110) ----

func TestListAllPaginates(t *testing.T) {
	f := newListFake(t)
	f.keys[listFakeBucket] = []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	f.pageSize = 2

	got, err := listAll(context.Background(), f.client(t), listFakeBucket, "")
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	want := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	if len(got) != len(want) {
		t.Fatalf("listAll returned %d keys, want %d: %v", len(got), len(want), got)
	}
	for i, k := range want {
		if got[i].Key != k {
			t.Fatalf("listAll[%d] = %q, want %q", i, got[i].Key, k)
		}
	}
}

func TestListAllTruncationStopsWithoutToken(t *testing.T) {
	// isTruncated=true but NextContinuationToken empty → loop must stop.
	// Simulated via a server that returns a truncated page without a next token.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<Contents><Key>a</Key><Size>1</Size><ETag>"1"</ETag><LastModified>2023-01-01T00:00:00Z</LastModified><StorageClass>STANDARD</StorageClass></Contents>` +
			`<IsTruncated>true</IsTruncated></ListBucketResult>`))
	}))
	t.Cleanup(srv.Close)

	keys, err := listAll(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("len = %d, want 1", len(keys))
	}
}

func TestListAllError(t *testing.T) {
	f := newListFake(t)
	f.failList = true
	got, err := listAll(context.Background(), f.client(t), listFakeBucket, "")
	if err == nil {
		t.Fatal("listAll must surface the list error instead of reporting an empty result")
	}
	if len(got) != 0 {
		t.Fatalf("listAll on error = %d keys, want 0", len(got))
	}
}

func TestListAllHardCap(t *testing.T) {
	const total = 100_000
	f := newListFake(t)
	keys := make([]string, total+1)
	for i := range keys {
		keys[i] = "obj" + strconv.Itoa(i)
	}
	f.keys[listFakeBucket] = keys
	f.pageSize = total + 1 // single page returns everything

	got, err := listAll(context.Background(), f.client(t), listFakeBucket, "")
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if len(got) != total {
		t.Fatalf("listAll hard cap returned %d, want %d", len(got), total)
	}
}

// ---- indexDst: pagination (line 140), error (127-128) ----

func TestIndexDstPaginates(t *testing.T) {
	f := newListFake(t)
	f.keys[listFakeBucket] = []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	f.pageSize = 3

	idx, err := indexDst(context.Background(), f.client(t), listFakeBucket, "")
	if err != nil {
		t.Fatalf("indexDst: %v", err)
	}
	if len(idx) != 5 {
		t.Fatalf("indexDst size = %d, want 5", len(idx))
	}
	for _, k := range []string{"a.txt", "c.txt", "e.txt"} {
		if _, ok := idx[k]; !ok {
			t.Fatalf("indexDst missing %q", k)
		}
	}
	if idx["a.txt"].Size != int64(len("a.txt")) {
		t.Fatalf("indexDst['a.txt'].Size = %d", idx["a.txt"].Size)
	}
}

func TestIndexDstError(t *testing.T) {
	f := newListFake(t)
	f.failList = true
	idx, err := indexDst(context.Background(), f.client(t), listFakeBucket, "")
	if err == nil {
		t.Fatal("indexDst must surface the list error instead of reporting an empty index")
	}
	if len(idx) != 0 {
		t.Fatalf("indexDst on error = %d entries, want 0", len(idx))
	}
}

// ---- zip.go: ctxCancelReader read/close/AfterFunc (lines 22-24, 28, 34) ----

func TestCtxCancelReader(t *testing.T) {
	t.Run("read passthrough", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cr := newCancelReader(ctx, io.NopCloser(strings.NewReader("ok")))
		buf := make([]byte, 2)
		n, err := cr.Read(buf)
		if err != nil || string(buf[:n]) != "ok" {
			t.Fatalf("read = %q err=%v", buf[:n], err)
		}
	})
	t.Run("close delegates", func(t *testing.T) {
		r := &closeCountingReader{r: strings.NewReader("x")}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cr := newCancelReader(ctx, r)
		if err := cr.Close(); err != nil {
			t.Fatalf("Close = %v", err)
		}
		if r.count() != 1 {
			t.Fatalf("underlying closed %d times, want 1", r.count())
		}
	})
	t.Run("afterfunc closes on cancel", func(t *testing.T) {
		r := &closeCountingReader{r: strings.NewReader("x")}
		ctx, cancel := context.WithCancel(context.Background())
		// 只关心注册的 AfterFunc 副作用，读端本身无需持有。
		newCancelReader(ctx, r)
		cancel()
		if !waitForCondition(2*time.Second, func() bool { return r.count() >= 1 }) {
			t.Fatal("AfterFunc did not close the underlying reader")
		}
	})
	t.Run("read after cancel errors", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cr := newCancelReader(ctx, io.NopCloser(strings.NewReader("x")))
		cancel()
		if _, err := cr.Read(make([]byte, 1)); err == nil {
			t.Fatal("cancelled ctx Read should return error")
		}
	})
}

// stuckContext mimics a context whose Done() never fires but Err() is non-nil.
// It lets the zip worker's pre-fetch ctx guard fire deterministically while the
// producer goroutine keeps sending jobs (the real ctx.Done() path would stop it).
type stuckContext struct{}

func (stuckContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (stuckContext) Done() <-chan struct{}       { return nil }
func (stuckContext) Err() error                  { return context.Canceled }
func (stuckContext) Value(any) any               { return nil }

func TestWriteObjectsZipWorkerPreGetGuard(t *testing.T) {
	var buf bytes.Buffer
	var getCalled atomic.Bool
	fails, err := WriteObjectsZip(stuckContext{}, func(context.Context, string) (io.ReadCloser, string, error) {
		getCalled.Store(true)
		return nil, "", errors.New("should not be called")
	}, []string{"a", "b", "c"}, &buf)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(fails) != 3 {
		t.Fatalf("failKeys = %v, want 3", fails)
	}
	if getCalled.Load() {
		t.Fatal("get invoked despite cancelled ctx; worker pre-fetch guard should bail")
	}
}

// TestWriteObjectsZipCreateHeaderError covers createHeader failure (lines 127-130):
// a long entry name overflows the zip writer's internal buffer so the local file
// header write flushes into the failing writer, making CreateHeader itself fail.
func TestWriteObjectsZipCreateHeaderError(t *testing.T) {
	longKey := "x" + strings.Repeat("a", 10_000)
	get := func(context.Context, string) (io.ReadCloser, string, error) {
		return io.NopCloser(strings.NewReader("body")), "text/plain", nil
	}
	fails, err := WriteObjectsZip(context.Background(), get, []string{longKey}, failWriter{})
	if err == nil {
		t.Fatal("CreateHeader failure should surface an error")
	}
	if len(fails) != 1 || fails[0] != longKey {
		t.Fatalf("failKeys = %v, want [%q]", fails, longKey)
	}
}

// ---- SyncKeys: ctx canceled before/during src listing ----
// 语义：客户端放弃（ctx 取消）不算错误——返回已完成的部分结果即可（review §B5 只要求
// 「真实的列举错误」不得被吞掉，取消是另一回事）。

func TestSync_CtxCanceledAfterSrcList(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{"src-bucket": {"x.txt"}, "dst-bucket": {}}
	s3FakeSize = map[string]int64{"src-bucket/x.txt": 5}
	s3FakeEtag = map[string]uint64{"src-bucket/x.txt": 0x1234}
	s3FakeMu.Unlock()
	src, dst, closer := makeFakePair(t)
	defer closer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 取消后 listAll 立即失败

	out, err := SyncKeys(ctx, src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 4, nil)
	if err != nil {
		t.Fatalf("ctx 取消不应作为错误上报，得到 %v", err)
	}
	if out.Scanned != 0 || out.Copied != 0 {
		t.Fatalf("canceled ctx result = %+v, want Scanned=0 Copied=0", out)
	}
}

// ---- SyncKeys: ctx canceled between src listing and dst index (sync.go lines 62-64) ----
// 目标端列表服务器在第 2 个 list 请求（indexDst）时取消 ctx，模拟「列举完源后中止」。

func TestSync_CtxCanceledDuringIndexDst(t *testing.T) {
	var mu sync.Mutex
	reqs := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		n := reqs
		mu.Unlock()
		if n >= 2 {
			cancel() // dst 列举被中止
		}
		bucket := firstSeg(r.URL.Path)
		var sb strings.Builder
		sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
		for _, k := range s3FakeStore[bucket] {
			fmt.Fprintf(&sb, `<Contents><Key>%s</Key><Size>1</Size><ETag>"x"</ETag><LastModified>2024-01-01T00:00:00Z</LastModified><StorageClass>STANDARD</StorageClass></Contents>`, k)
		}
		sb.WriteString(`<IsTruncated>false</IsTruncated></ListBucketResult>`)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(sb.String()))
	}))
	t.Cleanup(srv.Close)

	acc := &model.Account{Endpoint: srv.URL, AccessKey: "AK", SecretKey: "SK", Region: "us-east-1", Bucket: "src-bucket", PathStyle: true}
	c1, err := s3wrap.New(acc)
	if err != nil {
		t.Fatalf("src client: %v", err)
	}
	acc2 := *acc
	acc2.Bucket = "dst-bucket"
	c2, err := s3wrap.New(&acc2)
	if err != nil {
		t.Fatalf("dst client: %v", err)
	}

	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{"src-bucket": {"x.txt"}, "dst-bucket": {}}
	s3FakeMu.Unlock()

	out, err := SyncKeys(ctx, c1, c2, "src-bucket", "", "dst-bucket", "", CompareETag, 4, nil)
	if err != nil {
		t.Fatalf("ctx 取消不应作为错误上报，得到 %v", err)
	}
	if out.Scanned != 1 || out.Copied != 0 {
		t.Fatalf("canceled-during-index result = %+v, want Scanned=1 Copied=0", out)
	}
}

// ---- indexDst: hard cap（页数/总量双层上限，见 review §B6）----

func TestIndexDstHardCap(t *testing.T) {
	const total = 100_000
	f := newListFake(t)
	keys := make([]string, total+1)
	for i := range keys {
		keys[i] = "obj" + strconv.Itoa(i)
	}
	f.keys[listFakeBucket] = keys
	f.pageSize = total + 1 // 单页超过总量上限 → 命中内层 break

	idx, err := indexDst(context.Background(), f.client(t), listFakeBucket, "")
	if err != nil {
		t.Fatalf("indexDst: %v", err)
	}
	if len(idx) != total {
		t.Fatalf("indexDst hard cap = %d, want %d", len(idx), total)
	}
}
