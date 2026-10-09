package service

// schedule_cron.go —— 5 字段 cron 表达式解析与「下一次触发时刻」计算（ROADMAP §三 #6）。
//
// 为什么自研而不引 robfig/cron：本仓库坚持零新增依赖（同 ADR-008 / ADR-013 先例）——
// 新增直接依赖会牵动 go.mod、许可证清单重生成（third_party_licenses_gate）与
// govulncheck 攻击面；而计划任务只需「分钟粒度 + 下一次触发时刻」，语义可控的
// 5 字段子集约 200 行即可完整覆盖并 100% 测试。
//
// 语义与 Vixie cron 对齐（本实现的对外承诺）：
//   - 5 字段：分(0-59) 时(0-23) 日(1-31) 月(1-12) 周(0-7，7 与 0 等价于周日)；
//   - 每个字段支持 `*`、列表 `a,b`、区间 `a-b`（必须升序，不支持回绕）、
//     步进 `*/s`、`a-b/s`、`a/s`（从 a 到字段末尾，POSIX 语义）；
//   - 日与周**同时受限**（都不是字面 `*`）时取 OR：命中日 **或** 周任一即可
//     （Vixie 语义；只受限其一则只看那个字段）；
//   - **只接受数字**：MON/JAN 等名字明确报错，不静默按非法数字处理；
//   - Next 返回**严格晚于** after、按 after 所在时区、分钟对齐的下一时刻；
//     结构性不可能的表达式（如 2 月 30 日）返回零值。
//
// 搜索策略按「日 → 时 → 分」三级下钻（而非逐分钟暴力扫）：40 年视界 ≈ 1.5 万次
// 日期构造，命中日才进入时/分枚举——`0 0 30 2 *` 这类永不命中的表达式也只是
// 1.5 万次廉价日期判断，单次计算在微秒~毫秒级，足够调度器每个 tick 对每个计划调用。

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// cronHorizonDays 是 Next 的搜索视界（40 年）。
// 取 40 年的原因：「2 月 29 日 + 指定星期」在格里高利历下最长约 28 年一遇，
// 留足余量；视界内不命中即判定为「永不触发」（返回零值）。
const cronHorizonDays = 366 * 40

// Cron 是已解析的 5 字段 cron 表达式。零值不可用，必须经 ParseCron 构造。
// 原始表达式不在此保存——Schedule.Cron 已是回显的唯一来源（避免双源漂移）。
type Cron struct {

	// 位掩码：位 v 置位表示值 v 命中（dom 位 1..31、month 位 1..12、dow 位 0..6）。
	minute, hour, dom, month, dow uint64
	domStar, dowStar              bool // 该字段是否为字面 `*`（Vixie OR 语义的判据）

	// 升序枚举列表（由掩码展开），供 Next 按序下钻。
	minutes, hours []int
}

// cronField 描述单个字段的解析约束。
type cronField struct {
	name string
	lo   int
	hi   int
	// norm 把领域值映射为掩码位（仅周字段需要：7 → 0）。
	norm func(int) int
}

// ParseCron 解析 5 字段 cron 表达式；任何非法输入返回带字段名的错误。
func ParseCron(expr string) (*Cron, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron: expected 5 fields, got %d fields", len(fields))
	}
	specs := []cronField{
		{name: "minute", lo: 0, hi: 59, norm: ident},
		{name: "hour", lo: 0, hi: 23, norm: ident},
		{name: "day-of-month", lo: 1, hi: 31, norm: ident},
		{name: "month", lo: 1, hi: 12, norm: ident},
		{name: "day-of-week", lo: 0, hi: 7, norm: normDOW},
	}
	masks := make([]uint64, len(specs))
	for i, spec := range specs {
		mask, err := parseCronField(spec, fields[i])
		if err != nil {
			return nil, err
		}
		masks[i] = mask
	}
	c := &Cron{
		minute: masks[0],
		hour:   masks[1],
		dom:    masks[2],
		month:  masks[3],
		dow:    masks[4],
		// OR 语义只看字面 `*`：`*/1` 覆盖全天但语法上受限，与 Vixie 一致。
		domStar: fields[2] == "*",
		dowStar: fields[4] == "*",
		minutes: expandBits(masks[0]),
		hours:   expandBits(masks[1]),
	}
	return c, nil
}

// ident 恒等映射（分/时/日/月字段）。
func ident(v int) int { return v }

// normDOW 把周字段值 7 归一为 0（均为周日），保证 0/7 写法等价。
func normDOW(v int) int {
	if v == 7 {
		return 0
	}
	return v
}

