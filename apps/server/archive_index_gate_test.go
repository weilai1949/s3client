package main

// archive_index_gate_test.go —— `docs/archive/` 归档清单「计数 / 等集」门禁。
//
// 背景（2026-10-10 批次交接快照 §5 未做第 5 项）：归档操作的四步（`git mv` → 修引用 →
// 索引补行 → CHANGELOG）此前**全靠人工纪律**：`doc_link_gate` 只能保证清单里的链接不悬空，
// 保不住「新冻结件漏登记进清单」，也保不住索引页头那句人写的「当前**已归档 N 份**」
// 与目录实际文件数一致——两处都是「少写一行照样全绿」的盲区。
//
// 断言范围（三条，各带扫描面自检——低于基线即 Fatal，防解析口径塌缩后全绿但失明）：
//   1. 页头计数：`docs/archive/index.md` 的「当前**已归档 N 份**」== `docs/archive/` 下
//      除 `index.md` 外的 `*.md` 文件数；
//   2. 登记等集（双向）：目录里每个冻结件必须在「归档清单」表内被链接；表内每个链接目标
//      必须真实存在——双向不等即点名缺哪一行 / 哪一条是幽灵登记；
//   3. 扫描面自检：冻结件数与清单行数均不得低于基线阈值。
//
// 不断言：清单描述文字与冻结件内容是否一致（自然语言，属人工复核）、归档日期是否正确
// （需读 git 历史，门禁只保证「有登记、数一致」）。
//
// 变异验证（复核步骤，2026-10-10 实跑）：把页头「已归档 8 份」改回 `7 份` → 断言 1 红；
// 删掉 `handoff-20261010.md` 清单行 → 断言 2 红（点名该冻结件未登记）；还原后全绿。
// 复核命令：`cd apps/server && go test . -run TestArchiveIndexCountMatchesDir -count=1 -v`。
//
// 相关：`doc_index_gate_test.go`（导航覆盖）、`doc_link_gate_test.go`（链接与锚点）——
// 三者分别保证「有入口」「链接不死」「归档计数与登记不分叉」。

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	// archiveDirRel 是冻结件所在目录；archiveIndexRel 是其索引（清单 + 页头计数）。
	archiveDirRel   = "docs/archive"
	archiveIndexRel = "docs/archive/index.md"
	// archiveIndexMinFrozen 是扫描面自检阈值（实测基线 2026-10-10：8 份冻结件）。
	archiveIndexMinFrozen = 6
)

var (
	// archiveCountRe 匹配索引页头人工维护的计数声明：`当前**已归档 N 份**`。
	archiveCountRe = regexp.MustCompile(`当前\*\*已归档\s*(\d+)\s*份\*\*`)
	// archiveRowLinkRe 匹配表格行里的首个相对链接目标（`[...](x.md)`）。
	archiveRowLinkRe = regexp.MustCompile(`\]\(([^)]+)\)`)
)

// TestArchiveIndexCountMatchesDir 断言归档索引的页头计数、清单行、目录文件三者一致。
func TestArchiveIndexCountMatchesDir(t *testing.T) {
	t.Parallel()

	// 目录侧：除索引自身外的全部冻结件（只认顶层 `.md`，子目录不属归档件）。
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), filepath.FromSlash(archiveDirRel)))
	if err != nil {
		t.Fatalf("读取 %s 目录: %v", archiveDirRel, err)
	}
	frozen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "index.md" {
			continue
		}
		frozen[e.Name()] = true
	}
	if len(frozen) < archiveIndexMinFrozen {
		t.Fatalf("扫描面塌缩：%s 下只扫到 %d 个冻结件（基线 ≥%d）——目录口径或排除清单被改坏",
			archiveDirRel, len(frozen), archiveIndexMinFrozen)
	}

	text := readRepoFile(t, archiveIndexRel)

	// 断言 1：页头计数 == 目录冻结件数。
	m := archiveCountRe.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("%s 页头未找到计数声明形如「当前**已归档 N 份**」——措辞变更需同步本门禁",
			archiveIndexRel)
	}
	stated, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("%s 页头计数解析失败 %q: %v", archiveIndexRel, m[1], err)
	}
	if stated != len(frozen) {
		t.Errorf("%s 页头写「已归档 %d 份」，但 %s 下有 %d 个冻结件（除 index.md）——"+
			"归档 / 去归档后必须同 PR 更新计数",
			archiveIndexRel, stated, archiveDirRel, len(frozen))
	}

	// 断言 2：清单表行链接 ⇄ 目录冻结件双向等集。
	listed := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		lm := archiveRowLinkRe.FindStringSubmatch(line)
		if lm == nil {
			continue
		}
		tgt, _, _ := strings.Cut(lm[1], "#")
		if strings.Contains(tgt, "://") || !strings.HasSuffix(tgt, ".md") {
			continue
		}
		name := filepath.Base(tgt)
		if name == "index.md" {
			continue // 清单表内的自引用不算冻结件登记
		}
		listed[name] = true
	}
	if len(listed) < archiveIndexMinFrozen {
		t.Fatalf("扫描面塌缩：%s 归档清单只解析到 %d 行（基线 ≥%d）——表格形态被改坏",
			archiveIndexRel, len(listed), archiveIndexMinFrozen)
	}
	for name := range frozen {
		if !listed[name] {
			t.Errorf("冻结件 %s/%s 未登记进 %s 的「归档清单」——归档操作第 3 步漏做",
				archiveDirRel, name, archiveIndexRel)
		}
	}
	for name := range listed {
		if !frozen[name] {
			t.Errorf("%s 归档清单登记的 %s 在 %s 下不存在——幽灵登记或文件被误删",
				archiveIndexRel, name, archiveDirRel)
		}
	}
	t.Logf("归档冻结件 %d 份，清单行 %d 行，页头计数 %d", len(frozen), len(listed), stated)
}
