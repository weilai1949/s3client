package main

// data_model_gate_test.go —— 「数据模型 / 存储格式」文档与代码一致性门禁。
//
// 背景：账号存储的事实此前散落在四处——ADR-0006（驱动与原子写决策）、
// api/accounts.schema.json（字段契约）、compatibility.md §4（S3C2/S3C3 信封）、
// architecture.md（分层）。docs/data-model.md 把四者汇成一张地图，代价是**多了一处会漂移的副本**。
// 本门禁让「漂移」变成红灯，而不是靠人记得同步：
//   - 真值取自 **反射** `model.Account`（不是另抄一份字段表）；
//   - 驱动名取自 `store/open.go` 的 switch **源码**（不是另抄一份驱动清单）；
//   - schema / ADR 链接与信封版本号必须在文档里出现。
//
// 断言范围：**登记覆盖**，不评判文档叙述是否精彩。文档新增字段说明 → 自动进入断言面。
//
// 变异验证（复核步骤）：从 docs/data-model.md 删掉 `` `sqlite` `` 或任一 Account 字段名
// → 本门禁红灯点名 → 还原后绿灯。
//
// 相关：docs_naming_gate_test.go（命名登记）、doc_index_gate_test.go（导航覆盖）、
// doc_link_gate_test.go（链接与锚点）—— 本门禁补的是「文档 ⇔ 代码/契约」的一致性。

import (
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/weilai1949/s3client/apps/server/internal/model"
)

// 扫描面自检阈值：低于阈值说明解析口径塌缩（例如文档被清空 / 反射拿到零字段），
// 此时必须 Fatal 而不是「全绿但失明」。
const (
	dataModelDocMinBytes  = 2000
	dataModelMinFields    = 12 // model.Account 的 json 字段数
	dataModelMinDrivers   = 3  // json / sqlite / encrypted
	dataModelDocRel       = "docs/data-model.md"
	dataModelOpenRel      = "apps/server/internal/store/open.go"
	dataModelSchemaToken  = "accounts.schema.json"
	dataModelADRToken     = "0006-store-drivers-atomic-write.md"
	dataModelViewToken    = "AccountView"
	dataModelSecretToken  = "secretSet"
	dataModelEnvelopeV2   = "S3C2"
	dataModelEnvelopeV3   = "S3C3"
	openStoreDriverCaseRe = `(?m)^\s*case\s+([^:]+):`
	openStoreDriverNameRe = `"([^"]*)"`
)

// parseOpenStoreDrivers 从 open.go 的 switch 源码里提取驱动名（去空串、去重）。
// 真值取自源码而非文档里的清单，文档漏登记新驱动时会自动红灯。
func parseOpenStoreDrivers(src string) []string {
	caseRe := regexp.MustCompile(openStoreDriverCaseRe)
	nameRe := regexp.MustCompile(openStoreDriverNameRe)
	seen := map[string]bool{}
	var out []string
	for _, m := range caseRe.FindAllStringSubmatch(src, -1) {
		for _, n := range nameRe.FindAllStringSubmatch(m[1], -1) {
			d := strings.TrimSpace(n[1])
			if d == "" || seen[d] {
				continue
			}
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// TestDataModelDocCoversModelAndDrivers 断言 docs/data-model.md 覆盖
// model.Account 的全部 JSON 字段、store.Open 接受的全部驱动名，并链接 schema 与 ADR-0006。
func TestDataModelDocCoversModelAndDrivers(t *testing.T) {
	t.Parallel()
	doc := readRepoFile(t, dataModelDocRel)

	// 自检 1：文档不能是空壳。
	if len(doc) < dataModelDocMinBytes {
		t.Fatalf("docs/data-model.md 只有 %d 字节（阈值 %d）：文档疑似被清空或截断",
			len(doc), dataModelDocMinBytes)
	}

	// 1) model.Account 的每个 json 字段名都必须以行内代码（反引号）形式登记。
	//    只认反引号形态，避免散文里随口提到一个词就算覆盖。
	typ := reflect.TypeOf(model.Account{})
	var fields []string
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	if len(fields) < dataModelMinFields {
		t.Fatalf("反射只拿到 %d 个 json 字段（阈值 %d）：model.Account 结构或标签解析口径已变化",
			len(fields), dataModelMinFields)
	}
	var missing []string
	for _, f := range fields {
		if !strings.Contains(doc, "`"+f+"`") {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		t.Errorf("docs/data-model.md 未登记 %d 个 model.Account 字段（须以 `字段名` 形式出现）：%s\n"+
			"新增/改名字段时同步本文件（口径见 docs/DEVELOPMENT.md §4）。",
			len(missing), strings.Join(missing, ", "))
	}

	// 2) store.Open 的驱动名必须逐个登记。
	drivers := parseOpenStoreDrivers(readRepoFile(t, dataModelOpenRel))
	if len(drivers) < dataModelMinDrivers {
		t.Fatalf("从 %s 只解析出 %d 个驱动（阈值 %d）：switch 形态或正则口径已变化，门禁会失明",
			dataModelOpenRel, len(drivers), dataModelMinDrivers)
	}
	var missingDrivers []string
	for _, d := range drivers {
		if !strings.Contains(doc, "`"+d+"`") {
			missingDrivers = append(missingDrivers, d)
		}
	}
	if len(missingDrivers) > 0 {
		t.Errorf("docs/data-model.md 未登记驱动 %s（源码 %s 的 switch 里存在）",
			strings.Join(missingDrivers, ", "), dataModelOpenRel)
	}

	// 3) 契约与决策链接 + 信封版本 + 对外视图不得从文档里消失。
	for _, want := range []string{
		dataModelSchemaToken, // 指向字段级机器可读契约
		dataModelADRToken,    // 指向驱动/原子写的 ADR
		dataModelViewToken,   // 对外视图（secretKey 不进响应）
		dataModelSecretToken, // secretSet 语义
		dataModelEnvelopeV2,  // 旧信封仍可读
		dataModelEnvelopeV3,  // 当前写入信封
		"kdfParamsValid",     // 文件头 KDF 参数校验（读放大 DoS 防线）
		"atomicfile",         // 原子写实现
		"AcquireDataDirLock", // 单写者锁
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/data-model.md 缺少关键锚点 %q（该事实是存储格式契约的一部分，不能被删）", want)
		}
	}
}

// TestParseOpenStoreDrivers 用合成源码钉住解析口径：注释里的 case、非驱动字符串都不得误收。
func TestParseOpenStoreDrivers(t *testing.T) {
	t.Parallel()
	// 形如真实 open.go 的片段：注释里出现 case、default 分支、多值 case。
	src := "// case \"bogus\": 只是注释\nswitch d {\ncase \"sqlite\":\ncase \"encrypted\":\ncase \"\", \"json\":\ndefault:\n\treturn nil\n}"
	got := parseOpenStoreDrivers(src)
	want := []string{"sqlite", "encrypted", "json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseOpenStoreDrivers = %v，want %v（口径变化会让门禁漏报或误报）", got, want)
	}
	// 文件路径拼接：确保 dataModelOpenRel 是可被 readRepoFile 解析的仓库相对路径。
	if filepath.IsAbs(dataModelOpenRel) {
		t.Fatalf("%s 不应是绝对路径", dataModelOpenRel)
	}
}
