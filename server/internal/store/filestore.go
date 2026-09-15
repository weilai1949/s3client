package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/weilai1949/s3clinet/server/internal/model"
)

// fileCodec 是 fileStore 的落盘编解码策略：把磁盘字节还原为账号列表，
// 以及把内存快照序列化为磁盘字节。各驱动只差在格式（明文 json / S3C2 加密），
// CRUD 语义完全由 fileStore 统一实现。
type fileCodec interface {
	// missing 在文件不存在或为空时调用；加密驱动借此生成新盐，明文驱动无操作。
	missing() error
	// readError 包装「文件存在但读取失败」的错误，保留各驱动既有文案。
	readError(err error) error
	// decode 解析磁盘字节为账号列表。
	decode(data []byte) ([]*model.Account, error)
	// encode 把有序账号快照编码为磁盘字节。编码视为不可失败：
	// 明文 JSON / AES-256-GCM 对合法 32 字节密钥均不会失败，crypto/rand 在 Go 1.24+ 也不返回错误。
	encode(list []*model.Account) []byte
}

// fileStore 是 json / encrypted 两个文件驱动共享的内存状态与 CRUD 实现：
// accounts 映射 + 创建顺序切片，读写锁保护，落盘策略由 codec 注入。
// 落盘失败时 Create/Update/Delete 一律回滚内存状态，保持与磁盘一致。
type fileStore struct {
	mu       sync.RWMutex
	path     string
	accounts map[string]*model.Account
	order    []string // 保持创建顺序，便于列表展示稳定
	codec    fileCodec
}

// load 从磁盘加载：文件缺失或为空时交给 codec.missing；否则交给 codec.decode。
func (f *fileStore) load() error {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return f.codec.missing()
		}
		return f.codec.readError(err)
	}
	if len(data) == 0 {
		return f.codec.missing()
	}
	list, err := f.codec.decode(data)
	if err != nil {
		return err
	}
	f.accounts = make(map[string]*model.Account, len(list))
	f.order = f.order[:0]
	for _, a := range list {
		f.accounts[a.ID] = a
		f.order = append(f.order, a.ID)
	}
	return nil
}

// List 返回全部账号（脱敏副本），按创建顺序。
func (f *fileStore) List() ([]*model.Account, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]*model.Account, 0, len(f.order))
	for _, id := range f.order {
		if a, ok := f.accounts[id]; ok {
			out = append(out, a.Sanitized())
		}
	}
	return out, nil
}

// Get 返回指定账号副本（含密钥，供服务端内部使用）。
func (f *fileStore) Get(id string) (*model.Account, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	a, ok := f.accounts[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *a
	return &cp, nil
}

// Create 新增账号并持久化；存副本，调用方后续改动入参不影响已落库账号。
func (f *fileStore) Create(a *model.Account) (*model.Account, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.accounts[a.ID]; exists {
		return nil, fmt.Errorf("account %s already exists", a.ID)
	}
	cp := *a
	f.accounts[a.ID] = &cp
	f.order = append(f.order, a.ID)
	if err := f.persistLocked(); err != nil {
		// 写盘失败：回滚内存状态保持一致。
		delete(f.accounts, a.ID)
		f.order = f.order[:len(f.order)-1]
		return nil, err
	}
	return a.Sanitized(), nil
}

// Update 更新账号；id 不允许变更。
func (f *fileStore) Update(id string, a *model.Account) (*model.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.accounts[id]
	if !ok {
		return nil, ErrNotFound
	}
	// 持久化失败时回滚内存状态，保持与磁盘一致。
	prev := *cur
	// 保留 id 与创建时间；未提供的 SecretKey 不覆盖（避免脱敏值回写）
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
	if err := f.persistLocked(); err != nil {
		*cur = prev
		return nil, err
	}
	return cur.Sanitized(), nil
}

// Delete 删除账号并持久化；写盘失败时按原位置回滚创建顺序。
func (f *fileStore) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[id]
	if !ok {
		return ErrNotFound
	}
	idx := -1
	for i, oid := range f.order {
		if oid == id {
			idx = i
			break
		}
	}
	delete(f.accounts, id)
	if idx >= 0 {
		f.order = append(f.order[:idx], f.order[idx+1:]...)
	}
	if err := f.persistLocked(); err != nil {
		// 写盘失败：回滚内存状态，避免「磁盘仍在、内存已删」的漂移。
		f.accounts[id] = a
		if idx >= 0 {
			f.order = append(f.order[:idx], append([]string{id}, f.order[idx:]...)...)
		}
		return err
	}
	return nil
}

// Close 释放资源；文件驱动无额外资源需释放。
func (f *fileStore) Close() error { return nil }

// Ping 探测存储可用性（文件驱动始终视为可用）。
func (f *fileStore) Ping() error { return nil }

// snapshotLocked 按创建顺序导出账号快照；调用方必须已持有锁。
func (f *fileStore) snapshotLocked() []*model.Account {
	list := make([]*model.Account, 0, len(f.order))
	for _, id := range f.order {
		if a, ok := f.accounts[id]; ok {
			list = append(list, a)
		}
	}
	return list
}

// persistLocked 假定调用方已持有写锁，把当前快照交给 codec 编码后原子写盘。
func (f *fileStore) persistLocked() error {
	return atomicWriteFile(f.path, f.codec.encode(f.snapshotLocked()))
}

// marshalAccounts 把账号列表序列化为缩进 JSON（两个驱动的共同中间表示）。
// model.Account 字段全部为基本类型/时间，MarshalIndent 不会失败。
func marshalAccounts(list []*model.Account) []byte {
	data, _ := json.MarshalIndent(list, "", "  ")
	return data
}

// unmarshalAccounts 解析账号 JSON，错误文案在两个驱动间保持一致。
func unmarshalAccounts(data []byte) ([]*model.Account, error) {
	var list []*model.Account
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse account file: %w", err)
	}
	return list, nil
}
