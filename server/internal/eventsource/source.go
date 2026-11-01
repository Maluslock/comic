// Package eventsource 把外部漫展数据源抽象成可插拔的 Source 适配器。
//
// 设计约束（与 team goal 对齐）：
//   - 只负责「请求 -> 解析 -> 规范化 -> 去重」，不写库、不做业务判断，
//     编排层（internal/service 或 cmd）按优先级依次尝试，失败自动切备用源。
//   - 老的 internal/ingest 包（HTML 抓取）保持不动，本包是它的替代品。
//   - 统一 EventItem；AllcppID 是跨源唯一 upsert key：allcpp 用真实 id，
//     bilibili 用真实 project_id，方便 comic_events 做幂等写入。
//
// 抓取过程的诊断信息（提前停原因 / 部分页失败 / 被过滤掉多少条）通过
// Result.Report 返回，同时写日志，编排层可直接读来做数据源状态页。
package eventsource

import (
	"context"
	"log"
	"strings"
	"time"
)

// 数据源标识，落到 EventItem.Source / 数据源状态页。
const (
	SourceAllcpp   = "allcpp"
	SourceBilibili = "bilibili"
)

// 默认参数。
const (
	// DefaultTimeout 单次 Fetch 的整体超时。
	DefaultTimeout = 30 * time.Second

	// DefaultMaxPages 单源默认最多翻多少页（防止上游 pageCount 异常时无限翻页）。
	DefaultMaxPages = 10

	// DefaultAllcppPageSize allcpp 接口默认每页条数。
	DefaultAllcppPageSize = 50

	// DefaultBilibiliPageSize bilibili pagesize>=30 会返回
	// {code:81102084,message:'请求非法，请稍后重试'}，必须固定 20。
	DefaultBilibiliPageSize = 20

	// MaxBilibiliPageSize bilibili 允许的最大 pagesize。
	MaxBilibiliPageSize = 20

	// BrowserUA 不带浏览器 UA 请求 allcpp 会直接 403（实测），必须带。
	BrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// Shanghai 是漫展时间统一使用的时区。
var shanghai = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	// 容器里可能没有 tzdata，退回固定 +8。
	return time.FixedZone("CST", 8*3600)
}()

// Shanghai 返回适配器内部使用的时区（Asia/Shanghai，缺失时退化为 +8）。
func Shanghai() *time.Location { return shanghai }

// EventItem 是跨源统一的漫展条目。
type EventItem struct {
	Source    string    // allcpp | bilibili
	AllcppID  int32     // 跨源唯一 upsert key：allcpp 真实 id / bilibili 真实 project_id
	Name      string    // 活动名
	City      string    // 城市（已去掉「市/省」尾缀）
	Province  string    // 省（上游没给就为空）
	Venue     string    // 场馆/展馆名
	Address   string    // 详细地址
	Start     time.Time // 开始时间
	End       time.Time // 结束时间
	CoverURL  string    // 原图直链，http(s) 开头
	Tags      []string  // 标签
	TypeName  string    // 分类名
	SourceURL string    // 详情页链接
	Raw       []byte    // 原始 JSON，落 comic_events.raw_data
}

// Options 是单次抓取的参数，零值即取默认。
type Options struct {
	MaxPages int           // 最多翻页数，0 => DefaultMaxPages
	PageSize int           // 每页条数，0 => 源默认；bilibili 会被限制在 20 以内
	Timeout  time.Duration // 整体超时，0 => DefaultTimeout
}

// Timeout 返回生效的超时。
func (o Options) effectiveTimeout(def time.Duration) time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return def
}

// effectiveMaxPages 返回生效的最大翻页数。
func (o Options) effectiveMaxPages(def int) int {
	if o.MaxPages > 0 {
		return o.MaxPages
	}
	return def
}

// effectivePageSize 返回生效的每页条数，并夹在 [1, max] 区间。
func (o Options) effectivePageSize(def, max int) int {
	size := o.PageSize
	if size <= 0 {
		size = def
	}
	if max > 0 && size > max {
		size = max
	}
	if size <= 0 {
		size = 1
	}
	return size
}

// Source 是一个外部抓取源。
type Source interface {
	// Name 返回源标识（allcpp | bilibili）。
	Name() string
	// Fetch 抓取全量条目。全量失败返回 error；部分页失败返回已拿到的数据，
	// 失败原因记录在日志与 Result.Report.Warnings 里（Fetch 本身不返回 error）。
	Fetch(ctx context.Context, opt Options) ([]EventItem, error)
}

