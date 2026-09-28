package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

func TestSQLiteStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "accounts.db")
	s, err := openSQLite(dbPath, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	created, err := s.Create(&model.Account{
		Name: "sqlite", Endpoint: "http://127.0.0.1:9000", AccessKey: "ak", SecretKey: "sk", Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.Get(created.ID)
	if err != nil || got.SecretKey != "sk" {
		t.Fatalf("get: %+v err=%v", got, err)
	}
	updated, err := s.Update(created.ID, &model.Account{
		Name: "renamed", Endpoint: got.Endpoint, AccessKey: got.AccessKey, SecretKey: model.MaskedSecret,
	})
	if err != nil || updated.Name != "renamed" {
		t.Fatalf("update: %+v err=%v", updated, err)
	}
	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(created.ID); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestOpenSQLiteDriver(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "sqlite", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.(*SQLiteStore); !ok {
		t.Fatalf("expected *SQLiteStore, got %T", st)
	}
}

// TestOpenSQLiteSchemaErrorPropagates R12:建表/迁移错误必须上抛让 Open 返回错误走
// 启动失败路径——此前 `_, _ = db.Exec(sqliteSchema)` 把错误吞掉,坏盘下会「启动成功、
// 运行时全挂」。触发方式:预建一个缺 sort_order 列的同名 accounts 表,
// 使 sqliteSchema 里的 CREATE INDEX 报 "no such column"。
func TestOpenSQLiteSchemaErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "accounts.db")
	pre, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("pre open: %v", err)
	}
	if _, err := pre.Exec(`CREATE TABLE accounts (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatalf("pre-create table: %v", err)
	}
	if err := pre.Close(); err != nil {
		t.Fatalf("pre close: %v", err)
	}
	if _, err := openSQLite(dbPath, ""); err == nil {
		t.Fatal("openSQLite must fail when schema exec fails")
	}
	if _, err := Open(dir, "sqlite", ""); err == nil {
		t.Fatal("Open must fail when schema exec fails (startup failure path)")
	}
}

// TestSQLiteSidecarPerms0600 Nit:主库与 -wal/-shm 侧车必须 0600——侧车含明文页
// (secret_key 密文/明文行都在其中)。首次建库时侧车在主库收紧权限前创建,
// 在典型 umask 022 下是 0644,必须由 openSQLite 统一收紧;checkpoint 与重开
// 重建后同样保持。Windows 无 unix 权限位语义,跳过(与 lock_unix/lock_other 分流一致)。
func TestSQLiteSidecarPerms0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 无 unix 权限位语义")
	}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "accounts.db")
	assertPerms := func(stage string) {
		t.Helper()
		for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
			fi, err := os.Stat(p)
			if err != nil {
				t.Fatalf("%s: stat %s: %v", stage, filepath.Base(p), err)
			}
			if fi.Mode().Perm() != 0o600 {
				t.Fatalf("%s: %s perm = %04o, want 0600", stage, filepath.Base(p), fi.Mode().Perm())
			}
		}
	}
	st, err := Open(dir, "sqlite", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	assertPerms("after open")
	if _, err := st.Create(gapAcc("perm")); err != nil {
		t.Fatalf("create: %v", err)
	}
	assertPerms("after create")
	if _, err := st.(*SQLiteStore).db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	assertPerms("after checkpoint")
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// 重开重建侧车后仍须 0600。
	st2, err := Open(dir, "sqlite", "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := st2.Create(gapAcc("perm2")); err != nil {
		t.Fatalf("create after reopen: %v", err)
	}
	assertPerms("after reopen")
}

// TestSQLiteDSNBusyTimeout Nit:DSN 必须带 _busy_timeout=5000,防并发写锁时立刻
// 报 "database is locked";同时验证 pragma 在真实连接上生效(而非只拼串)。
func TestSQLiteDSNBusyTimeout(t *testing.T) {
	const p = "/var/lib/s3c/accounts.db"
	dsn := sqliteDSN(p)
	if !strings.HasPrefix(dsn, p+"?") {
		t.Fatalf("dsn %q must keep db path as query prefix", dsn)
	}
	for _, want := range []string{"_pragma=busy_timeout(5000)", "_pragma=foreign_keys(1)", "_pragma=journal_mode(WAL)"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn %q missing %q", dsn, want)
		}
	}
	s, err := openSQLite(filepath.Join(t.TempDir(), "accounts.db"), "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	var timeout int
	if err := s.db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", timeout)
	}
}
