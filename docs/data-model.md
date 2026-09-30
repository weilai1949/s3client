# 数据模型与存储格式

> **一句话**：账号数据只有一个模型（`model.Account`）、三种落盘方式（`json` / `sqlite` / `encrypted`）、
> 一种当前加密信封（`S3C3`）。
>
> **本文件不是所有细节的 SSOT**。它把散落在四处的内容集中成一张地图，并明确
> **冲突时谁说了算**——避免制造第五份会漂移的副本。

## 0. SSOT 分工（冲突时的裁决顺序）

| 关注点 | 权威来源 | 本文件的角色 |
|---|---|---|
| 账号字段集 / JSON 类型 / `required` | [`api/accounts.schema.json`](api/accounts.schema.json)（由 `TestAccountStoreSchemaMatchesModel` 反射比对 `model.Account`，漂移即红灯） | 汇总 + 解释 |
| 驱动选择 / 原子写 / fail-closed 的**决策依据** | [`decisions/0006-store-drivers-atomic-write.md`](decisions/0006-store-drivers-atomic-write.md)（ADR-0006） | 汇总 + 指向 |
| `S3C2` → `S3C3` 的**兼容承诺**与升级 / 回滚口径 | [`compatibility.md`](compatibility.md) §4 | 汇总 + 指向 |
| 分层与模块边界 | [`architecture.md`](architecture.md) | 不重复 |
| **字节级格式**（信封偏移、KDF 参数、权限位） | 生产代码：`apps/server/internal/store/{crypto.go,open.go,store.go,sqlite.go,filestore.go}` 与 `apps/server/internal/atomicfile` | 描述；与本文件不一致时**以代码为准** |
| 环境变量语义（`S3C_STORE_DRIVER` / `S3C_STORE_KEY` / `S3C_DATA_DIR`） | [`CONFIGURATION.md`](CONFIGURATION.md) | 只讲与数据结构相关的部分 |

> 口径：**代码是字节事实，schema 是字段契约，ADR 是决策记录，本文件是地图。**
> 任何一处改动都要按 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4「文档同步门禁」同步本文件，
> 且本文件与代码不一致时按上表回退到权威来源。

---

## 1. 账号模型（`model.Account`）

`apps/server/internal/model/account.go` 定义 12 个字段，**全部没有 `omitempty`**——
序列化写出的文件里字段恒存在（这也是 `accounts.schema.json` 把 12 个字段全标 `required` 的原因）。

| JSON 字段 | Go 类型 | 说明 |
|---|---|---|
| `id` | `string` | 账号 UUID；`Create` 未提供时由服务端生成（`uuid.NewString()`） |
| `name` | `string` | 展示名 |
| `endpoint` | `string` | 服务端访问地址，例如 `http://minio:9000` |
| `publicEndpoint` | `string` | 浏览器直传 / 下载用的公网地址；留空表示与 `endpoint` 相同 |
| `region` | `string` | 例如 `us-east-1` |
| `accessKey` | `string` | Access Key ID |
| `secretKey` | `string` | Secret Key；**永不进 HTTP 响应**，脱敏占位为 `******`（`model.MaskedSecret`） |
| `bucket` | `string` | 默认桶 |
| `pathStyle` | `bool` | 是否强制 path-style（MinIO 等第三方通常需要） |
| `useSSL` | `bool` | 是否启用 TLS |
| `createdAt` | `time.Time` | 创建时间，服务端写入 UTC |
| `updatedAt` | `time.Time` | 更新时间，`Update` 时刷新为 UTC |

**时间格式**：Go `time.Time` 默认 JSON 序列化为 RFC 3339（含纳秒与 `Z`），
`accounts.schema.json` 对应声明为 `format: date-time`。

**写入语义**（`fileStore`）：

