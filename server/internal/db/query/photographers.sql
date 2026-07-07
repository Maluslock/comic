-- name: GetRecommendedPhotographers :many
SELECT p.id, p.name, p.avatar, p.description, p.location, p.rating, p.review_count, p.order_count,
       array_agg(t.name) AS tags
FROM photographers p
LEFT JOIN photographer_tags pt ON p.id = pt.photographer_id
LEFT JOIN tags t ON pt.tag_id = t.id
GROUP BY p.id
ORDER BY p.rating DESC, p.order_count DESC
LIMIT $1;

-- name: GetPhotographerById :one
SELECT p.id, p.name, p.avatar, p.description, p.location, p.rating, p.review_count, p.order_count,
       array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL) AS tags
FROM photographers p
LEFT JOIN photographer_tags pt ON p.id = pt.photographer_id
LEFT JOIN tags t ON pt.tag_id = t.id
WHERE p.id = $1
GROUP BY p.id;
