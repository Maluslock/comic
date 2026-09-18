package service

import (
	"context"
	"errors"

	"github.com/Maluslock/comic/server/internal/repository"
)

var (
	// ErrInvalidContentStatus signals a work status outside {active, down}.
	ErrInvalidContentStatus = errors.New("invalid content status")
	// ErrTagExists signals a duplicate tag name (unique violation).
	ErrTagExists = errors.New("tag exists")
	ErrSameTag   = errors.New("cannot merge a tag into itself")
)

type adminContentStore interface {
	ListWorks(ctx context.Context, photographerID *int64, status string, limit, offset int) ([]repository.AdminWork, int64, error)
	SetWorkStatus(ctx context.Context, id int64, status string) error
	ListReviews(ctx context.Context, keyword string, photographerID *int64, limit, offset int) ([]repository.AdminReview, int64, error)
	DeleteReview(ctx context.Context, id int64) error
	ListTags(ctx context.Context) ([]repository.AdminTag, error)
	CreateTag(ctx context.Context, name string) (int64, error)
	UpdateTag(ctx context.Context, id int64, name string) error
	DeleteTag(ctx context.Context, id int64) error
	MergeTag(ctx context.Context, fromID, toID int64) error
}

type AdminContentService struct {
	store adminContentStore
}

func NewAdminContentService(repo *repository.AdminContentRepo) *AdminContentService {
	return &AdminContentService{store: repo}
}

func (s *AdminContentService) ListWorks(ctx context.Context, photographerID *int64, status string, page, pageSize int) ([]repository.AdminWork, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.ListWorks(ctx, photographerID, status, pageSize, (page-1)*pageSize)
}

func (s *AdminContentService) SetWorkStatus(ctx context.Context, id int64, status string) error {
	if status != "active" && status != "down" {
		return ErrInvalidContentStatus
	}
	return s.store.SetWorkStatus(ctx, id, status)
}

func (s *AdminContentService) ListReviews(ctx context.Context, keyword string, photographerID *int64, page, pageSize int) ([]repository.AdminReview, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.ListReviews(ctx, keyword, photographerID, pageSize, (page-1)*pageSize)
}

func (s *AdminContentService) DeleteReview(ctx context.Context, id int64) error {
	return s.store.DeleteReview(ctx, id)
}

func (s *AdminContentService) ListTags(ctx context.Context) ([]repository.AdminTag, error) {
	return s.store.ListTags(ctx)
}

func (s *AdminContentService) CreateTag(ctx context.Context, name string) (int64, error) {
	id, err := s.store.CreateTag(ctx, name)
	if err != nil && isUniqueViolation(err) {
		return 0, ErrTagExists
	}
	return id, err
}

func (s *AdminContentService) UpdateTag(ctx context.Context, id int64, name string) error {
	err := s.store.UpdateTag(ctx, id, name)
	if err != nil && isUniqueViolation(err) {
		return ErrTagExists
	}
	return err
}

func (s *AdminContentService) DeleteTag(ctx context.Context, id int64) error {
	return s.store.DeleteTag(ctx, id)
}

func (s *AdminContentService) MergeTag(ctx context.Context, fromID, toID int64) error {
	if fromID == toID {
		return ErrSameTag
	}
	return s.store.MergeTag(ctx, fromID, toID)
}
