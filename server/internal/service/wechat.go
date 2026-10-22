package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WeChatClient 把 wx.login 拿到的临时 code 换成 openid。
// 抽成接口是为了让单测不依赖真实微信网关。
type WeChatClient interface {
	Code2Session(ctx context.Context, code string) (openid string, unionid string, err error)
}

var (
	ErrWeChatDisabled = errors.New("wechat login is not configured")
	ErrWeChatCodeBad  = errors.New("invalid wechat code")
	// ErrWeChatConfig 表示 AppID/AppSecret 不匹配（40125）等配置问题，和「用户 code 过期」要区分开。
	ErrWeChatConfig = errors.New("wechat appid or secret is invalid")
)

// HTTPWeChatClient 调 https://api.weixin.qq.com/sns/jscode2session。
// 微信这个接口即使出错也返回 200，必须看 errcode。
type HTTPWeChatClient struct {
	appID   string
	secret  string
	baseURL string
	http    *http.Client
}

func NewWeChatClient(appID, secret, baseURL string) *HTTPWeChatClient {
	if baseURL == "" {
		baseURL = "https://api.weixin.qq.com"
	}
	return &HTTPWeChatClient{
		appID:   appID,
		secret:  secret,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPWeChatClient) Code2Session(ctx context.Context, code string) (string, string, error) {
	if code == "" {
		return "", "", ErrWeChatCodeBad
	}
	q := url.Values{}
	q.Set("appid", c.appID)
	q.Set("secret", c.secret)
	q.Set("js_code", code)
	q.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/sns/jscode2session?"+q.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("wechat request: %w", err)
	}
	defer resp.Body.Close()

	var body struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", fmt.Errorf("wechat decode: %w", err)
	}
	if body.ErrCode != 0 {
		// 40029/40163：code 无效或已被使用；40125：AppSecret 错误。
		switch body.ErrCode {
		case 40029, 40163:
			return "", "", ErrWeChatCodeBad
		case 40125:
			return "", "", ErrWeChatConfig
		}
		return "", "", fmt.Errorf("wechat errcode=%d errmsg=%s", body.ErrCode, body.ErrMsg)
	}
	if body.OpenID == "" {
		return "", "", ErrWeChatCodeBad
	}
	return body.OpenID, body.UnionID, nil
}

// MockWeChatClient 仅用于本地/演示环境：WX_APPID 未配置时接受 mock_ 开头的 code，
// openid = code 本身。生产必须配置真实 AppID/Secret。
type MockWeChatClient struct{}

func NewMockWeChatClient() *MockWeChatClient { return &MockWeChatClient{} }

func (m *MockWeChatClient) Code2Session(_ context.Context, code string) (string, string, error) {
	if !strings.HasPrefix(code, "mock_") {
		return "", "", ErrWeChatCodeBad
	}
	return code, "", nil
}
