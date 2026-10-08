package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/weilai1949/s3client/apps/server/internal/model"
)

// SQLiteStore 基于 SQLite 的账号存储（modernc.org/sqlite，纯 Go 无 CGO）。
//
// secret_key 列在 S3C_STORE_KEY 非空时以 AES-256-GCM 密文落盘（S3C3 参数），
// 空 key 时保持明文（向后兼容既有库与无 key 的本地开发）。读取时按魔数判别，
// 因此升级前写入的明文行仍可读，写回时自动加密（ASSESSMENT M1）。
type SQLiteStore struct {
	mu       sync.RWMutex
	db       *sql.DB
	path     string
	storeKey string
}

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS accounts (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  public_endpoint TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL DEFAULT '',
  access_key TEXT NOT NULL,
  secret_key TEXT NOT NULL,
  bucket TEXT NOT NULL DEFAULT '',
  path_style INTEGER NOT NULL DEFAULT 0,
  use_ssl INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_accounts_sort ON accounts(sort_order);
`

// encryptSecret 在配置了 storeKey 时把明文密钥加密为 S3C3 列值；空 key 原样返回。
// deriveKey 恒返回 32 字节合法密钥，故 AES-GCM 加密不会失败（无冗余错误分支）。
func (s *SQLiteStore) encryptSecret(secret string) string {
	if s.storeKey == "" || secret == "" {
		return secret
	}
	salt := randomSalt()
	enc, _ := encryptAESGCM(deriveKey(s.storeKey, salt, currentParams), []byte(secret))
	return string(envelope(salt, enc))
}

// decryptSecret 解析库中的 secret_key 列：S3C2/S3C3 密文按 storeKey 解密，
// 其余（历史明文行）原样返回。密文但缺 key 或密钥不符时返回错误。
func (s *SQLiteStore) decryptSecret(raw string) (string, error) {
	if !isEncryptedBlob([]byte(raw)) {
		return raw, nil
	}
	if s.storeKey == "" {
		return "", errors.New("encrypted secret_key found but S3C_STORE_KEY is not set")
	}
	params, salt, ciphertext, err := parseEnvelope([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("parse encrypted secret_key: %w", err)
	}
	plain, err := decryptAESGCM(deriveKey(s.storeKey, salt, params), ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt secret_key: %w", err)
	}
	return string(plain), nil
}

func openSQLite(dbPath, storeKey string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, err
	}
	// modernc sqlite 驱动注册于 init，sql.Open 仅解析 DSN 不实际打开；
	// 后续 Ping 才是真实 IO 探活，故此处不处理 Open 错误。
	db, _ := sql.Open("sqlite", sqliteDSN(dbPath))
	db.SetMaxOpenConns(1) // SQLite 单写；避免并发写锁冲突
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	// 建表/迁移错误必须让启动失败：`_, _ =` 吞错的话，坏盘/半成品库会「启动成功、
	// 运行时全挂」（R12）。两个错误合并到单一出口，db.Close 后上抛，避免出现
	// 「migrate 失败但建表成功」这种无人测试的分叉分支。
	_, err := db.Exec(sqliteSchema)
	if err == nil {
		err = migrateSQLiteSchema(db)
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init sqlite schema: %w", err)
	}
	s := &SQLiteStore{db: db, path: dbPath, storeKey: storeKey}
	chmodSQLitePerms(dbPath)
	return s, nil
}

// chmodSQLitePerms 把 SQLite 主库与两侧车统一收紧为 0600：侧车含明文页
// （secret_key 列数据在其中），首次建库时侧车在主库 chmod 之前创建，典型 umask 022
// 下是 0644（Nit）。侧车可能不存在（非 WAL / checkpoint 后），Chmod 对不存在文件
// 报错与只读挂载等失败同属「尽力而为的加固」，一律忽略，不影响开库。
func chmodSQLitePerms(dbPath string) {
	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		_ = os.Chmod(p, 0o600)
	}
}

// sqliteDSN 拼 SQLite DSN：外键约束、WAL、busy_timeout 三个 pragma。
// busy_timeout=5000 让并发写锁等待而非立刻报 "database is locked"
// （默认 0ms：第二个写事务一上来就失败，UI 偶发 500）。
func sqliteDSN(dbPath string) string {
	return dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
}

const sqliteUserVersion = 1

func migrateSQLiteSchema(db *sql.DB) error {
	// PRAGMA user_version 在 fresh DB 恒为 0，ver >= sqliteUserVersion 直接 no-op；
	// 极端错误（db 已 close）会通过后续 SQL 立刻暴露，无需在此防御。
	var ver int
	_ = db.QueryRow(`PRAGMA user_version`).Scan(&ver)
	if ver >= sqliteUserVersion {
		return nil
	}
	// v1：基线 schema 已由 CREATE IF NOT EXISTS 建立；后续增量 ALTER 写在此。
	_, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, sqliteUserVersion))
	return err
}

func insertAccountExec(exec interface {
	Exec(query string, args ...any) (sql.Result, error)
}, a *model.Account, sortOrder int) error {
	_, err := exec.Exec(`
INSERT INTO accounts (id,name,endpoint,public_endpoint,region,access_key,secret_key,bucket,path_style,use_ssl,created_at,updated_at,sort_order)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.Name, a.Endpoint, a.PublicEndpoint, a.Region, a.AccessKey, a.SecretKey, a.Bucket,
		sqliteBool(a.PathStyle), sqliteBool(a.UseSSL),
		a.CreatedAt.UTC().Format(time.RFC3339Nano), a.UpdatedAt.UTC().Format(time.RFC3339Nano), sortOrder,
	)
	return err
}