// Report 记录一次抓取的运行过程，供编排层/数据源状态页展示。
type Report struct {
	Source     string   // 源标识
	Pages      int      // 实际翻了几页（含失败的页）
	Total      int      // 上游声明的 total
	PageCount  int      // 上游声明的 pageCount
	Fetched    int      // 解析到的条目数（未去重、未过滤）
	Duplicates int      // 因重复 id 被丢掉的条目数
	Filtered   int      // 因分类黑名单/白名单被丢掉的条目数
	Invalid    int      // 无法解析或缺 id 被丢掉的条目数
	Kept       int      // 最终返回的条目数
	StopReason string   // 提前停/正常停的原因
	Warnings   []string // 部分页失败、未知分类等告警
}

// Result 是一次抓取的完整结果。
type Result struct {
	Items  []EventItem
	Report Report
}

// Reporter 由适配器额外实现：除了数据还能拿到抓取过程的诊断信息。
// 编排层应该优先用 FetchResult(ctx, src, opt) 而不是直接 src.Fetch()。
type Reporter interface {
	FetchResult(ctx context.Context, opt Options) (Result, error)
}

// FetchResult 取一次带诊断信息的抓取结果。
// 若适配器没有实现 Reporter，则退化为 Fetch 并补一个最小 Report。
func FetchResult(ctx context.Context, src Source, opt Options) (Result, error) {
	if r, ok := src.(Reporter); ok {
		return r.FetchResult(ctx, opt)
	}
	items, err := src.Fetch(ctx, opt)
	res := Result{Items: items, Report: Report{Source: src.Name(), Kept: len(items)}}
	if err != nil {
		res.Report.StopReason = "失败: " + err.Error()
		res.Report.Warnings = append(res.Report.Warnings, err.Error())
	}
	return res, err
}

// All 返回默认优先级顺序的抓取源：先 allcpp（数据更全），失败再切 bilibili。
func All() []Source {
	return []Source{NewAllcpp(), NewBilibili()}
}

// ---------------------------------------------------------------------------
// 规范化工具
// ---------------------------------------------------------------------------

// NormalizeCity 去掉城市名尾缀「市」「省」「自治区」等，得到「广州」这种短名。
func NormalizeCity(s string) string {
	s = strings.TrimSpace(s)
	for _, suffix := range []string{"特别行政区", "自治区", "自治州", "地区", "市", "省", "县", "区"} {
		if strings.HasSuffix(s, suffix) && len([]rune(s)) > len([]rune(suffix)) {
			s = strings.TrimSuffix(s, suffix)
			break
		}
	}
	return strings.TrimSpace(s)
}

// NormalizeProvince 去掉省名尾缀。
func NormalizeProvince(s string) string {
	s = strings.TrimSpace(s)
	for _, suffix := range []string{"特别行政区", "自治区", "省", "市"} {
		if strings.HasSuffix(s, suffix) && len([]rune(s)) > len([]rune(suffix)) {
			s = strings.TrimSuffix(s, suffix)
			break
		}
	}
	return strings.TrimSpace(s)
}

// NormalizeCover 补全封面直链：//x.jpg => https://x.jpg，/2026/7/x.jpg 加 base 前缀。
// 返回空字符串表示上游没给封面。
func NormalizeCover(raw, basePrefix string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return v
	case strings.HasPrefix(v, "//"):
		return "https:" + v
	case basePrefix != "":
		return strings.TrimRight(basePrefix, "/") + "/" + strings.TrimLeft(v, "/")
	default:
		return ""
	}
}

// SplitTags 把「a|b|c」这类管道分隔的标签拆成切片。
func SplitTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// millisToTime 把毫秒时间戳转成北京时间；0/负数返回零值。
func millisToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).In(shanghai)
}

// unixSecToTime 把秒时间戳转成北京时间；0/负数返回零值。
func unixSecToTime(sec int64) time.Time {
	if sec <= 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).In(shanghai)
}

// dateLayouts 是上游可能给出的日期格式（按 Asia/Shanghai 解析成本地 00:00）。
var dateLayouts = []string{
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006/01/02",
	"2006-01-02T15:04:05Z07:00",
	time.RFC3339,
}

// parseDate 解析上游的日期字符串；失败返回零值。
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, shanghai); err == nil {
			return t
		}
	}
	return time.Time{}
}

// logf 统一带包名前缀打日志，方便编排层 grep。
func logf(format string, args ...any) {
	log.Printf("eventsource: "+format, args...)
}
