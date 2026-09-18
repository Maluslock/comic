package service

import (
	"context"
	"errors"

	"github.com/Maluslock/comic/server/internal/repository"
)

var ErrInvalidStatus = errors.New("invalid status")

type adminUserStore interface {
	ListUsers(ctx context.Context, keyword, role, status string, limit, offset int) ([]repository.AdminUser, int64, error)
	GetUserDetail(ctx context.Context, id int64) (*repository.AdminUserDetail, error)
	SetUserStatus(ctx context.Context, id int64, status string) error
}

type AdminUserService struct {
	store adminUserStore
}

func NewAdminUserService(repo *repository.AdminUserRepo) *AdminUserService {
	return &AdminUserService{store: repo}
}

func (s *AdminUserService) List(ctx context.Context, keyword, role, status string, page, pageSize int) ([]repository.AdminUser, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.ListUsers(ctx, keyword, role, status, pageSize, (page-1)*pageSize)
}

func (s *AdminUserService) ExportUsers(ctx context.Context, keyword, role, status string) ([]repository.AdminUser, error) {
	items, _, err := s.store.ListUsers(ctx, keyword, role, status, 100000, 0)
	return items, err
}

func (s *AdminUserService) Detail(ctx context.Context, id int64) (*repository.AdminUserDetail, error) {
	return s.store.GetUserDetail(ctx, id)
}

func (s *AdminUserService) SetStatus(ctx context.Context, id int64, status string) error {
	if status != "active" && status != "disabled" {
		return ErrInvalidStatus
	}
	return s.store.SetUserStatus(ctx, id, status)
}
