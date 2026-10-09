package s3wrap

// checksum_test.go —— 端到端校验和：Head 暴露存储端校验和、CRC64NVME 实现向量、
// VerifyObjectChecksum 的算法阶梯与降级（合成校验和跳过 / 无校验和回退 ETag-MD5 / 均无则 none）。
//
// 向量来源（独立于实现）：
//   - CRC-64/NVME：CRC RevEng 目录参数（poly 0xad93d23594c93659 反射、init/xorout 全 1）
//     与 check("123456789")=0xAE8B14860A799888；
//   - "checksum probe payload" 的 CRC64NVME / CRC32C / SHA256 / SHA1 四值由 Python 独立实现算出，
//     并与真实 RustFS 返回的 base64（N4bktbEKNg8= / OL0iOA== / r+tj1ERvRP2f7jYwQzQIixedBQG2kUn3ZrRI1O6CdCU=
//     / nY7HDn3gD4GCCi0zieizVEU5TTY=）一致；etag 与 md5 hex 一致。

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestCRC64NVMEVectors CRC64NVME 实现的外部已知向量（含 base64 大端编码，走生产编码器）。
func TestCRC64NVMEVectors(t *testing.T) {
	cases := []struct {
		in   string
		want uint64
		b64  string
	}{
		{"", 0x0000000000000000, "AAAAAAAAAAA="},
		{"123456789", 0xAE8B14860A799888, "rosUhgp5mIg="},
		{"checksum probe payload", 0x3786E4B5B10A360F, "N4bktbEKNg8="},
	}
	encode := checksumAlgos[methodCRC64NVME].encode
	for _, tc := range cases {
		h := newCRC64NVME()
		if _, err := h.Write([]byte(tc.in)); err != nil {
			t.Fatalf("write: %v", err)
		}
		if got := h.Sum64(); got != tc.want {
			t.Fatalf("crc64nvme(%q) = 0x%016X, want 0x%016X", tc.in, got, tc.want)
		}
		if got := encode(h); got != tc.b64 {
			t.Fatalf("encode(crc64nvme(%q)) = %q, want %q", tc.in, got, tc.b64)
		}
	}
}

// TestCRC64NVMEStreaming 分块写入与整块等价；hash.Hash 接口语义（Sum 大端 / Reset 回初值）。
func TestCRC64NVMEStreaming(t *testing.T) {
	h := newCRC64NVME()
	for _, part := range [][]byte{[]byte("check"), []byte("sum probe "), []byte("payload")} {
		if _, err := h.Write(part); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if got, want := h.Sum64(), uint64(0x3786E4B5B10A360F); got != want {
		t.Fatalf("chunked crc64nvme = 0x%016X, want 0x%016X", got, want)
	}
	if h.Size() != 8 || h.BlockSize() != 1 {
		t.Fatalf("Size/BlockSize = %d/%d, want 8/1", h.Size(), h.BlockSize())
	}
	sum := h.Sum(nil)
	if len(sum) != 8 || hex.EncodeToString(sum) != "3786e4b5b10a360f" {
		t.Fatalf("Sum() = %x, want 3786e4b5b10a360f (big-endian)", sum)
	}
	h.Reset()
	if h.Sum64() != 0x0000000000000000 {
		t.Fatalf("Reset() must restore init state, got 0x%016X", h.Sum64())
	}
}

// TestHeadObjectMetaExposesChecksums Head(ChecksumMode=ENABLED) 返回的校验和进入 DTO。
func TestHeadObjectMetaExposesChecksums(t *testing.T) {
	var gotMode string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMode = r.Header.Get("x-amz-checksum-mode")
		w.Header().Set("x-amz-checksum-crc64nvme", "N4bktbEKNg8=")
		w.Header().Set("x-amz-checksum-sha256", "r+tj1ERvRP2f7jYwQzQIixedBQG2kUn3ZrRI1O6CdCU=")
		w.Header().Set("x-amz-checksum-crc32c", "OL0iOA==")
		w.Header().Set("x-amz-checksum-sha1", "nY7HDn3gD4GCCi0zieizVEU5TTY=")
		w.Header().Set("x-amz-checksum-type", "FULL_OBJECT")
		w.Header().Set("Content-Length", "22")
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	}))
	meta, err := c.HeadObjectMeta(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("HeadObjectMeta: %v", err)
	}
	if gotMode != "ENABLED" {
		t.Fatalf("x-amz-checksum-mode = %q, want ENABLED", gotMode)
	}
	if meta.Checksums == nil {
		t.Fatal("Checksums = nil, want parsed values")
	}
	cs := meta.Checksums
	if cs.CRC64NVME != "N4bktbEKNg8=" || cs.CRC32C != "OL0iOA==" ||
		cs.SHA256 != "r+tj1ERvRP2f7jYwQzQIixedBQG2kUn3ZrRI1O6CdCU=" ||
		cs.SHA1 != "nY7HDn3gD4GCCi0zieizVEU5TTY=" || cs.Type != "FULL_OBJECT" {
		t.Fatalf("checksums = %+v", cs)
	}
}

