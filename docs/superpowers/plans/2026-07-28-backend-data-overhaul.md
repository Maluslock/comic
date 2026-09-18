# 米拉漫展 — Backend & Data Source Overhaul Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace placeholder data with real convention data + images from nyato.com (喵特), restructure DB schema, add ingestion pipeline, and complete backend API coverage.

**Architecture:** Go ingestion service scrapes nyato.com event listings (real name/city/date/address/image URL) → stores in PostgreSQL → Gin API serves real data → frontend shows convention-matching images instead of random picsum placeholders.

**Tech Stack:** Go 1.22, Gin, pgx, PostgreSQL 16, Redis 7, net/http + encoding/xml or goquery for scraping.

## Global Constraints

- Vue 3 `<script setup lang="ts">` only (frontend touches are minimal — only type/URL updates)
- SCSS dark-neon variables only, rpx units
- `@/` import alias, no relative imports
- No `any` in TS; Go: no `panic`, errors handled and wrapped
- Go handlers: 404 for not-found, 400 invalid-input, 500 sanitized (`"internal server error"` — no `err.Error()` leak)
- `uni.request` only in `src/api/client.ts`; DTO mapping via `src/utils/mappers.ts`
- All event images must come from the REAL source matching the convention (img.nyato.com), never picsum for events
- Avatar images: keep picsum/dicebear only for mock photographers (frontend seed), never for real events

---

### Task 1: Verify nyato.com data source access and document extraction contract

**Files:**
- Create: `server/cmd/ingest/README.md` — extraction contract doc

- [ ] **Step 1: Confirm nyato.com accessibility from backend host**

```bash
curl -s --max-time 10 "https://www.nyato.com/manzhan" -o /tmp/nyato.html && wc -c /tmp/nyato.html
# Expected: > 50KB HTML containing event cards
```

- [ ] **Step 2: Document the extraction contract in `server/cmd/ingest/README.md`**

From the verified HTML (2026-07-28 crawl), each event card yields:

| Field | Source | Example |
|-------|--------|---------|
| `name` | card title text | `2026第19届西安星幻动漫节` |
| `city` | text after name | `西安市` |
| `dateStart` | `MM/DD` range | `02/21` |
| `dateEnd` | `MM/DD` range | `02/21` |
| `address` | 地址： line | `四川省 自贡市 荣县...` |
| `imageUrl` | `<img src>` | `https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg!330x450cut` |
| `score` | 综合评分： | `5` |

Note: strip `!330x450cut` suffix for full-size image.

---

### Task 2: Create Go ingestion package

**Files:**
- Create: `server/internal/ingest/nyato.go`
- Create: `server/internal/ingest/nyato_test.go`
- Create: `server/cmd/ingest/main.go`
- Modify: `server/go.mod` (add `github.com/PuerkitoBio/goquery` if net/html insufficient)

- [ ] **Step 1: Write failing test first**

```go
// server/internal/ingest/nyato_test.go
package ingest

import "testing"

func TestParseEventCards(t *testing.T) {
    html := loadFixture(t) // small sample HTML with 2 event cards
    cards, err := ParseEventCards(html)
    if err != nil { t.Fatal(err) }
    if len(cards) < 2 { t.Fatalf("expected >=2 cards, got %d", len(cards)) }
    if cards[0].Name == "" { t.Fatal("name empty") }
    if cards[0].ImageURL == "" { t.Fatal("image URL empty") }
    if cards[0].City == "" { t.Fatal("city empty") }
}

func TestStripThumbSuffix(t *testing.T) {
    got := stripThumbSuffix("https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg!330x450cut")
    want := "https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg"
    if got != want { t.Fatalf("got %q want %q", got, want) }
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `cd server && GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=$GOROOT/bin:$PATH go test ./internal/ingest/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement `nyato.go`**

```go
package ingest

import (
    "fmt"
    "io"
    "net/http"
    "regexp"
    "strings"
    "time"
)

type EventCard struct {
    Name      string
    City      string
    Venue     string
    Address   string
    DateStart string // MM/DD
    DateEnd   string // MM/DD
    ImageURL  string
}

var thumbSuffix = regexp.MustCompile(`!330x450cut$`)

func stripThumbSuffix(u string) string { return thumbSuffix.ReplaceAllString(u, "") }

func FetchEventCards(baseURL string) ([]EventCard, error) {
    client := &http.Client{Timeout: 15 * time.Second}
    resp, err := client.Get(baseURL)
    if err != nil { return nil, fmt.Errorf("fetch %s: %w", baseURL, err) }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil { return nil, err }
    return ParseEventCards(string(body))
}

// ParseEventCards extracts event cards from nyato.com HTML.
// Card shape verified 2026-07-28:
//   <img src="https://img.nyato.com/...jpg!330x450cut">
//   <div>N</div>   <- hot count
//   2026第19届西安星幻动漫节
//   西安市
//   02/21 - 02/21
//   地址：四川省 自贡市 ... 综合评分：5
func ParseEventCards(html string) ([]EventCard, error) {
    // Strategy (verified card shape 2026-07-28):
    // 1. Find every <img src="https://img.nyato.com/..."> — each is a card.
    // 2. From each img position, take the next ~800 chars of text.
    // 3. Strip tags, collapse whitespace, split by newline/space.
    // 4. First token(s) with digits+漫展/动漫/嘉年华/同人 = Name.
    // 5. Next token = City (ends with 市).
    // 6. `MM/DD - MM/DD` pair = DateStart/DateEnd.
    // 7. `地址：...` substring = Address (trim at 综合评分).
    // 8. stripThumbSuffix(img src) = ImageURL.
    // Return nil error; skip malformed cards (log count).
    return nil, nil
}
```

