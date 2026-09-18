# nyato.com Ingestion — Extraction Contract

Extraction contract for the nyato.com (喵特) event listing data source. This document is the
single source of truth for what the ingest pipeline extracts from each event card on the
nyato.com 漫展 list page and how the raw values must be normalized before storage.

## Source

| Item | Value |
|------|-------|
| URL | `https://www.nyato.com/manzhan` |
| Access check | `curl -s --max-time 10 https://www.nyato.com/manzhan` → 61,269 bytes HTML (threshold: > 50 KB) |
| Card count on page | 16 event cards (2026-07-28 crawl, re-verified live on ingest of this doc) |
| Encoding note | Event text is Chinese; treat extracted values as UTF-8 strings |

## Contract Fields

Each event card on the listing page yields the following fields (values below are the verified
examples from the 2026-07-28 crawl):

| Field | Source | Example |
|-------|--------|---------|
| `name` | card title text | `2026第19届西安星幻动漫节` |
| `city` | text after name | `西安市` |
| `dateStart` | `MM/DD` range | `02/21` |
| `dateEnd` | `MM/DD` range | `02/21` |
| `address` | 地址： line | `四川省 自贡市 荣县...` |
| `imageUrl` | `<img src>` | `https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg!330x450cut` |
| `score` | 综合评分： | `5` |

### Image Normalization Rule

Strip the `!330x450cut` thumbnail suffix from `imageUrl` to obtain the full-size image.
The suffix is a URL fragment (`<full-url>.jpg!330x450cut` → `<full-url>.jpg`).

```
input:  https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg!330x450cut
output: https://img.nyato.com/data/upload/expo/2026/0528/02/6a17369950bbb.jpg
```

This normalization is applied to `imageUrl` before storage; only the normalized full-size URL
is persisted.

## Card Shape (verified 2026-07-28)

Each card contains, in order:

```
<img src="https://img.nyato.com/...jpg!330x450cut">   -> imageUrl (strip !330x450cut)
<div>N</div>                                          -> hot count (not extracted)
2026第19届西安星幻动漫节                              -> name
西安市                                               -> city
02/21 - 02/21                                         -> dateStart - dateEnd
地址：四川省 自贡市 荣县... 综合评分：5                -> address (trim at 综合评分) / score
```

Parsing notes:
- `name` — first title token on the card.
- `city` — text immediately after the name, ends with `市`.
- `dateStart` / `dateEnd` — the `MM/DD - MM/DD` pair; both values must be present.
- `address` — the 地址： line; trim at `综合评分` so the score text does not leak into it.
- `score` — numeric value after `综合评分：`.

## Downstream Consumption

The ingestion package (`server/internal/ingest`, implemented in a later task) maps these
contract fields into its `EventCard` struct and upserts rows into `comic_events`. The contract
fields map as:

| Contract field | EventCard field |
|----------------|-----------------|
| `name` | `Name` |
| `city` | `City` |
| `dateStart` | `DateStart` |
| `dateEnd` | `DateEnd` |
| `address` | `Address` |
| `imageUrl` (normalized) | `ImageURL` |
| `score` | *(not yet in struct — see note)* |

Note: the planned `EventCard` struct also carries a `Venue` field that this contract does not
source. Reconciliation of `Venue` (add to contract) vs `score` (add to struct) is left to the
ingest implementation task.
