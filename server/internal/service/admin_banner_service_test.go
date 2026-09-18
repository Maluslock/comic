package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
)

type fakeBannerStore struct {
	createID    int64
	createErr   error
	updateErr   error
	statusErr   error
	createCalls int
	updateCalls int

	lastCreateImageURL  string
	lastCreateTitle     string
	lastCreateLinkType  string
	lastCreateLinkID    int32
	lastCreateSortOrder int32

	lastUpdateID       int64
	lastUpdateIsActive bool
}

func (f *fakeBannerStore) List(_ context.Context) ([]repository.AdminBanner, error) {
	return nil, nil
}

func (f *fakeBannerStore) Create(_ context.Context, imageURL, title, linkType string, linkID, sortOrder int32) (int64, error) {
	f.createCalls++
	f.lastCreateImageURL = imageURL
	f.lastCreateTitle = title
	f.lastCreateLinkType = linkType
	f.lastCreateLinkID = linkID
	f.lastCreateSortOrder = sortOrder
	return f.createID, f.createErr
}

func (f *fakeBannerStore) GetByID(_ context.Context, _ int64) (*repository.AdminBanner, error) {
	return nil, nil
}

func (f *fakeBannerStore) Update(_ context.Context, id int64, _, _, _ string, _, _ int32, isActive bool) error {
	f.updateCalls++
	f.lastUpdateID = id
	f.lastUpdateIsActive = isActive
	return f.updateErr
}

func (f *fakeBannerStore) SetStatus(_ context.Context, _ int64, _ bool) error {
	return f.statusErr
}

func newBannerTestSvc(store adminBannerStore) *AdminBannerService {
	return &AdminBannerService{store: store}
}

func TestCreate_ValidLinkType(t *testing.T) {
	store := &fakeBannerStore{createID: 7}
	svc := newBannerTestSvc(store)

	id, err := svc.Create(context.Background(), BannerUpsert{
		ImageURL:  "https://picsum.photos/seed/bannertest/750/360",
		Title:     "测试轮播",
		LinkType:  "event",
		LinkID:    1,
		SortOrder: 0,
	})
	if err != nil {
		t.Fatalf("Create(event) unexpected error: %v", err)
	}
	if id != 7 {
		t.Fatalf("want id=7, got %d", id)
	}
	if store.createCalls != 1 {
		t.Fatalf("want repo.Create called once, got %d", store.createCalls)
	}
	if store.lastCreateImageURL != "https://picsum.photos/seed/bannertest/750/360" ||
		store.lastCreateTitle != "测试轮播" ||
		store.lastCreateLinkType != "event" ||
		store.lastCreateLinkID != 1 ||
		store.lastCreateSortOrder != 0 {
		t.Fatalf("repo.Create got wrong params: imageURL=%q title=%q linkType=%q linkID=%d sortOrder=%d",
			store.lastCreateImageURL, store.lastCreateTitle, store.lastCreateLinkType,
			store.lastCreateLinkID, store.lastCreateSortOrder)
	}
}

func TestCreate_InvalidLinkType(t *testing.T) {
	store := &fakeBannerStore{}
	svc := newBannerTestSvc(store)

	_, err := svc.Create(context.Background(), BannerUpsert{
		ImageURL: "https://picsum.photos/seed/bannertest/750/360",
		Title:    "测试轮播",
		LinkType: "page",
		LinkID:   1,
	})
	if !errors.Is(err, ErrInvalidLinkType) {
		t.Fatalf("want ErrInvalidLinkType, got %v", err)
	}
	if store.createCalls != 0 {
		t.Fatalf("repo.Create must not be called on invalid link type, got %d calls", store.createCalls)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	store := &fakeBannerStore{updateErr: repository.ErrBannerNotFound}
	svc := newBannerTestSvc(store)

	err := svc.Update(context.Background(), 999, BannerUpsert{
		ImageURL: "https://picsum.photos/seed/bannertest/750/360",
		Title:    "测试轮播改",
		LinkType: "event",
		LinkID:   1,
	}, true)
	if !errors.Is(err, repository.ErrBannerNotFound) {
		t.Fatalf("want ErrBannerNotFound, got %v", err)
	}
	if store.updateCalls != 1 {
		t.Fatalf("want repo.Update called once, got %d", store.updateCalls)
	}
}

func TestUpdate_InvalidLinkType(t *testing.T) {
	store := &fakeBannerStore{}
	svc := newBannerTestSvc(store)

	err := svc.Update(context.Background(), 4, BannerUpsert{
		ImageURL: "https://picsum.photos/seed/bannertest/750/360",
		Title:    "测试轮播改",
		LinkType: "page",
	}, true)
	if !errors.Is(err, ErrInvalidLinkType) {
		t.Fatalf("want ErrInvalidLinkType, got %v", err)
	}
	if store.updateCalls != 0 {
		t.Fatalf("repo.Update must not be called on invalid link type, got %d calls", store.updateCalls)
	}
}

func TestBannerSetStatus_NotFound(t *testing.T) {
	store := &fakeBannerStore{statusErr: repository.ErrBannerNotFound}
	svc := newBannerTestSvc(store)

	err := svc.SetStatus(context.Background(), 999, false)
	if !errors.Is(err, repository.ErrBannerNotFound) {
		t.Fatalf("want ErrBannerNotFound, got %v", err)
	}
}
