package store

// open_storekey_test.go —— KNOWN_ISSUES #64：`store.Open` 的 json 分支是否丢弃入参
// storeKey。
//
// 修复前的形态：`case "", "json": return New(path)`，`New` 回头读环境变量
// S3C_STORE_KEY —— 于是**入参被无条件丢弃**，与 sqlite / encrypted 两个分支
// 「显式透传 storeKey」的契约不一致：非 FromEnv 的调用方（它们通常明确持有
// key 并期望它生效）传了 key 仍会明文落盘且无任何报错。今天唯一的生产调用方
// 是 `main.go` 的 `store.Open(cfg.DataDir, cfg.StoreDriver, cfg.StoreKey)`，
// 而 `cfg.StoreKey` 恰恰来自同一个环境变量，所以**零生产影响**；这是一条
// 契约一致性问题，不是线上缺陷。
//
// 本文件用「行为」而非结构把契约钉住：不看调了谁，只看**入参是否真的生效**。

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOpenJSONUsesExplicitStoreKey KNOWN_ISSUES #64：环境变量里是 key-B，
// 显式入参给 key-A —— 磁盘上的密文必须能被**入参**那把 key 解开。
//
// 修复前：Open 走 `New` 读环境变量得到 key-B，用 key-B 加密；
// 随后 `Open(dir, "json", "key-A")` 重开时仍读环境变量 key-B → 侥幸能解开，
// 于是这个断言**抓不到**入参被丢弃（这正是当年该 Nit 难以处置的原因）。
// 因此必须再看第二条：环境变量清空后入参仍然生效（见下一条）。
func TestOpenJSONUsesExplicitStoreKey(t *testing.T) {
	const (
		explicit = "explicit-key-0123456789"
	)
	dir := t.TempDir()
	// 环境变量刻意设成**另一把** key：入参若被丢弃，就会用它加密。
	t.Setenv("S3C_STORE_KEY", "env-key-abcdefghijklmnop")

	st, err := Open(dir, "json", explicit)
	if err != nil {
		t.Fatalf("Open(json, explicit key): %v", err)
	}
	if _, err := st.Create(encAtRestAcct()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// 有 key 就该加密落盘——无论这把 key 来自何处。
	assertEncryptedFile(t, filepath.Join(dir, "accounts.json"))
}

// TestOpenJSONStoreKeyBeatsEmptyEnv KNOWN_ISSUES #64 的**决定性**断言：
// 环境变量为空（或不存在）时，显式入参仍必须生效。
//
// 修复前：入参被丢弃 → `New` 读到空环境变量 → 明文落盘且无任何报错。
// 这就是「非 FromEnv 调用方传了 key 仍明文落盘」的可观测形态。
// 环境变量兜底自身由 TestOpenJSONFallsBackToEnvWhenKeyEmpty 保留。
func TestOpenJSONStoreKeyBeatsEmptyEnv(t *testing.T) {
	const explicit = "explicit-key-0123456789"
	t.Setenv("S3C_STORE_KEY", "") // 显式清空：排除从进程继承到值的可能

	dir := t.TempDir()
	st, err := Open(dir, "json", explicit)
	if err != nil {
		t.Fatalf("Open(json, explicit key, empty env): %v", err)
	}
	if _, err := st.Create(encAtRestAcct()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	p := filepath.Join(dir, "accounts.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	if !isEncryptedBlob(raw) {
		t.Fatal("Open(json) 丢弃了入参 storeKey：S3C_STORE_KEY 为空时 secretKey 明文落盘且无报错 —— " +
			"与 sqlite / encrypted 两分支「显式透传 storeKey」的契约不一致（KNOWN_ISSUES #64）")
	}

	// 密文必须能被**入参那把 key** 读回，且微妙不变。
	reopened, err := Open(dir, "json", explicit)
	if err != nil {
		t.Fatalf("reopen with explicit key: %v", err)
	}
	defer reopened.Close()
	list, err := reopened.List()
	if err != nil {
		t.Fatalf("List after reopen: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List = %d accounts, want 1", len(list))
	}
	full, err := reopened.Get(list[0].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if full.SecretKey != "sk-secret" {
		t.Fatalf("reopened secret = %q, want %q", full.SecretKey, "sk-secret")
	}
}

// TestOpenJSONFallsBackToEnvWhenKeyEmpty 反向钉住**不被本次改动推翻的既有语义**：
// json 分支 + 入参为空时仍回退环境变量 `S3C_STORE_KEY`（`New` 的历史行为）。
//
// 这条防止有人把「入参优先」理解成「只用入参」——那么 `Open(dir, driver, "")`
// 这种靠环境配置的老用法会从「加密」静默退化成「明文」，方向正好相反。
func TestOpenJSONFallsBackToEnvWhenKeyEmpty(t *testing.T) {
	const envKey = "env-key-abcdefghijklmnop"
	t.Setenv("S3C_STORE_KEY", envKey)

	dir := t.TempDir()
	st, err := Open(dir, "json", "")
	if err != nil {
		t.Fatalf("Open(json, empty key): %v", err)
	}
	if _, err := st.Create(encAtRestAcct()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// 入参为空但环境变量非空 ⇒ 仍须加密（回退路径没被误删）。
	assertEncryptedFile(t, filepath.Join(dir, "accounts.json"))

	// 换一把错误的 key 重开必须解不开——证明加密用的是环境变量那把。
	if _, err := Open(dir, "json", "totally-different-key-000"); err == nil {
		t.Fatal("Open with wrong key must fail when the file was encrypted with S3C_STORE_KEY")
	}
}

// TestOpenJSONNoKeyAnywhereStaysPlaintext 钉住「两处都为空」：仍明文落盘
// （permissive 向后兼容），不得因为这次改动报错或偷偷加密。
func TestOpenJSONNoKeyAnywhereStaysPlaintext(t *testing.T) {
	t.Setenv("S3C_STORE_KEY", "")

	dir := t.TempDir()
	st, err := Open(dir, "json", "")
	if err != nil {
		t.Fatalf("Open(json, no key anywhere): %v", err)
	}
	if _, err := st.Create(encAtRestAcct()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	assertPlaintextFile(t, filepath.Join(dir, "accounts.json"))
}
