package service

import (
	"strings"
	"testing"
	"time"

	// 测试自嵌 tzdata：LoadLocation 不依赖系统 zoneinfo（评审 R3 的 DST 三例必须
	// 在任何 runner 镜像上可复现）。
	_ "time/tzdata"
)

// cronLoc 固定 +08:00：Next 按传入时间所在时区计算，固定偏移让断言不随机器时区漂移。
var cronLoc = time.FixedZone("UTC+8", 8*3600)

// at 构造 cronLoc 下的时间。
func at(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, cronLoc)
}

// TestParseCronNext 覆盖合法表达式的解析与下一次触发时刻计算。
// want 为零值表示「永不触发」（结构性不可能的日期，如 2 月 30 日）。
func TestParseCronNext(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		expr  string
		after time.Time
		want  time.Time
	}{
		{"每分钟", "* * * * *", at(2026, 10, 8, 10, 15), at(2026, 10, 8, 10, 16)},
		{"秒被截断后进位", "* * * * *", at(2026, 10, 8, 10, 15).Add(30 * time.Second), at(2026, 10, 8, 10, 16)},
		{"整点时刻严格晚于", "0 2 * * *", at(2026, 10, 8, 2, 0), at(2026, 10, 9, 2, 0)},
		{"步进", "*/15 * * * *", at(2026, 10, 8, 10, 7), at(2026, 10, 8, 10, 15)},
		{"步进从整点出发", "*/15 * * * *", at(2026, 10, 8, 10, 15), at(2026, 10, 8, 10, 30)},
		{"指定日", "30 4 1 * *", at(2026, 10, 8, 5, 0), at(2026, 11, 1, 4, 30)},
		{"周一", "0 0 * * 1", at(2026, 10, 10, 9, 0), at(2026, 10, 12, 0, 0)}, // 2026-10-10 是周六
		{"闰日跨年", "0 12 29 2 *", at(2026, 10, 8, 0, 0), at(2028, 2, 29, 12, 0)},
		{"永不触发", "0 0 30 2 *", at(2026, 10, 8, 0, 0), time.Time{}},
		{"列表", "5,20,35 8-10 * * *", at(2026, 10, 8, 9, 40), at(2026, 10, 8, 10, 5)},
		{"列表命中下一分钟", "5,20,35 8-10 * * *", at(2026, 10, 8, 9, 19), at(2026, 10, 8, 9, 20)},
		{"周日写 7 等价于 0", "0 0 * * 7", at(2026, 10, 10, 9, 0), at(2026, 10, 11, 0, 0)},
		{"元旦", "0 0 1 1 *", at(2026, 10, 8, 0, 0), at(2027, 1, 1, 0, 0)},
		{"区间步进", "0-30/10 * * * *", at(2026, 10, 8, 10, 11), at(2026, 10, 8, 10, 20)},
		{"单值带步进=到末尾", "10/15 * * * *", at(2026, 10, 8, 10, 16), at(2026, 10, 8, 10, 25)},
		// Vixie 语义：日与周都受限时取 OR——10-08 是周四，先命中次日周五（dow=5）而非 1 号。
		{"日周双受限取或-先命中周五", "0 0 1 * 5", at(2026, 10, 8, 0, 0), at(2026, 10, 9, 0, 0)},
		// 日受限周为 * ：只看日——10-11（周日）不命中，命中 15 号。
		{"周为星号只看日", "0 0 15 * *", at(2026, 10, 8, 0, 0), at(2026, 10, 15, 0, 0)},
		// 日=15（受限）周=0（受限，非星号）：Vixie OR——先命中 10-11（周日）而非 15 号。
		{"日受限周亦受限取或-先命中周日", "0 0 15 * 0", at(2026, 10, 8, 0, 0), at(2026, 10, 11, 0, 0)},
		{"13 号或周五", "0 0 13 * 5", at(2026, 10, 8, 0, 0), at(2026, 10, 9, 0, 0)},
		{"当日时刻已过则跨月", "30 4 1 * *", at(2026, 10, 1, 4, 30), at(2026, 11, 1, 4, 30)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := ParseCron(tc.expr)
			if err != nil {
				t.Fatalf("ParseCron(%q) = %v, want nil", tc.expr, err)
			}
			got := c.Next(tc.after)
			if !got.Equal(tc.want) {
				t.Errorf("Next(%v) = %v, want %v", tc.after, got, tc.want)
			}
			if got.IsZero() != tc.want.IsZero() {
				t.Errorf("Next 零值判定 = %v, want %v", got.IsZero(), tc.want.IsZero())
			}
			// 非零结果必须严格晚于 after 且分钟对齐（cron 粒度为 1 分钟）。
			if !got.IsZero() {
				if !got.After(tc.after) {
					t.Errorf("Next(%v) = %v 不晚于 after", tc.after, got)
				}
				if got.Second() != 0 || got.Nanosecond() != 0 {
					t.Errorf("Next = %v 不是分钟对齐", got)
				}
				if got.Location() != tc.after.Location() {
					t.Errorf("Next 时区 = %v, want %v", got.Location(), tc.after.Location())
				}
			}
		})
	}
}

