package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

// TestSQLiteListReturnsErrorAfterClose 回归：SQLite 查询失败不得静默返回空列表
// （用户会误以为账号全丢），必须把错误返回给调用方以映射 500。
func TestSQLiteListReturnsErrorAfterClose(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "sqlite", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := st.Create(&model.Account{Name: "a", Endpoint: "http://127.0.0.1:9000", AccessKey: "ak", SecretKey: "sk", Region: "us-east-1"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	accounts, listErr := st.List()
	if listErr == nil {
		t.Fatalf("List after close: want error, got nil with %d accounts", len(accounts))
	}
	if len(accounts) != 0 {
		t.Fatalf("List after close returned accounts: %d", len(accounts))
	}
}

// TestSQLiteListReturnsErrorOnIterationInterruption R13:迭代中途失败(生产中如优雅关停
// 并发 Close;此处用「尾页损坏」确定性复现同类中断)必须返回错误,不得把截断的列表当
// 成成功返回——否则 UI 会静默丢账号。
//
// 配方:写满多页 → checkpoint 落主库 → 关闭 → 覆写最后一个物理页 → 重开 List。
// modernc 驱动在 Query 时只步进第一行,后续行惰性步进,故首行可正常返回、
// 错误在 rows.Err() 暴露,正好命中本次要修的分支。
func TestSQLiteListReturnsErrorOnIterationInterruption(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "accounts.db")
	st, err := openSQLite(dbPath, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// 300 行 × 约 70 字符名字 ≈ 24 页,保证数据横跨多个页。
	for i := 0; i < 300; i++ {
		if _, err := st.Create(gapAcc(fmt.Sprintf("acct-%03d-padding-to-fill-pages-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", i))); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	var pageSize int
	if err := st.db.QueryRow(`PRAGMA page_size`).Scan(&pageSize); err != nil {
		t.Fatalf("page_size: %v", err)
	}
	if _, err := st.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// 覆写最后一个物理页(0xFF 非法页内容)。
	f, err := os.OpenFile(dbPath, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open db file: %v", err)
	}
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat db file: %v", err)
	}
	if _, err := f.WriteAt(make([]byte, pageSize), fi.Size()-int64(pageSize)); err != nil {
		t.Fatalf("corrupt tail page: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close db file: %v", err)
	}

	st2, err := openSQLite(dbPath, "")
	if err != nil {
		t.Fatalf("reopen corrupted db: %v", err)
	}
	accounts, listErr := st2.List()
	if listErr == nil {
		t.Fatalf("List with corrupted tail page: want error, got nil with %d accounts (truncated list treated as success)", len(accounts))
	}
	if len(accounts) != 0 {
		t.Fatalf("List on iteration error must not return partial accounts, got %d", len(accounts))
	}
}
