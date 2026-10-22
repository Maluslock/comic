package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
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

// WeChatLoginRequest 是小程序 wx.login 的结果：code 只能用一次，5 分钟内有效。
type WeChatLoginRequest struct {
	Code     string `json:"code" binding:"required"`
	NickName string `json:"nickName"`
	Avatar   string `json:"avatar"`
}

var (
	ErrAccountDisabled = errors.New("account disabled")
	// ErrPhoneLoginDisabled 在 APP_ENV=production 时返回：手机号验证码登录是演示实现，
	// 生产只允许微信登录，直到接入真实短信服务。
	ErrPhoneLoginDisabled = errors.New("phone login is disabled")
)

type authUserStore interface {
	UpsertByPhone(ctx context.Context, phone, name, avatar string) (repository.User, error)
	UpsertByWeChat(ctx context.Context, openid, unionid, name, avatar string) (repository.User, error)
	CreateToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error
}

type AuthService struct {
	users   authUserStore
	queries *repository.Queries
	wechat  WeChatClient
	// allowMockLogin 控制演示用的「1234 开头即通过」手机号登录，默认 true 以兼容单测与本地开发。
	allowMockLogin bool
}

// NewAuthService 的第三个参数可选：不传时微信登录返回 ErrWeChatDisabled，
// 老调用方（单测）无需改签名。
func NewAuthService(users authUserStore, queries *repository.Queries, wechat ...WeChatClient) *AuthService {
	s := &AuthService{users: users, queries: queries, allowMockLogin: true}
	if len(wechat) > 0 {
		s.wechat = wechat[0]
	}
	return s
}

// SetMockLoginAllowed 由 main 按 APP_ENV 调用：生产环境关闭假验证码登录。
func (s *AuthService) SetMockLoginAllowed(allow bool) { s.allowMockLogin = allow }

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if !s.allowMockLogin {
		return nil, ErrPhoneLoginDisabled
	}
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
	return s.issueToken(ctx, u)
}

// WeChatLogin 用小程序 wx.login 的 code 换 openid 并登录/建号。
// 微信用户默认没有手机号（phone 为空），后续可在个人中心绑定。
func (s *AuthService) WeChatLogin(ctx context.Context, req WeChatLoginRequest) (*LoginResponse, error) {
	if s.wechat == nil {
		return nil, ErrWeChatDisabled
	}
	openid, unionid, err := s.wechat.Code2Session(ctx, req.Code)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.NickName)
	if name == "" {
		suffix := openid
		if len(suffix) > 4 {
			suffix = suffix[len(suffix)-4:]
		}
		name = "微信用户" + suffix
	}
	avatar := strings.TrimSpace(req.Avatar)
	if avatar == "" {
		avatar = "/static/img/avatar-user.svg"
	}

	u, err := s.users.UpsertByWeChat(ctx, openid, unionid, name, avatar)
	if err != nil {
		return nil, err
	}
	return s.issueToken(ctx, u)
}

// issueToken 是手机号登录和微信登录共用的收尾：禁用校验、发 token、查摄影师身份。
func (s *AuthService) issueToken(ctx context.Context, u repository.User) (*LoginResponse, error) {
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
