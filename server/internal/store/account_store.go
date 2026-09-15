package store

import (
	"errors"

	"github.com/weilai1949/s3clinet/server/internal/model"
)

// ErrNotFound 表示指定 id 的账号不存在。
var ErrNotFound = errors.New("account not found")

// AccountStore 账号持久化抽象（JSON 文件或 SQLite）。
type AccountStore interface {
	List() ([]*model.Account, error)
	Get(id string) (*model.Account, error)
	Create(a *model.Account) (*model.Account, error)
	Update(id string, a *model.Account) (*model.Account, error)
	Delete(id string) error
	Ping() error
	Close() error
}
