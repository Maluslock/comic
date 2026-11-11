package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
)

// stubStylePrefsStore 是 stylePrefsStore 的单测 fake，记录调用轨迹用于断言。
type stubStylePrefsStore struct {
	tagsRows []repository.StyleTagRow
	chosen   bool
	tagCount int // CountTags 的返回值（存在 tag 数）
	// chosenOnReplace 记录「ReplaceStylePrefs 被调用时 fake 是否同时置了 chosen」，
	// 用来固化「chosen 与偏好同事务」这条不变量：service 层不许再单独调一次置位。
	replaceErr     error
	countErr       error
	replaced       [][]int64 // 每次 ReplaceStylePrefs 收到的 tagIDs
	replaceCalls   int
}

func (s *stubStylePrefsStore) ListStyleTagsByUser(_ context.Context, _ int64) ([]repository.StyleTagRow, error) {
	return s.tagsRows, nil
}

func (s *stubStylePrefsStore) ReplaceStylePrefs(_ context.Context, _ int64, tagIDs []int64) error {
	if s.replaceErr != nil {
		return s.replaceErr
	}
	s.replaced = append(s.replaced, tagIDs)
	s.replaceCalls++
	// 真实实现会在同一个事务里置 chosen=true，fake 同步模拟，
	// 这样 Get() 回读到的 chosen 与真库行为一致。
	s.chosen = true
	return nil
}

func (s *stubStylePrefsStore) GetStylePrefChosen(_ context.Context, _ int64) (bool, error) {
	return s.chosen, nil
}

func (s *stubStylePrefsStore) CountTags(_ context.Context, _ []int64) (int, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}
	return s.tagCount, nil
}

// 正常路径：勾 1 个风格。PUT 成功后必须置 chosen=true（MarkStylePrefChosen 被调用）。
func TestStylePrefService_Replace_SingleTag(t *testing.T) {
	store := &stubStylePrefsStore{tagCount: 1, tagsRows: []repository.StyleTagRow{{ID: 3, Name: "暗黑"}}}
	got, err := NewStylePrefService(store).Replace(context.Background(), 1, []int64{3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.replaced) != 1 || len(store.replaced[0]) != 1 || store.replaced[0][0] != 3 {
		t.Fatalf("replace want [3], got %v", store.replaced)
	}
	if store.replaceCalls != 1 {
		t.Fatalf("ReplaceStylePrefs must be called once, got %d", store.replaceCalls)
	}
	if len(got.Tags) != 1 || got.Tags[0].ID != 3 || got.Tags[0].Name != "暗黑" {
		t.Fatalf("response tags: want [{3 暗黑}], got %+v", got.Tags)
	}
}

// 正常路径：勾满 5 个（上限边界，必须通过）。
func TestStylePrefService_Replace_FiveTags(t *testing.T) {
	store := &stubStylePrefsStore{tagCount: 5}
	_, err := NewStylePrefService(store).Replace(context.Background(), 1, []int64{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatalf("5 tags must be allowed: %v", err)
	}
}

// 空数组 = 主动清空，允许，且同样置 chosen=true（首启弹窗靠它判断别再弹）。
func TestStylePrefService_Replace_EmptyAllowed(t *testing.T) {
	store := &stubStylePrefsStore{}
	got, err := NewStylePrefService(store).Replace(context.Background(), 1, []int64{})
	if err != nil {
		t.Fatalf("empty selection must be allowed: %v", err)
	}
	if len(store.replaced) != 1 || len(store.replaced[0]) != 0 {
		t.Fatalf("replace want empty slice, got %v", store.replaced)
	}
	// chosen 与偏好同事务：清空也必须算「已表过态」。
	if !got.Chosen {
		t.Fatal("chosen must be true after an empty replace (same-transaction semantics)")
	}
}

// 6 个 = 超上限，必须 400（ErrInvalidStyleSelection），且不能碰库。
func TestStylePrefService_Replace_SixTagsRejected(t *testing.T) {
	store := &stubStylePrefsStore{tagCount: 6}
	_, err := NewStylePrefService(store).Replace(context.Background(), 1, []int64{1, 2, 3, 4, 5, 6})
	if !errors.Is(err, ErrInvalidStyleSelection) {
		t.Fatalf("want ErrInvalidStyleSelection, got %v", err)
	}
	if len(store.replaced) != 0 || store.replaceCalls != 0 {
		t.Fatal("rejected replace must not touch the store")
	}
}

// 重复勾选：5+5 全重复，去重后只剩 5 个 = 上限内，必须通过（重复不算超限）。
func TestStylePrefService_Replace_DuplicatesDeduped(t *testing.T) {
	store := &stubStylePrefsStore{tagCount: 5}
	_, err := NewStylePrefService(store).Replace(context.Background(), 1, []int64{1, 2, 3, 4, 5, 1, 2, 3, 4, 5})
	if err != nil {
		t.Fatalf("dedup to 5 must pass: %v", err)
	}
	if len(store.replaced) != 1 || len(store.replaced[0]) != 5 {
		t.Fatalf("store must receive deduped 5 ids, got %v", store.replaced)
	}
}

// 含不存在的 tagId：tags 表里查不满 → 400（ErrInvalidStyleSelection）。
func TestStylePrefService_Replace_UnknownTagRejected(t *testing.T) {
	store := &stubStylePrefsStore{tagCount: 1} // 只存在 1 个，但请求了 2 个
	_, err := NewStylePrefService(store).Replace(context.Background(), 1, []int64{3, 999})
	if !errors.Is(err, ErrInvalidStyleSelection) {
		t.Fatalf("want ErrInvalidStyleSelection for unknown tag, got %v", err)
	}
	if len(store.replaced) != 0 || store.replaceCalls != 0 {
		t.Fatal("invalid selection must not touch the store")
	}
}

// GET 的 chosen 必须读 users.style_pref_chosen，不能靠 tags 非空推断：
// tags 为空 + chosen=true（主动清空）与 tags 为空 + chosen=false（从未选择）都要如实透出。
func TestStylePrefService_Get_ChosenReadsColumn(t *testing.T) {
	store := &stubStylePrefsStore{chosen: true}
	got, err := NewStylePrefService(store).Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Chosen {
		t.Fatal("chosen=true must be surfaced even when tags are empty")
	}
	if got.Tags == nil {
		t.Fatal("tags must be an empty slice, not nil (frontend v-for)")
	}
}

func TestStylePrefService_Get_NeverChosen(t *testing.T) {
	store := &stubStylePrefsStore{chosen: false, tagsRows: []repository.StyleTagRow{{ID: 2, Name: "古风"}}}
	got, err := NewStylePrefService(store).Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Chosen {
		t.Fatal("chosen must be false when the column says false, even with tags present")
	}
	if len(got.Tags) != 1 || got.Tags[0].ID != 2 || got.Tags[0].Name != "古风" {
		t.Fatalf("want [{2 古风}], got %+v", got.Tags)
	}
}
