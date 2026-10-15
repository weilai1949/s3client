package handler

// listing.go —— 分页列举的公共骨架（2026-10-14 评审 O2 / KNOWN_ISSUES #89）。
//
// `listPrefixKeys` 与 `collectStorageReportObjects` 原本各写一份**逐字相同**的循环，
// service 侧 listAll / indexDst / deletePrefix 也各有一份同形态骨架——五处的 ctx 检查、
// 页数上限、token 不前进守卫完全一致，已统一收敛于 service.EnumeratePages；
// 本文件的 collectListing 只保留这两处共有的「逐项按上限收集」语义
//（截断口径见 review §B6 与 R6；deletePrefix 的按页带副作用删除留在 service 内）。

import (
	"context"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/service"
)

// keyOf 是 collectListing 的常用投影：只取对象 key（供 listPrefixKeys 复用）。
func keyOf(o s3wrap.ObjectItem) string { return o.Key }

// collectListing 逐页列举 prefix 下的对象，按 maxItems 上限收集 mapFn(o) 的结果。
//
// 返回的 truncated 表示「有对象未被收录」，四种触发：本页装不下、收满 maxItems 后对端仍称未完、
// 对端声称未完但 token 缺失/不前进、页数上限 maxPages 耗尽——后两种由 service.EnumeratePages 统一兜住。
func collectListing[T any](ctx context.Context, client *s3wrap.Client, bucket, prefix string, maxItems, maxPages int, pageSize int32, mapFn func(s3wrap.ObjectItem) T) ([]T, bool, error) {
	var out []T
	truncated, err := service.EnumeratePages(ctx, client, bucket, prefix, maxPages, pageSize,
		func(p *s3wrap.ListPage) (bool, bool, error) {
			for _, o := range p.Objects {
				if len(out) >= maxItems {
					// 本页还有对象装不进上限：明确截断，不得静默丢弃。
					return true, true, nil
				}
				out = append(out, mapFn(o))
			}
			if len(out) >= maxItems {
				// 正好收满且本页无丢弃：对端还声称有下一页才算截断（正好等于上限且对端声明
				// 列举完成 ⇒ 全部收录，不是截断）。
				return true, p.IsTruncated, nil
			}
			return false, false, nil
		})
	if err != nil {
		return nil, false, err
	}
	return out, truncated, nil
}
