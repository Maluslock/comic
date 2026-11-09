package service

import (
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
)

// 互勉 hasFree 兜底的四象限（t6 口径）：
// ① free+0 套餐 → HasFree=true  ② both+0 套餐 → HasFree=true
// ③ both+有套餐 → HasFree=false ④ pay+0 套餐 → HasFree=false
// 三个出口（列表 mergePriceSummaries、首页与漫展详情共用 mergePhotographerItemPriceSummaries）
// 各跑一遍同一组用例，防止口径漂移。

var mutualHasFreeMatrix = []struct {
	id   int64
	mode string
	want bool
}{
	{1, "free", true},  // ①
	{2, "both", true},  // ②
	{3, "both", false}, // ③
	{4, "pay", false},  // ④
}

// 出口 1/3：摄影师列表（List → mergePriceSummaries）。
func TestMutualHasFree_ListOutlet(t *testing.T) {
	items := []PhotographerListItem{
		{PhotographerItem: PhotographerItem{ID: 1, Mode: "free"}}, // 0 套餐：不在 sums 里
		{PhotographerItem: PhotographerItem{ID: 2, Mode: "both"}}, // 0 套餐：不在 sums 里
		{PhotographerItem: PhotographerItem{ID: 3, Mode: "both"}}, // 有 2 个上架套餐
		{PhotographerItem: PhotographerItem{ID: 4, Mode: "pay"}},  // 0 套餐
	}
	sums := []repository.ServicePriceSummary{
		{PhotographerID: 3, MinPrice: i32(399), ActiveCount: 2, HasFree: false},
	}
	got := mergePriceSummaries(items, sums)
	if len(got) != 4 {
		t.Fatalf("want 4 items, got %d", len(got))
	}
	for _, it := range got {
		for _, c := range mutualHasFreeMatrix {
			if it.ID != c.id {
				continue
			}
			if it.HasFree != c.want {
				t.Errorf("id=%d mode=%s ServiceCount=%d: HasFree want %v got %v", it.ID, it.Mode, it.ServiceCount, c.want, it.HasFree)
			}
		}
	}
	if got[2].ServiceCount != 2 || got[2].MinPrice == nil || *got[2].MinPrice != 399 {
		t.Errorf("real roll-up must survive the fallback, got %+v", got[2])
	}
	if got[0].ServiceCount != 0 || got[1].ServiceCount != 0 {
		t.Errorf("0-package photographers must keep ServiceCount 0, got %d/%d", got[0].ServiceCount, got[1].ServiceCount)
	}
}

// 出口 2/3：首页推荐摄影师（GetHomeData → mergePhotographerItemPriceSummaries）。
func TestMutualHasFree_HomeOutlet(t *testing.T) {
	items := []PhotographerItem{
		{ID: 1, Mode: "free"},
		{ID: 2, Mode: "both"},
		{ID: 3, Mode: "both"},
		{ID: 4, Mode: "pay"},
	}
	sums := []repository.ServicePriceSummary{
		{PhotographerID: 3, MinPrice: i32(399), ActiveCount: 2, HasFree: false},
	}
	got := mergePhotographerItemPriceSummaries(items, sums)
	for _, it := range got {
		for _, c := range mutualHasFreeMatrix {
			if it.ID != c.id {
				continue
			}
			if it.HasFree != c.want {
				t.Errorf("id=%d mode=%s ServiceCount=%d: HasFree want %v got %v", it.ID, it.Mode, it.ServiceCount, c.want, it.HasFree)
			}
		}
	}
}

// 出口 3/3：漫展详情推荐摄影师（GetEventDetail → mergePhotographerItemPriceSummaries，
// 与首页共用同一实现，此处守住事件侧调用面）。
func TestMutualHasFree_EventOutlet(t *testing.T) {
	items := []PhotographerItem{
		{ID: 1, Mode: "free"},
		{ID: 2, Mode: "both"},
		{ID: 3, Mode: "both"},
		{ID: 4, Mode: "pay"},
	}
	sums := []repository.ServicePriceSummary{
		{PhotographerID: 3, ActiveCount: 1},
	}
	got := mergePhotographerItemPriceSummaries(items, sums)
	for _, it := range got {
		for _, c := range mutualHasFreeMatrix {
			if it.ID != c.id {
				continue
			}
			if it.HasFree != c.want {
				t.Errorf("id=%d mode=%s ServiceCount=%d: HasFree want %v got %v", it.ID, it.Mode, it.ServiceCount, c.want, it.HasFree)
			}
		}
	}
	// 兜底不得覆盖真实 0 元套餐的 roll-up（pay 摄影师有真免费套餐时仍为 true）。
	withFree := mergePhotographerItemPriceSummaries(
		[]PhotographerItem{{ID: 9, Mode: "pay"}},
		[]repository.ServicePriceSummary{{PhotographerID: 9, HasFree: true, ActiveCount: 1}},
	)
	if !withFree[0].HasFree {
		t.Error("summary HasFree=true must survive for pay mode with a real free package")
	}
}

// 出口 4/4：GetDetail 详情（applyDetailMutualHasFree，与 GetDetail 同一函数）。
// both + 有套餐 → false 是 captain 实测抓到的回归锁：缺 ServiceCount==0 守卫时
// GetDetail 会把 mode=both 的收费摄影师标成 hasFree=true，盖掉 minPrice（详情显示
// 「互勉」而列表显示「¥399 起」，口径漂移 = 放走付款）。
func TestMutualHasFree_DetailOutlet(t *testing.T) {
	for _, c := range mutualHasFreeMatrix {
		// 与 GetDetail 同一构造：ActiveCount 先取真实上架套餐数，有聚合行再覆盖。
		priceSummary := repository.ServicePriceSummary{ActiveCount: 0}
		if c.id == 3 {
			priceSummary = repository.ServicePriceSummary{ActiveCount: 2, MinPrice: i32(399)}
		}
		applyDetailMutualHasFree(c.mode, &priceSummary)
		if priceSummary.HasFree != c.want {
			t.Errorf("id=%d mode=%s ActiveCount=%d: HasFree want %v got %v", c.id, c.mode, priceSummary.ActiveCount, c.want, priceSummary.HasFree)
		}
		if c.id == 3 && (priceSummary.MinPrice == nil || *priceSummary.MinPrice != 399) {
			t.Errorf("both+packages must keep its minPrice roll-up, got %+v", priceSummary)
		}
	}
	// 真实 0 元套餐的汇总 HasFree 不受兜底影响（有套餐也不抹掉）。
	withFree := repository.ServicePriceSummary{ActiveCount: 1, HasFree: true}
	applyDetailMutualHasFree("pay", &withFree)
	if !withFree.HasFree {
		t.Error("real free package roll-up HasFree=true must survive the fallback")
	}
}

func TestIsMutualMode(t *testing.T) {
	cases := map[string]bool{
		"free":  true,
		"both":  true,
		"pay":   false,
		"":      false,
		"FREE":  false, // 精确匹配，不认大小写变体
		"both2": false,
	}
	for mode, want := range cases {
		if got := isMutualMode(mode); got != want {
			t.Errorf("isMutualMode(%q) want %v got %v", mode, want, got)
		}
	}
}
