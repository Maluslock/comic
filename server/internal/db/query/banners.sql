-- name: GetActiveBanners :many
SELECT id, image_url, title, link_type, link_id, sort_order
FROM banners
WHERE is_active = true
ORDER BY sort_order ASC
LIMIT $1;
