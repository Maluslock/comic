package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Maluslock/comic/server/internal/ingest"
	"github.com/Maluslock/comic/server/internal/repository"
)

var (
	baseURL  = flag.String("url", "https://www.nyato.com/manzhan", "nyato event list URL")
	pages    = flag.Int("pages", 1, "number of pages to crawl (pagination: ?p=N)")
	dsn      = flag.String("dsn", "", "postgres DSN (overrides DB_* env vars)")
	schedule = flag.String("schedule", "", "cron spec: \"0 H * * *\" (H=0-23); empty = run once")
	bilibili = flag.Bool("bilibili", false, "also scrape Bilibili convention listings (best-effort)")
	config   = flag.String("config", "", "path to cron.yaml with task definitions (overrides -schedule/-pages/-url)")
)

// runParams carries per-run settings, parameterized so both flag mode and
// YAML-driven cron mode can dispatch the same pipeline.
type runParams struct {
	url      string
	pages    int
	withBili bool
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// cleanText replaces invalid UTF-8 byte sequences (scraped HTML can contain
// them — e.g. truncated multi-byte chars) so Postgres upserts never fail
// with SQLSTATE 22021.
func cleanText(s string) string {
	if !utf8.ValidString(s) {
		return strings.ToValidUTF8(s, "")
	}
	return s
}

func cleanTextPtr(p *string) *string {
	if p == nil {
		return nil
	}
	return strPtr(cleanText(*p))
}

func buildDSN() string {
	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")
	user := envOrDefault("DB_USER", "comic")
	pass := envOrDefault("DB_PASSWORD", "comic123")
	name := envOrDefault("DB_NAME", "comic")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)
}

