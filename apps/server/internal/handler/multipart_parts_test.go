package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

const listPartsXMLNS = `xmlns="http://s3.amazonaws.com/doc/2006-03-01/"`

// newListPartsFixture 起一个只回 ListParts XML 的假 S3，并返回已注册路由与账号 ID。
// accBucket 为空表示账号无默认桶（用于 bucket 缺失的 400 分支）。
func newListPartsFixture(t *testing.T, s3Resp func(w http.ResponseWriter, r *http.Request), accBucket string) (http.Handler, string) {
	t.Helper()
	s3fake := httptest.NewServer(http.HandlerFunc(s3Resp))
	t.Cleanup(s3fake.Close)

	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "fake", Endpoint: s3fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: accBucket, PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes(), acc.ID
}

// TestMultipartParts 只读端点：返回服务端真实分段清单（含段号 / ETag / 大小 / 修改时间）。
func TestMultipartParts(t *testing.T) {
	h, accID := newListPartsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("uploadId") != "UPLOAD123" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><ListPartsResult `+listPartsXMLNS+`>`+
			`<Bucket>b</Bucket><Key>big.bin</Key><UploadId>UPLOAD123</UploadId><IsTruncated>false</IsTruncated>`+
			`<Part><PartNumber>1</PartNumber><LastModified>2026-10-08T05:00:00.000Z</LastModified><ETag>"e1"</ETag><Size>10485760</Size></Part>`+
			`<Part><PartNumber>2</PartNumber><LastModified>2026-10-08T05:01:00.000Z</LastModified><ETag>"e2"</ETag><Size>5</Size></Part>`+
			`</ListPartsResult>`)
	}, "b")

	rr := doJSON(t, h, http.MethodGet, "/api/accounts/"+accID+"/multipart/parts?bucket=b&key=big.bin&uploadId=UPLOAD123", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Parts []struct {
			PartNumber   int32  `json:"partNumber"`
			ETag         string `json:"etag"`
			Size         int64  `json:"size"`
			LastModified string `json:"lastModified"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
	}
	if len(resp.Parts) != 2 {
		t.Fatalf("parts = %+v", resp.Parts)
	}
	if resp.Parts[0].PartNumber != 1 || resp.Parts[0].ETag != "e1" || resp.Parts[0].Size != 10485760 {
		t.Fatalf("parts[0] = %+v", resp.Parts[0])
	}
	if resp.Parts[0].LastModified != "2026-10-08T05:00:00Z" {
		t.Fatalf("lastModified = %q", resp.Parts[0].LastModified)
	}
	if resp.Parts[1].PartNumber != 2 || resp.Parts[1].ETag != "e2" || resp.Parts[1].Size != 5 {
		t.Fatalf("parts[1] = %+v", resp.Parts[1])
	}
}

// TestMultipartPartsDefaultBucket 未传 bucket 时回退账号默认桶。
func TestMultipartPartsDefaultBucket(t *testing.T) {
	h, accID := newListPartsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><ListPartsResult `+listPartsXMLNS+`><IsTruncated>false</IsTruncated><Part><PartNumber>1</PartNumber><ETag>"e1"</ETag><Size>1</Size></Part></ListPartsResult>`)
	}, "default-bkt")
	rr := doJSON(t, h, http.MethodGet, "/api/accounts/"+accID+"/multipart/parts?key=k&uploadId=U1", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Parts []struct {
			PartNumber int32 `json:"partNumber"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil || len(resp.Parts) != 1 {
		t.Fatalf("parts = %+v, err=%v", resp.Parts, err)
	}
}

// TestMultipartPartsValidation 参数边界：缺 key / 缺 uploadId / 缺 bucket 一律 400。
func TestMultipartPartsValidation(t *testing.T) {
	h, accID := newListPartsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("S3 不应被调用")
	}, "")
	cases := []struct {
		name string
		url  string
	}{
		{"缺 key", "/api/accounts/" + accID + "/multipart/parts?uploadId=U1"},
		{"缺 uploadId", "/api/accounts/" + accID + "/multipart/parts?key=k"},
		{"缺 bucket（账号也无默认桶）", "/api/accounts/" + accID + "/multipart/parts?key=k&uploadId=U1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := doJSON(t, h, http.MethodGet, tc.url, "")
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestMultipartPartsUnknownAccount 账号不存在 → 404（accountClient 失败分支）。
func TestMultipartPartsUnknownAccount(t *testing.T) {
	h, _ := newListPartsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("S3 不应被调用")
	}, "b")
	rr := doJSON(t, h, http.MethodGet, "/api/accounts/does-not-exist/multipart/parts?bucket=b&key=k&uploadId=U1", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
}

// TestMultipartPartsS3Error ListParts 失败（如 uploadId 失效）→ 5xx，供前端回退重新 init。
func TestMultipartPartsS3Error(t *testing.T) {
	h, accID := newListPartsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		writeS3ErrorXML(w, http.StatusNotFound, "NoSuchUpload", "no such upload")
	}, "b")
	rr := doJSON(t, h, http.MethodGet, "/api/accounts/"+accID+"/multipart/parts?bucket=b&key=k&uploadId=U1", "")
	if rr.Code < 400 {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
}

// writeS3ErrorXML 返回标准 S3 错误 XML（handler 假 S3 用）。
func writeS3ErrorXML(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>`+code+`</Code><Message>`+message+`</Message></Error>`)
}
