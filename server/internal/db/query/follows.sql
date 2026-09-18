-- name: InsertFollow :one
INSERT INTO event_follows (user_id, event_id) VALUES ($1, $2)
ON CONFLICT (user_id, event_id) DO NOTHING
RETURNING id, user_id, event_id, created_at;

-- name: DeleteFollow :exec
DELETE FROM event_follows WHERE user_id = $1 AND event_id = $2;

-- name: ListFollows :many
SELECT f.id, f.event_id, f.created_at, e.name, e.location, e.venue, e.start_date, e.end_date, e.cover_url, e.status
FROM event_follows f
JOIN comic_events e ON e.id = f.event_id
WHERE f.user_id = $1
ORDER BY e.start_date ASC;

-- name: InsertSubscription :one
INSERT INTO event_subscriptions (user_id, event_id, template_id) VALUES ($1, $2, $3)
ON CONFLICT (user_id, event_id) DO UPDATE SET template_id = EXCLUDED.template_id, status = 'accepted'
RETURNING id, user_id, event_id, template_id, status;
