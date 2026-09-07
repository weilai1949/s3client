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
		bEndpoint, bRegion string
		want               bool
	}{
		{"explicit equal, trailing slash", "http://minio:9000", "us-east-1", "http://minio:9000/", "us-east-1", true},
		{"scheme differs", "https://a.com", "us-east-1", "http://a.com", "us-east-1", false},
		{"https trailing slash", "https://a.com", "us-east-1", "https://a.com/", "us-east-1", true},
		{"different hosts", "http://a.com", "us-east-1", "http://b.com", "us-east-1", false},
		{"empty vs explicit", "", "us-east-1", "http://localhost:9000", "us-east-1", false},
		{"both blank default", "  ", "  ", "  ", "  ", true},
		{"both empty same region", "", "us-east-1", "", "us-east-1", true},
		{"both empty different region", "", "us-east-1", "", "eu-west-1", false},
		{"both empty region case/trim", "", " eu-west-1 ", "", "EU-WEST-1", true},
		{"explicit same endpoint ignores region", "http://a.com", "us-east-1", "http://a.com", "eu-west-1", true},
	}
	for _, c := range cases {
		if got := SameEndpoint(c.aEndpoint, c.aRegion, c.bEndpoint, c.bRegion); got != c.want {
			t.Errorf("%s: SameEndpoint(%q,%q,%q,%q)=%v want %v", c.name, c.aEndpoint, c.aRegion, c.bEndpoint, c.bRegion, got, c.want)
		}
	}
}
