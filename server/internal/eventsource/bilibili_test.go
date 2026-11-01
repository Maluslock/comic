package eventsource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// bilibiliProjectItem 造桩用的条目，字段名与真实接口一致。
// 注意：真实响应里 name 是 null，标题在 project_name。
func bilibiliProjectItem(projectID int, projectName, city, district, venue, startTime, endTime, cover, category, label string) map[string]any {
	return map[string]any{
		"id":                  projectID,
		"project_id":          projectID,
		"project_name":        projectName,
		"name":                nil,
		"city":                city,
		"district_name":       district,
		"venue_name":          venue,
		"start_time":          startTime,
		"end_time":            endTime,
		"start_unix":          0,
		"cover":               cover,
		"third_category_name": category,
		"label":               label,
		"jump_url":            "https://mall.bilibili.com/neul-next/ticket-renovation/detail.html?id=" + strconv.Itoa(projectID),
	}
}

// bilibiliEnvelope 包一层真实接口的响应信封。
func bilibiliEnvelope(page, pageSize, total int, result []map[string]any) map[string]any {
	if result == nil {
		result = []map[string]any{}
	}
	return map[string]any{
		"errno":   0,
		"code":    0,
		"msg":     "success",
		"message": "success",
		"data": map[string]any{
			"total":      total,
			"numResults": len(result),
			"page":       page,
			"pagesize":   pageSize,
			"result":     result,
		},
	}
}

// newBilibiliServer 按页号返回预设数据，pages[0] 对应 page=1。
// 返回的 requestedPages 记录每次请求的 page/pagesize，便于断言「pagesize 恒为 20」。
func newBilibiliServer(t *testing.T, pages []map[string]any) (*httptest.Server, *atomic.Int32, *[]string) {
	t.Helper()
	var hits atomic.Int32
	var reqs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != bilibiliListPath {
			http.NotFound(w, r)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page <= 0 {
			page = 1
		}
		reqs = append(reqs, "page="+strconv.Itoa(page)+"&pagesize="+r.URL.Query().Get("pagesize"))
		hits.Add(1)
		if page > len(pages) {
			_ = json.NewEncoder(w).Encode(bilibiliEnvelope(page, 20, 0, nil))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pages[page-1])
	}))
	t.Cleanup(srv.Close)
	return srv, &hits, &reqs
}

func newTestBilibili(t *testing.T, srv *httptest.Server) *Bilibili {
	t.Helper()
	return NewBilibiliWithConfig(BilibiliConfig{
		BaseURL:       srv.URL,
		RetryBackoffs: []time.Duration{}, // 测试不等待退避
		PageDelay:     -1,                // 翻页不 sleep
	})
}

// TestBilibiliCategoryFilter 覆盖验收③：主题餐厅被丢、漫展保留。
func TestBilibiliCategoryFilter(t *testing.T) {
	page1 := []map[string]any{
		bilibiliProjectItem(1001, "广州·CICF漫展", "广州市", "海珠区", "保利世贸博览馆", "2026-11-14", "2026-11-16", "//i0.hdslb.com/a.jpg", "漫展", "独家"),
		bilibiliProjectItem(1002, "上海·主题餐厅联名", "上海市", "黄浦区", "某商场", "2026-11-01", "2026-11-30", "//i0.hdslb.com/b.jpg", "主题餐厅", ""),
		bilibiliProjectItem(1003, "北京·Only同人展", "北京市", "朝阳区", "朝阳公园", "2026-11-21", "2026-11-21", "//i0.hdslb.com/c.jpg", "Only同人展", ""),
		bilibiliProjectItem(1004, "深圳·演唱会", "深圳市", "南山区", "春茧体育馆", "2026-12-01", "2026-12-01", "//i0.hdslb.com/d.jpg", "演唱会", ""),
		bilibiliProjectItem(1005, "成都·同好会", "成都市", "武侯区", "世纪城", "2026-12-05", "2026-12-05", "", "同好会", ""),
	}
	srv, _, reqs := newBilibiliServer(t, []map[string]any{bilibiliEnvelope(1, 20, 5, page1)})
	src := newTestBilibili(t, srv)

	res, err := src.FetchResult(context.Background(), Options{})
	if err != nil {
		t.Fatalf("FetchResult 失败: %v", err)
	}

	if got, want := len(res.Items), 3; got != want {
		t.Fatalf("保留条数 = %d, want %d（漫展/Only同人展/同好会保留，主题餐厅与演唱会丢弃）", got, want)
	}
	wantIDs := []int32{1001, 1003, 1005}
	for i, want := range wantIDs {
		if res.Items[i].AllcppID != want {
			t.Fatalf("Items[%d].AllcppID = %d, want %d", i, res.Items[i].AllcppID, want)
		}
		if res.Items[i].Source != SourceBilibili {
			t.Fatalf("Items[%d].Source = %q, want %q", i, res.Items[i].Source, SourceBilibili)
		}
	}
	// 主题餐厅与演唱会必须被丢掉。
	for _, it := range res.Items {
		if it.Name == "上海·主题餐厅联名" {
			t.Fatalf("主题餐厅不应保留")
		}
		if it.Name == "深圳·演唱会" {
			t.Fatalf("演唱会不应保留")
		}
	}
	if res.Report.Filtered != 2 {
		t.Fatalf("Report.Filtered = %d, want 2", res.Report.Filtered)
	}
	if res.Report.Kept != 3 {
		t.Fatalf("Report.Kept = %d, want 3", res.Report.Kept)
	}
	// pagesize 必须恒为 20（pagesize=30 会被上游判非法）。
	for _, r := range *reqs {
		if !strings.HasSuffix(r, "pagesize=20") {
			t.Fatalf("请求参数 %q 的 pagesize 必须为 20", r)
		}
	}
}

