# ADR-006：账号存储三驱动（json / sqlite / encrypted）+ 原子写 + 单写者锁

## Status

Accepted

## Date

2026-09-18（回溯记录；`filestore.go` / `open.go` 于 2026-09-18 首次引入；flock 单写者锁
2026-09-19 落地 `store/lock.go`；原子写 2026-09-28 收敛进 `internal/atomicfile`）

## Context

部署形态多样：本地开发希望文件可直接查看、容器生产希望无明文密钥、单机自托管没有外部
数据库。与 [ADR-002](0002-store-fail-closed.md) 的关系：ADR-002 决定「存储不可用则硬失败」
是**策略层**决策；本 ADR 是**存储实现层**决策——用哪几种存储形态、怎么保证落盘原子性与
单写者，使「硬失败」策略在三种形态上有一致语义。

现状（2026-09-30 回读源码核实，路径与常量逐条比对）：

- [`apps/server/internal/store/open.go`](../../apps/server/internal/store/open.go)：
  `Open(dataDir, driver, storeKey)` 按 `S3C_STORE_DRIVER` 分派 `json`（默认）/ `sqlite` /
  `encrypted`；驱动名去空白 + 小写后分发，**未知值拒绝启动**（历史：静默落回 `json` 会让
  `S3C_STORE_DRIVER=sqlite` 的部署悄悄明文落盘，绕过明文闸，R4）。
- `json`：明文 `accounts.json`（0600）；`storeKey` 非空时 S3C3 加密。
  `sqlite`：`modernc.org/sqlite`（**纯 Go 无 CGO**，`sqlite.go` 注释）；`storeKey` 非空时
  `secret_key` 列 AES-256-GCM 加密，读取按魔数判别兼容历史明文行。
  `encrypted`：整文件 AES-256-GCM（`accounts.json.enc`，需 `storeKey`）。
- [`apps/server/internal/store/crypto.go`](../../apps/server/internal/store/crypto.go)：加密信封
  `S3C2` 只读兼容、新写入一律 `S3C3`（magic + KDF 参数头 + salt + nonce|ciphertext）；KDF 为
  Argon2id，`S3C3` 当前参数 time=2、memory=64MiB（threads=4 见
  [`PERFORMANCE.md`](../PERFORMANCE.md) 实测记录）；参数随文件头保存，支持将来调参。
- [`apps/server/internal/atomicfile/atomicfile.go`](../../apps/server/internal/atomicfile/atomicfile.go)：
  临时文件 `O_EXCL` + `chmod 0600` + `fsync` + `rename` + 父目录 `fsync`；store 与 service
  任务清单共用（review R11 收敛两份复制实现，独立成零依赖叶子包避免分层倒置）。
- [`apps/server/internal/store/lock.go`](../../apps/server/internal/store/lock.go)：
  `AcquireDataDirLock` 对 `S3C_DATA_DIR` 加 `flock` **单写者锁**，第二实例立即失败；锁由内核在
  进程退出（含 panic / SIGKILL）时释放；非 unix 平台为 no-op（`lock_other.go`）。
- [`apps/server/internal/store/filestore.go`](../../apps/server/internal/store/filestore.go)：
  落盘失败时 Create / Update / Delete 一律**回滚内存状态**，保持内存与磁盘一致。

## Decision

**账号存储提供 `json` / `sqlite` / `encrypted` 三个驱动，统一入口 `store.Open(dataDir,
driver, storeKey)`；所有落盘走 `atomicfile` 原子写；`S3C_DATA_DIR` 加 flock 单写者锁。**
未知驱动拒绝启动；加密从 S3C2 只读兼容演进到 S3C3（参数入头、可调参）。

## Alternatives Considered

### 只保留一个 json 驱动
- Pros：实现最简。
- Cons：无法兼顾「本地易读 / 开发便捷」与「生产无明文」——`sqlite` 提供列级加密与关系查询、
  `encrypted` 提供整文件加密；三驱动是部署形态差异下的最小集合。
- 被拒：生产加固要求（compose 强制 `encrypted`）与开发便利要求并存。

### 外部数据库（如 PostgreSQL）
- Pros：并发 / 复制 / 备份生态成熟。
- Cons：引入外部依赖与运维负担；单实例自托管定位下用不上（见 [ADR-011](0011-single-instance-no-ha.md)）。
- （「当时是否认真评估过外部 DB」未在仓库文档中找到记录，此对比为回溯补记，**未验证**。）

### 直接覆盖写原文件（无原子写）
- Pros：实现最简。
- Cons：断电 / 崩溃可能得到半截文件——账号库损坏（`atomicfile.go` 注释 R11：rename 只改目录项，
  不保证数据落盘）。
- 已被推翻：`atomicfile` 为唯一落盘路径。

### 不加单写者锁，只靠部署纪律
- Pros：少一处系统调用。
- Cons：两个实例共享数据卷会**静默互相覆盖写入、重复执行迁移任务**（`lock.go` 注释，依据
  [`ROADMAP.md`](../ROADMAP.md) §5.1 R4）。
- 已被推翻：flock + 启动即失败。

## Consequences

- 配置面新增 `S3C_STORE_DRIVER` / `S3C_STORE_KEY` / `S3C_ALLOW_PLAINTEXT_STORE`，SSOT 在
  [`CONFIGURATION.md`](../CONFIGURATION.md)。
- 加密换钥不可行：`S3C_STORE_KEY` 丢失 = 账号库不可解密（[`OPERATIONS.md`](../OPERATIONS.md) §6.2
  明确「两者缺一不可」）。
- 单写者锁把「多副本」从部署选项中剔除，单实例约束见 [ADR-011](0011-single-instance-no-ha.md)。