// TestHeadObjectMetaWithoutChecksums 厂商未返回校验和 → Checksums 为 nil（降级口径）。
func TestHeadObjectMetaWithoutChecksums(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
	}))
	meta, err := c.HeadObjectMeta(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("HeadObjectMeta: %v", err)
	}
	if meta.Checksums != nil {
		t.Fatalf("Checksums = %+v, want nil", meta.Checksums)
	}
}

// TestHeadObjectMetaTypeOnlyHeader 只有 checksum-type、无任何值 → 仍视为「无校验和」。
func TestHeadObjectMetaTypeOnlyHeader(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-amz-checksum-type", "FULL_OBJECT")
		w.WriteHeader(http.StatusOK)
	}))
	meta, err := c.HeadObjectMeta(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("HeadObjectMeta: %v", err)
	}
	if meta.Checksums != nil {
		t.Fatalf("Checksums = %+v, want nil", meta.Checksums)
	}
}

// TestVerifyObjectChecksumCRC64NVME 首选 CRC64NVME：本地重算与存储端一致 → match。
func TestVerifyObjectChecksumCRC64NVME(t *testing.T) {
	var gotQueries []string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueries = append(gotQueries, r.Method+" "+r.URL.RawQuery)
		if r.Method == http.MethodHead {
			w.Header().Set("x-amz-checksum-crc64nvme", "N4bktbEKNg8=")
			w.Header().Set("x-amz-checksum-type", "FULL_OBJECT")
			w.Header().Set("Content-Length", "22")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "checksum probe payload")
	}))
	v, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", "v-7")
	if err != nil {
		t.Fatalf("VerifyObjectChecksum: %v", err)
	}
	if v.Method != "crc64nvme" || !v.Match {
		t.Fatalf("verify = %+v, want crc64nvme match", v)
	}
	if v.Local != "N4bktbEKNg8=" || v.Remote != "N4bktbEKNg8=" {
		t.Fatalf("local/remote = %q/%q", v.Local, v.Remote)
	}
	if len(gotQueries) != 2 || !strings.HasPrefix(gotQueries[0], "HEAD ") {
		t.Fatalf("want HEAD-then-GET, got %v", gotQueries)
	}
	if !strings.Contains(gotQueries[1], "versionId=v-7") {
		t.Fatalf("GET must forward versionId: %q", gotQueries[1])
	}
}

// TestVerifyObjectChecksumMismatch 存储端值与本地重算不一致 → match=false（诚实报告）。
func TestVerifyObjectChecksumMismatch(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-amz-checksum-crc64nvme", "AAAAAAAAAAA=")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "checksum probe payload")
	}))
	v, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("VerifyObjectChecksum: %v", err)
	}
	if v.Method != "crc64nvme" || v.Match {
		t.Fatalf("verify = %+v, want crc64nvme mismatch", v)
	}
}

// TestVerifyObjectChecksumAlgorithmLadder 无 CRC64NVME 时按 crc32c → sha256 → sha1 阶梯回退。
func TestVerifyObjectChecksumAlgorithmLadder(t *testing.T) {
	payload := "checksum probe payload"
	cases := []struct {
		name   string
		header string
		value  string
		method string
	}{
		{"crc32c", "x-amz-checksum-crc32c", "OL0iOA==", "crc32c"},
		{"sha256", "x-amz-checksum-sha256", "r+tj1ERvRP2f7jYwQzQIixedBQG2kUn3ZrRI1O6CdCU=", "sha256"},
		{"sha1", "x-amz-checksum-sha1", "nY7HDn3gD4GCCi0zieizVEU5TTY=", "sha1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hdr, val := tc.header, tc.value
			c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.Header().Set(hdr, val)
					w.Header().Set("ETag", `"someetag"`)
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, payload)
			}))
			v, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", "")
			if err != nil {
				t.Fatalf("VerifyObjectChecksum: %v", err)
			}
			if v.Method != tc.method || !v.Match {
				t.Fatalf("verify = %+v, want %s match", v, tc.method)
			}
		})
	}
}

