package eventsource

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// maxBodyBytes 限制单次响应体大小，避免上游返回怪东西把内存打爆。
const maxBodyBytes = 8 << 20 // 8 MiB

// defaultRetryBackoffs 是「重试 2 次，退避 1s、3s」的默认值。
var defaultRetryBackoffs = []time.Duration{1 * time.Second, 3 * time.Second}

// backoffsOrDefault 区分 nil（用默认）与空切片（调用方显式要求不等待）。
func backoffsOrDefault(v []time.Duration) []time.Duration {
	if v == nil {
		return defaultRetryBackoffs
	}
	return v
}

// defaultHTTPClient 构造内置 client。
//
// 参考 internal/ingest 的经验（nyato 的 WAF 拒绝 Go 默认 HTTP/2），这里统一关掉
// HTTP/2 并给足连接/握手超时：接口层走 HTTPS，强制 HTTP/1.1 更稳。
func defaultHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			ForceAttemptHTTP2:   false,
			DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        10,
			IdleConnTimeout:     60 * time.Second,
		},
	}
}

// getWithRetry 带 ctx 与重试的 GET。
//   - HTTP 非 200 / 网络错误 -> 可重试；
//   - ctx 取消/超时 -> 立即返回；
//   - 重试次数 = len(backoffs)，退避时间依次取 backoffs。
func getWithRetry(ctx context.Context, client *http.Client, url string, headers map[string]string, backoffs []time.Duration) ([]byte, error) {
	attempts := len(backoffs) + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if err := sleepCtx(ctx, backoffs[i-1]); err != nil {
				return nil, err
			}
		}
		body, retryable, err := getOnce(ctx, client, url, headers)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
		if i < attempts-1 {
			logf("请求失败(第 %d/%d 次)，准备重试: %s: %v", i+1, attempts, url, err)
		}
	}
	return nil, lastErr
}

// getOnce 发一次 GET，返回 body / 是否可重试 / 错误。
func getOnce(ctx context.Context, client *http.Client, url string, headers map[string]string) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("构造请求失败: %w", err)
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		return nil, true, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, rerr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if rerr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		return nil, true, fmt.Errorf("读取响应体失败: %w", rerr)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, true, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet)
	}
	return body, false, nil
}

// sleepCtx 可被 ctx 打断的 sleep。
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// pageDelay 返回翻页之间的等待时间。
//   - cfg < 0 : 不等待（测试用）
//   - cfg == 0: 默认 300-500ms 随机抖动（避免被上游风控）
//   - cfg > 0 : 用给定值
func pageDelay(cfg time.Duration) time.Duration {
	switch {
	case cfg < 0:
		return 0
	case cfg > 0:
		return cfg
	default:
		return 300*time.Millisecond + time.Duration(rand.Int63n(int64(200*time.Millisecond)))
	}
}

// browserHeaders 返回浏览器风格请求头。
func browserHeaders(ua, referer string) map[string]string {
	if ua == "" {
		ua = BrowserUA
	}
	return map[string]string{
		"User-Agent":      ua,
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		"Referer":         referer,
	}
}
