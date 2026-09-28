package service

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSanitizeZipName(t *testing.T) {
	cases := map[string]string{
		"a/b.txt": "a/b.txt",
		"../x":    "_/x",
		"a/../b":  "a/_/b",
		`a\..\b`:  "a/_/b",
		"foo.":    "foo",
		".. ":     "_",
		"":        "download",
		"a:b":     "a_b",
	}
	for in, want := range cases {
		if got := SanitizeZipName(in); got != want {
			t.Errorf("SanitizeZipName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestWriteObjectsZip(t *testing.T) {
	var buf bytes.Buffer
	get := func(_ context.Context, key string) (io.ReadCloser, string, error) {
		if key == "bad" {
			return nil, "", io.ErrUnexpectedEOF
		}
		return io.NopCloser(strings.NewReader(key)), "text/plain", nil
	}
	fails, err := WriteObjectsZip(context.Background(), get, []string{"a.txt", "bad", "b.txt"}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(fails) != 1 || fails[0] != "bad" {
		t.Fatalf("fails=%v", fails)
	}
	if buf.Len() < 50 {
		t.Fatalf("zip too small: %d", buf.Len())
	}
}

func TestWriteObjectsZipParallel(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	inflight := 0
	maxInflight := 0
	get := func(_ context.Context, key string) (io.ReadCloser, string, error) {
		mu.Lock()
		inflight++
		if inflight > maxInflight {
			maxInflight = inflight
		}
		mu.Unlock()
		defer func() {
			mu.Lock()
			inflight--
			mu.Unlock()
		}()
		time.Sleep(20 * time.Millisecond)
		return io.NopCloser(strings.NewReader(key)), "text/plain", nil
	}
	keys := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt", "f.txt"}
	fails, err := WriteObjectsZip(context.Background(), get, keys, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(fails) != 0 {
		t.Fatalf("fails=%v", fails)
	}
	if maxInflight < 2 {
		t.Fatalf("expected concurrent fetches, maxInflight=%d", maxInflight)
	}
	if maxInflight > zipFetchWorkers {
		t.Fatalf("maxInflight=%d > workers %d", maxInflight, zipFetchWorkers)
	}
}

func TestSameEndpoint(t *testing.T) {
	cases := []struct {
		name               string
		aEndpoint, aRegion string
		aUseSSL            bool
		bEndpoint, bRegion string
		bUseSSL            bool
		want               bool
	}{
		{"explicit equal, trailing slash", "http://minio:9000", "us-east-1", false, "http://minio:9000/", "us-east-1", false, true},
		{"scheme differs", "https://a.com", "us-east-1", false, "http://a.com", "us-east-1", false, false},
		{"https trailing slash", "https://a.com", "us-east-1", false, "https://a.com/", "us-east-1", false, true},
		{"different hosts", "http://a.com", "us-east-1", false, "http://b.com", "us-east-1", false, false},
		{"empty vs explicit", "", "us-east-1", false, "http://localhost:9000", "us-east-1", false, false},
		{"both blank default", "  ", "  ", false, "  ", "  ", false, true},
		{"both empty same region", "", "us-east-1", false, "", "us-east-1", false, true},
		{"both empty different region", "", "us-east-1", false, "", "eu-west-1", false, false},
		{"both empty region case/trim", "", " eu-west-1 ", false, "", "EU-WEST-1", false, true},
		{"explicit same endpoint ignores region", "http://a.com", "us-east-1", false, "http://a.com", "eu-west-1", false, true},
		// useSSL 参与裸端点补全（与建 client 的 BaseEndpoint 同一口径）：
		{"bare endpoints same TLS", "a.com", "us-east-1", true, "a.com", "us-east-1", true, true},
		{"bare endpoints differing TLS", "a.com", "us-east-1", true, "a.com", "us-east-1", false, false},
		{"bare https matches explicit https", "https://a.com", "us-east-1", false, "a.com", "us-east-1", true, true},
		{"bare http matches explicit http", "http://a.com", "us-east-1", false, "a.com", "us-east-1", false, true},
		{"bare https vs explicit http", "http://a.com", "us-east-1", false, "a.com", "us-east-1", true, false},
		// 显式 scheme 优先于账号开关（建 client 时同样如此），故两侧仍同端。
		{"explicit scheme beats useSSL", "http://a.com", "us-east-1", true, "http://a.com", "us-east-1", false, true},
		// 端点为空时账号开关不参与（未配 BaseEndpoint → SDK 恒走默认 https 端点）。
		{"both empty ignores useSSL", "", "us-east-1", true, "", "us-east-1", false, true},
	}
	for _, c := range cases {
		if got := SameEndpoint(c.aEndpoint, c.aRegion, c.aUseSSL, c.bEndpoint, c.bRegion, c.bUseSSL); got != c.want {
			t.Errorf("%s: SameEndpoint(%q,%q,%v,%q,%q,%v)=%v want %v",
				c.name, c.aEndpoint, c.aRegion, c.aUseSSL, c.bEndpoint, c.bRegion, c.bUseSSL, got, c.want)
		}
	}
}
