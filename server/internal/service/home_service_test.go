package service

import (
	"testing"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

func int32Ptr(i int32) *int32 { return &i }

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic("mustParseTime: " + err.Error())
	}
	return t
}

// --- derefString / derefInt32 ---

func TestDerefString_Nil(t *testing.T) {
	if derefString(nil) != "" {
		t.Error("expected empty string for nil")
	}
}

func TestDerefString_NonNil(t *testing.T) {
	s := "hello"
	if derefString(&s) != "hello" {
		t.Error("expected 'hello'")
	}
}

func TestDerefInt32_Nil(t *testing.T) {
	if derefInt32(nil) != 0 {
		t.Error("expected 0 for nil")
	}
}

func TestDerefInt32_NonNil(t *testing.T) {
	v := int32(42)
	if derefInt32(&v) != 42 {
		t.Error("expected 42")
	}
}

// --- numericToFloat64 ---

func TestNumericToFloat64_Invalid(t *testing.T) {
	n := pgtype.Numeric{Valid: false}
	if numericToFloat64(n) != 0 {
		t.Error("expected 0 for invalid Numeric")
	}
}

func TestNumericToFloat64_NaN(t *testing.T) {
	n := pgtype.Numeric{NaN: true, Valid: true}
	if numericToFloat64(n) != 0 {
		t.Error("expected 0 for NaN Numeric")
	} else {
		t.Log("correctly returned 0 for NaN")
	}
}

func TestNumericToFloat64_Valid(t *testing.T) {
	// Scan a float into a Numeric to get a valid value
	var n pgtype.Numeric
	if err := n.Scan("4.75"); err != nil {
		t.Fatalf("failed to scan: %v", err)
	}
	if got := numericToFloat64(n); got != 4.75 {
		t.Errorf("expected 4.75, got %v", got)
	}
}

// --- mapBanners ---

