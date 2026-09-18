package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUsernameExists     = errors.New("username exists")
	ErrInvalidRole        = errors.New("invalid role")
	ErrPrimaryAdmin       = errors.New("cannot disable primary admin")
)

type AdminAccount struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type adminStore interface {
	GetAdminByUsername(ctx context.Context, username string) (*repository.AdminRow, error)
	GetAdminByToken(ctx context.Context, token string) (*repository.AdminRow, error)
	SetToken(ctx context.Context, id int64, token string, expiresAt time.Time) error
	UpdatePassword(ctx context.Context, id int64, hash string) error
	GetByID(ctx context.Context, id int64) (*repository.AdminRow, error)
	List(ctx context.Context) ([]repository.AdminRow, error)
	SetStatus(ctx context.Context, id int64, status string) error
	ClearToken(ctx context.Context, id int64) error
	Create(ctx context.Context, username, passwordHash, role string) (int64, error)
}

type AdminService struct {
	store adminStore
}

func NewAdminService(repo *repository.AdminRepo) *AdminService { return &AdminService{store: repo} }

func newAdminToken(id int64) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("adm_%d_%s", id, hex.EncodeToString(b))
}

func (s *AdminService) Login(ctx context.Context, username, password string) (string, *repository.AdminRow, error) {
	a, err := s.store.GetAdminByUsername(ctx, username)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) != nil {
		return "", nil, ErrInvalidCredentials
	}
	if a.Status == "disabled" {
		return "", nil, ErrInvalidCredentials
	}
	token := newAdminToken(a.ID)
	if err := s.store.SetToken(ctx, a.ID, token, time.Now().Add(24*time.Hour)); err != nil {
		return "", nil, err
	}
	return token, a, nil
}

func (s *AdminService) Authenticate(ctx context.Context, token string) (*repository.AdminRow, error) {
	return s.store.GetAdminByToken(ctx, token)
}

func (s *AdminService) List(ctx context.Context) ([]AdminAccount, error) {
	rows, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	accounts := make([]AdminAccount, 0, len(rows))
	for _, a := range rows {
		accounts = append(accounts, AdminAccount{
			ID:        a.ID,
			Username:  a.Username,
			Role:      a.Role,
			Status:    a.Status,
			CreatedAt: a.CreatedAt,
		})
	}
	return accounts, nil
}

func (s *AdminService) Create(ctx context.Context, username, password, role string) (int64, error) {
	if role != "admin" && role != "super" {
		return 0, ErrInvalidRole
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	id, err := s.store.Create(ctx, username, string(hash), role)
	if err != nil && isUniqueViolation(err) {
		return 0, ErrUsernameExists
	}
	return id, err
}

func (s *AdminService) SetStatus(ctx context.Context, id int64, status string) error {
	if status != "active" && status != "disabled" {
		return ErrInvalidStatus
	}
	if id == 1 && status == "disabled" {
		return ErrPrimaryAdmin
	}
	return s.store.SetStatus(ctx, id, status)
}

func (s *AdminService) ResetPassword(ctx context.Context, id int64, password string) error {
	if _, err := s.store.GetByID(ctx, id); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.store.UpdatePassword(ctx, id, string(hash)); err != nil {
		return err
	}
	return s.store.ClearToken(ctx, id)
}