// insertAccount 加密 secret_key 后写入。
func (s *SQLiteStore) insertAccount(a *model.Account, sortOrder int) error {
	stored := *a
	stored.SecretKey = s.encryptSecret(a.SecretKey)
	return insertAccountExec(s.db, &stored, sortOrder)
}

func sqliteBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

// sqliteFillCommon 回填 Get/List 两条扫描路径共用的派生字段（bool 与时间），
// 保证两条路径产出的 Account 形态永远一致，避免只改一处造成的字段漂移。
func sqliteFillCommon(a *model.Account, pathStyle, useSSL int, created, updated string) {
	a.PathStyle = pathStyle != 0
	a.UseSSL = useSSL != 0
	if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
		a.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, updated); err == nil {
		a.UpdatedAt = t
	}
}

// sqliteScan 扫描一行并解密 secret_key（历史明文行原样通过）。Get/Update 路径用，
// 它们需要真实密钥；对外返回前由调用方 Sanitized 脱敏。
func (s *SQLiteStore) sqliteScan(row interface{ Scan(...any) error }) (*model.Account, error) {
	var a model.Account
	var pathStyle, useSSL int
	var created, updated string
	if err := row.Scan(
		&a.ID, &a.Name, &a.Endpoint, &a.PublicEndpoint, &a.Region,
		&a.AccessKey, &a.SecretKey, &a.Bucket, &pathStyle, &useSSL, &created, &updated,
	); err != nil {
		return nil, err
	}
	secret, err := s.decryptSecret(a.SecretKey)
	if err != nil {
		return nil, err
	}
	a.SecretKey = secret
	sqliteFillCommon(&a, pathStyle, useSSL, created, updated)
	return &a, nil
}

const sqliteAccountCols = `id,name,endpoint,public_endpoint,region,access_key,secret_key,bucket,path_style,use_ssl,created_at,updated_at`

// sqliteListCols 是 List 的专用投影：不取 secret_key 密文本身，只合成 secret_set
// 布尔（列值是否非空）。列表路径完全不碰密文，也就无需逐行 Argon2id 解密——
// 旧实现解密完立刻被 Sanitized 丢弃，20 个加密账号 ≈0.7s/次白算（R14）。
// 脱敏形态与 Get+Sanitized 完全一致：非空 → ******，空 → 空。
const sqliteListCols = `id,name,endpoint,public_endpoint,region,access_key,` +
	`(secret_key != '') AS secret_set,bucket,path_style,use_ssl,created_at,updated_at`

