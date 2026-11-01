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

// allcpp（无差别同人站）适配器。
//
// 实测事实（2026-11-01 本机验证，不要凭直觉"优化"掉）：
//   - 列表接口 https://www.allcpp.cn/allcpp/event/getList.do?order=1&sort=0&pageindex=N&pagesize=50
//   - 不带浏览器 UA 直接 403，必须带 UA；Referer 可带可不带（这里带上更稳）。
//   - 返回 {"result":{pageCount,total,currentPage,list[]},"isSuccess":true,"message":"有记录"}。
//   - total=200 / pageCount=4，但第 4 页与第 3 页有 48 条重叠，实测 4 页共 200 行只有
//     100 个唯一 id。所以必须按 id 去重，并且「本页没有新 id」就提前停，不能盲翻 pageCount。
//   - 封面 appLogoPicUrl 形如 /2026/9/xxx.png，必须拼 https://imagecdn3.allcpp.cn/upload 前缀
//     （不带 /upload 是 404，已实测）。
//   - enterTime / endTime 是毫秒时间戳，换算到 Asia/Shanghai 后都是当天 00:00（实测 100/100）。
//   - isOnline=1 的线上活动保留，但 TypeName 统一写成「线上活动」。
//
// 详情页：/allcpp/event/eventMain.do?eventId=N 会 302 到 cp.allcpp.cn 登录页（不可用）；
// /d/N.do 可正常 200，但只是个通用 SPA 壳（没有活动正文）。两者都不可用的情况下，
// SourceURL 取 /d/N.do——它至少不带登录跳转，用户点进去能落在站内。
const (
	allcppDefaultBaseURL   = "https://www.allcpp.cn"
	allcppListPath         = "/allcpp/event/getList.do"
	allcppDefaultImageBase = "https://imagecdn3.allcpp.cn/upload"
	allcppDetailURLFormat  = "https://www.allcpp.cn/d/%d.do"
	allcppDefaultReferer   = "https://www.allcpp.cn/"
	allcppMaxPageSize      = 100
	allcppOnlineTypeName   = "线上活动"
	allcppDefaultTypeName  = "综合"
	allcppOrderParam       = "1"
	allcppSortParam        = "0"
)

// AllcppConfig 覆盖 Allcpp 适配器的默认端点与节奏，零值即取默认。
// 生产代码用 NewAllcpp()；单元测试用 httptest.Server 的 URL 填 BaseURL。
type AllcppConfig struct {
	// BaseURL 网站根地址，默认 https://www.allcpp.cn。
	BaseURL string
	// ImageBaseURL 封面图 CDN 前缀，默认 https://imagecdn3.allcpp.cn/upload。
	ImageBaseURL string
	// DetailURLFormat 详情页格式（含一个 %d），默认 https://www.allcpp.cn/d/%d.do。
	DetailURLFormat string
	// HTTPClient 自定义 client，nil 用内置（20s 单请求超时）。
	HTTPClient *http.Client
	// RetryBackoffs 重试退避；nil 用默认 [1s,3s]，空切片表示不重试不等待（测试用）。
	RetryBackoffs []time.Duration
	// PageDelay 翻页间隔；0 = 默认 300-500ms 随机，<0 = 不等待（测试用）。
	PageDelay time.Duration
}

// Allcpp 是 allcpp（无差别同人站）数据源。
type Allcpp struct {
	baseURL         string
	imageBase       string
	detailURLFormat string
	client          *http.Client
	backoffs        []time.Duration
	pageDelay       time.Duration
}

// NewAllcpp 用默认配置构造 allcpp 适配器。
func NewAllcpp() *Allcpp { return NewAllcppWithConfig(AllcppConfig{}) }

