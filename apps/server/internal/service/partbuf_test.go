package service

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// drainPartBufPool 取空池中残留的空闲缓冲，让后续断言只针对本次调用归还的块。
func drainPartBufPool() {
	for {
		select {
		case <-partBufPool:
		default:
			return
		}
	}
}

// TestPartBufReuse 归还后的分段缓冲应被复用（同一底层数组），
// 避免每个大对象流式复制都新分配一份 64MB。
func TestPartBufReuse(t *testing.T) {
	// 使用一个其它测试不会用到的独特尺寸，保证池中不会有更大/同尺寸的干扰项。
	const size = int64(1<<20 + 12345)

	first := acquirePartBuf(size)
	if int64(len(first)) != size {
		t.Fatalf("len = %d, want %d", len(first), size)
	}
	for i := range first {
		first[i] = 0 // 触碰整块，确保是真实可写内存
	}
	first[0] = 0xAB
	releasePartBuf(first)

	second := acquirePartBuf(size)
	defer releasePartBuf(second)
	if int64(len(second)) != size {
		t.Fatalf("len = %d, want %d", len(second), size)
	}
	if &first[0] != &second[0] {
		t.Fatal("buffer not reused: acquire returned a different backing array")
	}
	if second[0] != 0xAB {
		t.Fatalf("reused buffer content = %#x, want 0xAB", second[0])
	}
}

// TestPartBufNeverUndersized 池中尺寸不足的旧缓冲不得返回给更大的请求。
func TestPartBufNeverUndersized(t *testing.T) {
	releasePartBuf(make([]byte, 8))
	got := acquirePartBuf(4096)
	defer releasePartBuf(got)
	if cap(got) < 4096 || len(got) != 4096 {
		t.Fatalf("len/cap = %d/%d, want >= 4096", len(got), cap(got))
	}
}

// TestPartBufNoDoubleCheckout 同一块缓冲不得同时发给两个持有者（避免别名写入）。
func TestPartBufNoDoubleCheckout(t *testing.T) {
	a := acquirePartBuf(1024)
	b := acquirePartBuf(1024)
	defer releasePartBuf(a)
	defer releasePartBuf(b)
	if &a[0] == &b[0] {
		t.Fatal("pool handed out the same backing array to two live holders")
	}
}

// TestPartBufReleaseEmpty 空缓冲（cap=0）归还时应直接忽略，不得入池污染后续获取。
func TestPartBufReleaseEmpty(t *testing.T) {
	drainPartBufPool()
	releasePartBuf(nil)
	releasePartBuf([]byte{})
	got := acquirePartBuf(4)
	if len(got) != 4 || cap(got) != 4 {
		t.Fatalf("len/cap = %d/%d, want 4/4 (empty buffers must not be pooled)", len(got), cap(got))
	}
}

// TestPartBufReleaseIsBounded 池有上限：超量归还不会阻塞，也不会无限占用内存。
func TestPartBufReleaseIsBounded(t *testing.T) {
	drainPartBufPool()
	const n = maxIdlePartBufs * 4
	for i := 0; i < n; i++ {
		releasePartBuf(make([]byte, 16))
	}
	retained := 0
	for {
		select {
		case <-partBufPool:
			retained++
		default:
			if retained != maxIdlePartBufs {
				t.Fatalf("pool retained %d idle buffers, want %d", retained, maxIdlePartBufs)
			}
			return
		}
	}
}

// TestMultipartStreamCopyReleasesBufferOnAbort 出错/abort 返回路径也必须归还分段缓冲，
// 否则「每失败一次大对象复制」就永久漏掉一块 64MB。
func TestMultipartStreamCopyReleasesBufferOnAbort(t *testing.T) {
	drainPartBufPool()
	old := multipartPartSize
	multipartPartSize = 1 << 16
	t.Cleanup(func() { multipartPartSize = old })

	f := newSvcFake(t)
	f.failPart = true
	if err := MultipartStreamCopy(context.Background(), f.dstClient(t), "dst", "x", "", strings.NewReader("data")); err == nil {
		t.Fatal("expected part error")
	}
	if f.aborts != 1 {
		t.Fatalf("aborts = %d, want 1", f.aborts)
	}
	buf := acquirePartBuf(multipartPartSize)
	defer releasePartBuf(buf)
	if int64(cap(buf)) < multipartPartSize {
		t.Fatal("buffer not returned to pool on abort path")
	}
}

// TestMultipartStreamCopyReleasesBufferOnSuccess 成功路径同样归还缓冲（供下一个大对象复用）。
func TestMultipartStreamCopyReleasesBufferOnSuccess(t *testing.T) {
	drainPartBufPool()
	old := multipartPartSize
	multipartPartSize = 1 << 16
	t.Cleanup(func() { multipartPartSize = old })

	f := newSvcFake(t)
	if err := MultipartStreamCopy(context.Background(), f.dstClient(t), "dst", "ok", "", strings.NewReader("data")); err != nil {
		t.Fatalf("MultipartStreamCopy: %v", err)
	}
	buf := acquirePartBuf(multipartPartSize)
	defer releasePartBuf(buf)
	if int64(cap(buf)) < multipartPartSize {
		t.Fatal("buffer not returned to pool on success path")
	}
}

// TestPartBufConcurrent 并发获取/归还（go test -race 下验证无数据竞争）。
func TestPartBufConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				b := acquirePartBuf(64)
				if len(b) != 64 {
					t.Errorf("len = %d, want 64", len(b))
					return
				}
				b[0], b[63] = 1, 2
				releasePartBuf(b)
			}
		}()
	}
	wg.Wait()
}
