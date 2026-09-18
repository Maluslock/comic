package service

import (
	"context"
	"errors"

	"github.com/Maluslock/comic/server/internal/repository"
)

var ErrInvalidLinkType = errors.New("invalid link type")

type BannerUpsert struct {
	ImageURL  string
	Title     string
	LinkType  string
	LinkID    int32
	SortOrder int32
}

type adminBannerStore interface {
	List(ctx context.Context) ([]repository.AdminBanner, error)
	Create(ctx context.Context, imageURL, title, linkType string, linkID, sortOrder int32) (int64, error)
	GetByID(ctx context.Context, id int64) (*repository.AdminBanner, error)
	Update(ctx context.Context, id int64, imageURL, title, linkType string, linkID, sortOrder int32, isActive bool) error
	SetStatus(ctx context.Context, id int64, isActive bool) error
}

type AdminBannerService struct {
	store adminBannerStore
}

func NewAdminBannerService(repo *repository.AdminBannerRepo) *AdminBannerService {
	return &AdminBannerService{store: repo}
}

func (s *AdminBannerService) List(ctx context.Context) ([]repository.AdminBanner, error) {
	return s.store.List(ctx)
}

func (s *AdminBannerService) Create(ctx context.Context, req BannerUpsert) (int64, error) {
	if req.LinkType != "event" {
		return 0, ErrInvalidLinkType
	}
	return s.store.Create(ctx, req.ImageURL, req.Title, req.LinkType, req.LinkID, req.SortOrder)
}

func (s *AdminBannerService) Update(ctx context.Context, id int64, req BannerUpsert, isActive bool) error {
	if req.LinkType != "event" {
		return ErrInvalidLinkType
	}
	return s.store.Update(ctx, id, req.ImageURL, req.Title, req.LinkType, req.LinkID, req.SortOrder, isActive)
}

func (s *AdminBannerService) SetStatus(ctx context.Context, id int64, isActive bool) error {
	return s.store.SetStatus(ctx, id, isActive)
}
