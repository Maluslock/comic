package eventsource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// allcppItem 是造桩用的条目构造器，字段名与真实接口一致。
func allcppItem(id int, name, city, prov, area, address, typeName string, enterMs, endMs int64, cover, tag string, online int) map[string]any {
	return map[string]any{
		"id":            id,
		"name":          name,
		"cityName":      city,
		"provName":      prov,
		"areaName":      area,
		"enterAddress":  address,
		"enterPoint":    "",
		"enterTime":     enterMs,
		"endTime":       endMs,
		"appLogoPicUrl": cover,
		"logoPicUrl":    "",
		"mLogPicUrl":    "",
		"tag":           tag,
		"typeName":      typeName,
		"isOnline":      online,
	}
}

// shanghaiMillis 把 "2026-11-07" 这类日期转成北京时间当天 00:00 的毫秒时间戳。
func shanghaiMillis(t *testing.T, date string) int64 {
	t.Helper()
	tm, err := time.ParseInLocation("2006-01-02", date, Shanghai())
	if err != nil {
		t.Fatalf("parse date %q: %v", date, err)
	}
	return tm.UnixMilli()
}

// allcppEnvelope 包一层真实接口的响应信封。
func allcppEnvelope(pageCount, total, currentPage int, list []map[string]any) map[string]any {
	if list == nil {
		list = []map[string]any{}
	}
	return map[string]any{
		"result": map[string]any{
			"pageCount":   pageCount,
			"total":       total,
			"currentPage": currentPage,
			"list":        list,
		},
		"isSuccess": true,
		"message":   "有记录",
	}
}

// newAllcppServer 起一个按页号返回预设数据的桩，pages[0] 对应 pageindex=1。
// hint 记录每页实际请求的 UA / 页号，便于断言。
func newAllcppServer(t *testing.T, pages []map[string]any) (*httptest.Server, *atomic.Int32, *[]string) {
	t.Helper()
	var hits atomic.Int32
	var uas []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != allcppListPath {
			http.NotFound(w, r)
			return
		}
		uas = append(uas, r.Header.Get("User-Agent"))
		page, _ := strconv.Atoi(r.URL.Query().Get("pageindex"))
		if page <= 0 {
			page = 1
		}
		hits.Add(1)
		if page > len(pages) {
			// 真实上游翻过尾页会返回空列表。
			_ = json.NewEncoder(w).Encode(allcppEnvelope(len(pages), 0, page, nil))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pages[page-1])
	}))
	t.Cleanup(srv.Close)
	return srv, &hits, &uas
}

func newTestAllcpp(t *testing.T, srv *httptest.Server) *Allcpp {
	t.Helper()
	return NewAllcppWithConfig(AllcppConfig{
		BaseURL:       srv.URL,
		ImageBaseURL:  allcppDefaultImageBase,
		RetryBackoffs: []time.Duration{}, // 测试不等待重试退避
		PageDelay:     -1,                // 翻页不 sleep
	})
}

