package eventsource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// bilibili 会员购票务适配器。
//
// 实测事实（2026-11-01 本机验证，不要凭直觉"优化"掉）：
//   - 列表接口 https://show.bilibili.com/api/ticket/project/listV2?version=134&page=N&pagesize=20&area=0&filter=&platform=web
//   - pagesize 上限 20：pagesize=30 直接返回 {code:81102084,message:"请求非法，请稍后重试"}，
//     所以这里把 pageSize 硬夹在 20。
//   - page=2 起返回的是同一批数据（page1 20 条唯一，page2 与之完全重复）→ 同样按 id 去重，
//     并在「本页没有新 id」时提前停。
//   - name 字段是 null，真实标题在 project_name；project_id 就是 id。
//   - city 形如「广州市」，要按 NormalizeCity 去尾缀；district_name 形如「白云区」。
//   - start_time / end_time 是 "YYYY-MM-DD" 字符串，按 Asia/Shanghai 解析成当地 00:00；
//     start_unix 是秒级时间戳（详情时间，比 start_time 精确，优先用）。
//   - cover 以 // 开头，要补 https:。
//   - 分类用 third_category_name 白名单过滤：漫展 / Only同人展 / 其他展览 / 同好会；
//     黑名单（主题餐厅、观影会、音乐节、演唱会、电竞赛事、其他演出、其他活动）丢弃。
//     实测 20 条里主题餐厅占 9 条，必须丢掉。
const (
	bilibiliDefaultBaseURL = "https://show.bilibili.com"
	bilibiliListPath       = "/api/ticket/project/listV2"
	bilibiliDetailFormat   = "https://show.bilibili.com/platform/detail.html?id=%d"
	bilibiliDefaultReferer = "https://show.bilibili.com/"
	bilibiliVersion        = "134"
	bilibiliArea           = "0"
	bilibiliPlatform       = "web"
)

// bilibiliCategoryAllow 是保留的分类（漫展相关）。
var bilibiliCategoryAllow = map[string]struct{}{
	"漫展":      {},
	"Only同人展": {},
	"其他展览":    {},
	"同好会":     {},
}

// bilibiliCategoryDeny 是明确丢弃的分类（餐厅/演出/赛事，不是漫展）。
// 白名单已经足够，黑名单只用于日志里把"有意丢弃"和"未知分类"区分开。
var bilibiliCategoryDeny = map[string]struct{}{
	"主题餐厅": {},
	"观影会":  {},
	"音乐节":  {},
	"演唱会":  {},
	"电竞赛事": {},
	"其他演出": {},
	"其他活动": {},
}

// BilibiliConfig 覆盖 Bilibili 适配器的默认端点与节奏，零值即取默认。
type BilibiliConfig struct {
	BaseURL       string
	HTTPClient    *http.Client
	RetryBackoffs []time.Duration
	PageDelay     time.Duration
}

// Bilibili 是 bilibili 会员购漫展数据源。
type Bilibili struct {
	baseURL   string
	client    *http.Client
	backoffs  []time.Duration
	pageDelay time.Duration
}

// NewBilibili 用默认配置构造 bilibili 适配器。
func NewBilibili() *Bilibili { return NewBilibiliWithConfig(BilibiliConfig{}) }

// NewBilibiliWithConfig 用给定配置构造 bilibili 适配器。
func NewBilibiliWithConfig(cfg BilibiliConfig) *Bilibili {
	b := &Bilibili{
		baseURL:   strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		client:    cfg.HTTPClient,
		backoffs:  backoffsOrDefault(cfg.RetryBackoffs),
		pageDelay: cfg.PageDelay,
	}
	if b.baseURL == "" {
		b.baseURL = bilibiliDefaultBaseURL
	}
	if b.client == nil {
		b.client = defaultHTTPClient(DefaultTimeout)
	}
	return b
}

// Name 实现 Source。
func (b *Bilibili) Name() string { return SourceBilibili }

