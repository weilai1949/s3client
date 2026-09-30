# 性能基线与基准测试

> 本文件记录后端**本地热路径**的基准与解读，供改动时判断「有没有把什么弄慢」。
> 它不是性能承诺（无 SLO 数字），也**不覆盖**端到端吞吐——那类结论需要真实 RustFS 与真实网络，
> 属 [`DEVELOPMENT.md`](DEVELOPMENT.md) §2 的 E2E 范畴。
> 运维侧的容量与水位见 [`OPERATIONS.md`](OPERATIONS.md) §10。

## 1. 怎么跑

```bash
cd apps/server
# 全部基准（约 2 分钟；store 的加密写入单次 ~30ms，占大头）
go test ./internal/s3wrap/ ./internal/service/ ./internal/store/ -run '^$' -bench 'Benchmark' -benchmem

# 只跑某一组
go test ./internal/store/ -run '^$' -bench 'BenchmarkStore' -benchmem

# 小样本对照（用来看「随规模变化」的趋势，见 §3.3）
go test ./internal/store/ -run '^$' -bench 'BenchmarkStoreCreate/json$' -benchtime=200x
```

**回归门禁 + 原始基准一条命令**（与 CI 的 [`.github/workflows/perf.yml`](../.github/workflows/perf.yml) 同命令）：

```bash
make bench                                                   # 预算门禁 + 原始数字（见 §4.1）
cd apps/server && go test . -run 'TestBench' -count=1 -v     # 只跑预算门禁
```

基准全部是**纯本地计算或本地文件 I/O**，不依赖网络与对端，因此结果可跨机器比较趋势（绝对值不可比）。

## 2. 基线（本仓库实测）

采集环境：`go1.26.6 linux/amd64`，13th Gen Intel(R) Core(TM) i5-13400F，`GOMAXPROCS=16`，
`-benchtime=1s`。**绝对值仅供趋势判断**，换机器请按同机前后对比。

### 2.1 `s3wrap` —— 端点归一化与签名

| 基准 | ns/op | B/op | allocs/op |
|---|---|---|---|
| `NormalizeEndpoint/s3.example.com/` | 72.6 | 24 | 1 |
| `NormalizeEndpoint/http://127.0.0.1:9000/` | 77.3 | 24 | 1 |
| `NormalizeEndpoint/http://127.0.0.1:9000` | 77.8 | 24 | 1 |
| `NormalizeEndpoint/127.0.0.1:9000` | 79.6 | 24 | 1 |
| `NormalizeEndpoint/http://minio.internal:9000` | 93.3 | 32 | 1 |
| `NormalizeEndpoint/HTTPS://S3.Example.COM` | 144.0 | 48 | 3 |
| `PresignPut` | **33 268** | 27 497 | **362** |

### 2.2 `service` —— key 映射内核

| 基准 | ns/op | B/op | allocs/op |
|---|---|---|---|
| `RelKey/无前缀` | 11.8 | 0 | **0** |
| `RelKey/段边界不命中` | 14.3 | 0 | **0** |
| `RelKey/段边界命中` | 15.0 | 0 | **0** |
| `RelKey/深前缀` | 14.6 | 0 | **0** |
| `RelKey/含中文与空格` | 42.7 | 48 | 1 |
| `BaseKey/段边界命中` | 31.1 | 16 | 1 |
| `BaseKey/深前缀` | 39.0 | 16 | 1 |
| `BaseKey/段边界不命中` | 41.2 | 16 | 1 |
| `BaseKey/无前缀` | 48.5 | 24 | 1 |
| `BaseKey/含中文与空格` | 52.1 | 24 | 1 |

### 2.3 `store` —— 账号读写

| 基准 | ns/op | B/op | allocs/op |
|---|---|---|---|
| `StoreList/json`（32 个账号） | 2 977 | 6 400 | 33 |
| `StoreList/json+key` | 2 463 | 6 400 | 33 |
| `StoreList/encrypted` | 2 581 | 6 400 | 33 |
| `StoreCreate/json` | 11 210 303 | 6 357 708 | 7 413 |
| `StoreCreate/json+key` | 29 609 321 | 67 193 959 | 150 |
| `StoreCreate/encrypted` | 29 638 345 | 67 199 428 | 153 |

## 3. 三条需要知道的结论

### 3.1 加密写入单次约 30 ms、瞬时分配约 64 MiB —— 这是**有意设计**

`json+key` 与 `encrypted` 的写入都在 `storeCodec` 里做一次 Argon2id 派生，参数为
`t=2, m=64 MiB, p=4`（`argonTimeV3` / `argonMemoryV3` / `argonThreadsV3`，见 `internal/store/crypto.go`）。
Argon2id 是**内存硬**函数，慢与吃内存正是它抗暴力破解的方式，不该被「优化」掉。

需要留意的不是单次 30 ms，而是**并发写时的峰值内存**：每次写入瞬时占用约 64 MiB，
N 个并发写理论上叠加 N × 64 MiB。当前写路径受 store 写锁串行化，因此实际不会叠加；
但若将来把写路径改成并发，必须把这一项纳入内存预算（容器内存上限见
[`OPERATIONS.md`](OPERATIONS.md) §10）。

### 3.2 账号写入是 **O(n)**：每次写都重写整个文件

`fileStore.persistLocked` 把**全部账号**序列化后原子写盘（temp → fsync → rename → fsync 父目录），
而不是追加或按条更新。这一点在基准里能直接看到：同一基准换样本量，单次成本差两个数量级——

