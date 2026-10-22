package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestPhoneLogin_DisabledInProduction(t *testing.T) {
	svc := NewAuthService(&fakeAuthUserStore{}, nil)
	svc.SetMockLoginAllowed(false)
	_, err := svc.Login(context.Background(), LoginRequest{Phone: "13800138000", Code: "123456"})
	if !errors.Is(err, ErrPhoneLoginDisabled) {
		t.Fatalf("want ErrPhoneLoginDisabled, got %v", err)
	}
}

func TestHTTPWeChatClient_MapsErrCodes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want error
	}{
		{"invalid-code", `{"errcode":40029,"errmsg":"invalid code"}`, ErrWeChatCodeBad},
		{"used-code", `{"errcode":40163,"errmsg":"code been used"}`, ErrWeChatCodeBad},
		{"bad-secret", `{"errcode":40125,"errmsg":"invalid appsecret"}`, ErrWeChatConfig},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			client := NewWeChatClient("appid", "secret", srv.URL)
			_, _, err := client.Code2Session(context.Background(), "any-code")
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}