// Fetch 实现 Source。
func (b *Bilibili) Fetch(ctx context.Context, opt Options) ([]EventItem, error) {
	res, err := b.FetchResult(ctx, opt)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// bilibiliListResponse 是列表接口的响应信封。
type bilibiliListResponse struct {
	Errno   int    `json:"errno"`
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
	Data    struct {
		Total      int               `json:"total"`
		NumResults int               `json:"numResults"`
		Page       int               `json:"page"`
		PageSize   int               `json:"pagesize"`
		Result     []json.RawMessage `json:"result"`
	} `json:"data"`
}

// bilibiliProject 只声明用得到的字段；完整 JSON 留在 EventItem.Raw。
type bilibiliProject struct {
	ID                int64   `json:"id"`
	ProjectID         int64   `json:"project_id"`
	ProjectName       string  `json:"project_name"`
	Name              *string `json:"name"` // 实测为 null，真实标题在 project_name
	City              string  `json:"city"`
	DistrictName      string  `json:"district_name"`
	VenueName         string  `json:"venue_name"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
	StartUnix         int64   `json:"start_unix"`
	Cover             string  `json:"cover"`
	ThirdCategoryName string  `json:"third_category_name"`
	JumpURL           string  `json:"jump_url"`
	Label             string  `json:"label"`
	TLabel            string  `json:"tlabel"`
}

// listURL 拼出第 page 页的请求地址。pagesize 由调用方夹到 20。
func (b *Bilibili) listURL(page, pageSize int) string {
	q := url.Values{}
	q.Set("version", bilibiliVersion)
	q.Set("page", strconv.Itoa(page))
	q.Set("pagesize", strconv.Itoa(pageSize))
	q.Set("area", bilibiliArea)
	q.Set("filter", "")
	q.Set("platform", bilibiliPlatform)
	return b.baseURL + bilibiliListPath + "?" + q.Encode()
}

// headers 返回 bilibili 需要的请求头（缺 UA/Referer 会被拦）。
func (b *Bilibili) headers() map[string]string {
	h := browserHeaders(BrowserUA, bilibiliDefaultReferer)
	h["Origin"] = b.baseURL
	return h
}

// FetchResult 抓取全量条目并返回诊断信息。
func (b *Bilibili) FetchResult(ctx context.Context, opt Options) (Result, error) {
	timeout := opt.effectiveTimeout(DefaultTimeout)
	maxPages := opt.effectiveMaxPages(DefaultMaxPages)
	// pagesize=30 会被上游判非法（code 81102084），必须夹到 20。
	pageSize := opt.effectivePageSize(DefaultBilibiliPageSize, MaxBilibiliPageSize)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rep := Report{Source: SourceBilibili}
	items := make([]EventItem, 0, pageSize)
	seen := make(map[int64]struct{}, pageSize)
	unknown := make(map[string]int)

	for page := 1; page <= maxPages; page++ {
		if page > 1 {
			if err := sleepCtx(ctx, pageDelay(b.pageDelay)); err != nil {
				rep.StopReason = fmt.Sprintf("第 %d 页前等待被中断，提前停止: %v", page, err)
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				logf("bilibili: %s", rep.StopReason)
				break
			}
		}

		reqURL := b.listURL(page, pageSize)
		body, err := getWithRetry(ctx, b.client, reqURL, b.headers(), b.backoffs)
		rep.Pages++
		if err != nil {
			if page == 1 {
				rep.StopReason = "首页请求失败: " + err.Error()
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				logf("bilibili: %s", rep.StopReason)
				return Result{Report: rep}, fmt.Errorf("bilibili 抓取失败（第 1 页）: %w", err)
			}
			rep.StopReason = fmt.Sprintf("第 %d 页请求失败，已保留前 %d 页 %d 条: %v", page, page-1, len(items), err)
			rep.Warnings = append(rep.Warnings, rep.StopReason)
			logf("bilibili: %s", rep.StopReason)
			return Result{Items: items, Report: rep}, nil
		}

		var resp bilibiliListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			if page == 1 {
				rep.StopReason = "首页响应解析失败: " + err.Error()
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				return Result{Report: rep}, fmt.Errorf("bilibili 响应解析失败（第 1 页）: %w", err)
			}
			rep.StopReason = fmt.Sprintf("第 %d 页响应解析失败，已保留前 %d 页 %d 条: %v", page, page-1, len(items), err)
			rep.Warnings = append(rep.Warnings, rep.StopReason)
			logf("bilibili: %s", rep.StopReason)
			return Result{Items: items, Report: rep}, nil
		}
		if resp.Code != 0 || resp.Errno != 0 {
			msg := firstNonEmpty(resp.Message, resp.Msg, "code="+strconv.Itoa(resp.Code))
			if page == 1 {
				rep.StopReason = "首页返回失败: " + msg
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				return Result{Report: rep}, fmt.Errorf("bilibili 接口返回失败（第 1 页）: %s", msg)
			}
			rep.StopReason = fmt.Sprintf("第 %d 页返回失败，已保留前 %d 页 %d 条: %s", page, page-1, len(items), msg)
			rep.Warnings = append(rep.Warnings, rep.StopReason)
			logf("bilibili: %s", rep.StopReason)
			return Result{Items: items, Report: rep}, nil
		}

		if page == 1 {
			rep.Total = resp.Data.Total
		}

		pageNew := 0
		for _, raw := range resp.Data.Result {
			rep.Fetched++
			item, id, keep, reason := b.parseItem(raw)
			// 记录非白名单分类的分布，方便状态页看「上游有多少非漫展内容」。
			if !keep && reason == "" {
				if cat := categoryOf(raw); cat != "" {
					if _, ok := bilibiliCategoryDeny[cat]; !ok {
						unknown[cat]++
					}
				}
			}
			if id != 0 {
				if _, dup := seen[id]; dup {
					rep.Duplicates++
					continue
				}
				seen[id] = struct{}{}
			}
			switch {
			case reason != "":
				rep.Invalid++
				rep.Warnings = append(rep.Warnings, fmt.Sprintf("第 %d 页跳过 1 条: %s", page, reason))
				continue
			case !keep:
				rep.Filtered++
				continue
			}
			items = append(items, item)
			pageNew++
		}

		logf("bilibili: 第 %d/%d 页取回 %d 条，新增 %d 条，累计 %d 条（过滤 %d 条）",
			page, maxPages, len(resp.Data.Result), pageNew, len(items), rep.Filtered)

		switch {
		case len(resp.Data.Result) == 0:
			rep.StopReason = fmt.Sprintf("第 %d 页返回空列表，提前停止", page)
		case pageNew == 0:
			// 上游 page>=2 会重复下发同一批（实测 page2 与 page1 完全重复），没新 id 就停。
			rep.StopReason = fmt.Sprintf("第 %d 页没有新 id（上游翻页重复下发），提前停止", page)
		case page == maxPages:
			rep.StopReason = fmt.Sprintf("达到 MaxPages=%d 上限", maxPages)
		}
		if rep.StopReason != "" {
			logf("bilibili: %s", rep.StopReason)
			break
		}
	}
	if rep.StopReason == "" {
		rep.StopReason = fmt.Sprintf("达到 MaxPages=%d 上限", maxPages)
	}
	for cat, n := range unknown {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("未知分类 %q 共 %d 条，已按非漫展过滤", cat, n))
	}

	rep.Kept = len(items)
	return Result{Items: items, Report: rep}, nil
}

// parseItem 解析一条 bilibili 项目。
// id 为 0 表示这条连 id 都没有（计入 Invalid）；keep=false 表示分类被过滤（计入 Filtered）。
func (b *Bilibili) parseItem(raw json.RawMessage) (item EventItem, id int64, keep bool, reason string) {
	var p bilibiliProject
	if err := json.Unmarshal(raw, &p); err != nil {
		return EventItem{}, 0, false, "解析条目 JSON 失败: " + err.Error()
	}
	id = p.ProjectID
	if id == 0 {
		id = p.ID
	}
	if id <= 0 {
		return EventItem{}, 0, false, "条目缺少 project_id"
	}

	category := strings.TrimSpace(p.ThirdCategoryName)
	if _, ok := bilibiliCategoryAllow[category]; !ok {
		// 白名单之外一律丢弃：主题餐厅/观影会/演唱会等实测占大头，必须挡掉。
		return EventItem{}, id, false, ""
	}

	name := strings.TrimSpace(p.ProjectName)
	if name == "" && p.Name != nil {
		name = strings.TrimSpace(*p.Name)
	}
	if name == "" {
		return EventItem{}, id, false, fmt.Sprintf("project_id=%d 条目缺少标题", id)
	}

	// start_unix 是精确到场时刻（秒），start_time 是日期字符串；优先用后者解析成当地 00:00
	// 以与 allcpp 口径一致，缺失时退回 start_unix。
	start := parseDate(p.StartTime)
	if start.IsZero() {
		start = unixSecToTime(p.StartUnix)
	}
	if start.IsZero() {
		return EventItem{}, id, false, fmt.Sprintf("project_id=%d 条目缺少开始时间", id)
	}
	end := parseDate(p.EndTime)
	if end.IsZero() {
		end = start
	}

	return EventItem{
		Source:   SourceBilibili,
		AllcppID: int32(id),
		Name:     name,
		// City 用去尾缀的短名（广州市 => 广州）供列表展示；Address 保留上游原文
		// （杭州市 萧山区 xxx），与 allcpp 的拼接口径一致，便于直接展示给人看。
		City:      NormalizeCity(p.City),
		Province:  "", // 上游只给城市/区，不给省
		Venue:     strings.TrimSpace(p.VenueName),
		Address:   joinNonEmpty(p.City, p.DistrictName, p.VenueName),
		Start:     start,
		End:       end,
		CoverURL:  NormalizeCover(p.Cover, ""),
		Tags:      bilibiliTags(category, p.Label),
		TypeName:  category,
		SourceURL: fmt.Sprintf(bilibiliDetailFormat, id),
		Raw:       append([]byte(nil), raw...),
	}, id, true, ""
}

// categoryOf 取一条原始 JSON 的 third_category_name（失败返回空）。
func categoryOf(raw json.RawMessage) string {
	var p struct {
		ThirdCategoryName string `json:"third_category_name"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return ""
	}
	return strings.TrimSpace(p.ThirdCategoryName)
}

// bilibiliTags 组合标签：分类名 + 上游角标。
func bilibiliTags(category, label string) []string {
	tags := make([]string, 0, 2)
	if category != "" {
		tags = append(tags, category)
	}
	if l := strings.TrimSpace(label); l != "" && l != category {
		tags = append(tags, l)
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}