- `Create` 复制入参后存库，调用方后续改动入参不影响已落库账号；
- `Update` 不允许改 `id` 与 `createdAt`；`secretKey` 为空或等于脱敏占位（`******`）时**不覆盖**已有密钥；
- `Delete` 按原位置回滚创建顺序；
- **落盘失败一律回滚内存状态**——不允许出现「磁盘仍在、内存已删」的漂移；
- `List` 返回按创建顺序的**脱敏副本**（`Sanitized()`）；`Get` 返回含密钥副本，仅供服务端内部使用。

## 2. 对外视图（`model.AccountView`）

HTTP 响应**不使用** `Account`，而是 `AccountView`：结构与 `Account` 相同，但
`secretKey` 被替换为布尔 `secretSet`——客户端拿不到任何密钥（连占位值都拿不到）。

| 视图字段 | 说明 |
|---|---|
| `secretSet` | `true` = 当前持有密钥（真实密钥或脱敏占位都算）；空串才算未设置 |

**为什么不留占位值**：早期设计用 `******` 回传，存在「客户端把占位值当真实密钥回写」的风险
（见 `model.IsMaskedSecret` 的防御）。视图化后该风险在类型层面消失。

## 3. 存储驱动

`store.Open(dataDir, driver, storeKey)` 是唯一入口。驱动名**先去空白 + 转小写**再分发，
大小写 / 空白变体与小写行为一致；**未知驱动直接报错**（不静默回退 `json`——静默回退会让
`S3C_STORE_DRIVER=sqlite` 的部署悄悄明文落盘）。空串 = 未显式指定，等价 `json`。

| `S3C_STORE_DRIVER` | 落盘路径（相对 `S3C_DATA_DIR`） | 需要 key | 当前写入格式 | 可读格式 | 严格性 |
|---|---|---|---|---|---|
| `json`（默认） | `accounts.json` | 否 | 无 key：明文 JSON；有 key：`S3C3` 信封（**每次写盘换新盐**） | 明文 JSON / `S3C2` / `S3C3` | permissive |
| `sqlite` | `accounts.db` | 否 | 有 key：`secret_key` 列逐值 `S3C3` 信封（**每条随机盐**）；无 key：明文列 | 明文列 / `S3C2` / `S3C3` | 逐值判别 |
| `encrypted` | `accounts.json.enc` | **是** | 恒 `S3C3` 信封（文件盐在 `missing` / `decode` 时随机一次并复用） | `S3C2` / `S3C3` | strict |

**StoreKey 契约**：三个分支**一律优先使用显式入参 `storeKey`**；只有「`json` 分支 + 入参为空」
才回退环境变量 `S3C_STORE_KEY`（历史 `New` 语义）。`sqlite` / `encrypted` 分支不回退——
`encrypted` 缺 key 直接报错。

**接口**（`store.AccountStore`）：`List` / `Get` / `Create` / `Update` / `Delete` / `Ping` / `Close`，
不存在时统一返回 `store.ErrNotFound`。

> `Ping()` 的真实性：`json` / `encrypted` 驱动**恒返回 `nil`**（文件型存储没有连接可探），
> 只有 `sqlite` 会真的查库。这条差异直接影响 `/api/health` 与 `s3c_store_up` 的解读，
> 口径见 [`OPERATIONS.md`](OPERATIONS.md) §4。

## 4. 落盘格式

### 4.1 明文 `accounts.json`

`json.MarshalIndent(list, "", "  ")` 产出的**账号数组**（创建顺序），缩进 2 空格：

```json
[
  {
    "id": "0b0f…",
    "name": "minio",
    "endpoint": "http://minio:9000",
    "publicEndpoint": "https://s3.example.com",
    "region": "us-east-1",
    "accessKey": "ak",
    "secretKey": "sk",
    "bucket": "b",
    "pathStyle": true,
    "useSSL": false,
    "createdAt": "2026-09-01T00:00:00Z",
    "updatedAt": "2026-09-01T00:00:00Z"
  }
]
```

> **明文驱动的 `secretKey` 就是明文**。需要静态加密时必须配置 `S3C_STORE_KEY`
> 或改用 `encrypted` 驱动，见 [`threat-model.md`](threat-model.md)。

### 4.2 加密信封 `S3C2` / `S3C3`

