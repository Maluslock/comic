package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Maluslock/comic/server/internal/repository"
)

type fakeContentStore struct {
	worksErr           error
	setStatusErr       error
	reviewsErr         error
	deleteReviewErr    error
	tagsErr            error
	createID           int64
	createErr          error
	updateErr          error
	deleteTagErr       error
	lastPhotographerID *int64
	lastStatus         string
	lastKeyword        string
	lastLimit          int
	lastOffset         int
	setStatusCalls     int
}

func (f *fakeContentStore) ListWorks(_ context.Context, photographerID *int64, status string, limit, offset int) ([]repository.AdminWork, int64, error) {
	f.lastPhotographerID = photographerID
	f.lastStatus = status
	f.lastLimit = limit
	f.lastOffset = offset
	return nil, 0, f.worksErr
}

func (f *fakeContentStore) SetWorkStatus(_ context.Context, _ int64, _ string) error {
	f.setStatusCalls++
	return f.setStatusErr
}

func (f *fakeContentStore) ListReviews(_ context.Context, keyword string, photographerID *int64, limit, offset int) ([]repository.AdminReview, int64, error) {
	f.lastKeyword = keyword
	f.lastPhotographerID = photographerID
	f.lastLimit = limit
	f.lastOffset = offset
	return nil, 0, f.reviewsErr
}

func (f *fakeContentStore) DeleteReview(_ context.Context, _ int64) error {
	return f.deleteReviewErr
}

func (f *fakeContentStore) ListTags(_ context.Context) ([]repository.AdminTag, error) {
	return nil, f.tagsErr
}

func (f *fakeContentStore) CreateTag(_ context.Context, _ string) (int64, error) {
	return f.createID, f.createErr
}

func (f *fakeContentStore) UpdateTag(_ context.Context, _ int64, _ string) error {
	return f.updateErr
}

func (f *fakeContentStore) DeleteTag(_ context.Context, _ int64) error {
	return f.deleteTagErr
}

func (f *fakeContentStore) MergeTag(_ context.Context, _, _ int64) error {
	return nil
}

func newTestContentSvc(store adminContentStore) *AdminContentService {
	return &AdminContentService{store: store}
}

func TestContentSetWorkStatus_Invalid(t *testing.T) {
	store := &fakeContentStore{}
	svc := newTestContentSvc(store)
	err := svc.SetWorkStatus(context.Background(), 1, "banana")
	if !errors.Is(err, ErrInvalidContentStatus) {
		t.Fatalf("want ErrInvalidContentStatus, got %v", err)
	}
	if store.setStatusCalls != 0 {
		t.Fatalf("repo.SetWorkStatus must not be called on invalid status, got %d calls", store.setStatusCalls)
	}
}

func TestContentSetWorkStatus_NotFound(t *testing.T) {
	svc := newTestContentSvc(&fakeContentStore{setStatusErr: repository.ErrContentNotFound})
	err := svc.SetWorkStatus(context.Background(), 9999, "down")
	if !errors.Is(err, repository.ErrContentNotFound) {
		t.Fatalf("want ErrContentNotFound, got %v", err)
	}
}

func TestContentDeleteReview_NotFound(t *testing.T) {
	svc := newTestContentSvc(&fakeContentStore{deleteReviewErr: repository.ErrContentNotFound})
	err := svc.DeleteReview(context.Background(), 9999)
	if !errors.Is(err, repository.ErrContentNotFound) {
		t.Fatalf("want ErrContentNotFound, got %v", err)
	}
}

func TestContentCreateTag_Duplicate(t *testing.T) {
	dupErr := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint \"tags_name_key\""}
	svc := newTestContentSvc(&fakeContentStore{createErr: dupErr})
	_, err := svc.CreateTag(context.Background(), "测试标签")
	if !errors.Is(err, ErrTagExists) {
		t.Fatalf("want ErrTagExists, got %v", err)
	}
}

func TestContentUpdateTag_Duplicate(t *testing.T) {
	dupErr := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint \"tags_name_key\""}
	svc := newTestContentSvc(&fakeContentStore{updateErr: dupErr})
	err := svc.UpdateTag(context.Background(), 1, "测试标签")
	if !errors.Is(err, ErrTagExists) {
		t.Fatalf("want ErrTagExists, got %v", err)
	}
}

func TestContentDeleteTag_InUse(t *testing.T) {
	svc := newTestContentSvc(&fakeContentStore{deleteTagErr: repository.ErrTagInUse})
	err := svc.DeleteTag(context.Background(), 1)
	if !errors.Is(err, repository.ErrTagInUse) {
		t.Fatalf("want ErrTagInUse, got %v", err)
	}
}

func TestContentDeleteTag_NoRefs(t *testing.T) {
	svc := newTestContentSvc(&fakeContentStore{})
	if err := svc.DeleteTag(context.Background(), 14); err != nil {
		t.Fatalf("DeleteTag unexpected error: %v", err)
	}
}

func TestContentListWorks_Filters(t *testing.T) {
	store := &fakeContentStore{}
	svc := newTestContentSvc(store)

	var pid int64 = 3
	if _, _, err := svc.ListWorks(context.Background(), &pid, "down", 2, 10); err != nil {
		t.Fatalf("ListWorks unexpected error: %v", err)
	}
	if store.lastPhotographerID == nil || *store.lastPhotographerID != 3 {
		t.Fatalf("want photographerID=3, got %v", store.lastPhotographerID)
	}
	if store.lastStatus != "down" {
		t.Fatalf("want status=down, got %q", store.lastStatus)
	}
	if store.lastLimit != 10 || store.lastOffset != 10 {
		t.Fatalf("want limit=10 offset=10 (page=2,pageSize=10), got limit=%d offset=%d",
			store.lastLimit, store.lastOffset)
	}

	if _, _, err := svc.ListWorks(context.Background(), nil, "", 1, 20); err != nil {
		t.Fatalf("ListWorks(no filters) unexpected error: %v", err)
	}
	if store.lastPhotographerID != nil || store.lastStatus != "" {
		t.Fatalf("want empty filters, got photographerID=%v status=%q", store.lastPhotographerID, store.lastStatus)
	}
	if store.lastLimit != 20 || store.lastOffset != 0 {
		t.Fatalf("want default limit=20 offset=0, got limit=%d offset=%d", store.lastLimit, store.lastOffset)
	}
}
