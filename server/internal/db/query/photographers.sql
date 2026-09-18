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

-- name: SearchPhotographers :many
SELECT p.id, p.name, p.avatar, p.description, p.location, p.rating, p.review_count, p.order_count,
       array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL) AS tags
FROM photographers p
LEFT JOIN photographer_tags pt ON p.id = pt.photographer_id
LEFT JOIN tags t ON pt.tag_id = t.id
WHERE ($1::text IS NULL OR p.name ILIKE '%' || $1 || '%' OR p.description ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR p.location ILIKE '%' || $2 || '%')
  AND ($3::text IS NULL OR EXISTS (
    SELECT 1 FROM photographer_tags pt2
    JOIN tags t2 ON pt2.tag_id = t2.id
    WHERE pt2.photographer_id = p.id AND t2.name = $3
  ))
GROUP BY p.id
ORDER BY p.rating DESC, p.order_count DESC
LIMIT $4 OFFSET $5;