// TestBilibiliCoverAndCityNormalization 覆盖验收④⑤：
// cover 补 https:、city 去「市」尾缀、标题取 project_name。
func TestBilibiliCoverAndCityNormalization(t *testing.T) {
	page1 := []map[string]any{
		bilibiliProjectItem(2001, "杭州·CP30", "杭州市", "萧山区", "杭州国际博览中心", "2026-11-07", "2026-11-08", "//i0.hdslb.com/bfs/arch/cover.jpg", "漫展", ""),
		bilibiliProjectItem(2002, "武汉·同人展", "武汉市", "江汉区", "武汉国际会展中心", "2026-12-12", "2026-12-12", "https://i0.hdslb.com/full.png", "其他展览", ""),
	}
	srv, _, _ := newBilibiliServer(t, []map[string]any{bilibiliEnvelope(1, 20, 2, page1)})
	src := newTestBilibili(t, srv)

	items, err := src.Fetch(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("条目数 = %d, want 2", len(items))
	}

	// ④ cover 以 // 开头时补 https:。
	if got, want := items[0].CoverURL, "https://i0.hdslb.com/bfs/arch/cover.jpg"; got != want {
		t.Fatalf("CoverURL = %q, want %q", got, want)
	}
	if !strings.HasPrefix(items[0].CoverURL, "https://") {
		t.Fatalf("CoverURL 必须以 https:// 开头, got %q", items[0].CoverURL)
	}
	// 已经是 https 的原样保留。
	if got, want := items[1].CoverURL, "https://i0.hdslb.com/full.png"; got != want {
		t.Fatalf("CoverURL = %q, want %q", got, want)
	}

	// ⑤ city 去「市」尾缀。
	if got, want := items[0].City, "杭州"; got != want {
		t.Fatalf("City = %q, want %q", got, want)
	}
	if got, want := items[1].City, "武汉"; got != want {
		t.Fatalf("City = %q, want %q", got, want)
	}

	// 标题必须取 project_name（name 字段是 null）。
	if items[0].Name != "杭州·CP30" {
		t.Fatalf("Name = %q, want 杭州·CP30", items[0].Name)
	}
	// 时间按 Asia/Shanghai 解析成当地 00:00。
	if got, want := items[0].Start.In(Shanghai()).Format("2006-01-02 15:04"), "2026-11-07 00:00"; got != want {
		t.Fatalf("Start = %s, want %s", got, want)
	}
	if got, want := items[0].End.In(Shanghai()).Format("2006-01-02 15:04"), "2026-11-08 00:00"; got != want {
		t.Fatalf("End = %s, want %s", got, want)
	}
	// SourceURL 用详情页。
	if got, want := items[0].SourceURL, "https://show.bilibili.com/platform/detail.html?id=2001"; got != want {
		t.Fatalf("SourceURL = %q, want %q", got, want)
	}
	if items[0].Venue != "杭州国际博览中心" {
		t.Fatalf("Venue = %q, want 杭州国际博览中心", items[0].Venue)
	}
	if !strings.Contains(items[0].Address, "杭州市") || !strings.Contains(items[0].Address, "萧山区") {
		t.Fatalf("Address = %q, 应包含城市与区", items[0].Address)
	}
	// Raw 必须是原始 JSON（落 raw_data 用）。
	var probe struct {
		ProjectID int `json:"project_id"`
	}
	if err := json.Unmarshal(items[0].Raw, &probe); err != nil {
		t.Fatalf("Raw 不是合法 JSON: %v", err)
	}
	if probe.ProjectID != 2001 {
		t.Fatalf("Raw.project_id = %d, want 2001", probe.ProjectID)
	}
}

