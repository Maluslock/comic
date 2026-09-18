package ingest

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// EventCard represents a single event extracted from a nyato.com listing card.
type EventCard struct {
	Name      string
	City      string
	Venue     string // no venue source in nyato contract; set to "" by parser
	Address   string
	DateStart string // MM/DD
	DateEnd   string // MM/DD
	ImageURL  string // full-size URL (thumb suffix stripped)
}

var (
	thumbSuffixRE = regexp.MustCompile(`!330x450cut$`)
	imgTagRE      = regexp.MustCompile(`<img[^>]*src="([^"]*img\.nyato\.com[^"]*)"[^>]*>`)
	tagStripRE    = regexp.MustCompile(`<[^>]*>`)
	dateRE        = regexp.MustCompile(`(\d{2}/\d{2})\s*-\s*(\d{2}/\d{2})`)
)

// nyatoClient is a shared HTTP client tuned for nyato.com.
//
// IMPORTANT (2026-08-21): nyato's WAF rejects Go's default HTTP/2 client
// with "stream error: INTERNAL_ERROR". Forcing HTTP/1.1 plus a browser-like
// UA (and Referer) passes cleanly. Keep these settings — do not "clean up"
// into a plain http.Client, or fetching silently breaks again.
var nyatoClient = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ForceAttemptHTTP2:   false, // nyato WAF: HTTP/2 -> INTERNAL_ERROR
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        10,
	},
}

func nyatoRequest(baseURL string) (*http.Request, error) {
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", "https://www.nyato.com/manzhan/")
	return req, nil
}

// stripThumbSuffix removes the !330x450cut thumbnail suffix from an image URL.
func stripThumbSuffix(u string) string {
	return thumbSuffixRE.ReplaceAllString(u, "")
}

// FetchEventCards fetches the nyato.com listing page and extracts event cards.
// baseURL should be the full URL (e.g. https://www.nyato.com/manzhan).
func FetchEventCards(baseURL string) ([]EventCard, error) {
	req, err := nyatoRequest(baseURL)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", baseURL, err)
	}
	resp, err := nyatoClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", baseURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return ParseEventCards(string(body))
}

// ParseEventCards extracts event cards from nyato.com HTML.
//
// Card shape verified 2026-07-28:
//
//	<img src="https://img.nyato.com/...jpg!330x450cut">
//	<div>N</div>
//	2026第19届西安星幻动漫节
//	西安市
//	02/21 - 02/21
//	地址：四川省 自贡市 荣县... 综合评分：5
//
// Strategy:
//  1. Find every <img src="https://img.nyato.com/..."> — each is a card.
//  2. From each img position, take the next ~800 chars of text.
//  3. Strip tags, collapse whitespace, split by newline/space.
//  4. Parse fields positionally (verified card shape): hot count → name → city → dates → address.
//  5. stripThumbSuffix(img src) = ImageURL.
//  6. Return nil error; skip malformed cards.
func ParseEventCards(html string) ([]EventCard, error) {
	matches := imgTagRE.FindAllStringSubmatchIndex(html, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var cards []EventCard

	for i, match := range matches {
		// match[0], match[1] = full match start/end
		// match[2], match[3] = src capture group start/end
		src := html[match[2]:match[3]]
		imgEnd := match[1]

		// Determine text chunk boundaries
		var textEnd int
		if i+1 < len(matches) {
			textEnd = matches[i+1][0] // start of next img tag
		} else {
			textEnd = len(html)
		}

		// Get text chunk after this img, limited to 800 chars
		chunk := html[imgEnd:textEnd]
		if len(chunk) > 800 {
			chunk = chunk[:800]
		}

		// Strip HTML tags
		text := tagStripRE.ReplaceAllString(chunk, "")

		card := parseCardText(text, src)
		// Valid card requires name + city + both dates; the real nyato page
		// has a demo card ("喵特门票购买演示") with empty city/dates. Address optional.
		if card.Name != "" && card.City != "" && card.DateStart != "" && card.DateEnd != "" {
			cards = append(cards, card)
		}
	}

	return cards, nil
}

// parseCardText parses a single card's stripped text and image src into an EventCard.
func parseCardText(text string, imgSrc string) EventCard {
	card := EventCard{
		ImageURL: stripThumbSuffix(imgSrc),
		Venue:    "", // no venue source in nyato contract
	}

	// Split by newlines, trim, filter empty
	lines := strings.Split(text, "\n")
	var clean []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}

	for _, line := range clean {
		// Skip hot count (pure digits)
		if isNumeric(line) {
			continue
		}

		// Date range: MM/DD - MM/DD
		if m := dateRE.FindStringSubmatch(line); m != nil {
			card.DateStart = m[1]
			card.DateEnd = m[2]
			continue
		}

		// Address line: starts with 地址： or 地址:
		if strings.HasPrefix(line, "地址：") || strings.HasPrefix(line, "地址:") {
			card.Address = parseAddress(line)
			continue
		}

		// City: ends with 市
		if strings.HasSuffix(line, "市") {
			card.City = line
			continue
		}

		// Name: first remaining non-empty, non-numeric, non-special line
		if card.Name == "" {
			card.Name = line
		}
	}

	return card
}

// parseAddress extracts the address text from an address line.
// Strips the "地址：" prefix and trims at "综合评分：".
func parseAddress(line string) string {
	line = strings.TrimPrefix(line, "地址：")
	line = strings.TrimPrefix(line, "地址:")

	// Trim at 综合评分： or 综合评分:
	if idx := strings.Index(line, "综合评分："); idx >= 0 {
		line = line[:idx]
	} else if idx := strings.Index(line, "综合评分:"); idx >= 0 {
		line = line[:idx]
	}

	return strings.TrimSpace(line)
}

// isNumeric returns true if s contains only ASCII digits and is non-empty.
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
