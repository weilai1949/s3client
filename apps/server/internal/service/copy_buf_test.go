package service

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

// countingWriter 记录每次 Write 的字节数，用于断言「大流量走大缓冲」。
// 不实现 io.ReaderFrom，避免 io.CopyBuffer 走 ReaderFrom 捷径绕过缓冲池。
type countingWriter struct {
	writes []int
	total  int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.writes = append(c.writes, len(p))
	c.total += len(p)
	return len(p), nil
}

// opaqueReader 只暴露 Read，屏蔽底层可能实现的 io.WriterTo（bytes.Reader 有），
// 从而真正测到 io.CopyBuffer 的缓冲路径。
type opaqueReader struct{ r io.Reader }

func (o opaqueReader) Read(p []byte) (int, error) { return o.r.Read(p) }

// TestCopyBufferedUsesLargeBuffer 64KiB 负载：通用 32KiB 缓冲需 2 次写、
// 256KiB 有界缓冲只需 1 次，且字节数守恒。
func TestCopyBufferedUsesLargeBuffer(t *testing.T) {
	const total = 64 << 10
	dst := &countingWriter{}
	src := opaqueReader{bytes.NewReader(make([]byte, total))}
	n, err := CopyBuffered(dst, src)
	if err != nil {
		t.Fatalf("CopyBuffered: %v", err)
	}
	if n != total || dst.total != total {
		t.Fatalf("n=%d total=%d, want %d", n, dst.total, total)
	}
	if len(dst.writes) != 1 {
		t.Fatalf("写调用 %d 次（每次 %v），want 1（256KiB 缓冲）", len(dst.writes), dst.writes)
	}
	if got := dst.writes[0]; got > copyBufSize {
		t.Fatalf("单次写 %d > copyBufSize %d", got, copyBufSize)
	}
}

// TestCopyBufferedPreservesBigStream 1MiB 负载：写调用数远少于 32KiB 通用缓冲的 32 次，
// 且总字节数与尾部哨兵字节完全一致（不得截断 / 重排）。
func TestCopyBufferedPreservesBigStream(t *testing.T) {
	const total = 1 << 20
	data := bytes.Repeat([]byte("abcdefgh"), total/8)
	data[total-1] = 0x5A
	dst := &countingWriter{}
	n, err := CopyBuffered(dst, opaqueReader{bytes.NewReader(data)})
	if err != nil {
		t.Fatalf("CopyBuffered: %v", err)
	}
	if n != total {
		t.Fatalf("n=%d, want %d", n, total)
	}
	if len(dst.writes) > 8 {
		t.Fatalf("写调用 %d 次，want ≤8（32KiB 通用缓冲会是 32 次）", len(dst.writes))
	}
}

// TestCopyBufferedPropagatesError 源读错误必须原样返回给调用方（流式下载失败要可观测）。
func TestCopyBufferedPropagatesError(t *testing.T) {
	sentinel := io.ErrUnexpectedEOF
	_, err := CopyBuffered(&countingWriter{}, opaqueReader{errReader{sentinel}})
	if err != sentinel {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func drainCopyBufPool() {
	for {
		select {
		case <-copyBufPool:
		default:
			return
		}
	}
}

// TestCopyBufPoolReuse 归还的缓冲应被复用（同一底层数组），避免每路流式下载新分配 256KiB。
func TestCopyBufPoolReuse(t *testing.T) {
	drainCopyBufPool()
	first := acquireCopyBuf()
	if len(first) != copyBufSize {
		t.Fatalf("len = %d, want %d", len(first), copyBufSize)
	}
	first[0] = 0xAB
	releaseCopyBuf(first)
	second := acquireCopyBuf()
	defer releaseCopyBuf(second)
	if &first[0] != &second[0] {
		t.Fatal("buffer not reused: acquire returned a different backing array")
	}
}

// TestCopyBufPoolIsBounded 池有上限：超量归还不会阻塞，也不会无限占用内存。
func TestCopyBufPoolIsBounded(t *testing.T) {
	drainCopyBufPool()
	for i := 0; i < maxIdleCopyBufs*4; i++ {
		releaseCopyBuf(make([]byte, copyBufSize))
	}
	retained := 0
	for {
		select {
		case <-copyBufPool:
			retained++
		default:
			if retained != maxIdleCopyBufs {
				t.Fatalf("pool retained %d idle buffers, want %d", retained, maxIdleCopyBufs)
			}
			return
		}
	}
}

// TestCopyBufNoDoubleCheckout 同一块缓冲不得同时发给两个持有者。
func TestCopyBufNoDoubleCheckout(t *testing.T) {
	a := acquireCopyBuf()
	b := acquireCopyBuf()
	defer releaseCopyBuf(a)
	defer releaseCopyBuf(b)
	if &a[0] == &b[0] {
		t.Fatal("pool handed out the same backing array to two live holders")
	}
}

// TestCopyBufferedConcurrent 并发调用（go test -race 下验证无数据竞争）。
func TestCopyBufferedConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				dst := &countingWriter{}
				if _, err := CopyBuffered(dst, opaqueReader{bytes.NewReader(make([]byte, 4096))}); err != nil {
					t.Errorf("CopyBuffered: %v", err)
					return
				}
				if dst.total != 4096 {
					t.Errorf("total = %d, want 4096", dst.total)
					return
				}
			}
		}()
	}
	wg.Wait()
}
