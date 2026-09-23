package service

// zip_leak_test.go —— docs/archive/review-2026-09-19.md §B4：写 ZIP 时首个拷贝错误不能 break，
// 否则无缓冲 results 通道无人消费 → worker 永久阻塞、已取回的 body 不关闭。

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// errAfter 读取 prefix 后返回错误，用于在 io.Copy 阶段制造拷贝失败（而不是 get 阶段失败）。
type errAfter struct {
	r      io.Reader
	remain int
}

func (e *errAfter) Read(p []byte) (int, error) {
	if e.remain <= 0 {
		return 0, errors.New("read failed")
	}
	if len(p) > e.remain {
		p = p[:e.remain]
	}
	n, err := e.r.Read(p)
	e.remain -= n
	if err == nil && e.remain <= 0 {
		err = errors.New("read failed")
	}
	return n, err
}

func (e *errAfter) Close() error { return nil }

// TestWriteObjectsZipCopyErrorDoesNotLeak 首个对象拷贝失败后：后续在途 body 必须被关闭，
// 且所有 key 都要出现在失败清单里（旧实现的 break 让它们既不关闭也不上报）。
func TestWriteObjectsZipCopyErrorDoesNotLeak(t *testing.T) {
	keys := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}
	var mu sync.Mutex
	created, closed := 0, 0
	get := func(_ context.Context, key string) (io.ReadCloser, string, error) {
		mu.Lock()
		created++
		mu.Unlock()
		return &countingBody{
			Reader: &errAfter{r: strings.NewReader(strings.Repeat(key, 64)), remain: 8},
			onClose: func() {
				mu.Lock()
				closed++
				mu.Unlock()
			},
		}, "text/plain", nil
	}

	fails, err := WriteObjectsZip(context.Background(), get, keys, io.Discard)
	if err != nil {
		t.Fatalf("WriteObjectsZip: %v", err)
	}
	if len(fails) != len(keys) {
		t.Fatalf("fails = %v, want all %d keys", fails, len(keys))
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		c, cl := created, closed
		mu.Unlock()
		if c > 0 && cl == c {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("body 泄漏：created=%d closed=%d（worker 仍阻塞在 results 上）", c, cl)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// countingBody 统计 Close 次数。
type countingBody struct {
	io.Reader
	once    sync.Once
	onClose func()
}

func (b *countingBody) Close() error {
	b.once.Do(b.onClose)
	return nil
}
