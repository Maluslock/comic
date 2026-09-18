package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
)

type fakeAdminUserStore struct {
	setStatusErr error
}

func (f *fakeAdminUserStore) ListUsers(_ context.Context, _, _, _ string, _, _ int) ([]repository.AdminUser, int64, error) {
	return nil, 0, nil
}

func (f *fakeAdminUserStore) GetUserDetail(_ context.Context, _ int64) (*repository.AdminUserDetail, error) {
	return nil, nil
}

func (f *fakeAdminUserStore) SetUserStatus(_ context.Context, _ int64, _ string) error {
	return f.setStatusErr
}

type fakeAuthUserStore struct {
	user repository.User
	err  error
}

func (f *fakeAuthUserStore) UpsertByPhone(_ context.Context, _, _, _ string) (repository.User, error) {
	return f.user, f.err
}

func (f *fakeAuthUserStore) CreateToken(_ context.Context, _ string, _ int64, _ time.Time) error {
	return nil
}

func TestSetStatus_Valid(t *testing.T) {
	svc := &AdminUserService{store: &fakeAdminUserStore{}}
	if err := svc.SetStatus(context.Background(), 1, "disabled"); err != nil {
		t.Fatalf("SetStatus(disabled) unexpected error: %v", err)
	}
	if err := svc.SetStatus(context.Background(), 1, "active"); err != nil {
		t.Fatalf("SetStatus(active) unexpected error: %v", err)
	}
}

func TestSetStatus_Invalid(t *testing.T) {
	svc := &AdminUserService{store: &fakeAdminUserStore{}}
	err := svc.SetStatus(context.Background(), 1, "banana")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("want ErrInvalidStatus, got %v", err)
	}
}

func TestSetStatus_NotFound(t *testing.T) {
	svc := &AdminUserService{store: &fakeAdminUserStore{setStatusErr: repository.ErrManageNotFound}}
	err := svc.SetStatus(context.Background(), 99999999, "disabled")
	if !errors.Is(err, repository.ErrManageNotFound) {
		t.Fatalf("want ErrManageNotFound, got %v", err)
	}
}

func TestAuthLogin_Disabled(t *testing.T) {
	svc := NewAuthService(&fakeAuthUserStore{
		user: repository.User{ID: 16, Phone: "13800138000", Name: "用户8000", Status: "disabled"},
	}, nil)
	_, err := svc.Login(context.Background(), LoginRequest{Phone: "13800138000", Code: "123456"})
	if !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("want ErrAccountDisabled, got %v", err)
	}
}
