-- name: GetHotTags :many
SELECT name, usage_count
FROM tags
ORDER BY usage_count DESC
LIMIT $1;
