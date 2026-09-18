package ingest

import (
	"io"
	"log"
	"net/http"
	"time"
)

// FetchBilibiliEvents scrapes Bilibili anime convention listings.
// Best-effort: returns (nil, nil) on any parse/fetch failure so the
// caller can continue with other sources. Never fatal.
//
// Implementation reality (2026-07-28):
//   - GET https://www.bilibili.com/ with a browser-like UA header.
//   - Bilibili's main page is a JS-rendered SPA; the HTML body contains
//     essentially no static event/convention content — the DOM is
//     populated client-side by React/Vue bundles.
//   - Known Bilibili activity endpoints (e.g. /blackboard/activity-*.html)
//     are event-specific deep links, not a listing feed. No public
//     unauthenticated API for convention/event listings was identified.
//   - Consequently this function currently returns (nil, nil) after a
//     best-effort HTTP fetch attempt.
//
// Future work: if Bilibili exposes a public API endpoint (or a
// server-rendered listing page) for anime conventions, drop it in here.
// The signature is intentionally stable: []EventCard, error.
func FetchBilibiliEvents() ([]EventCard, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", "https://www.bilibili.com/", nil)
	if err != nil {
		log.Printf("bilibili: request creation failed: %v", err)
		return nil, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("bilibili: fetch failed: %v", err)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("bilibili: unexpected status %d", resp.StatusCode)
		return nil, nil
	}

	// Read body just to confirm it arrived; the SPA HTML contains no
	// usable event listing data.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		log.Printf("bilibili: read body failed: %v", err)
		return nil, nil
	}

	_ = len(body) // silence unused; body is discarded

	return nil, nil
}