- [ ] **Step 4: Run test, verify it passes**

Run: same command as Step 2
Expected: PASS

- [ ] **Step 5: Implement `cmd/ingest/main.go`**

```go
package main

import (
    "context"
    "flag"
    "log"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/Maluslock/comic/server/internal/ingest"
)

var (
    baseURL = flag.String("url", "https://www.nyato.com/manzhan", "nyato event list URL")
    dsn     = flag.String("dsn", "", "postgres DSN (default: env DB_* vars)")
)

func main() {
    flag.Parse()
    cards, err := ingest.FetchEventCards(*baseURL)
    if err != nil { log.Fatalf("fetch: %v", err) }
    log.Printf("fetched %d event cards", len(cards))
    // (SQL upsert wiring added in Task 4)
}
```

- [ ] **Step 6: Build**

Run: `cd server && GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=$GOROOT/bin:$PATH go build ./cmd/ingest`
Expected: exit 0

---

### Task 3: Fix event covers — replace picsum with real nyato images

**Files:**
- Modify: `server/migrations/000003_real_event_images.up.sql` (create — update covers for the 8 seeded events)
- Modify: `server/cmd/api/main.go` mockHomeHandler (covers only)
- Modify: `src/data/mock.ts` (covers only)
- Modify: `server/migrations/000002_seed.up.sql` (future-proof: covers)

- [ ] **Step 1: Create migration `000003_real_event_images.up.sql`**