两种信封都以 4 字节 ASCII magic 开头；`isEncryptedBlob` 认这两种 magic。

**`S3C3`（当前唯一写入格式）**——总头长 **29 字节**：

| 偏移 | 长度 | 内容 |
|---|---|---|
| 0 | 4 | magic `S3C3` |
| 4 | 4 | Argon2id `time` cost（`uint32`，大端） |
| 8 | 4 | Argon2id `memory`（KiB，`uint32`，大端） |
| 12 | 1 | Argon2id `threads`（`uint8`） |
| 13 | 16 | `salt` |
| 29 | 12 | AES-GCM `nonce` |
| 41 | n | `ciphertext`（含 16 字节 GCM tag） |

**`S3C2`（只读，兼容既有加密库）**：

| 偏移 | 长度 | 内容 |
|---|---|---|
| 0 | 4 | magic `S3C2` |
| 4 | 16 | `salt` |
| 20 | 12 | AES-GCM `nonce` |
| 32 | n | `ciphertext`（含 16 字节 GCM tag） |

**密钥派生**：`Argon2id(password, salt, time, memory, threads) → 32 字节`（`keyLen = 32`），
用于 AES-256-GCM。`S3C2` 不携带参数，按硬编码历史值 `t=1, m=64 MiB, p=4` 派生；
`S3C3` 参数随文件头保存，**当前写入值 `t=2, m=64 MiB, p=4`**（OWASP 建议 Argon2id time cost ≥ 2）。

**文件头参数会校验**：`S3C3` 的参数来自文件内容、可被篡改，派生前必须通过 `kdfParamsValid`——
`time ∈ [1,10]`、`memory ∈ [1,524288]` KiB（上界 512 MiB）、`threads ∈ [1,16]`。
零值与越界值合并为同一罚则 `invalid KDF params`，避免攻击者从文案差异推断上界。
**为什么必须校验**：把 `memory` 改成 4 GiB 就能让一次普通读取吃掉数 GiB 内存（读放大 DoS）。

### 4.3 `accounts.db`（SQLite）

纯 Go `modernc.org/sqlite` 驱动，无 CGO。表结构（`sqliteSchema`）：

```sql
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
```

- DSN pragma：`foreign_keys(1)`、`journal_mode(WAL)`、`busy_timeout(5000)`；
- schema 版本用 `PRAGMA user_version` 管理；
- `List` 走专用投影，**不取 `secret_key` 密文本身**，只合成 `(secret_key != '') AS secret_set`；
- 历史明文行原样可读，**写回时自动加密**（逐值信封）。

> 与文件驱动不同，SQLite 的「密文」是**列值**而不是整个文件，所以同一张表里
> 可能同时存在加密行与历史明文行——这是有意的向后兼容，不是不一致。

## 5. 原子写与崩溃安全

所有文件落盘都走 `internal/atomicfile.WriteFile`（`store` 与 `service` 共用的零依赖叶子包）。
顺序**每一步都必要**：

1. `OpenFile(path + ".tmp", O_WRONLY|O_CREATE|O_EXCL, 0600)`（先 `os.Remove` 残骸；`O_EXCL` 防抢占与符号链接重定向）；
2. `Write` 全量数据（短写即失败）；
3. `Chmod(tmp, 0600)`——放在 `rename` **之前**，权限位随 `fsync` 一起落盘，
   避免「先 rename 再收紧」中间窗口权限取决于 umask（可能 0644）；
4. `f.Sync()`——数据落盘；
5. `f.Close()`；
6. `os.Rename(tmp, path)`——目录项原子替换；
7. `fsync` 父目录——**目录项**落盘。

`rename` 之前的任何错误都清理 `.tmp`；`rename` 之后只上抛（目标已是新内容，不能回滚成旧内容）。
**为什么 rename 不够**：`rename` 只改目录项，不保证数据 / 新文件名进入磁盘；断电可能得到
「旧内容 + 新文件名」或目录项丢失。

## 6. 权限位

