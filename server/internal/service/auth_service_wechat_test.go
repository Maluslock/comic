package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Maluslock/comic/server/internal/repository"
)

func TestWeChatLogin_DisabledUser(t *testing.T) {
	store := &fakeAuthUserStore{user: repository.User{ID: 21, Name: "微信用户", Status: "disabled"}}
	svc := NewAuthService(store, nil, NewMockWeChatClient())

	_, err := svc.WeChatLogin(context.Background(), WeChatLoginRequest{Code: "mock_openid_1234"})
	if !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("want ErrAccountDisabled, got %v", err)
	}
	if store.lastOpenID != "mock_openid_1234" {
		t.Fatalf("openid not passed through: %q", store.lastOpenID)
	}
}

func TestWeChatLogin_BadCode(t *testing.T) {
	svc := NewAuthService(&fakeAuthUserStore{}, nil, NewMockWeChatClient())
	_, err := svc.WeChatLogin(context.Background(), WeChatLoginRequest{Code: "not-a-mock-code"})
	if !errors.Is(err, ErrWeChatCodeBad) {
		t.Fatalf("want ErrWeChatCodeBad, got %v", err)
	}
}

func TestWeChatLogin_NotConfigured(t *testing.T) {
	svc := NewAuthService(&fakeAuthUserStore{}, nil)
	_, err := svc.WeChatLogin(context.Background(), WeChatLoginRequest{Code: "mock_x"})
	if !errors.Is(err, ErrWeChatDisabled) {
		t.Fatalf("want ErrWeChatDisabled, got %v", err)
	}
}