// listScan 扫描 List 投影行：按 secret_set 合成脱敏，不解密也不接触密文，
// 因此密文损坏 / 未配置 S3C_STORE_KEY 时 List 依然可用（解密只在 Get 路径要求）。
func listScan(row interface{ Scan(...any) error }) (*model.Account, error) {
	var a model.Account
	var secretSet, pathStyle, useSSL int
	var created, updated string
	if err := row.Scan(
		&a.ID, &a.Name, &a.Endpoint, &a.PublicEndpoint, &a.Region,
		&a.AccessKey, &secretSet, &a.Bucket, &pathStyle, &useSSL, &created, &updated,
	); err != nil {
		return nil, err
	}
	if secretSet != 0 {
		a.SecretKey = model.MaskedSecret
	}
	sqliteFillCommon(&a, pathStyle, useSSL, created, updated)
	return &a, nil
}

func (s *SQLiteStore) List() ([]*model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT ` + sqliteListCols + ` FROM accounts ORDER BY sort_order ASC`)
	if err != nil {
		// 不得伪装成「无账号」；把错误交给调用方映射 500。
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()
	out := make([]*model.Account, 0)
	for rows.Next() {
		a, err := listScan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan account row: %w", err)
		}
		// listScan 已按 secret_set 脱敏；再过一遍 Sanitized 是防御性兜底，
		// 保证即使投影列将来漏改，密钥也不会流出 List。
		out = append(out, a.Sanitized())
	}
	// 迭代中途失败（并发 Close、库文件损坏）必须上抛：把截断列表当成功返回，
	// UI 会静默丢账号（R13）。返回 nil 而非已收集的部分行，避免半份数据被当结果用。
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return out, nil
}

// Ping 探测 SQLite 连接是否可用。
func (s *SQLiteStore) Ping() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Close 不置 nil db；连接若已关闭由 db.Ping 报「database is closed」。
	return s.db.Ping()
}

func (s *SQLiteStore) Get(id string) (*model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row := s.db.QueryRow(`SELECT `+sqliteAccountCols+` FROM accounts WHERE id = ?`, id)
	a, err := s.sqliteScan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *SQLiteStore) nextSortOrder() (int, error) {
	var max sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(sort_order) FROM accounts`).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64) + 1, nil
}

func (s *SQLiteStore) Create(a *model.Account) (*model.Account, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	ord, err := s.nextSortOrder()
	if err != nil {
		return nil, err
	}
	if err := s.insertAccount(a, ord); err != nil {
		noteWriteFailure()
		return nil, err
	}
	return a.Sanitized(), nil
}

func (s *SQLiteStore) Update(id string, a *model.Account) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, err := s.getLocked(id)
	if err != nil {
		return nil, err
	}
	if a.SecretKey != "" && !model.IsMaskedSecret(a.SecretKey) {
		cur.SecretKey = a.SecretKey
	}
	cur.Name = a.Name
	cur.Endpoint = a.Endpoint
	cur.PublicEndpoint = a.PublicEndpoint
	cur.Region = a.Region
	cur.AccessKey = a.AccessKey
	cur.Bucket = a.Bucket
	cur.PathStyle = a.PathStyle
	cur.UseSSL = a.UseSSL
	cur.UpdatedAt = time.Now().UTC()
	secret := s.encryptSecret(cur.SecretKey)
	if _, err := s.db.Exec(`
UPDATE accounts SET name=?,endpoint=?,public_endpoint=?,region=?,access_key=?,secret_key=?,bucket=?,path_style=?,use_ssl=?,updated_at=?
WHERE id=?`,
		cur.Name, cur.Endpoint, cur.PublicEndpoint, cur.Region, cur.AccessKey, secret, cur.Bucket,
		sqliteBool(cur.PathStyle), sqliteBool(cur.UseSSL), cur.UpdatedAt.UTC().Format(time.RFC3339Nano), id,
	); err != nil {
		noteWriteFailure()
		return nil, fmt.Errorf("sqlite update: %w", err)
	}
	return cur.Sanitized(), nil
}

func (s *SQLiteStore) getLocked(id string) (*model.Account, error) {
	row := s.db.QueryRow(`SELECT `+sqliteAccountCols+` FROM accounts WHERE id = ?`, id)
	a, err := s.sqliteScan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *SQLiteStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		noteWriteFailure()
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Close 关闭 SQLite 连接（优雅关停时由 main 调用）。
func (s *SQLiteStore) Close() error {
	// Close 不置 nil db；重复 Close 会被 sql 驱动返回错误但无害。
	return s.db.Close()
}