// TestParseCronNextDST（评审 R3）：非 UTC 时区的 DST 边界行为。America/New_York 2026
// 实测三例（2026-03-08 02:00 EST→03:00 EDT 春季跳变、2026-11-01 02:00 EDT→01:00 EST
// 秋季回拨）。时刻一律用绝对 UTC 锚定再转本地，避免构造歧义（秋季 01:xx 出现两次）：
//  1. `30 2` after 01:00 EST → 当日 03:30 EDT：间隙墙钟按跳变前偏移解释
//     （02:30 EST ≡ 07:30 UTC ≡ 本地 03:30 EDT），晚触发优于漏跑，且小时 3 ≠
//     表达式小时 2 是「间隙归一化」的预期，不是违约；
//  2. `30 2` after 01:45 EST → 仍是当日 03:30 EDT：候选 01:30 已过期时不得直接
//     跳到次日，当天必须触发一次；
//  3. `30 1` after 11-01 01:45 EDT → 当日第二次 01:30 EST（06:30 UTC，45 分钟后）：
//     重复小时的第二次出现同样可触发，不漏跑。
func TestParseCronNextDST(t *testing.T) {
	t.Parallel()
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	cases := []struct {
		name  string
		expr  string
		after time.Time
		want  time.Time
	}{
		{
			"春季间隙-跳变前偏移解释",
			"30 2 * * *",
			time.Date(2026, 3, 8, 6, 0, 0, 0, time.UTC).In(ny),  // 01:00 EST
			time.Date(2026, 3, 8, 7, 30, 0, 0, time.UTC).In(ny), // 03:30 EDT
		},
		{
			"春季间隙-01:45 后当天仍触发一次",
			"30 2 * * *",
			time.Date(2026, 3, 8, 6, 45, 0, 0, time.UTC).In(ny), // 01:45 EST
			time.Date(2026, 3, 8, 7, 30, 0, 0, time.UTC).In(ny), // 03:30 EDT
		},
		{
			"秋季重复小时-第二次出现不漏跑",
			"30 1 * * *",
			time.Date(2026, 11, 1, 5, 45, 0, 0, time.UTC).In(ny), // 01:45 EDT
			time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC).In(ny), // 01:30 EST
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, perr := ParseCron(tc.expr)
			if perr != nil {
				t.Fatalf("ParseCron(%q) = %v", tc.expr, perr)
			}
			got := c.Next(tc.after)
			if !got.Equal(tc.want) {
				t.Errorf("Next(%v, %s) = %v (%s), want %v (%s)",
					tc.after, tc.expr, got, got.Format("15:04 MST"), tc.want, tc.want.Format("15:04 MST"))
			}
			if !got.IsZero() {
				if !got.After(tc.after) {
					t.Errorf("Next = %v 不晚于 after %v", got, tc.after)
				}
				if got.Location() != ny {
					t.Errorf("Next 时区 = %v, want %v", got.Location(), ny)
				}
			}
		})
	}
}

// TestParseCronNextDSTMidnightGap 覆盖回验中的**跨日剔除**分支（评审 R3）：
// 智利 2026-09-06 00:00 本地墙钟因跳变不存在（09-05 24:00 → 01:00）。cronCandidates
// 会按 ±2h 探针构造两个偏移的候选——旧偏移落点是前一日 23:00（本地日期不符，必须
// 剔除，否则会把 09-05 的触发误算到 09-06），跳变前偏移解释的落点是 01:00（跳变后
// 本地时刻）。墙钟不存在 → 按间隙语义当天触发一次（晚触发优于漏跑）。
func TestParseCronNextDSTMidnightGap(t *testing.T) {
	t.Parallel()
	scl, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	c, perr := ParseCron("0 0 * * *")
	if perr != nil {
		t.Fatalf("ParseCron: %v", perr)
	}
	after := time.Date(2026, 9, 5, 16, 0, 0, 0, time.UTC).In(scl) // 09-05 12:00 -04
	got := c.Next(after)
	want := time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC).In(scl) // 09-06 01:00 -03（跳变后）
	if !got.Equal(want) {
		t.Errorf("Next(%v, 0 0 * * *) = %v (%s), want %v (%s)",
			after, got, got.Format("2006-01-02 15:04 MST"), want, want.Format("2006-01-02 15:04 MST"))
	}
	if !got.IsZero() {
		if !got.After(after) {
			t.Errorf("Next = %v 不晚于 after %v", got, after)
		}
		if got.Location() != scl {
			t.Errorf("Next 时区 = %v, want %v", got.Location(), scl)
		}
	}
}

