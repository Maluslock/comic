package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Maluslock/comic/server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type fakeAdminStore struct {
	byUsername      *repository.AdminRow
	byUsernameErr   error
	byID            *repository.AdminRow
	byIDErr         error
	setStatusErr    error
	setStatusCalls  int
	clearTokenCalls int
	createID        int64
	createErr       error
}

func (f *fakeAdminStore) GetAdminByUsername(_ context.Context, _ string) (*repository.AdminRow, error) {
	return f.byUsername, f.byUsernameErr
}

func (f *fakeAdminStore) GetAdminByToken(_ context.Context, _ string) (*repository.AdminRow, error) {
	return nil, repository.ErrAdminNotFound
}

func (f *fakeAdminStore) SetToken(_ context.Context, _ int64, _ string, _ time.Time) error {
	return nil
}

func (f *fakeAdminStore) UpdatePassword(_ context.Context, _ int64, _ string) error {
	return nil
}

func (f *fakeAdminStore) GetByID(_ context.Context, _ int64) (*repository.AdminRow, error) {
	return f.byID, f.byIDErr
}

func (f *fakeAdminStore) List(_ context.Context) ([]repository.AdminRow, error) {
	return nil, nil
}

func (f *fakeAdminStore) SetStatus(_ context.Context, _ int64, _ string) error {
	f.setStatusCalls++
	return f.setStatusErr
}

func (f *fakeAdminStore) ClearToken(_ context.Context, _ int64) error {
	f.clearTokenCalls++
	return nil
}

func (f *fakeAdminStore) Create(_ context.Context, _, _, _ string) (int64, error) {
	return f.createID, f.createErr
}

func TestAdminLogin_Disabled(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	svc := &AdminService{store: &fakeAdminStore{
		byUsername: &repository.AdminRow{ID: 2, PasswordHash: string(hash), Status: "disabled"},
	}}
	_, _, err := svc.Login(context.Background(), "testadmin", "admin123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestAdminSetStatus_PrimaryAdmin(t *testing.T) {
	store := &fakeAdminStore{}
	svc := &AdminService{store: store}
	err := svc.SetStatus(context.Background(), 1, "disabled")
	if !errors.Is(err, ErrPrimaryAdmin) {
		t.Fatalf("want ErrPrimaryAdmin, got %v", err)
	}
	if store.setStatusCalls != 0 {
		t.Fatalf("repo.SetStatus must not be called for primary admin, got %d calls", store.setStatusCalls)
	}
}

func TestAdminSetStatus_Invalid(t *testing.T) {
	store := &fakeAdminStore{}
	svc := &AdminService{store: store}
	err := svc.SetStatus(context.Background(), 2, "banana")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("want ErrInvalidStatus, got %v", err)
	}
	if store.setStatusCalls != 0 {
		t.Fatalf("repo.SetStatus must not be called on invalid status, got %d calls", store.setStatusCalls)
	}
}

func TestAdminSetStatus_NotFound(t *testing.T) {
	svc := &AdminService{store: &fakeAdminStore{setStatusErr: repository.ErrAdminNotFound}}
	err := svc.SetStatus(context.Background(), 9999, "disabled")
	if !errors.Is(err, repository.ErrAdminNotFound) {
		t.Fatalf("want ErrAdminNotFound, got %v", err)
	}
}

func TestAdminCreate_UsernameExists(t *testing.T) {
	dupErr := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint \"admins_username_key\""}
	svc := &AdminService{store: &fakeAdminStore{createErr: dupErr}}
	_, err := svc.Create(context.Background(), "admin", "admin123", "admin")
	if !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("want ErrUsernameExists, got %v", err)
	}
}

func TestAdminResetPassword_ClearsToken(t *testing.T) {
	store := &fakeAdminStore{byID: &repository.AdminRow{ID: 2}}
	svc := &AdminService{store: store}
	if err := svc.ResetPassword(context.Background(), 2, "newpass123"); err != nil {
		t.Fatalf("ResetPassword unexpected error: %v", err)
	}
	if store.clearTokenCalls != 1 {
		t.Fatalf("want ClearToken called once, got %d", store.clearTokenCalls)
	}
}