// TestAllcppDedupAndEarlyStop 覆盖验收①：两页数据、第二页整页重复
// → 去重后条数正确，且在第 2 页就提前停（不再翻第 3/4 页）。
func TestAllcppDedupAndEarlyStop(t *testing.T) {
	enter := shanghaiMillis(t, "2026-11-07")
	end := shanghaiMillis(t, "2026-11-08")
	page1 := []map[string]any{
		allcppItem(101, "上海·CP31", "上海", "上海", "浦东新区", "上海新国际博览中心", "综合同人展", enter, end, "/2026/9/a.png", "原创|漫画|", 0),
		allcppItem(102, "广州·笔至量力", "广州", "广东", "白云", "西城智汇park", "综合同人展", enter, end, "/2026/9/b.png", "", 0),
		allcppItem(103, "北京·Only展", "北京", "北京", "朝阳", "朝阳公园", "ONLY", enter, end, "", "", 0),
	}
	// 第 2 页与第 1 页完全相同（真实上游第 4 页全重复），应当 0 新增并停下。
	page2 := []map[string]any{
		allcppItem(101, "上海·CP31", "上海", "上海", "浦东新区", "上海新国际博览中心", "综合同人展", enter, end, "/2026/9/a.png", "原创|漫画|", 0),
		allcppItem(102, "广州·笔至量力", "广州", "广东", "白云", "西城智汇park", "综合同人展", enter, end, "/2026/9/b.png", "", 0),
		allcppItem(103, "北京·Only展", "北京", "北京", "朝阳", "朝阳公园", "ONLY", enter, end, "", "", 0),
	}
	page3 := []map[string]any{
		allcppItem(999, "不该被请求到", "杭州", "浙江", "西湖", "某某馆", "综合展", enter, end, "", "", 0),
	}
	pages := []map[string]any{
		allcppEnvelope(4, 200, 1, page1),
		allcppEnvelope(4, 200, 2, page2),
		allcppEnvelope(4, 200, 3, page3),
	}
	srv, hits, uas := newAllcppServer(t, pages)
	src := newTestAllcpp(t, srv)

	res, err := src.FetchResult(context.Background(), Options{MaxPages: 10})
	if err != nil {
		t.Fatalf("FetchResult 返回错误: %v", err)
	}

	if got, want := len(res.Items), 3; got != want {
		t.Fatalf("去重后条数 = %d, want %d", got, want)
	}
	if got := int(hits.Load()); got != 2 {
		t.Fatalf("实际请求页数 = %d, want 2（第 2 页无新 id 必须提前停）", got)
	}
	if res.Report.Duplicates != 3 {
		t.Fatalf("Report.Duplicates = %d, want 3", res.Report.Duplicates)
	}
	if res.Report.Pages != 2 {
		t.Fatalf("Report.Pages = %d, want 2", res.Report.Pages)
	}
	if !strings.Contains(res.Report.StopReason, "没有新 id") {
		t.Fatalf("StopReason = %q, 应说明「没有新 id」", res.Report.StopReason)
	}
	if res.Report.Total != 200 || res.Report.PageCount != 4 {
		t.Fatalf("Total/PageCount = %d/%d, want 200/4", res.Report.Total, res.Report.PageCount)
	}
	if res.Report.Kept != 3 {
		t.Fatalf("Report.Kept = %d, want 3", res.Report.Kept)
	}
	// id 顺序保持上游顺序，AllcppID 用真实 id。
	wantIDs := []int32{101, 102, 103}
	for i, want := range wantIDs {
		if res.Items[i].AllcppID != want {
			t.Fatalf("Items[%d].AllcppID = %d, want %d", i, res.Items[i].AllcppID, want)
		}
		if res.Items[i].Source != SourceAllcpp {
			t.Fatalf("Items[%d].Source = %q, want %q", i, res.Items[i].Source, SourceAllcpp)
		}
	}
	// 必须带浏览器 UA（实测不带直接 403）。
	for i, ua := range *uas {
		if ua != BrowserUA {
			t.Fatalf("第 %d 次请求 UA = %q, want %q", i+1, ua, BrowserUA)
		}
	}
	// 时间按 Asia/Shanghai 解析成本地 00:00。
	if got, want := res.Items[0].Start.In(Shanghai()).Format("2006-01-02 15:04"), "2026-11-07 00:00"; got != want {
		t.Fatalf("Items[0].Start = %s, want %s", got, want)
	}
	if got, want := res.Items[0].End.In(Shanghai()).Format("2006-01-02 15:04"), "2026-11-08 00:00"; got != want {
		t.Fatalf("Items[0].End = %s, want %s", got, want)
	}
}

// TestAllcppForbidden 覆盖验收②：HTTP 403 → 返回 error（编排层据此切备用源）。
func TestAllcppForbidden(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	t.Cleanup(srv.Close)

	src := newTestAllcpp(t, srv)
	items, err := src.Fetch(context.Background(), Options{})
	if err == nil {
		t.Fatalf("403 必须返回 error，实际返回 items=%d, err=nil", len(items))
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error 里应包含状态码 403, got %q", err.Error())
	}
	if hits.Load() != 1 {
		t.Fatalf("无重试退避时只应请求 1 次, got %d", hits.Load())
	}

	// 带默认退避时会重试 2 次（共 3 次请求），但为了不拖慢测试这里只验证首屏失败语义。
	res, ferr := src.FetchResult(context.Background(), Options{})
	if ferr == nil {
		t.Fatalf("FetchResult 也必须返回 error")
	}
	if len(res.Items) != 0 {
		t.Fatalf("失败时不应返回条目, got %d", len(res.Items))
	}
	if res.Report.Source != SourceAllcpp {
		t.Fatalf("Report.Source = %q, want %q", res.Report.Source, SourceAllcpp)
	}
}

