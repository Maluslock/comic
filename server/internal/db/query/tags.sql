-- name: GetHotTags :many
SELECT name, usage_count
FROM tags
ORDER BY usage_count DESC
LIMIT $1;

-- name: GetAllTags :many
SELECT id, name, usage_count
FROM tags
ORDER BY name ASC;
