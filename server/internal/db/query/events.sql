-- name: GetUpcomingEvents :many
SELECT id, allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status
FROM comic_events
WHERE status = 'upcoming'
ORDER BY start_date ASC
LIMIT $1;

-- name: GetEventByAllcppId :one
SELECT id, allcpp_id, name, location, venue, start_date, end_date, cover_url, tags, type_name, status
FROM comic_events
WHERE allcpp_id = $1;

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
ORDER BY start_date ASC
LIMIT $2 OFFSET $3;