| 对象 | 权限 | 实现 |
|---|---|---|
| 数据目录 | `0700` | `ensureDataDirPerm`（**尽力而为**：只读挂载、非属主等 `chmod` 失败不影响启动） |
| `accounts.json` / `accounts.json.enc` | `0600` | `atomicfile` 在 `rename` 前 chmod |
| `accounts.db` 与两侧车 `-wal` / `-shm` | `0600` | `chmodSQLitePerms`（侧车含明文页，必须与主库一起收紧） |

> `os.MkdirAll` 的 mode 只在**新建**时生效：Docker volume / systemd `StateDirectory`
> 预建的 `0755` 目录此后不会自动变 `0700`，故需要主动 `chmod`。

## 7. fail-closed 行为（不静默降级）

| 情形 | 行为 |
|---|---|
| 未知 `S3C_STORE_DRIVER` | `Open` 报错 `unknown store driver %q (want json\|sqlite\|encrypted)` |
| `encrypted` 驱动缺 `S3C_STORE_KEY` | 构造即报错 `S3C_STORE_KEY is required for encrypted store` |
| `S3C3` KDF 参数为零 / 越界 | 派生前报错 `S3C3 file has invalid KDF params` |
| strict 驱动遇到非 `S3C2` / `S3C3` magic | 报错 `encrypted account file magic %q is not S3C2/S3C3`，**不回退明文解析** |
| SQLite 列是密文但未配置 `S3C_STORE_KEY` | 报错 `encrypted secret_key found but S3C_STORE_KEY is not set` |
| 落盘失败 | `Create` / `Update` / `Delete` 回滚内存状态后上抛 |

## 8. 单写者约束

`store.AcquireDataDirLock(dataDir)` 对数据目录加**单写者锁**（`flock`）。store 是文件型的、
`JobRegistry` 在内存中、单 token 模型没有租约或选主——两个副本共享同一数据卷会静默互相覆盖
写入并重复执行迁移任务。锁由内核在进程退出（含 `panic` / `SIGKILL`）时释放，不留需手工清理的
陈旧锁文件；非 Unix 平台当前降级为 no-op（**单副本约束仍由部署方式保证**）。决策依据见
[`decisions/0011-single-instance-no-ha.md`](decisions/0011-single-instance-no-ha.md)。

## 9. 升级、回滚与密钥轮换

- **`S3C2` → `S3C3` 不需要手工迁移**：旧库照常读取，新写入一律 `S3C3`，首次写入后旧库自然转 `S3C3`；
- **回滚注意**：`S3C2` 由新版本写入后不会被改回，回滚到只认旧格式的历史版本前请先备份 `/data`；
- **换 `S3C_STORE_KEY` 的后果**：已加密的数据（`encrypted` 驱动整文件、`json` 信封、
  SQLite 的密文列）**无法再解密**，且仓库**没有密钥轮换 / 导出工具**、`AccountView` 也不回传
  `secretKey`——**等价于重新录入全部账号**。手工轮换步骤见 [`OPERATIONS.md`](OPERATIONS.md)。

## 10. 相关文档

| 文档 | 用途 |
|---|---|
| [`api/accounts.schema.json`](api/accounts.schema.json) | 字段级机器可读契约（校验 / 生成 `accounts.json`） |
| [`decisions/0006-store-drivers-atomic-write.md`](decisions/0006-store-drivers-atomic-write.md) | 驱动与原子写的决策记录 |
| [`decisions/0011-single-instance-no-ha.md`](decisions/0011-single-instance-no-ha.md) | 单副本决策 |
| [`compatibility.md`](compatibility.md) | 存储格式兼容承诺与弃用政策 |
| [`CONFIGURATION.md`](CONFIGURATION.md) | `S3C_DATA_DIR` / `S3C_STORE_DRIVER` / `S3C_STORE_KEY` |
| [`threat-model.md`](threat-model.md) | 明文落盘、权限位与 DoS 边界的威胁建模 |
| [`architecture.md`](architecture.md) | `store → model` 在整体分层中的位置 |