// TestParseCronRejectsInvalid 非法表达式必须报错并点名字段，不得静默匹配任意时间。
func TestParseCronRejectsInvalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		expr string
		want string // 错误信息必须包含的片段；空串 = 只断言报错
	}{
		{"", "5 fields"},
		{"* * * *", "4 fields"},
		{"* * * * * *", "6 fields"},
		{"60 * * * *", "minute"},
		{"* 24 * * *", "hour"},
		{"0 0 32 * *", "day-of-month"},
		{"0 0 0 * *", "day-of-month"},
		{"0 0 * 13 *", "month"},
		{"0 0 * * 8", "day-of-week"},
		{"a * * * *", "minute"},
		{"0 0 x * *", "day-of-month"},
		{"*/0 * * * *", "step"},
		{"30-10 * * * *", "ascending"}, // 区间必须升序（不支持回绕）
		{"1,,2 * * * *", "empty"},
		{"1-2-3 * * * *", ""}, // 只断言报错
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			t.Parallel()
			_, err := ParseCron(tc.expr)
			if err == nil {
				t.Fatalf("ParseCron(%q) = nil error, want error", tc.expr)
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Errorf("ParseCron(%q) err = %q, want substring %q", tc.expr, err, tc.want)
			}
		})
	}
}

// TestParseCronNumericOnly 本实现只接受数字字段；MON/JAN 这类名字必须明确报错，
// 而不是被当成非法数字后给出误导信息。
func TestParseCronNumericOnly(t *testing.T) {
	t.Parallel()
	for _, expr := range []string{"0 0 * * MON", "0 0 1 JAN *"} {
		if _, err := ParseCron(expr); err == nil {
			t.Errorf("ParseCron(%q) = nil, want error（不支持月份/星期名字）", expr)
		}
	}
}

// FuzzParseCronNoPanic 解析任意输入不得 panic；可解析时 Next 结果必须严格晚于 after 且分钟对齐。
func FuzzParseCronNoPanic(f *testing.F) {
	for _, seed := range []string{
		"* * * * *", "0 2 * * *", "*/15 * * * *", "0 0 13 * 5",
		"", "* * *", "60 * * * *", "1,,2 * * * *", "0 0 30 2 *",
		"59 23 31 12 7", "-1 * * * *", "0 0 * * 7", "*/999999999999 * * * *",
	} {
		f.Add(seed)
	}
	after := time.Date(2026, 10, 8, 10, 15, 30, 0, time.UTC)
	f.Fuzz(func(t *testing.T, expr string) {
		c, err := ParseCron(expr)
		if err != nil {
			return
		}
		next := c.Next(after)
		if next.IsZero() {
			return
		}
		if !next.After(after) {
			t.Errorf("Next(%v) = %v 不晚于 after", after, next)
		}
		if next.Second() != 0 || next.Nanosecond() != 0 {
			t.Errorf("Next = %v 不是分钟对齐", next)
		}
		if next.Location() != after.Location() {
			t.Errorf("Next 时区 = %v, want %v", next.Location(), after.Location())
		}
	})
}

// TestParseCronStepAndRangeErrors 步进与区间内部的细分错误分支。
func TestParseCronStepAndRangeErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		expr string
		want string
	}{
		{"*/x * * * *", "invalid step"},  // 步进非数字
		{"0 0 x-10 * *", "day-of-month"}, // 区间低端值非法（点名字段）
		{"0 0 5-x * *", "day-of-month"},  // 区间高端值非法
		{"0 0 10-5 * *", "ascending"},    // 降序区间
		{"0 0 * * 1-2-3", "day-of-week"}, // 多段破折号
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			t.Parallel()
			_, err := ParseCron(tc.expr)
			if err == nil {
				t.Fatalf("ParseCron(%q) = nil, want error", tc.expr)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want substring %q", err, tc.want)
			}
		})
	}
}
