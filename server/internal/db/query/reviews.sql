-- name: GetReviewsByPhotographer :many
SELECT id, photographer_id, user_id, user_name, user_avatar, rating, content, images, created_at
FROM reviews
WHERE photographer_id = $1
ORDER BY created_at DESC;