// TestAllcppRetriesOnTransientError 验证「HTTP 非 200 重试 2 次」的语义：
// 前两次 500、第三次 200 → 最终成功。
func TestAllcppRetriesOnTransientError(t *testing.T) {
	var hits atomic.Int32
	enter := shanghaiMillis(t, "2026-12-05")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
			return
		}
		_ = json.NewEncoder(w).Encode(allcppEnvelope(1, 1, 1, []map[string]any{
			allcppItem(201, "成都·同人展", "成都", "四川", "武侯", "世纪城", "综合展", enter, enter, "", "", 0),
		}))
	}))
	t.Cleanup(srv.Close)

	src := NewAllcppWithConfig(AllcppConfig{
		BaseURL:       srv.URL,
		RetryBackoffs: []time.Duration{time.Millisecond, time.Millisecond},
		PageDelay:     -1,
	})
	items, err := src.Fetch(context.Background(), Options{})
	if err != nil {
		t.Fatalf("重试后应成功, got err=%v", err)
	}
	if len(items) != 1 {
		t.Fatalf("条目数 = %d, want 1", len(items))
	}
	if hits.Load() != 3 {
		t.Fatalf("请求次数 = %d, want 3（1 次原始 + 2 次重试）", hits.Load())
	}
}

// TestAllcppPartialPageFailure 验证「部分页失败返回已拿到的数据 + 记录原因」。
func TestAllcppPartialPageFailure(t *testing.T) {
	enter := shanghaiMillis(t, "2026-11-14")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("pageindex"))
		if page >= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("upstream down"))
			return
		}
		_ = json.NewEncoder(w).Encode(allcppEnvelope(4, 200, 1, []map[string]any{
			allcppItem(301, "南京·Only", "南京", "江苏", "建邺", "博览中心", "ONLY", enter, enter, "", "", 0),
			allcppItem(302, "杭州·综合展", "杭州", "浙江", "萧山", "国博", "综合展", enter, enter, "", "", 0),
		}))
	}))
	t.Cleanup(srv.Close)

	src := newTestAllcpp(t, srv)
	res, err := src.FetchResult(context.Background(), Options{})
	if err != nil {
		t.Fatalf("部分页失败不应返回 error, got %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("应保留第 1 页的 2 条, got %d", len(res.Items))
	}
	if len(res.Report.Warnings) == 0 {
		t.Fatalf("Report.Warnings 应记录部分页失败原因")
	}
	if !strings.Contains(strings.Join(res.Report.Warnings, "|"), "500") {
		t.Fatalf("Warnings 应包含 500, got %v", res.Report.Warnings)
	}
}

