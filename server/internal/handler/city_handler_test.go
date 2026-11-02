package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// stubCityStore 是城市接口的假仓储（service.cityStore 的实现）。
type stubCityStore struct {
	rows []repository.CityCount
	err  error
}

func (s *stubCityStore) List(_ context.Context) ([]repository.CityCount, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

func TestCityHandler_List_OK(t *testing.T) {
	store := &stubCityStore{rows: []repository.CityCount{
		{Name: "上海", EventCount: 48},
		{Name: "杭州", EventCount: 45},
	}}
	rec, c := newGinContext("GET", "/api/v1/cities", nil)
	NewCityHandler(service.NewCityService(store), nil).List(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var body service.CityListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v (%s)", err, rec.Body.String())
	}
	if body.Total != 2 || len(body.List) != 2 {
		t.Fatalf("want 2 items, got total=%d len=%d (%s)", body.Total, len(body.List), rec.Body.String())
	}
	if body.List[0].Name != "上海" || body.List[0].EventCount != 48 {
		t.Fatalf("unexpected first item: %+v", body.List[0])
	}
}

// 空库时 list 必须是 [] 而不是 null —— 前端城市弹层直接 v-for。
func TestCityHandler_List_EmptyIsNotNull(t *testing.T) {
	rec, c := newGinContext("GET", "/api/v1/cities", nil)
	NewCityHandler(service.NewCityService(&stubCityStore{}), nil).List(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if string(raw["list"]) != "[]" {
		t.Fatalf("list must be [] when empty, got %s", string(raw["list"]))
	}
	if string(raw["total"]) != "0" {
		t.Fatalf("total must be 0 when empty, got %s", string(raw["total"]))
	}
}

func TestCityHandler_List_StoreErrorIs500(t *testing.T) {
	rec, c := newGinContext("GET", "/api/v1/cities", nil)
	NewCityHandler(service.NewCityService(&stubCityStore{err: errors.New("boom")}), nil).List(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d (%s)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["error"] != "internal server error" {
		t.Fatalf("unexpected error body: %s", rec.Body.String())
	}
}
