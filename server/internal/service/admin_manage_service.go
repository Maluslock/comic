package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type adminManageStore interface {
	ListPhotographers(ctx context.Context, certified *bool) ([]repository.AdminPhotographer, error)
	GetPhotographerDetail(ctx context.Context, id int64) (repository.AdminPhotographerDetail, error)
	SetCertified(ctx context.Context, id int64, certified bool) error
	ListOrders(ctx context.Context, status string, limit, offset int32) ([]repository.AdminOrder, int64, error)
	ListEvents(ctx context.Context) ([]repository.AdminEvent, error)
	SetEventStatus(ctx context.Context, id int64, delFlag bool) error
}

type AdminManageService struct {
	store      adminManageStore
	bookingSvc *BookingService
}

func NewAdminManageService(repo *repository.AdminManageRepo, bookingSvc *BookingService) *AdminManageService {
	return &AdminManageService{store: repo, bookingSvc: bookingSvc}
}

func (s *AdminManageService) ListPhotographers(ctx context.Context, certified *bool) ([]repository.AdminPhotographer, error) {
	return s.store.ListPhotographers(ctx, certified)
}

func (s *AdminManageService) ExportOrders(ctx context.Context, status string) ([]repository.AdminOrder, error) {
	items, _, err := s.store.ListOrders(ctx, status, 100000, 0)
	return items, err
}

func (s *AdminManageService) ExportPhotographers(ctx context.Context, certified *bool) ([]repository.AdminPhotographer, error) {
	return s.store.ListPhotographers(ctx, certified)
}

func (s *AdminManageService) SetCertified(ctx context.Context, id int64, certified bool) error {
	return s.store.SetCertified(ctx, id, certified)
}

func (s *AdminManageService) GetPhotographerDetail(ctx context.Context, id int64) (repository.AdminPhotographerDetail, error) {
	return s.store.GetPhotographerDetail(ctx, id)
}

func (s *AdminManageService) ListOrders(ctx context.Context, status string, page, pageSize int) ([]repository.AdminOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.ListOrders(ctx, status, int32(pageSize), int32((page-1)*pageSize))
}

func (s *AdminManageService) SetOrderStatus(ctx context.Context, bookingID int64, newStatus string) (*BookingItem, error) {
	return s.bookingSvc.AdminUpdateStatus(ctx, bookingID, newStatus)
}

func (s *AdminManageService) ListEvents(ctx context.Context) ([]repository.AdminEvent, error) {
	return s.store.ListEvents(ctx)
}

func (s *AdminManageService) SetEventStatus(ctx context.Context, id int64, delFlag bool) error {
	return s.store.SetEventStatus(ctx, id, delFlag)
}
