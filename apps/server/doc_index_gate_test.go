package main

// doc_index_gate_test.go —— 「文档导航覆盖」门禁：`docs/` 下每篇文档都必须有导航入口。
//
// 背景（2026-09-29 第 2 层「元信息 / 导航」盘点）：仓库有 4 个导航面——根 `README.md`「文档」段
// （人类）、`AGENTS.md`「文档入口」表（agent）、`llms.txt`（LLM）、`docs/README.md`（docs 目录落地页）。
// 它们此前全靠人工同步，且**已经漂移过**：README 归档清单漏 2 项、AGENTS 入口表曾落后、
// `DEVELOPMENT.md` 的文档清单曾漏 `SUPPORT.md`。
//
// `doc_link_gate_test.go` 只能保证「**已有的**链接不悬空」，**不能**保证「**新文档被登记进导航**」——
// 一篇新文档没写进任何索引，照样全绿、谁都找不到它。本门禁补的就是这一条。
//
// 断言范围：
//   - `docs/*.md`（顶层）必须出现在 `docs/README.md`（人类导航 SSOT）；
//   - `docs/<子目录>/*.md` 必须出现在 `docs/README.md` **或**该子目录自己的 `index.md`
//     （`archive/`、`decisions/` 已有各自索引，不强制重复登记）；
//   - 自检阈值：扫到的文档数不得低于基线（防路径口径塌缩后「全绿但失明」）。
//
// 盲区（刻意不做）：
//   - 不校验导航里的**描述文字**是否与文档内容一致——自然语言，属人工审查；
//   - 不强制根 `README.md` / `AGENTS.md` / `llms.txt` 三面集合完全一致：它们面向不同读者
//     （用户 / agent / LLM），允许各有取舍与不同粒度，只钉住「docs 下每篇文档都有人指路」。

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// docsNavIndex 是人类导航的 SSOT：GitHub 在浏览 `docs/` 时按字面名渲染本文件。
const docsNavIndex = "docs/README.md"

// minDocsInNavigation 是 docs 文档数的自检阈值（实测基线 30 篇，取 25 留余量）。
// 低于它说明扫描口径塌缩（目录改名 / WalkDir 失效），此时必须红灯而不是安静通过。
const minDocsInNavigation = 25

// docsMarkdownFiles 返回 `docs/` 下全部 `.md`（相对仓库根，`/` 分隔，升序）。
func docsMarkdownFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var files []string
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 docs 目录: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("未扫到任何 docs/*.md——扫描口径已失效")
	}
	sort.Strings(files)
	return files
}

// navTargets 把「某个导航文件里的相对链接」解析成仓库根相对的**目标文件路径**集合。
// 外链、纯锚点、指向仓库外的相对路径一律忽略。
func navTargets(links []mdLinkSite, navFile string) map[string]bool {
	out := map[string]bool{}
	for _, l := range links {
		if l.file != navFile || isExternalLink(l.tgt) {
			continue
		}
		tgt, _, _ := strings.Cut(l.tgt, "#")
		tgt, _, _ = strings.Cut(tgt, "?")
		if tgt == "" {
			continue
		}
		out[filepath.ToSlash(filepath.Clean(
			filepath.Join(filepath.Dir(navFile), filepath.FromSlash(tgt))))] = true
	}
	return out
}

// TestDocsAreReachableFromNavigation 断言 docs 下每篇文档都能从导航面到达。
func TestDocsAreReachableFromNavigation(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(docsNavIndex))); err != nil {
		t.Fatalf("导航 SSOT %s 不存在：%v", docsNavIndex, err)
	}
	files := docsMarkdownFiles(t)
	if len(files) < minDocsInNavigation {
		t.Fatalf("只扫到 %d 篇 docs 文档（阈值 %d）：导航扫描口径已失效", len(files), minDocsInNavigation)
	}

	links, _ := collectMdLinks(t)
	if len(links) == 0 {
		t.Fatal("未解析到任何 Markdown 链接：链接解析口径已失效")
	}
	mainIndex := navTargets(links, docsNavIndex)
	subIndexCache := map[string]map[string]bool{}

	var uncovered []string
	for _, f := range files {
		if f == docsNavIndex {
			continue // 导航 SSOT 自身不需要被自己登记。
		}
		if filepath.Dir(f) == "docs" {
			if !mainIndex[f] {
				uncovered = append(uncovered, f+"（应登记进 "+docsNavIndex+"）")
			}
			continue
		}
		subIndex := filepath.ToSlash(filepath.Join(filepath.Dir(f), "index.md"))
		targets, ok := subIndexCache[subIndex]
		if !ok {
			targets = navTargets(links, subIndex)
			subIndexCache[subIndex] = targets
		}
		if mainIndex[f] || targets[f] {
			continue
		}
		uncovered = append(uncovered, f+"（应登记进 "+docsNavIndex+" 或 "+subIndex+"）")
	}
	if len(uncovered) > 0 {
		t.Errorf("以下 %d 篇文档**没有任何导航入口**——新增文档必须同时登记进导航索引，"+
			"否则读者 / agent / LLM 都找不到它：\n  %s",
			len(uncovered), strings.Join(uncovered, "\n  "))
	}
}
