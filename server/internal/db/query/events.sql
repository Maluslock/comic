-- name: GetUpcomingEvents :many
SELECT id, allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status
FROM comic_events
WHERE status = 'upcoming' AND del_flag = false
ORDER BY start_date ASC
LIMIT $1;

-- name: GetEventById :one
SELECT id, allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status
FROM comic_events
WHERE id = $1 AND del_flag = false;

-- name: GetEventByAllcppId :one
SELECT id, allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status
FROM comic_events
WHERE allcpp_id = $1 AND del_flag = false;

-- name: UpsertEvent :one
INSERT INTO comic_events (
  allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status, raw_data, synced_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
)
ON CONFLICT (allcpp_id) DO UPDATE SET
  name = EXCLUDED.name,
  location = EXCLUDED.location,
  venue = EXCLUDED.venue,
  start_date = EXCLUDED.start_date,
  end_date = EXCLUDED.end_date,
  cover_url = EXCLUDED.cover_url,
  tags = EXCLUDED.tags,
  type_name = EXCLUDED.type_name,
  status = EXCLUDED.status,
  raw_data = EXCLUDED.raw_data,
  updated_at = NOW(),
  synced_at = NOW()
RETURNING id, allcpp_id, name;

-- name: SearchEvents :many
SELECT id, allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status
FROM comic_events
WHERE ($1::text IS NULL OR location ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR status = $2)
  AND del_flag = false
ORDER BY start_date ASC
LIMIT $3 OFFSET $4;

-- name: UpsertEventFromIngest :one
INSERT INTO comic_events (allcpp_id, name, location, venue, address, start_date, end_date, cover_url, tags, type_name, status, source_url, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'upcoming', $11, NOW())
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
  synced_at = NOW(),
  updated_at = NOW()
RETURNING id;

-- name: GetAllEvents :many
SELECT id, allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status
FROM comic_events
WHERE del_flag = false
ORDER BY start_date DESC
LIMIT $1 OFFSET $2;