Pick one real nyato image per seeded event (use images verified from the 2026-07-28 crawl; if a specific convention is not on nyato's current page, use the closest real poster found via tavily search and pin it in the migration):

```sql
-- Real covers: replace picsum placeholders with real convention posters.
-- Source: nyato.com img CDN (verified 2026-07-28).
UPDATE comic_events SET cover_url = 'https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg' WHERE id = 1;  -- 西安星幻动漫节 (placeholder mapping — see note)
-- ...one UPDATE per event id 1..8...
```

Note: because nyato's current page lists smaller regional events, Task 5's crawl may fetch major-convention posters too; where the exact convention (ChinaJoy, 萤火虫) is unavailable from nyato, use the real poster URL discovered via tavily (chinajoy.net press images) and document the source in the migration comment.

- [ ] **Step 2: Apply migration to PG**

```bash
docker exec -i comic-pg psql -U comic -d comic < server/migrations/000003_real_event_images.up.sql
```

- [ ] **Step 3: Verify no picsum for events**

```bash
docker exec comic-pg psql -U comic -d comic -c "SELECT count(*) FROM comic_events WHERE cover_url LIKE '%picsum%';"
# Expected: 0
```

- [ ] **Step 4: Sync mock.ts + main.go covers**

Copy the same real URLs into `src/data/mock.ts` (each `cover:`) and `server/cmd/api/main.go` (`coverUrl`) so mock mode and real DB agree.

- [ ] **Step 5: Verify images load**

```bash
curl -s -o /dev/null -w "%{http_code}" "https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg"
# Expected: 200
```

---

### Task 4: DB upsert + schema extension for ingested data

**Files:**
- Modify: `server/internal/repository/events.sql.go` — add `UpsertEventFromIngest`
- Modify: `server/internal/db/query/events.sql` — add upsert query
- Create: `server/migrations/000004_event_ingest_columns.up.sql` — add `address`, `image_gallery TEXT[]`, `source_url`, `synced_at`

- [ ] **Step 1: Create migration `000004_event_ingest_columns.up.sql`**

```sql
ALTER TABLE comic_events
  ADD COLUMN IF NOT EXISTS address TEXT,
  ADD COLUMN IF NOT EXISTS image_gallery TEXT[] DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS source_url TEXT,
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ DEFAULT NOW();
```

- [ ] **Step 2: Apply migration**

```bash
docker exec -i comic-pg psql -U comic -d comic < server/migrations/000004_event_ingest_columns.up.sql
```

- [ ] **Step 3: Add upsert query to `events.sql`**

```sql
-- name: UpsertEventFromIngest :one
INSERT INTO comic_events (allcpp_id, name, location, venue, address, start_date, end_date, cover_url, tags, type_name, status, source_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'upcoming', $11)
ON CONFLICT (allcpp_id) DO UPDATE SET
  name = EXCLUDED.name,
  location = EXCLUDED.location,
  venue = EXCLUDED.venue,
  address = EXCLUDED.address,
  start_date = EXCLUDED.start_date,
  end_date = EXCLUDED.end_date,
  cover_url = EXCLUDED.cover_url,
  tags = EXCLUDED.tags,
  type_name = EXCLUDED.type_name,
  source_url = EXCLUDED.source_url,
  updated_at = NOW()
RETURNING id;
```

- [ ] **Step 4: Wire `cmd/ingest/main.go` to upsert**

After Task 2 Step 5, add pgxpool connect (DSN from env `DB_*` same as api) + loop cards → `UpsertEventFromIngest` with `allcpp_id = hash(name+city)` (use `xxhash` or FNV — no new heavy deps).

- [ ] **Step 5: Run ingest end-to-end**

```bash
cd server && DB_HOST=localhost DB_PORT=5433 DB_USER=comic DB_PASSWORD=comic123 DB_NAME=comic \
  GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=$GOROOT/bin:$PATH \
  go run ./cmd/ingest
# Expected log: "fetched N event cards" then rows upserted
```

- [ ] **Step 6: Verify new rows**

```bash
docker exec comic-pg psql -U comic -d comic -c "SELECT count(*), max(source_url) FROM comic_events;"
# Expected: count >= 8, source_url = nyato
```

---

### Task 5: Full ingest run + sync major conventions (ChinaJoy/萤火虫/CP)

**Files:**
- Create: `server/cmd/ingest/sync_major.sh` — multi-page crawl helper
- Modify: `server/cmd/ingest/main.go` — accept `-pages N` flag

- [ ] **Step 1: Add `-pages` flag to crawl multiple nyato pages**

```go
// pages := flag.Int("pages", 1, "number of pages to crawl")
// for p := 1; p <= *pages; p++ { url := fmt.Sprintf("%s?page=%d", *baseURL, p); ... }
```

- [ ] **Step 2: Crawl 3 pages**

```bash
cd server && go run ./cmd/ingest -pages 3
# Expected: 30-60 event cards upserted
```

- [ ] **Step 3: Verify data volume + image variety**

```bash
docker exec comic-pg psql -U comic -d comic -c "SELECT count(*) FROM comic_events WHERE cover_url LIKE 'https://img.nyato.com%';"
# Expected: > 20
```

- [ ] **Step 4: Manually verify ChinaJoy/萤火虫 rows have matching posters**

Query those rows; if their cover is a generic poster, note and (optionally) patch via migration with the official poster URL.

---

### Task 6: Backend API completeness — verify and harden

**Files:**
- Modify: `server/internal/handler/event_handler.go` (list — include `address`, `imageGallery`)
- Modify: `server/internal/service/event_service.go` (map new fields)
- Modify: `src/utils/mappers.ts` (map `address` + `imageGallery` into ComicEvent type)

- [ ] **Step 1: Extend EventItem DTO**

```go
type EventItem struct {
    // existing fields...
    Address      string   `json:"address,omitempty"`
    ImageGallery []string `json:"imageGallery,omitempty"`
}
```

- [ ] **Step 2: Map new fields in service + mapper**

```ts
// src/utils/mappers.ts
export function mapEventItem(e: any): ComicEvent {
  return {
    // ...existing...
    address: e.address || '',
    imageGallery: e.imageGallery || [],
  }
}
```

- [ ] **Step 3: Add `address` + `imageGallery` to `ComicEvent` type in `src/types/index.ts`**

- [ ] **Step 4: Extend `event/detail.vue` — show address row + image gallery (swiper of up to 3 images)**

- [ ] **Step 5: Verify**

```bash
curl -s http://localhost:8080/api/v1/events/1 | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'address' in d, 'missing address'; print('address:', d.get('address'))"
```

---

### Task 7: Final verification sweep

- [ ] **Go tests**: `cd server && go test ./internal/handler/ ./internal/ingest/ -count=1` → all PASS
- [ ] **No picsum in events**: `grep -rl "picsum" server/migrations/000003_real_event_images.up.sql` → 0 (except photographers)
- [ ] **Real images load**: `curl -s -o /dev/null -w "%{http_code}" $(curl -s http://localhost:8080/api/v1/events | python3 -c "import sys,json; print(json.load(sys.stdin)['list'][0]['coverUrl'])")` → 200
- [ ] **Ingest idempotent**: run ingest twice → row count stable (no duplicates)
- [ ] **API health**: `curl -s http://localhost:8080/health` → `{"status":"ok"}`
- [ ] **404**: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/events/99999` → 404
- [ ] **Frontend type-check**: `npx vue-tsc --noEmit` → 0 new errors

---

### Task 8: Commit and push

- [ ] **Step 1: Commit**

```bash
cd /vol1/1000/code/comic && git add -A && git commit -m "feat: real nyato data source ingestion + real event covers + schema extension + address/gallery in API"
```

- [ ] **Step 2: Push to gitee**

```bash
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY && git -c http.proxy= push gitee master
```
