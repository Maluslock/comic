package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMapToParams_StatusMapping(t *testing.T) {
	s := &EventSyncer{cdnBase: "https://imagecdn3.allcpp.cn/upload"}

	tests := []struct {
		enabled  int
		expected string
	}{
		{0, "upcoming"},
		{5, "cancelled"},
		{6, "ended"},
		{7, "ended"},
	}

	for _, tt := range tests {
		e := AllcppEvent{Enabled: tt.enabled}
		params := s.mapToParams(e)
		if params.Status != tt.expected {
			t.Errorf("enabled=%d: expected status=%q, got %q", tt.enabled, tt.expected, params.Status)
		}
	}
}

func TestMapToParams_TagParsing(t *testing.T) {
	s := &EventSyncer{cdnBase: "https://imagecdn3.allcpp.cn/upload"}

	e := AllcppEvent{Tag: "义炭|你的距离|"}
	params := s.mapToParams(e)

	if len(params.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(params.Tags), params.Tags)
	}
	if params.Tags[0] != "义炭" {
		t.Errorf("expected tag[0]=义炭, got %q", params.Tags[0])
	}
	if params.Tags[1] != "你的距离" {
		t.Errorf("expected tag[1]=你的距离, got %q", params.Tags[1])
	}
}

func TestMapToParams_CoverUrl(t *testing.T) {
	s := &EventSyncer{cdnBase: "https://imagecdn3.allcpp.cn/upload"}

	// Relative path
	e := AllcppEvent{AppLogoPicUrl: "/2026/5/test.png"}
	params := s.mapToParams(e)
	expected := "https://imagecdn3.allcpp.cn/upload/2026/5/test.png"
	if params.CoverUrl == nil {
		t.Fatal("expected non-nil CoverUrl")
	}
	if *params.CoverUrl != expected {
		t.Errorf("expected cover=%q, got %q", expected, *params.CoverUrl)
	}

	// Absolute URL (should pass through)
	e2 := AllcppEvent{AppLogoPicUrl: "https://example.com/img.png"}
	params2 := s.mapToParams(e2)
	if params2.CoverUrl == nil {
		t.Fatal("expected non-nil CoverUrl for absolute URL")
	}
	if *params2.CoverUrl != "https://example.com/img.png" {
		t.Errorf("expected cover=%q, got %q", "https://example.com/img.png", *params2.CoverUrl)
	}

	// Empty URL (should be nil)
	e3 := AllcppEvent{AppLogoPicUrl: ""}
	params3 := s.mapToParams(e3)
	if params3.CoverUrl != nil {
		t.Errorf("expected nil CoverUrl for empty input, got %q", *params3.CoverUrl)
	}
}

func TestMapToParams_Location(t *testing.T) {
	s := &EventSyncer{cdnBase: "https://imagecdn3.allcpp.cn/upload"}

	e := AllcppEvent{CityName: "广州", ProvName: "广东"}
	params := s.mapToParams(e)
	if params.Location == nil {
		t.Fatal("expected non-nil Location")
	}
	if *params.Location != "广东广州" {
		t.Errorf("expected location=广东广州, got %q", *params.Location)
	}
}

func TestMapToParams_TimestampConversion(t *testing.T) {
	s := &EventSyncer{cdnBase: "https://imagecdn3.allcpp.cn/upload"}

	ts := int64(1783699200000) // milliseconds
	e := AllcppEvent{EnterTime: ts, EndTime: ts}
	params := s.mapToParams(e)

	expected := time.UnixMilli(ts)
	if !params.StartDate.Equal(expected) {
		t.Errorf("expected start=%v, got %v", expected, params.StartDate)
	}
	if !params.EndDate.Equal(expected) {
		t.Errorf("expected end=%v, got %v", expected, params.EndDate)
	}
}

func TestMapToParams_RawData(t *testing.T) {
	s := &EventSyncer{cdnBase: "https://imagecdn3.allcpp.cn/upload"}

	e := AllcppEvent{ID: 6803, Name: "广州义炭ONLY", CityName: "广州"}
	params := s.mapToParams(e)

	if len(params.RawData) == 0 {
		t.Error("expected non-empty RawData")
	}

	// Verify RawData round-trips
	var restored AllcppEvent
	if err := json.Unmarshal(params.RawData, &restored); err != nil {
		t.Fatalf("failed to unmarshal RawData: %v", err)
	}
	if restored.Name != "广州义炭ONLY" {
		t.Errorf("expected Name=广州义炭ONLY, got %q", restored.Name)
	}
}

func TestFetchPage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := allcppResponse{
			Result: struct {
				PageCount   int           `json:"pageCount"`
				Total       int           `json:"total"`
				CurrentPage int           `json:"currentPage"`
				List        []AllcppEvent `json:"list"`
			}{
				PageCount:   1,
				Total:       2,
				CurrentPage: 1,
				List: []AllcppEvent{
					{ID: 6803, Name: "广州义炭ONLY", CityName: "广州", ProvName: "广东", EnterTime: 1783699200000, EndTime: 1783699200000, Tag: "义炭|"},
					{ID: 7032, Name: "你的距离主题店", CityName: "上海", ProvName: "上海", EnterTime: 1781884800000, EndTime: 1784044800000, Tag: "你的距离|"},
				},
			},
			IsSuccess: true,
			Message:   "有记录",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	s := &EventSyncer{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	resp, err := s.fetchPage(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("fetchPage failed: %v", err)
	}

	if len(resp.Result.List) != 2 {
		t.Errorf("expected 2 events, got %d", len(resp.Result.List))
	}
	if resp.Result.List[0].Name != "广州义炭ONLY" {
		t.Errorf("expected first event=广州义炭ONLY, got %q", resp.Result.List[0].Name)
	}
}

func TestFetchPage_ApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(allcppResponse{IsSuccess: false, Message: "错误"})
	}))
	defer server.Close()

	s := &EventSyncer{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := s.fetchPage(context.Background(), 1, 50)
	if err == nil {
		t.Error("expected error for failed API response")
	}
}

func TestFetchPage_PaginationParams(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		resp := allcppResponse{
			Result: struct {
				PageCount   int           `json:"pageCount"`
				Total       int           `json:"total"`
				CurrentPage int           `json:"currentPage"`
				List        []AllcppEvent `json:"list"`
			}{
				PageCount:   1,
				Total:       0,
				CurrentPage: 3,
				List:        []AllcppEvent{},
			},
			IsSuccess: true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	s := &EventSyncer{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	_, err := s.fetchPage(context.Background(), 3, 25)
	if err != nil {
		t.Fatalf("fetchPage failed: %v", err)
	}

	if !contains(capturedURL, "pageindex=3") || !contains(capturedURL, "pagesize=25") {
		t.Errorf("expected pageindex=3 and pagesize=25 in URL, got %q", capturedURL)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