// TestVerifyObjectChecksumETagMD5 无存储校验和但 ETag 是单段 MD5 → 以 ETag-MD5 尽力比对。
func TestVerifyObjectChecksumETagMD5(t *testing.T) {
	etag := `"` + md5Of("checksum probe payload") + `"`
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "checksum probe payload")
	}))
	v, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("VerifyObjectChecksum: %v", err)
	}
	if v.Method != "etag-md5" || !v.Match {
		t.Fatalf("verify = %+v, want etag-md5 match", v)
	}
	if v.Local != md5Of("checksum probe payload") || v.Local != v.Remote {
		t.Fatalf("local/remote = %q/%q", v.Local, v.Remote)
	}
}

// TestVerifyObjectChecksumETagMD5Mismatch ETag-MD5 路径的不匹配同样必须如实报告。
func TestVerifyObjectChecksumETagMD5Mismatch(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("ETag", `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "checksum probe payload")
	}))
	v, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("VerifyObjectChecksum: %v", err)
	}
	if v.Method != "etag-md5" || v.Match {
		t.Fatalf("verify = %+v, want etag-md5 mismatch", v)
	}
}

// TestVerifyObjectChecksumNone 分段 ETag（含 "-"）且无存储校验和 → method=none，不再发起 GET。
func TestVerifyObjectChecksumNone(t *testing.T) {
	var getServed bool
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("ETag", `"0123456789abcdef0123456789abcdef-3"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		getServed = true
		w.WriteHeader(http.StatusOK)
	}))
	v, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("VerifyObjectChecksum: %v", err)
	}
	if v.Method != "none" || v.Match || v.Local != "" || v.Remote != "" {
		t.Fatalf("verify = %+v, want none/empty", v)
	}
	if getServed {
		t.Fatal("must not fetch body when no verifiable checksum exists")
	}
}

// TestVerifyObjectChecksumCompositeSkipped 分段合成（COMPOSITE）校验和不可全对象比对：
// 按类型字段识别（COMPOSITE_* 前缀）与按值内 "-" 识别两条路径都必须跳过并落到 none。
func TestVerifyObjectChecksumCompositeSkipped(t *testing.T) {
	cases := []struct {
		name  string
		hdr   string
		value string
		ctype string
	}{
		{"type-composite", "x-amz-checksum-crc64nvme", "N4bktbEKNg8=", "COMPOSITE_CRC64NVME"},
		{"value-composite", "x-amz-checksum-crc32c", "OL0iOA==-4", "FULL_OBJECT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hdr, val, ctype := tc.hdr, tc.value, tc.ctype
			c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.Header().Set(hdr, val)
					w.Header().Set("x-amz-checksum-type", ctype)
					w.Header().Set("ETag", `"abc-2"`)
					w.WriteHeader(http.StatusOK)
					return
				}
				t.Errorf("must not GET object with composite checksum")
				w.WriteHeader(http.StatusOK)
			}))
			v, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", "")
			if err != nil {
				t.Fatalf("VerifyObjectChecksum: %v", err)
			}
			if v.Method != "none" {
				t.Fatalf("verify = %+v, want none", v)
			}
		})
	}
}

// TestVerifyObjectChecksumNotFound 对象不存在 → 错误上抛（handler 映射 404）。
func TestVerifyObjectChecksumNotFound(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusNotFound, "NoSuchKey", "gone")
	}))
	if _, err := c.VerifyObjectChecksum(context.Background(), "bkt", "gone.txt", ""); err == nil {
		t.Fatal("want error for missing object")
	}
}

// TestVerifyObjectChecksumGetFailure HEAD 成功但 GET 失败 → 错误上抛（不得吞成 mismatch）。
func TestVerifyObjectChecksumGetFailure(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-amz-checksum-crc64nvme", "N4bktbEKNg8=")
			w.WriteHeader(http.StatusOK)
			return
		}
		writeS3Error(w, http.StatusForbidden, "AccessDenied", "no")
	}))
	if _, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", ""); err == nil {
		t.Fatal("want error when body fetch fails")
	}
}

// TestVerifyObjectChecksumBodyReadFailure 响应体中途读失败（截断）→ 错误上抛。
func TestVerifyObjectChecksumBodyReadFailure(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("x-amz-checksum-crc64nvme", "N4bktbEKNg8=")
			w.Header().Set("Content-Length", "22")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Length", "22")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "short") // 承诺 22 字节只给 5 → 客户端读到 unexpected EOF
	}))
	if _, err := c.VerifyObjectChecksum(context.Background(), "bkt", "k.txt", ""); err == nil {
		t.Fatal("want error when body stream is truncated")
	}
}

// md5Of 返回十六进制 MD5（etag-md5 用例的期望值来源）。
func md5Of(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