// NewAllcppWithConfig 用给定配置构造 allcpp 适配器。
func NewAllcppWithConfig(cfg AllcppConfig) *Allcpp {
	a := &Allcpp{
		baseURL:         strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		imageBase:       strings.TrimSpace(cfg.ImageBaseURL),
		detailURLFormat: strings.TrimSpace(cfg.DetailURLFormat),
		client:          cfg.HTTPClient,
		backoffs:        backoffsOrDefault(cfg.RetryBackoffs),
		pageDelay:       cfg.PageDelay,
	}
	if a.baseURL == "" {
		a.baseURL = allcppDefaultBaseURL
	}
	if a.imageBase == "" {
		a.imageBase = allcppDefaultImageBase
	}
	if a.detailURLFormat == "" {
		a.detailURLFormat = allcppDetailURLFormat
	}
	if a.client == nil {
		a.client = defaultHTTPClient(DefaultTimeout)
	}
	return a
}

// Name 实现 Source。
func (a *Allcpp) Name() string { return SourceAllcpp }

// Fetch 实现 Source：全量失败返回 error；部分页失败返回已拿到的数据（原因见 Report）。
func (a *Allcpp) Fetch(ctx context.Context, opt Options) ([]EventItem, error) {
	res, err := a.FetchResult(ctx, opt)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// allcppListResponse 是列表接口的响应信封。
type allcppListResponse struct {
	Result struct {
		PageCount   int               `json:"pageCount"`
		Total       int               `json:"total"`
		CurrentPage int               `json:"currentPage"`
		List        []json.RawMessage `json:"list"`
	} `json:"result"`
	IsSuccess bool   `json:"isSuccess"`
	Message   string `json:"message"`
}

// allcppEvent 只声明用得到的字段；其余 58 个字段留在 Raw 里。
type allcppEvent struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	CityName      string `json:"cityName"`
	ProvName      string `json:"provName"`
	AreaName      string `json:"areaName"`
	EnterAddress  string `json:"enterAddress"`
	EnterPoint    string `json:"enterPoint"`
	EnterTime     int64  `json:"enterTime"`
	EndTime       int64  `json:"endTime"`
	AppLogoPicURL string `json:"appLogoPicUrl"`
	LogoPicURL    string `json:"logoPicUrl"`
	MLogPicURL    string `json:"mLogPicUrl"`
	Tag           string `json:"tag"`
	TypeName      string `json:"typeName"`
	IsOnline      int    `json:"isOnline"`
}

// listURL 拼出第 page 页的请求地址。
func (a *Allcpp) listURL(page, pageSize int) string {
	q := url.Values{}
	q.Set("order", allcppOrderParam)
	q.Set("sort", allcppSortParam)
	q.Set("pageindex", strconv.Itoa(page))
	q.Set("pagesize", strconv.Itoa(pageSize))
	return a.baseURL + allcppListPath + "?" + q.Encode()
}

// detailURL 拼出详情页地址。
func (a *Allcpp) detailURL(id int64) string {
	return fmt.Sprintf(a.detailURLFormat, id)
}

