-- name: GetFeaturedWorks :many
SELECT w.id, w.photographer_id, w.title, w.images, w.description, w.created_at,
       p.name AS photographer_name
FROM works w
JOIN photographers p ON w.photographer_id = p.id
ORDER BY w.created_at DESC
LIMIT $1;

-- name: GetWorksByPhotographer :many
SELECT id, photographer_id, title, images, description, created_at
FROM works
WHERE photographer_id = $1
ORDER BY created_at DESC;
