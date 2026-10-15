package service

import (
	"context"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// EnumeratePages 是分页列举的公共循环骨架（2026-10-14 评审 O2 / KNOWN_ISSUES #89）。
//
// 此前同一骨架在五处各写一份：service 的 listAll / indexDst / deletePrefix、
// handler 的 listPrefixKeys / collectStorageReportObjects——都是「逐页 ListObjectsV2 →
// 交给调用方处理本页 → 三条终止守卫兜底」。收敛后每处只剩「本页做什么」：
//
//  1. ctx 取消在每次列举前检查——不依赖 SDK 调用间接响应取消；
//  2. 页数上限 maxPages（service 与 handler 统一传 ListMaxPages，与 s3wrap 包内
//     列举守卫同口径）——空页 + 前进 token 时调用方的计数永不增长，只有页数上限拦得住；
//  3. token 缺失或不前进即停并标记截断——对端异常时避免空转（review §B6）。
//
// 每页的处理（纯收集 / 收进 map / 按页带副作用删除）交给 fn：fn 返回 stop=true 即停止，
// truncated 由 fn 判定（如达到总量上限）；p.IsTruncated=false 与 token 守卫由本函数统一处理，
// 结束于页数上限时恒为截断（review R6：静默截断必须透出）。
func EnumeratePages(
	ctx context.Context,
	client *s3wrap.Client,
	bucket, prefix string,
	maxPages int,
	pageSize int32,
	fn func(p *s3wrap.ListPage) (stop, truncated bool, err error),
) (truncated bool, err error) {
	token := ""
	for page := 0; page < maxPages; page++ {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		p, err := client.ListObjectsPage(ctx, bucket, prefix, "", token, "", pageSize)
		if err != nil {
			return false, err
		}
		stop, trunc, err := fn(p)
		if err != nil {
			return trunc, err
		}
		if stop {
			return trunc, nil
		}
		if !p.IsTruncated {
			return false, nil
		}
		if p.NextToken == "" || p.NextToken == token {
			return true, nil
		}
		token = p.NextToken
	}
	return true, nil
}
