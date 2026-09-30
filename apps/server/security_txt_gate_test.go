package main

// security_txt_gate_test.go —— RFC 9116 `security.txt` 的机械门禁。
//
// 背景：`.well-known/security.txt` 是机器可读的漏洞披露入口，但「文件存在」不等于
// 「内容可信」——Expires 过期后安全研究员就不该再信任文件里的联系方式（RFC 9116 §5.3）。
// 若没有门禁，「续期」只能靠人记得；本文件把「过期」变成红灯。
//
// 断言范围（RFC 9116 §2.5）：
//   - Expires：**恰好一个**、按 RFC 3339 可解析、**未过期**（过期即红灯强制续期）、
//     且不超过一年（§2.5.5 建议）；
//   - Contact：至少一个 https URI，首选项必须复用 `.github/SECURITY.md` 登记的
//     GitHub 私有漏洞报告渠道（本仓库无公开安全邮箱，禁止发明 mailto）；
//   - Policy：至少一个，且每个都指向仓库内**磁盘上真实存在**的文件；
//   - Canonical：唯一且指向本文件自身；
//   - Preferred-Languages：非空。
//
// 自检纪律（同其它门禁）：解析出的字段数低于阈值 → Fatal，防止格式变更后
// 「抽不到任何字段却全绿」的失明。
//
// 变异验证（复核步骤）：把 Expires 改成过去时间（如 2020-01-01T00:00:00Z）→
// 本门禁红灯点名「已过期」→ 还原后绿灯。命令：
//
//	cd apps/server && go test . -run TestSecurityTxt -count=1

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	// securityTxtRelPath 是 security.txt 在工作区中的相对路径（RFC 9116 §3 的
	// 标准位置；本仓库把它放在仓库根，部署时映射到站点 `/.well-known/`）。
	securityTxtRelPath = ".well-known/security.txt"

	// securityTxtBlobPrefix 是仓库内文件在 GitHub 上的 blob URL 前缀（main 分支）。
	// Policy / Canonical 必须用该前缀指向仓库内文件，门禁才能回落到磁盘校验存在性。
	securityTxtBlobPrefix = "https://github.com/weilai1949/s3client/blob/main/"

	// securityTxtAdvisoryPath 是 `.github/SECURITY.md` 登记的唯一私下报告渠道
	// （GitHub 私有漏洞报告：仓库 → Security → Report a vulnerability）。
	securityTxtAdvisoryPath = "/security/advisories/new"

	// securityTxtMinFields 是解析面自检阈值（当前文件含 Contact/Expires 等 6 个字段）。
	securityTxtMinFields = 5

	// securityTxtMaxFuture 是 Expires 的上界（RFC 9116 §2.5.5 建议小于一年；
	// 留一天闰年余量，避免合法续期被误杀）。
	securityTxtMaxFuture = 366 * 24 * time.Hour
)

// securityTxtField 是一行解析后的字段（含行号，便于报错定位）。
type securityTxtField struct {
	name  string
	value string
	line  int
}

// parseSecurityTxt 按 RFC 9116 的朴素格式解析：`Field: value`，`#` 开头为注释，空行忽略。
func parseSecurityTxt(raw string) []securityTxtField {
	var out []securityTxtField
	for i, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(strings.TrimRight(line, "\r"))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}
		out = append(out, securityTxtField{
			name:  strings.TrimSpace(trimmed[:idx]),
			value: strings.TrimSpace(trimmed[idx+1:]),
			line:  i + 1,
		})
	}
	return out
}

// securityTxtValues 返回指定字段的全部取值（保留出现顺序，用于 Contact 优先级语义）。
func securityTxtValues(fields []securityTxtField, name string) []string {
	var out []string
	for _, f := range fields {
		if f.name == name {
			out = append(out, f.value)
		}
	}
	return out
}

// securityTxtRepoPath 把仓库 blob URL 映射回工作区相对路径；不是本仓库 blob URL 时 ok=false。
func securityTxtRepoPath(u string) (string, bool) {
	if !strings.HasPrefix(u, securityTxtBlobPrefix) {
		return "", false
	}
	rel := strings.TrimPrefix(u, securityTxtBlobPrefix)
	if rel == "" || strings.HasPrefix(rel, "/") {
		return "", false
	}
	return rel, true
}