// FetchResult 抓取全量条目并返回诊断信息。
func (a *Allcpp) FetchResult(ctx context.Context, opt Options) (Result, error) {
	timeout := opt.effectiveTimeout(DefaultTimeout)
	maxPages := opt.effectiveMaxPages(DefaultMaxPages)
	pageSize := opt.effectivePageSize(DefaultAllcppPageSize, allcppMaxPageSize)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rep := Report{Source: SourceAllcpp}
	items := make([]EventItem, 0, pageSize)
	seen := make(map[int64]struct{}, pageSize)

	for page := 1; page <= maxPages; page++ {
		if page > 1 {
			if err := sleepCtx(ctx, pageDelay(a.pageDelay)); err != nil {
				rep.StopReason = fmt.Sprintf("第 %d 页前等待被中断，提前停止: %v", page, err)
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				logf("allcpp: %s", rep.StopReason)
				break
			}
		}

		reqURL := a.listURL(page, pageSize)
		body, err := getWithRetry(ctx, a.client, reqURL, browserHeaders(BrowserUA, allcppDefaultReferer), a.backoffs)
		rep.Pages++
		if err != nil {
			// 首页就失败 = 整源失败，必须让编排层知道（好切备用源）。
			if page == 1 {
				rep.StopReason = "首页请求失败: " + err.Error()
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				logf("allcpp: %s", rep.StopReason)
				return Result{Report: rep}, fmt.Errorf("allcpp 抓取失败（第 1 页）: %w", err)
			}
			// 部分页失败：保留已拿到的数据，把原因记进 Report.Warnings。
			rep.StopReason = fmt.Sprintf("第 %d 页请求失败，已保留前 %d 页 %d 条: %v", page, page-1, len(items), err)
			rep.Warnings = append(rep.Warnings, rep.StopReason)
			logf("allcpp: %s", rep.StopReason)
			return Result{Items: items, Report: rep}, nil
		}

		var resp allcppListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			if page == 1 {
				rep.StopReason = "首页响应解析失败: " + err.Error()
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				return Result{Report: rep}, fmt.Errorf("allcpp 响应解析失败（第 1 页）: %w", err)
			}
			rep.StopReason = fmt.Sprintf("第 %d 页响应解析失败，已保留前 %d 页 %d 条: %v", page, page-1, len(items), err)
			rep.Warnings = append(rep.Warnings, rep.StopReason)
			logf("allcpp: %s", rep.StopReason)
			return Result{Items: items, Report: rep}, nil
		}
		if !resp.IsSuccess {
			msg := strings.TrimSpace(resp.Message)
			if msg == "" {
				msg = "isSuccess=false"
			}
			if page == 1 {
				rep.StopReason = "首页返回失败: " + msg
				rep.Warnings = append(rep.Warnings, rep.StopReason)
				return Result{Report: rep}, fmt.Errorf("allcpp 接口返回失败（第 1 页）: %s", msg)
			}
			rep.StopReason = fmt.Sprintf("第 %d 页返回失败，已保留前 %d 页 %d 条: %s", page, page-1, len(items), msg)
			rep.Warnings = append(rep.Warnings, rep.StopReason)
			logf("allcpp: %s", rep.StopReason)
			return Result{Items: items, Report: rep}, nil
		}

		if page == 1 {
			rep.Total = resp.Result.Total
			rep.PageCount = resp.Result.PageCount
		}

		pageNew := 0
		for _, raw := range resp.Result.List {
			rep.Fetched++
			item, id, err := a.parseItem(raw)
			if err != nil {
				rep.Invalid++
				rep.Warnings = append(rep.Warnings, fmt.Sprintf("第 %d 页跳过 1 条: %v", page, err))
				continue
			}
			if _, dup := seen[id]; dup {
				rep.Duplicates++
				continue
			}
			seen[id] = struct{}{}
			items = append(items, item)
			pageNew++
		}
		logf("allcpp: 第 %d/%d 页取回 %d 条，新增 %d 条，累计 %d 条",
			page, maxPages, len(resp.Result.List), pageNew, len(items))

		switch {
		case len(resp.Result.List) == 0:
			rep.StopReason = fmt.Sprintf("第 %d 页返回空列表，提前停止", page)
		case pageNew == 0:
			// 上游翻页会重复下发同一批数据（实测第 4 页全重复），没新 id 就没必要再翻。
			rep.StopReason = fmt.Sprintf("第 %d 页没有新 id（上游翻页重复下发），提前停止", page)
		case resp.Result.PageCount > 0 && page >= resp.Result.PageCount:
			rep.StopReason = fmt.Sprintf("已翻到上游 pageCount=%d", resp.Result.PageCount)
		case page == maxPages:
			rep.StopReason = fmt.Sprintf("达到 MaxPages=%d 上限", maxPages)
		}
		if rep.StopReason != "" {
			logf("allcpp: %s", rep.StopReason)
			break
		}
	}
	if rep.StopReason == "" {
		rep.StopReason = fmt.Sprintf("达到 MaxPages=%d 上限", maxPages)
	}

	rep.Kept = len(items)
	return Result{Items: items, Report: rep}, nil
}