// TestBilibiliDedupAndEarlyStop 覆盖验收①：page2 与 page1 完全重复 → 去重且提前停。
func TestBilibiliDedupAndEarlyStop(t *testing.T) {
	page1 := []map[string]any{
		bilibiliProjectItem(3001, "广州·漫展A", "广州市", "海珠区", "保利世贸", "2026-11-14", "2026-11-16", "", "漫展", ""),
		bilibiliProjectItem(3002, "广州·漫展B", "广州市", "天河区", "琶洲", "2026-11-20", "2026-11-21", "", "Only同人展", ""),
	}
	// 真实上游 page>=2 会重复下发同一批数据。
	page2 := []map[string]any{
		bilibiliProjectItem(3001, "广州·漫展A", "广州市", "海珠区", "保利世贸", "2026-11-14", "2026-11-16", "", "漫展", ""),
		bilibiliProjectItem(3002, "广州·漫展B", "广州市", "天河区", "琶洲", "2026-11-20", "2026-11-21", "", "Only同人展", ""),
	}
	page3 := []map[string]any{
		bilibiliProjectItem(3999, "不该被请求到", "广州市", "海珠区", "某馆", "2026-11-30", "2026-11-30", "", "漫展", ""),
	}
	srv, hits, _ := newBilibiliServer(t, []map[string]any{
		bilibiliEnvelope(1, 20, 21, page1),
		bilibiliEnvelope(2, 20, 21, page2),
		bilibiliEnvelope(3, 20, 21, page3),
	})
	src := newTestBilibili(t, srv)

	res, err := src.FetchResult(context.Background(), Options{MaxPages: 10})
	if err != nil {
		t.Fatalf("FetchResult 失败: %v", err)
	}
	if got, want := len(res.Items), 2; got != want {
		t.Fatalf("去重后条数 = %d, want %d", got, want)
	}
	if got := int(hits.Load()); got != 2 {
		t.Fatalf("实际请求页数 = %d, want 2（第 2 页无新 id 必须提前停）", got)
	}
	if res.Report.Duplicates != 2 {
		t.Fatalf("Report.Duplicates = %d, want 2", res.Report.Duplicates)
	}
	if !strings.Contains(res.Report.StopReason, "没有新 id") {
		t.Fatalf("StopReason = %q, 应说明「没有新 id」", res.Report.StopReason)
	}
	if res.Report.Total != 21 {
		t.Fatalf("Report.Total = %d, want 21", res.Report.Total)
	}
}

// TestBilibiliForbidden 覆盖验收②：HTTP 403 → 返回 error。
func TestBilibiliForbidden(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	t.Cleanup(srv.Close)

	src := newTestBilibili(t, srv)
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
}

// TestBilibiliUpstreamCodeError 验证上游业务错误码（如 pagesize 非法）被当成失败。
func TestBilibiliUpstreamCodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":81102084,"message":"请求非法，请稍后重试","data":{}}`))
	}))
	t.Cleanup(srv.Close)

	src := newTestBilibili(t, srv)
	_, err := src.Fetch(context.Background(), Options{})
	if err == nil {
		t.Fatalf("上游 code!=0 必须返回 error")
	}
	if !strings.Contains(err.Error(), "请求非法") {
		t.Fatalf("error 应带上游 message, got %q", err.Error())
	}
}

// TestBilibiliPartialPageFailure 验证部分页失败返回已拿到的数据 + 记录原因。
func TestBilibiliPartialPageFailure(t *testing.T) {
	page1 := []map[string]any{
		bilibiliProjectItem(4001, "南京·漫展", "南京市", "建邺区", "南京国际博览中心", "2026-11-21", "2026-11-22", "", "漫展", ""),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page >= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("upstream down"))
			return
		}
		_ = json.NewEncoder(w).Encode(bilibiliEnvelope(1, 20, 40, page1))
	}))
	t.Cleanup(srv.Close)

	src := newTestBilibili(t, srv)
	res, err := src.FetchResult(context.Background(), Options{})
	if err != nil {
		t.Fatalf("部分页失败不应返回 error, got %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("应保留第 1 页的 1 条, got %d", len(res.Items))
	}
	if len(res.Report.Warnings) == 0 {
		t.Fatalf("Report.Warnings 应记录部分页失败原因")
	}
	if !strings.Contains(strings.Join(res.Report.Warnings, "|"), "500") {
		t.Fatalf("Warnings 应包含 500, got %v", res.Report.Warnings)
	}
}

// TestBilibiliMaxPageSizeClamped 验证 pageSize 被硬夹到 20。
func TestBilibiliMaxPageSizeClamped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("pagesize"); got != "20" {
			t.Errorf("pagesize = %q, want 20", got)
		}
		_ = json.NewEncoder(w).Encode(bilibiliEnvelope(1, 20, 0, nil))
	}))
	t.Cleanup(srv.Close)

	src := newTestBilibili(t, srv)
	if _, err := src.Fetch(context.Background(), Options{PageSize: 100}); err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
}
