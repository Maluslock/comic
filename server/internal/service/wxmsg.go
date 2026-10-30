package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WxMsgService 负责微信订阅消息下发：缓存 stable access_token，按模板 ID 调
// subscribe/send。任何失败都只记日志、不向上抛 —— 推送是旁路，绝不能把已成功的
// 下单/确认变成失败（与 notification 的 best-effort 语义一致）。
type WxMsgService struct {
	appid   string
	secret  string
	apiBase string
	state   string // miniprogram_state: developer / trial / formal
	enabled bool

	mu       sync.Mutex
	token    string
	expireAt time.Time
	client   *http.Client
}

// NewWxMsgService 从环境变量装配。缺 appid/secret 或显式关闭时返回 nil，
// 调用方据此跳过推送（本地/CI 没有微信凭据也能跑）。
func NewWxMsgService() *WxMsgService {
	appid := strings.TrimSpace(os.Getenv("WX_APPID"))
	secret := strings.TrimSpace(os.Getenv("WX_APP_SECRET"))
	if appid == "" || secret == "" || os.Getenv("WX_MSG_ENABLED") == "false" {
		return nil
	}
	base := strings.TrimRight(os.Getenv("WX_API_BASE"), "/")
	if base == "" {
		base = "https://api.weixin.qq.com"
	}
	state := os.Getenv("WX_MP_STATE")
	if state == "" {
		state = "formal"
	}
	return &WxMsgService{
		appid:   appid,
		secret:  secret,
		apiBase: base,
		state:   state,
		enabled: true,
		client:  &http.Client{Timeout: 8 * time.Second},
	}
}

type stableTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
}

// accessToken 走 stable_token（不会被其它服务顶掉），提前 5 分钟过期以便重试。
func (s *WxMsgService) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Now().Before(s.expireAt) {
		return s.token, nil
	}
	body, _ := json.Marshal(map[string]any{
		"grant_type":    "client_credential",
		"appid":         s.appid,
		"secret":        s.secret,
		"force_refresh": false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBase+"/cgi-bin/stable_token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var tr stableTokenResp
	if err := json.Unmarshal(raw, &tr); err != nil {
		return "", fmt.Errorf("stable_token decode: %w", err)
	}
	if tr.Errcode != 0 || tr.AccessToken == "" {
		return "", fmt.Errorf("stable_token errcode=%d errmsg=%s", tr.Errcode, tr.Errmsg)
	}
	ttl := tr.ExpiresIn
	if ttl <= 0 {
		ttl = 7200
	}
	s.token = tr.AccessToken
	s.expireAt = time.Now().Add(time.Duration(ttl-300) * time.Second)
	return s.token, nil
}

// clamp 按微信字段长度限制截断：thing 20 字、phrase 5 字、其它保守 32。
func clamp(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "—"
	}
	r := []rune(v)
	if len(r) > 20 {
		return string(r[:19]) + "…"
	}
	return v
}

// SendSubscribe 下发一条订阅消息。data 的 key 必须是后台生成的字段 ID
// （thing1/time2/...），value 直接是展示文本，这里统一做长度裁剪。
func (s *WxMsgService) SendSubscribe(ctx context.Context, openid, tmplID, page string, data map[string]string) {
	if s == nil || !s.enabled || openid == "" || tmplID == "" {
		return
	}
	token, err := s.accessToken(ctx)
	if err != nil {
		log.Printf("wxmsg token failed: %v", err)
		return
	}
	fields := make(map[string]map[string]string, len(data))
	for k, v := range data {
		fields[k] = map[string]string{"value": clamp(v)}
	}
	payload, _ := json.Marshal(map[string]any{
		"touser":            openid,
		"template_id":       tmplID,
		"page":              page,
		"miniprogram_state": s.state,
		"lang":              "zh_CN",
		"data":              fields,
	})
	url := fmt.Sprintf("%s/cgi-bin/message/subscribe/send?access_token=%s", s.apiBase, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("wxmsg send failed: %v", err)
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
	var out struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(raw, &out)
	// 43101 = 用户没订阅或额度用尽，属正常，不打错误日志刷屏。
	if out.Errcode == 43101 {
		return
	}
	if out.Errcode != 0 {
		log.Printf("wxmsg send errcode=%d errmsg=%s tmpl=%s", out.Errcode, out.Errmsg, tmplID)
	}
}

// TmplIDs 汇总三张模板 ID 与各自的字段 ID 顺序。
//
// 字段 ID（thing1/time2/amount3…）由微信在提交模板时按关键词顺序生成，代码里
// 猜不出来，所以放 env；不配时按 thing1..thingN 兜底，配错会在下发时收到 47003，
// 日志里有模板 ID 可直接定位。
type TmplIDs struct {
	NewBooking string   // 收到预约通知（发给摄影师）
	Result     string   // 预约结果通知（发给 coser）
	Delivered  string   // 相册交付状态通知（发给 coser）
	NewFields  []string // 顺序：拍摄日期、拍摄类型、价格规格、备注
	ResFields  []string // 顺序：预约对象、预约结果、预约拍摄日期、备注、预约单号
	DelFields  []string // 顺序：摄影师、拍摄时间、备注
}

func fieldsOf(env string, n int) []string {
	raw := strings.TrimSpace(os.Getenv(env))
	var out []string
	if raw != "" {
		for _, f := range strings.Split(raw, ",") {
			if f = strings.TrimSpace(f); f != "" {
				out = append(out, f)
			}
		}
	}
	for len(out) < n {
		out = append(out, fmt.Sprintf("thing%d", len(out)+1))
	}
	return out[:n]
}

func LoadTmplIDs() TmplIDs {
	return TmplIDs{
		NewBooking: strings.TrimSpace(os.Getenv("WX_TMPL_NEW_BOOKING")),
		Result:     strings.TrimSpace(os.Getenv("WX_TMPL_RESULT")),
		Delivered:  strings.TrimSpace(os.Getenv("WX_TMPL_DELIVERED")),
		NewFields:  fieldsOf("WX_FIELDS_NEW_BOOKING", 4),
		ResFields:  fieldsOf("WX_FIELDS_RESULT", 5),
		DelFields:  fieldsOf("WX_FIELDS_DELIVERED", 3),
	}
}

// buildData 把有序的值按字段 ID 组装成下发 data，长度不齐时截断到二者最小值。
func buildData(ids, values []string) map[string]string {
	n := len(ids)
	if len(values) < n {
		n = len(values)
	}
	out := make(map[string]string, n)
	for i := 0; i < n; i++ {
		out[ids[i]] = values[i]
	}
	return out
}

// priceText 让「价格规格」既适配 amount（纯数字）也适配 thing（带文字）。
// 面议单没有一口价，直接给「面议」，避免下发一个 0。
func priceText(total int32, mode string) string {
	switch mode {
	case "mutual":
		return "互勉"
	case "negotiable":
		if total > 0 {
			return strconv.Itoa(int(total))
		}
		return "面议"
	}
	if total <= 0 {
		return "面议"
	}
	return strconv.Itoa(int(total))
}

// dateTimeText 微信 time 类字段吃 "2006-01-02 15:04"，date 只有日期。
func dateTimeText(date, clock string) string {
	if date == "" {
		return "—"
	}
	if clock == "" {
		return date
	}
	return date + " " + clock
}
