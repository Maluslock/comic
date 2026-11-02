package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
)

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

// 仓储按场次降序返回什么，service 就必须原样透出什么 —— 排序是仓储的职责，
// service 不许再排一次（再排会掩盖仓储 SQL 写错的情况）。
func TestCityService_List_PreservesOrder(t *testing.T) {
	store := &stubCityStore{rows: []repository.CityCount{
		{Name: "上海", EventCount: 48},
		{Name: "杭州", EventCount: 45},
		{Name: "广州", EventCount: 37},
	}}
	got, err := NewCityService(store).List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Total != 3 || len(got.List) != 3 {
		t.Fatalf("want 3 items, got total=%d len=%d", got.Total, len(got.List))
	}
	want := []string{"上海", "杭州", "广州"}
	for i, name := range want {
		if got.List[i].Name != name {
			t.Fatalf("index %d: want %q, got %q", i, name, got.List[i].Name)
		}
	}
	if got.List[0].EventCount != 48 {
		t.Fatalf("上海 eventCount: want 48, got %d", got.List[0].EventCount)
	}
}

// 没有任何在办漫展时，list 必须是 []（不是 null），total 是 0。
// 前端城市弹层会直接 v-for，null 会在小程序端抛 "Cannot read property 'length' of null"。
func TestCityService_List_EmptyIsEmptySlice(t *testing.T) {
	got, err := NewCityService(&stubCityStore{}).List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.List == nil {
		t.Fatal("list must be an empty slice, not nil")
	}
	if len(got.List) != 0 || got.Total != 0 {
		t.Fatalf("want empty, got total=%d len=%d", got.Total, len(got.List))
	}
}

func TestCityService_List_StoreErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewCityService(&stubCityStore{err: boom}).List(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("want boom, got %v", err)
	}
}