func TestMapBanners_Empty(t *testing.T) {
	result := mapBanners(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestMapBanners_Success(t *testing.T) {
	linkID := int32(100)
	banners := []repository.Banner{
		{
			ID:       1,
			ImageUrl: "https://example.com/banner1.jpg",
			Title:    strPtr("Summer Event"),
			LinkType: strPtr("event"),
			LinkID:   &linkID,
		},
		{
			ID:       2,
			ImageUrl: "https://example.com/banner2.jpg",
			Title:    nil,
			LinkType: nil,
			LinkID:   nil,
		},
	}
	result := mapBanners(banners)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].ID != 1 || result[0].ImageURL != "https://example.com/banner1.jpg" {
		t.Error("banner 0 mismatch")
	}
	if result[0].Title != "Summer Event" || result[0].LinkType != "event" || *result[0].LinkID != 100 {
		t.Error("banner 0 fields mismatch")
	}
	if result[1].ID != 2 || result[1].Title != "" || result[1].LinkType != "" || result[1].LinkID != nil {
		t.Error("banner 1 should have zero/default fields")
	}
}

// --- mapEvents ---

func TestMapEvents_Empty(t *testing.T) {
	result := mapEvents(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestMapEvents_Success(t *testing.T) {
	events := []repository.ComicEvent{
		{
			ID:        1,
			Name:      "ComicCon 2026",
			Location:  strPtr("Shanghai"),
			Venue:     strPtr("Expo Center"),
			StartDate: mustParseTime("2026-07-01T09:00:00Z"),
			EndDate:   mustParseTime("2026-07-03T18:00:00Z"),
			CoverUrl:  strPtr("https://example.com/cover.jpg"),
			Tags:      []string{"cosplay", "anime"},
			Status:    "upcoming",
			TypeName:  strPtr("大型综合"),
		},
	}
	result := mapEvents(events)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	e := result[0]
	if e.ID != 1 || e.Name != "ComicCon 2026" || e.Location != "Shanghai" {
		t.Error("event fields mismatch")
	}
	if e.Venue != "Expo Center" || e.Status != "upcoming" || e.TypeName != "大型综合" {
		t.Error("event fields mismatch")
	}
	if len(e.Tags) != 2 || e.Tags[0] != "cosplay" || e.Tags[1] != "anime" {
		t.Error("event tags mismatch")
	}
}

func TestMapEvents_NilTags(t *testing.T) {
	events := []repository.ComicEvent{
		{
			ID:        1,
			Name:      "TinyCon",
			StartDate: mustParseTime("2026-07-01T09:00:00Z"),
			EndDate:   mustParseTime("2026-07-01T18:00:00Z"),
			Tags:      nil,
			Status:    "upcoming",
		},
	}
	result := mapEvents(events)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Tags == nil || len(result[0].Tags) != 0 {
		t.Errorf("expected empty slice for nil tags, got %v", result[0].Tags)
	}
}

// --- mapTags ---

func TestMapTags_Empty(t *testing.T) {
	result := mapTags(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestMapTags_Success(t *testing.T) {
	tags := []repository.Tag{
		{Name: "cosplay", UsageCount: int32Ptr(42)},
		{Name: "anime", UsageCount: nil},
	}
	result := mapTags(tags)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].Name != "cosplay" || result[0].UsageCount != 42 {
		t.Error("tag 0 mismatch")
	}
	if result[1].Name != "anime" || result[1].UsageCount != 0 {
		t.Error("tag 1 should have UsageCount 0 for nil")
	}
}

// --- mapPhotographers ---

func TestMapPhotographers_Empty(t *testing.T) {
	result := mapPhotographers(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestMapPhotographers_Success(t *testing.T) {
	var rating pgtype.Numeric
	_ = rating.Scan("4.5")
	reviewCount := int32(120)
	orderCount := int32(340)

	photographers := []repository.PhotographerWithTags{
		{
			ID:          1,
			Name:        "Alice",
			Avatar:      strPtr("https://example.com/avatar.jpg"),
			Location:    strPtr("Beijing"),
			Rating:      rating,
			ReviewCount: &reviewCount,
			OrderCount:  &orderCount,
			Tags:        []string{"日系", "古风"},
		},
		{
			ID:          2,
			Name:        "Bob",
			Avatar:      nil,
			Location:    nil,
			Rating:      pgtype.Numeric{Valid: false},
			ReviewCount: nil,
			OrderCount:  nil,
			Tags:        nil,
		},
	}
	result := mapPhotographers(photographers)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].ID != 1 || result[0].Name != "Alice" || result[0].Rating != 4.5 {
		t.Error("photographer 0 mismatch")
	}
	if result[0].ReviewCount != 120 || result[0].OrderCount != 340 {
		t.Error("photographer 0 count mismatch")
	}
	if result[0].Avatar != "https://example.com/avatar.jpg" || result[0].Location != "Beijing" {
		t.Error("photographer 0 fields mismatch")
	}
	if len(result[0].Tags) != 2 || result[0].Tags[0] != "日系" {
		t.Error("photographer 0 tags mismatch")
	}
	if result[1].ID != 2 || result[1].Name != "Bob" {
		t.Error("photographer 1 mismatch")
	}
	if result[1].Rating != 0 || result[1].ReviewCount != 0 || result[1].OrderCount != 0 {
		t.Error("photographer 1 should have zero counts")
	}
	if result[1].Avatar != "" || result[1].Location != "" {
		t.Error("photographer 1 should have empty strings")
	}
	if result[1].Tags == nil || len(result[1].Tags) != 0 {
		t.Errorf("photographer 1 should have empty tags slice, got %v", result[1].Tags)
	}
}

// --- mapWorks ---

func TestMapWorks_Empty(t *testing.T) {
	result := mapWorks(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestMapWorks_Success(t *testing.T) {
	works := []repository.FeaturedWork{
		{
			ID:               1,
			Title:            "Cosplay in the Park",
			Images:           []string{"img1.jpg", "img2.jpg"},
			PhotographerName: "Alice",
		},
		{
			ID:               2,
			Title:            "Night Shoot",
			Images:           nil,
			PhotographerName: "Bob",
		},
	}
	result := mapWorks(works)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].ID != 1 || result[0].Title != "Cosplay in the Park" {
		t.Error("work 0 mismatch")
	}
	if result[0].PhotographerName != "Alice" {
		t.Error("work 0 PhotographerName mismatch")
	}
	if len(result[0].Images) != 2 || result[0].Images[0] != "img1.jpg" {
		t.Error("work 0 Images mismatch")
	}
	if result[1].ID != 2 || result[1].PhotographerName != "Bob" {
		t.Error("work 1 mismatch")
	}
	if result[1].Images == nil || len(result[1].Images) != 0 {
		t.Errorf("work 1 should have empty images slice, got %v", result[1].Images)
	}
}

// --- NewHomeService ---

func TestNewHomeService(t *testing.T) {
	svc := NewHomeService(nil)
	if svc == nil {
		t.Error("NewHomeService returned nil")
	}
}