// parseDate converts a "MM/DD" string to time.Time.
// Uses the current year, or year+1 if the parsed date is before today (so it stays upcoming).
func parseDate(mmdd string) (time.Time, error) {
	today := time.Now()
	year := today.Year()

	t, err := time.ParseInLocation("01/02", mmdd, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %q: %w", mmdd, err)
	}
	// Set the year
	t = time.Date(year, t.Month(), t.Day(), 0, 0, 0, 0, time.Local)

	// If the date is before today (e.g. parsed date already passed this year), use next year
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	if t.Before(todayStart) {
		t = time.Date(year+1, t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	}

	return t, nil
}

func buildPageURL(base *url.URL, page int) string {
	u := *base // copy: don't mutate the shared base URL
	q := u.Query()
	q.Set("p", strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return u.String()
}

func main() {
	flag.Parse()

	if *config != "" {
		runCronMode(*config)
		return
	}

	if *schedule != "" {
		// Validate schedule spec early.
		if _, err := ingest.NextRunTime(*schedule, time.Now()); err != nil {
			log.Fatalf("bad -schedule: %v", err)
		}
		log.Printf("cron mode: schedule=%q", *schedule)
		for {
			start := time.Now()
			runOnce(runParams{url: *baseURL, pages: *pages, withBili: *bilibili})
			elapsed := time.Since(start)
			next, err := ingest.NextRunTime(*schedule, time.Now())
			if err != nil {
				log.Fatalf("schedule error: %v", err)
			}
			wait := time.Until(next)
			log.Printf("run took %s; next run at %s (%s from now)", elapsed.Round(time.Second), next.Format("2006-01-02 15:04:05"), wait.Round(time.Second))
			time.Sleep(wait)
		}
	}

	runOnce(runParams{url: *baseURL, pages: *pages, withBili: *bilibili})
}

func runOnce(p runParams) {
	var allCards []ingest.EventCard
	seen := make(map[string]bool) // deduplicate by name+city across pages

	base, err := url.Parse(p.url)
	if err != nil {
		log.Fatalf("parse base URL %q: %v", p.url, err)
	}

	var fetchErrors int
	for page := 1; page <= p.pages; page++ {
		pageURL := buildPageURL(base, page)
		log.Printf("crawling page %d/%d: %s", page, p.pages, pageURL)

		cards, err := ingest.FetchEventCards(pageURL)
		if err != nil {
			fetchErrors++
			log.Printf("fetch page %d (%s): %v", page, pageURL, err)
			continue
		}

		var pageCards []ingest.EventCard
		for _, card := range cards {
			key := card.Name + "|" + card.City
			if !seen[key] {
				seen[key] = true
				pageCards = append(pageCards, card)
			}
		}
		log.Printf("page %d: fetched %d cards, %d new (after dedup)", page, len(cards), len(pageCards))
		allCards = append(allCards, pageCards...)

		if page < p.pages {
			time.Sleep(500 * time.Millisecond)
		}
	}

	cards := allCards
	log.Printf("crawl finished: %d/%d pages OK (%d fetch errors), %d unique cards", p.pages-fetchErrors, p.pages, fetchErrors, len(cards))

	if len(cards) == 0 && !p.withBili {
		log.Println("no cards to upsert")
		return
	}

	// Build DSN: explicit flag takes priority, then env vars
	databaseURL := *dsn
	if databaseURL == "" {
		databaseURL = buildDSN()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	queries := repository.New(pool)
	sourceURL := p.url

	var upserted int
	for _, card := range cards {
		allcppID := ingest.AllcppID(card.Name, card.City)
		startDate, err := parseDate(card.DateStart)
		if err != nil {
			log.Printf("skip card %q: bad start_date: %v", card.Name, err)
			continue
		}
		endDate, err := parseDate(card.DateEnd)
		if err != nil {
			log.Printf("skip card %q: bad end_date: %v", card.Name, err)
			continue
		}

		location := strPtr(card.City)
		venue := strPtr("")
		address := strPtr(card.Address)
		if card.Address == "" {
			address = nil
		}
		coverURL := strPtr(card.ImageURL)
		if card.ImageURL == "" {
			coverURL = nil
		}
		typeName := strPtr("综合")

		// Build city tag: strip trailing "市" (e.g. "西安市" → "西安")
		tags := []string{}
		if card.City != "" {
			tags = append(tags, strings.TrimSuffix(card.City, "市"))
		}

		params := repository.UpsertEventFromIngestParams{
			AllcppID:  allcppID,
			Name:      cleanText(card.Name),
			Location:  cleanTextPtr(location),
			Venue:     venue,
			Address:   cleanTextPtr(address),
			StartDate: startDate,
			EndDate:   endDate,
			CoverUrl:  coverURL,
			Tags:      tags,
			TypeName:  typeName,
			SourceUrl: &sourceURL,
		}

		id, err := queries.UpsertEventFromIngest(ctx, params)
		if err != nil {
			log.Printf("upsert %q (allcpp_id=%d): %v", card.Name, allcppID, fmt.Errorf("upsert %q: %w", card.Name, err))
			continue
		}
		log.Printf("upserted %q -> id=%d allcpp_id=%d date=%s-%s tags=%v", card.Name, id, allcppID, card.DateStart, card.DateEnd, tags)
		upserted++
	}

	if p.withBili {
		bCards, err := ingest.FetchBilibiliEvents()
		if err != nil || len(bCards) == 0 {
			log.Printf("bilibili: skipped (%v)", err)
		} else {
			for _, c := range bCards {
				allcppID := ingest.AllcppID(c.Name, c.City)
				startDate, err := parseDate(c.DateStart)
				if err != nil {
					log.Printf("bilibili: skip %q: bad start_date: %v", c.Name, err)
					continue
				}
				endDate, err := parseDate(c.DateEnd)
				if err != nil {
					log.Printf("bilibili: skip %q: bad end_date: %v", c.Name, err)
					continue
				}

				location := strPtr(c.City)
				venue := strPtr("")
				address := strPtr(c.Address)
				if c.Address == "" {
					address = nil
				}
				coverURL := strPtr(c.ImageURL)
				if c.ImageURL == "" {
					coverURL = nil
				}
				typeName := strPtr("综合")

				tags := []string{}
				if c.City != "" {
					tags = append(tags, strings.TrimSuffix(c.City, "市"))
				}

				params := repository.UpsertEventFromIngestParams{
					AllcppID:  allcppID,
					Name:      cleanText(c.Name),
					Location:  cleanTextPtr(location),
					Venue:     venue,
					Address:   cleanTextPtr(address),
					StartDate: startDate,
					EndDate:   endDate,
					CoverUrl:  coverURL,
					Tags:      tags,
					TypeName:  typeName,
					SourceUrl: strPtr("https://www.bilibili.com/"),
				}

				id, err := queries.UpsertEventFromIngest(ctx, params)
				if err != nil {
					log.Printf("bilibili: upsert %q (allcpp_id=%d): %v", c.Name, allcppID, err)
					continue
				}
				log.Printf("bilibili: upserted %q -> id=%d allcpp_id=%d date=%s-%s tags=%v", c.Name, id, allcppID, c.DateStart, c.DateEnd, tags)
				upserted++
			}
		}
	}

	log.Printf("done: %d cards fetched, %d upserted", len(cards), upserted)

	// Soft-delete expired events so the home page stops showing "0天后".
	expired, err := pool.Exec(ctx, "UPDATE comic_events SET del_flag = true WHERE start_date < NOW() AND del_flag = false")
	if err != nil {
		log.Printf("mark expired events: %v", err)
	} else {
		log.Printf("marked %d expired events as deleted", expired.RowsAffected())
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
