package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5"
)

type LoginRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

type LoginUser struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Phone          string `json:"phone"`
	Avatar         string `json:"avatar"`
	Bio            string `json:"bio"`
	PhotographerID *int64 `json:"photographerId"`
}

type LoginResponse struct {
	Token string    `json:"token"`
	User  LoginUser `json:"user"`
}

var ErrAccountDisabled = errors.New("account disabled")

type authUserStore interface {
	UpsertByPhone(ctx context.Context, phone, name, avatar string) (repository.User, error)
	CreateToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error
}

type AuthService struct {
	users   authUserStore
	queries *repository.Queries
}

func NewAuthService(users authUserStore, queries *repository.Queries) *AuthService {
	return &AuthService{users: users, queries: queries}
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if len(req.Code) < 4 || req.Code[:4] != "1234" {
		return nil, errors.New("invalid verification code")
	}
	if len(req.Phone) < 4 {
		return nil, errors.New("invalid phone")
	}

	name := fmt.Sprintf("用户%s", req.Phone[len(req.Phone)-4:])
	avatar := "/static/img/avatar-user.svg"
	u, err := s.users.UpsertByPhone(ctx, req.Phone, name, avatar)
	if err != nil {
		return nil, err
	}
	if u.Status == "disabled" {
		return nil, ErrAccountDisabled
	}

	token := generateToken()
	if err := s.users.CreateToken(ctx, token, u.ID, time.Now().Add(7*24*time.Hour)); err != nil {
		return nil, err
	}

	var photographerID *int64
	profile, err := s.queries.GetPhotographerByUserID(ctx, u.ID)
	if err == nil {
		id := int64(profile.ID)
		photographerID = &id
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  LoginUser{ID: u.ID, Name: u.Name, Phone: u.Phone, Avatar: u.Avatar, Bio: derefString(u.Bio), PhotographerID: photographerID},
	}, nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