// securityTxtParseDate 解析 Expires：RFC 3339 优先，容忍纯 ISO 日期（YYYY-MM-DD）。
func securityTxtParseDate(v string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, v); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02", v); err == nil {
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("不是 RFC 3339 / ISO 8601 日期时间: %q", v)
}

// TestSecurityTxtRFC9116 校验 .well-known/security.txt 的必需字段、有效期与可达性。
func TestSecurityTxtRFC9116(t *testing.T) {
	raw := readRepoFile(t, securityTxtRelPath)
	fields := parseSecurityTxt(raw)
	// 解析面自检：格式变更导致字段抽不出来时直接失败，避免「全绿但失明」。
	if len(fields) < securityTxtMinFields {
		t.Fatalf("%s 只解析出 %d 个字段（阈值 %d），疑似解析口径失效",
			securityTxtRelPath, len(fields), securityTxtMinFields)
	}

	// —— Expires：恰好一个、可解析、未过期、不超过一年 ——
	expires := securityTxtValues(fields, "Expires")
	if len(expires) != 1 {
		t.Fatalf("%s 的 Expires 必须恰好出现一次（RFC 9116 §2.5.5），实际 %d 次",
			securityTxtRelPath, len(expires))
	}
	exp, err := securityTxtParseDate(expires[0])
	if err != nil {
		t.Fatalf("Expires=%q 无法解析：%v", expires[0], err)
	}
	now := time.Now()
	if !exp.After(now) {
		t.Errorf("Expires=%s 已过期（现在 %s）——过期的 security.txt 不应继续被信任；"+
			"请把 Expires 续期到当前时间之后、一年之内（RFC 9116 §2.5.5）",
			exp.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	if exp.After(now.Add(securityTxtMaxFuture)) {
		t.Errorf("Expires=%s 距现在超过一年（RFC 9116 §2.5.5 建议小于一年）", exp.Format(time.RFC3339))
	}

	// —— Contact：至少一个 https URI，且首选项必须是 SECURITY.md 登记的私下渠道 ——
	contacts := securityTxtValues(fields, "Contact")
	if len(contacts) == 0 {
		t.Fatal("缺少 Contact 字段（RFC 9116 §2.5.3 要求 MUST always be present）")
	}
	for _, c := range contacts {
		if !strings.HasPrefix(c, "https://") {
			t.Errorf("Contact=%q 不是 https URI（RFC 9116 §2.5.3）", c)
		}
	}
	if !strings.Contains(contacts[0], securityTxtAdvisoryPath) {
		t.Errorf("Contact=%q 不是 .github/SECURITY.md 登记的唯一渠道；应指向含 %q 的 GitHub 私有漏洞报告，"+
			"不得发明邮箱地址", contacts[0], securityTxtAdvisoryPath)
	}

	// —— Policy：至少一个，且每个都指向仓库内真实存在的文件 ——
	policies := securityTxtValues(fields, "Policy")
	if len(policies) == 0 {
		t.Fatal("缺少 Policy 字段（本仓库要求登记披露政策与威胁模型）")
	}
	for _, p := range policies {
		rel, ok := securityTxtRepoPath(p)
		if !ok {
			t.Errorf("Policy=%q 不是 %s 前缀的仓库内文件，无法校验存在性", p, securityTxtBlobPrefix)
			continue
		}
		if _, err := os.Stat(filepath.Join(repoRoot(t), rel)); err != nil {
			t.Errorf("Policy=%q 指向的仓库文件 %s 不存在：%v", p, rel, err)
		}
	}

	// —— Canonical：唯一，且指向本文件自身 ——
	canonicals := securityTxtValues(fields, "Canonical")
	if len(canonicals) != 1 {
		t.Errorf("Canonical 必须恰好一次（RFC 9116 §2.5.2），实际 %d 次", len(canonicals))
	} else if rel, ok := securityTxtRepoPath(canonicals[0]); !ok || rel != securityTxtRelPath {
		t.Errorf("Canonical=%q 必须指向 %s（仓库内可校验的规范位置），实际映射为 %q（ok=%v）",
			canonicals[0], securityTxtRelPath, rel, ok)
	}

	// —— Preferred-Languages：可选字段，但登记了就不得为空 ——
	langs := securityTxtValues(fields, "Preferred-Languages")
	if len(langs) == 0 || strings.TrimSpace(langs[0]) == "" {
		t.Error("缺少非空 Preferred-Languages（报告方据此选择语言）")
	}
}
