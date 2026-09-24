package service

import (
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
)

func i32(v int32) *int32 { return &v }

func TestMergePriceSummaries_AttachesRollup(t *testing.T) {
	items := []PhotographerListItem{
		{PhotographerItem: PhotographerItem{ID: 1}},
		{PhotographerItem: PhotographerItem{ID: 2}},
	}
	sums := []repository.ServicePriceSummary{
		{PhotographerID: 1, MinPrice: i32(399), HasFree: true, HasNegotiable: true, ActiveCount: 3},
		{PhotographerID: 99, MinPrice: i32(1), ActiveCount: 1}, // 不在本页，忽略
	}
	got := mergePriceSummaries(items, sums)

	if got[0].MinPrice == nil || *got[0].MinPrice != 399 {
		t.Errorf("want MinPrice 399, got %v", got[0].MinPrice)
	}
	if !got[0].HasFree || !got[0].HasNegotiable {
		t.Errorf("want both flags set, got free=%v negotiable=%v", got[0].HasFree, got[0].HasNegotiable)
	}
	if got[0].ServiceCount != 3 {
		t.Errorf("want ServiceCount 3, got %d", got[0].ServiceCount)
	}
}

// 摄影师没有任何上架套餐时，聚合查询不会返回该行 —— 此时必须保持零值，
// 由前端显示「暂未设置」，而不是沿用上一位摄影师的数字。
func TestMergePriceSummaries_LeavesAbsentPhotographerZeroed(t *testing.T) {
	items := []PhotographerListItem{{PhotographerItem: PhotographerItem{ID: 7}}}
	got := mergePriceSummaries(items, nil)

	if got[0].MinPrice != nil || got[0].HasFree || got[0].HasNegotiable || got[0].ServiceCount != 0 {
		t.Errorf("want zero roll-up for a photographer with no active package, got %+v", got[0])
	}
}