| `BenchmarkStoreCreate/json` | ns/op | B/op |
|---|---|---|
| `-benchtime=200x`（文件里账号数少） | 335 543 | 140 077 |
| `-benchtime=1s`（迭代数千次，文件持续增长） | 11 210 303 | 6 357 708 |

**结论与取舍**：账号是「配置」而非「业务数据」，量级通常是几个到几十个，O(n) 完全可接受，
换来的是**实现简单 + 落盘原子**（不存在半写状态）。真正的约束在
[`OPERATIONS.md`](OPERATIONS.md) §10：账号规模到数百时应改用 `sqlite` 驱动。

### 3.3 `PresignPut` 约 33 µs / 362 allocs —— 每次「直传 / 分享链接」一次

SigV4 签名要在本地做 canonical request、HMAC 链与 URI 编码，362 次分配是主要成本。
它发生在**用户点击**的路径上（生成上传直传 URL、复制 1 小时分享链接），33 µs 无感；
但**不要**把它放进按对象循环的路径（例如为一次列出的 1000 个对象各签一次名，
就是 33 ms 且 36 万次分配）。

### 3.4 端点归一化与 key 映射都很便宜，且几乎零分配

`NormalizeEndpoint` 约 73–144 ns、1–3 次分配（`HTTPS://S3.Example.COM` 更贵是因为要处理
大小写混合 scheme）；`RelKey` 在常见路径上**零分配**——它是逐 key 调用的内核，
迁移上万 key 也只有约 0.15 ms 的总开销。`BaseKey` 的 1 次分配来自 `path.Base` 的切片语义。

## 4. 改动时怎么用这份基线

1. **改到上述任一路径**，跑对应基准的前后对照（同机、同 `-benchtime`）。
2. 数字明显变差时，先判断是不是**有意的取舍**：例如给写路径加一次校验会让
   `StoreCreate` 变慢，这可能是对的；但**无理由**地把 `RelKey` 从 0 分配变成 2 次分配，
   就是需要解释的回归。
3. 新增基准时：放同包 `bench_test.go`，用 `b.ReportAllocs()`，并在文件头写清
   **为什么挑它**（哪些是热路径、为什么它值得被钉住）。

### 4.1 性能回归门禁（`apps/server/bench_budget_test.go`）

自 2026-09-30 起，「有没有把什么弄慢」不再只靠人记得跑基准：`bench_budget_test.go` 用
`testing.Benchmark()` 直接在单测里跑 §2.1、§2.3 的三条热路径，断言分三类——

| 断言 | 实测基线（2026-09-30，本仓库开发机） | 余量 | 为什么这样定 |
|---|---|---|---|
| **分配数 / 分配字节**（主力） | `PresignPut` 362 allocs/op、27.5 KB/op；`encrypted` 写入 212 allocs/op、67.3 MB/op | ~3 倍 | 固定 Go 版本 + 固定输入下**确定性**、跨机器稳定；多一次拷贝 / 一层编码即触发 |
| **单次耗时**（兜底） | `PresignPut` ~26 µs；`encrypted` 写入 ~29 ms | ~70–80 倍 | 耗时对 runner / cgroup / fsync 极敏感，只用于拦算法级崩塌（O(n)→O(n²)、Argon2id 参数放大一个数量级） |
| **账号持久化 O(n)**（结构） | 4 个账号 vs 256 个账号的单次写入**最小值**比 | 比值 ≤ 64，另有 10 ms 绝对地板 + 250 ms 兜底 | 线性实现实测约 25 倍；二次实现会到三个数量级（§3.2） |

`.github/workflows/perf.yml` 在 PR / push（相关路径）上跑门禁，并**每周定时**跑原始基准、
把数字作为 artifact 留存（保留 30 天），供人工比对「没有 PR 的缓慢退化」。本地同款：`make bench`。

### 4.2 flakiness 口径（诚实说明）

- **不承诺**捕捉 10 倍以内的**耗时**回归：那要求把阈值压到与 runner 噪声同量级，只会制造假失败。
  这一档交给 §4.1 的定时基准历史数字与人工判断。
- **分配类断言是确定性的**，可以也应该在 PR 上红灯；本仓库把「无理由多一次分配」视为回归
  （与根 `AGENTS.md` 的「死代码零容忍」同一取向）。
- **比值断言**取 5 次采样的**最小值**并设 100 ms 绝对地板，避免小库耗时趋零时被调度噪声放大。
- 若红灯确系环境异常（共享 runner 被超卖），**先复跑确认**；不允许用放宽阈值来「修」红灯——
  那等于把门禁退化成装饰。

### 4.3 读到回归时怎么办

1. 先分清是**耗时**还是**分配**超标：分配超标通常直接对应代码改动（新增拷贝 / 编码 / 接口层）；
2. 用 `git log -p` 对照改动，同机跑 `-benchtime` 前后对照确认；
3. 判断是有意取舍还是回归：**有意取舍** → 连同本节表格与 `bench_budget_test.go` 的预算常量一起
   在同一个 PR 更新；**回归** → 修代码；
4. 若是**工具链**导致的系统性变慢（如 Go 版本升级），连同 §2 的「采集环境」一起更新。

## 5. 相关文档

- 测试分层与必验门禁：[`DEVELOPMENT.md`](DEVELOPMENT.md) §2 / §3
- 容量水位与单实例约束：[`OPERATIONS.md`](OPERATIONS.md) §10
- 存储驱动与 O(n) 取舍的运维含义：[`CONFIGURATION.md`](CONFIGURATION.md)（`S3C_STORE_DRIVER`）
- 加密参数与文件格式：[`threat-model.md`](threat-model.md) · [`compatibility.md`](compatibility.md)