// parseItem 把一条原始 JSON 解析成 EventItem。
func (a *Allcpp) parseItem(raw json.RawMessage) (EventItem, int64, error) {
	var ev allcppEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return EventItem{}, 0, fmt.Errorf("解析条目 JSON 失败: %w", err)
	}
	if ev.ID <= 0 {
		return EventItem{}, 0, fmt.Errorf("条目缺少 id")
	}
	name := strings.TrimSpace(ev.Name)
	if name == "" {
		return EventItem{}, 0, fmt.Errorf("id=%d 条目缺少活动名", ev.ID)
	}

	start := millisToTime(ev.EnterTime)
	if start.IsZero() {
		return EventItem{}, 0, fmt.Errorf("id=%d 条目缺少 enterTime", ev.ID)
	}
	end := millisToTime(ev.EndTime)

	city := NormalizeCity(ev.CityName)
	if city == "" {
		// 少量条目 cityName 为空；直辖市时用 provName 兜底（实测 provName 与 cityName 同名）。
		if p := NormalizeProvince(ev.ProvName); isMunicipality(p) {
			city = p
		}
	}

	cover := NormalizeCover(firstNonEmpty(ev.AppLogoPicURL, ev.LogoPicURL, ev.MLogPicURL), a.imageBase)

	typeName := strings.TrimSpace(ev.TypeName)
	if ev.IsOnline == 1 {
		// 线上活动保留，但统一标签，方便前端/状态页区分线下场次。
		typeName = allcppOnlineTypeName
	} else if typeName == "" {
		typeName = allcppDefaultTypeName
	}

	// enterAddress 就是场馆/场地名（实测 100/100 都有值，形如「西城智汇park」）；
	// enterPoint 是补充说明（实测全为空，非空时当作场馆补充）。
	venue := strings.TrimSpace(ev.EnterAddress)
	if p := strings.TrimSpace(ev.EnterPoint); p != "" {
		venue = firstNonEmpty(venue, p)
	}
	// Address 存完整地址：省 + 市 + 区 + 场馆，去重后空格拼接（与前端 venue/address 展示口径一致）。
	address := joinNonEmpty(ev.ProvName, ev.CityName, ev.AreaName, ev.EnterAddress, ev.EnterPoint)
	if address == "" {
		address = venue
	}

	tags := SplitTags(ev.Tag)
	if len(tags) == 0 && city != "" {
		tags = []string{city}
	}

	return EventItem{
		Source:    SourceAllcpp,
		AllcppID:  int32(ev.ID), // 实测 id 最大值 7599，int32 足够
		Name:      name,
		City:      city,
		Province:  NormalizeProvince(ev.ProvName),
		Venue:     venue,
		Address:   address,
		Start:     start,
		End:       end,
		CoverURL:  cover,
		Tags:      tags,
		TypeName:  typeName,
		SourceURL: a.detailURL(ev.ID),
		Raw:       append([]byte(nil), raw...),
	}, ev.ID, nil
}

// municipalities 是直辖市集合，用于 cityName 缺失时用 provName 兜底。
var municipalities = map[string]struct{}{
	"北京": {},
	"上海": {},
	"天津": {},
	"重庆": {},
}

// isMunicipality 判断是否直辖市（已归一化过的短名）。
func isMunicipality(name string) bool {
	_, ok := municipalities[name]
	return ok
}

// joinNonEmpty 用空格拼接非空部分，并去掉相邻重复（省=市时只留一个）。
func joinNonEmpty(values ...string) string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if len(out) > 0 && out[len(out)-1] == v {
			continue
		}
		dup := false
		for _, prev := range out {
			if prev == v {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		out = append(out, v)
	}
	return strings.Join(out, " ")
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
