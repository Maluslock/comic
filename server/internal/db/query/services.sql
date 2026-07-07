-- name: GetServices :many
SELECT id, name, price, description, duration
FROM services
ORDER BY price ASC;
