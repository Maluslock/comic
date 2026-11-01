package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// stubEventSourceStore 是数据源状态接口的假仓储：
// 三个方法各自可注入错误，用来分别验证「哪一步挂了都必须是 500 而不是半截数据」。
type stubEventSourceStore struct {
	latest    []repository.SourceRun
	runs      []repository.SourceRun
	counts    []repository.SourceCount
	latestErr error
	runsErr   error
	countErr  error
}

func (s *stubEventSourceStore) LatestRunPerSource(_ context.Context) ([]repository.SourceRun, error) {
	if s.latestErr != nil {
		return nil, s.latestErr
	}
	return s.latest, nil
}

func (s *stubEventSourceStore) RecentRuns(_ context.Context, _ int) ([]repository.SourceRun, error) {
	if s.runsErr != nil {
		return nil, s.runsErr
	}
	return s.runs, nil
}

func (s *stubEventSourceStore) CountBySource(_ context.Context) ([]repository.SourceCount, error) {
	if s.countErr != nil {
		return nil, s.countErr
	}
	return s.counts, nil
}

func newAdminEventSourceHandlerForTest(s *stubEventSourceStore) *AdminEventSourceHandler {
	return NewAdminEventSourceHandler(service.NewEventSourceService(s))
}

func TestAdminEventSourceHandler_EventSources_OK(t *testing.T) {
	started := time.Now().Add(-2 * time.Hour)
	msg := "抓取完成"
	store := &stubEventSourceStore{
		latest: []repository.SourceRun{{
			ID: 7, Source: repository.SourceAllcpp, StartedAt: started, Status: repository.RunStatusOK,
			Items: 51, Inserted: 3, Updated: 48, Message: &msg,
		}},
		// 两次 RecentRuns 调用（表格 50 条 + 7 天窗口 500 条）共用同一份假数据。
		runs: []repository.SourceRun{
			{ID: 7, Source: repository.SourceAllcpp, StartedAt: started, Status: repository.RunStatusOK, Items: 51},
			{ID: 6, Source: repository.SourceBilibili, StartedAt: started.Add(-1 * time.Hour), Status: repository.RunStatusFailed},
		},
		counts: []repository.SourceCount{
			{Source: repository.SourceAllcpp, Total: 206, Active: 200},
			{Source: repository.SourceBilibili, Total: 42, Active: 42},
		},
	}

	rec, c := newGinContext("GET", "/api/admin/v1/event-sources", nil)
	newAdminEventSourceHandlerForTest(store).EventSources(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Sources []struct {
			Source        string  `json:"source"`
			LastStatus    string  `json:"lastStatus"`
			LastItems     int32   `json:"lastItems"`
			SuccessRate7d float64 `json:"successRate7d"`
			RunCount7d    int     `json:"runCount7d"`
			TotalItems    int64   `json:"totalItems"`
			ActiveItems   int64   `json:"activeItems"`
		} `json:"sources"`
		Runs   []repository.SourceRun `json:"runs"`
		Totals map[string]int64       `json:"totals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v (%s)", err, rec.Body.String())
	}

	// 白名单里的四个源必须全部出现（没数据的补零），顺序与 repository.EventSources 一致。
	if len(body.Sources) != len(repository.EventSources) {
		t.Fatalf("want %d sources, got %d (%s)", len(repository.EventSources), len(body.Sources), rec.Body.String())
	}
	for i, want := range repository.EventSources {
		if body.Sources[i].Source != want {
			t.Fatalf("source[%d]: want %q, got %q", i, want, body.Sources[i].Source)
		}
	}
	if body.Sources[0].LastStatus != repository.RunStatusOK {
		t.Fatalf("allcpp lastStatus: want ok, got %q", body.Sources[0].LastStatus)
	}
	if body.Sources[0].LastItems != 51 {
		t.Fatalf("allcpp lastItems: want 51, got %d", body.Sources[0].LastItems)
	}
	if body.Sources[0].TotalItems != 206 || body.Sources[0].ActiveItems != 200 {
		t.Fatalf("allcpp counts: want 206/200, got %d/%d", body.Sources[0].TotalItems, body.Sources[0].ActiveItems)
	}
	// allcpp 1 轮成功 → 1.0；bilibili 1 轮失败 → 0；nyato/manual 没跑过 → 0 且 runCount 0。
	if body.Sources[0].SuccessRate7d != 1 {
		t.Fatalf("allcpp successRate7d: want 1, got %v", body.Sources[0].SuccessRate7d)
	}
	if body.Sources[1].SuccessRate7d != 0 {
		t.Fatalf("bilibili successRate7d: want 0, got %v", body.Sources[1].SuccessRate7d)
	}
	if body.Sources[2].RunCount7d != 0 || body.Sources[2].LastStatus != "" {
		t.Fatalf("nyato should be zero-valued, got %+v", body.Sources[2])
	}
	if len(body.Runs) != 2 {
		t.Fatalf("want 2 runs, got %d", len(body.Runs))
	}
	if body.Totals[repository.SourceAllcpp] != 206 || body.Totals[repository.SourceBilibili] != 42 {
		t.Fatalf("totals mismatch: %v", body.Totals)
	}
}

func TestAdminEventSourceHandler_EventSources_EmptyIsNotNull(t *testing.T) {
	rec, c := newGinContext("GET", "/api/admin/v1/event-sources", nil)
	newAdminEventSourceHandlerForTest(&stubEventSourceStore{}).EventSources(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	// runs 必须是 []，不能是 null —— 前端 NDataTable 拿到 null 会直接报错。
	if string(body["runs"]) != "[]" {
		t.Fatalf("runs must be [] when empty, got %s", string(body["runs"]))
	}
	if string(body["totals"]) != "{}" {
		t.Fatalf("totals must be {} when empty, got %s", string(body["totals"]))
	}
}

func TestAdminEventSourceHandler_EventSources_StoreErrorIs500(t *testing.T) {
	cases := []struct {
		name  string
		store *stubEventSourceStore
	}{
		{"latest fails", &stubEventSourceStore{latestErr: errors.New("boom")}},
		{"runs fail", &stubEventSourceStore{runsErr: errors.New("boom")}},
		{"counts fail", &stubEventSourceStore{countErr: errors.New("boom")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, c := newGinContext("GET", "/api/admin/v1/event-sources", nil)
			newAdminEventSourceHandlerForTest(tc.store).EventSources(c)

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("want 500, got %d (%s)", rec.Code, rec.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			if body["error"] != "event sources unavailable" {
				t.Fatalf("unexpected error body: %s", rec.Body.String())
			}
		})
	}
}
