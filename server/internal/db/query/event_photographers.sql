-- name: ApplyEventPhotographer :one
INSERT INTO event_photographers (event_id, photographer_id, user_id, message)
VALUES ($1, $2, $3, $4)
RETURNING id, event_id, photographer_id, user_id, status, message, review_reason, applied_at, reviewed_at, admin_id;

-- name: GetMyEventApplication :one
SELECT id, event_id, photographer_id, user_id, status, message, review_reason, applied_at, reviewed_at, admin_id
FROM event_photographers
WHERE event_id = $1 AND user_id = $2
ORDER BY applied_at DESC
LIMIT 1;

-- name: ListMyEventApplications :many
SELECT a.id, a.event_id, a.photographer_id, a.user_id, a.status, a.message, a.review_reason,
       a.applied_at, a.reviewed_at, a.admin_id,
       e.name, e.start_date, e.end_date, e.location, e.venue, e.cover_url
FROM event_photographers a
JOIN comic_events e ON e.id = a.event_id
WHERE a.user_id = $1
ORDER BY a.applied_at DESC;

-- name: WithdrawEventApplication :execrows
UPDATE event_photographers
SET status = 'withdrawn', reviewed_at = NOW()
WHERE event_id = $1 AND user_id = $2 AND status IN ('pending', 'approved');

-- name: ListApprovedPhotographersByEvent :many
SELECT p.id, p.name, p.avatar, p.description, p.location, p.rating, p.review_count, p.order_count,
       p.user_id, p.mode, p.certified,
       array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL) AS tags
FROM event_photographers ep
JOIN photographers p ON p.id = ep.photographer_id
LEFT JOIN photographer_tags pt ON p.id = pt.photographer_id
LEFT JOIN tags t ON pt.tag_id = t.id
WHERE ep.event_id = $1 AND ep.status = 'approved'
GROUP BY p.id
ORDER BY p.rating DESC, p.order_count DESC, p.id ASC;

-- name: ReviveWithdrawnEventApplication :one
UPDATE event_photographers
SET status = 'pending',
    message = $3,
    applied_at = NOW(),
    reviewed_at = NULL,
    review_reason = NULL,
    admin_id = NULL
WHERE event_id = $1 AND photographer_id = $2 AND status = 'withdrawn'
RETURNING id, event_id, photographer_id, user_id, status, message, review_reason, applied_at, reviewed_at, admin_id;

-- name: CountApprovedByEvent :one
SELECT COUNT(*) FROM event_photographers WHERE event_id = $1 AND status = 'approved';

-- name: ListEventApplications :many
SELECT a.id, a.event_id, a.photographer_id, a.user_id, a.status, a.message, a.review_reason,
       a.applied_at, a.reviewed_at, a.admin_id,
       COALESCE(e.name, '') AS event_name, e.start_date, e.end_date,
       COALESCE(p.name, '') AS photographer_name,
       COALESCE(u.name, '') AS user_name
FROM event_photographers a
LEFT JOIN comic_events e ON e.id = a.event_id
LEFT JOIN photographers p ON p.id = a.photographer_id
LEFT JOIN users u ON u.id = a.user_id
WHERE ($1::text = '' OR a.status = $1)
ORDER BY a.applied_at DESC
LIMIT $2 OFFSET $3;

-- name: CountEventApplications :one
SELECT COUNT(*) FROM event_photographers WHERE ($1::text = '' OR status = $1);

-- name: ReviewEventApplication :execrows
UPDATE event_photographers
SET status = $2,
    review_reason = CASE WHEN $3 = '' THEN NULL ELSE $3 END,
    admin_id = $4,
    reviewed_at = NOW()
WHERE id = $1 AND status = 'pending';

-- name: GetEventApplicationByID :one
SELECT id, event_id, photographer_id, user_id, status, message, review_reason, applied_at, reviewed_at, admin_id
FROM event_photographers
WHERE id = $1;