// parseCronField 解析单个字段（可含列表/区间/步进）为位掩码。
func parseCronField(f cronField, s string) (uint64, error) {
	var mask uint64
	for _, elem := range strings.Split(s, ",") {
		if elem == "" {
			return 0, fmt.Errorf("cron: empty list element in %s field", f.name)
		}
		base := elem
		step := 1
		hasStep := false
		if i := strings.IndexByte(elem, '/'); i >= 0 {
			base = elem[:i]
			n, err := strconv.Atoi(elem[i+1:])
			if err != nil {
				return 0, fmt.Errorf("cron: invalid step %q in %s field", elem[i+1:], f.name)
			}
			if n < 1 {
				return 0, fmt.Errorf("cron: step must be >= 1, got %d in %s field", n, f.name)
			}
			step = n
			hasStep = true
		}

		lo, hi, err := cronSpan(f, base, hasStep)
		if err != nil {
			return 0, err
		}
		// 用「剩余距离」判断是否继续，避免 step 极大时 v += step 溢出后死循环
		//（fuzz 输入可能给出 MaxInt 量级的 step）。
		for v := lo; v <= hi; {
			mask |= 1 << uint(f.norm(v))
			if step > hi-v {
				break
			}
			v += step
		}
	}
	return mask, nil
}

// cronSpan 把一个列表元素（`*` / `a` / `a-b`）展开为闭区间 [lo, hi]。
// hasStep 表示该元素带 `/step`：POSIX 规定单值带步进（`a/s`）从 a 扫到字段末尾。
func cronSpan(f cronField, base string, hasStep bool) (lo, hi int, err error) {
	if base == "*" {
		return f.lo, f.hi, nil
	}
	if strings.Contains(base, "-") {
		parts := strings.Split(base, "-")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("cron: invalid range %q in %s field", base, f.name)
		}
		if lo, err = cronNum(f, parts[0]); err != nil {
			return 0, 0, err
		}
		if hi, err = cronNum(f, parts[1]); err != nil {
			return 0, 0, err
		}
		if lo > hi {
			return 0, 0, fmt.Errorf("cron: range %q in %s field must be ascending", base, f.name)
		}
		return lo, hi, nil
	}
	if lo, err = cronNum(f, base); err != nil {
		return 0, 0, err
	}
	if hasStep {
		return lo, f.hi, nil
	}
	return lo, lo, nil
}

// cronNum 解析并校验一个数值：必须是纯数字且落在字段范围内。
// 名字（MON/JAN）在此被明确拒绝——它们不是「超范围的数字」，错误信息给出专用提示。
func cronNum(f cronField, tok string) (int, error) {
	v, err := strconv.Atoi(tok)
	if err != nil {
		return 0, fmt.Errorf("cron: %s value %q is not a number (names like MON/JAN are not supported)", f.name, tok)
	}
	if v < f.lo || v > f.hi {
		return 0, fmt.Errorf("cron: %s value %d out of range %d-%d", f.name, v, f.lo, f.hi)
	}
	return v, nil
}

// expandBits 把掩码展开为升序整数列表（掩码位均 < 64，见各字段范围约束）。
func expandBits(mask uint64) []int {
	out := make([]int, 0, bitsOnesCount(mask))
	for v := 0; v < 64; v++ {
		if mask&(1<<uint(v)) != 0 {
			out = append(out, v)
		}
	}
	return out
}

// bitsOnesCount 只数本实现会用到的低位（避免为容量估算引入 math/bits 依赖面）。
func bitsOnesCount(mask uint64) int {
	n := 0
	for m := mask; m != 0; m &= m - 1 {
		n++
	}
	return n
}

// Next 返回严格晚于 after 的下一触发时刻（after 所在时区、分钟对齐）。
// 40 年视界内不命中（结构性不可能的日期）返回零值。
func (c *Cron) Next(after time.Time) time.Time {
	loc := after.Location()
	// 从 after 的下一分钟所在日期开始搜索；当日更早的时刻必然不晚于 after。
	y, m, d := after.Date()
	for i := 0; i < cronHorizonDays; i++ {
		day := time.Date(y, m, d+i, 0, 0, 0, 0, loc)
		if c.month&(1<<uint(day.Month())) == 0 {
			continue
		}
		if !c.dayMatches(day) {
			continue
		}
		// 时间构造对 DST 间隙/重叠的归一化是「已接受的取舍」：春季跳变日的
		// 不存在时刻会落到跳变后的合法时刻，与主流 cron 行为一致；备份场景
		// 晚触发一次优于不触发。
		for _, h := range c.hours {
			for _, mi := range c.minutes {
				cand := time.Date(day.Year(), day.Month(), day.Day(), h, mi, 0, 0, loc)
				if cand.After(after) {
					return cand
				}
			}
		}
	}
	return time.Time{}
}

// dayMatches 按 Vixie 语义判定日期是否命中：
// 日与周都受限（非字面 `*`）→ 取 OR；只受限其一 → 只看那个字段；都为 `*` → 恒真。
func (c *Cron) dayMatches(t time.Time) bool {
	domOK := c.dom&(1<<uint(t.Day())) != 0
	dowOK := c.dow&(1<<uint(t.Weekday())) != 0
	switch {
	case c.domStar && c.dowStar:
		return true
	case c.domStar:
		return dowOK
	case c.dowStar:
		return domOK
	default:
		return domOK || dowOK
	}
}
