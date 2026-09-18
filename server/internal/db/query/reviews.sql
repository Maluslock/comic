-- name: GetReviewsByPhotographer :many
SELECT id, photographer_id, user_id, user_name, user_avatar, rating, content, images, created_at
FROM reviews
WHERE photographer_id = $1
ORDER BY created_at DESC;

-- name: CreateReview :one
INSERT INTO reviews (photographer_id, user_id, user_name, user_avatar, rating, content, images)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, photographer_id, user_id, user_name, user_avatar, rating, content, images, created_at;