// TestAllcppCoverAndCityNormalization 覆盖验收④⑤：封面拼 CDN 前缀、city 去「市」尾缀、
// 线上活动保留且 TypeName 归一。
func TestAllcppCoverAndCityNormalization(t *testing.T) {
	enter := shanghaiMillis(t, "2026-11-21")
	srv, _, _ := newAllcppServer(t, []map[string]any{
		allcppEnvelope(1, 3, 1, []map[string]any{
			allcppItem(401, "西安·漫展", "西安市", "陕西省", "雁塔区", "曲江会展中心", "综合展", enter, enter, "/2026/7/cover.jpg", "", 0),
			allcppItem(402, "线上·绘画接龙", "", "", "", "", "ONLY", enter, enter, "", "", 1),
			allcppItem(403, "深圳·同人展", "深圳市", "广东省", "南山区", "深圳湾", "综合同人展", enter, enter, "//cdn.example.com/x.png", "", 0),
		}),
	})
	src := newTestAllcpp(t, srv)

	items, err := src.Fetch(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("条目数 = %d, want 3（线上活动不能被过滤）", len(items))
	}

	// ④ 相对路径封面必须拼 https://imagecdn3.allcpp.cn/upload 前缀。
	if got, want := items[0].CoverURL, allcppDefaultImageBase+"/2026/7/cover.jpg"; got != want {
		t.Fatalf("CoverURL = %q, want %q", got, want)
	}
	if !strings.HasPrefix(items[0].CoverURL, "https://") {
		t.Fatalf("CoverURL 必须以 https:// 开头, got %q", items[0].CoverURL)
	}
	// // 开头的也要补 https:。
	if got, want := items[2].CoverURL, "https://cdn.example.com/x.png"; got != want {
		t.Fatalf("CoverURL = %q, want %q", got, want)
	}

	// ⑤ city 去「市」尾缀。
	if got, want := items[0].City, "西安"; got != want {
		t.Fatalf("City = %q, want %q", got, want)
	}
	if got, want := items[2].City, "深圳"; got != want {
		t.Fatalf("City = %q, want %q", got, want)
	}
	if got, want := items[0].Province, "陕西"; got != want {
		t.Fatalf("Province = %q, want %q", got, want)
	}

	// 线上活动：保留，TypeName 统一「线上活动」。
	if got, want := items[1].TypeName, allcppOnlineTypeName; got != want {
		t.Fatalf("线上活动 TypeName = %q, want %q", got, want)
	}

	// SourceURL 用 /d/{id}.do（eventMain.do 会 302 到登录页，不可用）。
	if got, want := items[0].SourceURL, "https://www.allcpp.cn/d/401.do"; got != want {
		t.Fatalf("SourceURL = %q, want %q", got, want)
	}

	// Raw 必须是原始 JSON（落 raw_data 用）。
	var probe struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(items[0].Raw, &probe); err != nil {
		t.Fatalf("Raw 不是合法 JSON: %v", err)
	}
	if probe.ID != 401 {
		t.Fatalf("Raw.id = %d, want 401", probe.ID)
	}
	if items[0].Address == "" {
		t.Fatalf("Address 不应为空（省市区+场馆拼接）")
	}
	if !strings.Contains(items[0].Address, "西安市") || !strings.Contains(items[0].Address, "曲江会展中心") {
		t.Fatalf("Address = %q, 应包含城市与场馆", items[0].Address)
	}
	if items[0].Venue != "曲江会展中心" {
		t.Fatalf("Venue = %q, want 曲江会展中心", items[0].Venue)
	}
}

// TestAllcppInvalidRowsSkipped 验证缺 id / 缺时间的脏数据被丢弃并计入 Invalid。
func TestAllcppInvalidRowsSkipped(t *testing.T) {
	enter := shanghaiMillis(t, "2026-11-28")
	srv, _, _ := newAllcppServer(t, []map[string]any{
		allcppEnvelope(1, 3, 1, []map[string]any{
			allcppItem(0, "没有 id", "上海", "上海", "", "", "综合展", enter, enter, "", "", 0),
			allcppItem(501, "没有时间", "上海", "上海", "", "", "综合展", 0, 0, "", "", 0),
			allcppItem(502, "正常条目", "上海", "上海", "", "", "综合展", enter, enter, "", "", 0),
		}),
	})
	src := newTestAllcpp(t, srv)

	res, err := src.FetchResult(context.Background(), Options{})
	if err != nil {
		t.Fatalf("FetchResult 失败: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].AllcppID != 502 {
		t.Fatalf("应只保留正常条目 502, got %+v", res.Items)
	}
	if res.Report.Invalid != 2 {
		t.Fatalf("Report.Invalid = %d, want 2", res.Report.Invalid)
	}
}

// TestAll 验证默认优先级顺序：allcpp 先，bilibili 后。
func TestAll(t *testing.T) {
	sources := All()
	if len(sources) != 2 {
		t.Fatalf("All() 长度 = %d, want 2", len(sources))
	}
	if sources[0].Name() != SourceAllcpp {
		t.Fatalf("All()[0] = %q, want %q", sources[0].Name(), SourceAllcpp)
	}
	if sources[1].Name() != SourceBilibili {
		t.Fatalf("All()[1] = %q, want %q", sources[1].Name(), SourceBilibili)
	}
	// 两个适配器都要实现 Reporter，编排层才能读到 Report。
	for _, s := range sources {
		if _, ok := s.(Reporter); !ok {
			t.Fatalf("%s 未实现 Reporter", s.Name())
		}
	}
	_ = fmt.Sprint(sources[0])
}
