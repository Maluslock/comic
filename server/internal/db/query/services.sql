-- sqlc 源文件。仓库里实际使用的是 internal/repository/services.sql.go（手工维护，
-- 因为构建环境无法联网跑 sqlc）。两边的列清单必须逐字一致：任何一次只改一边，
-- 下一次有人跑 sqlc 就会把手写文件里的 price_unit / photo_count 覆盖掉，
-- 按张计价会在运行时静默退化成整套价（读到零值 → 兜底 fixed），金额随之改变。

-- name: GetServices :many
SELECT id, name, price, description, duration, price_unit, photo_count
FROM services
WHERE photographer_id IS NULL
ORDER BY sort_order ASC, id ASC;

-- name: GetServiceById :one
SELECT id, name, price, description, duration, price_unit, photo_count
FROM services
WHERE id = $1;

-- name: GetServicesByPhotographer :many
SELECT id, name, price, description, duration, price_unit, photo_count
FROM services
WHERE photographer_id = $1 AND is_active = true
ORDER BY sort_order ASC, id ASC;

-- name: GetAllServicesByPhotographer :many
SELECT id, name, price, description, duration, is_active, sort_order, price_unit, photo_count
FROM services
WHERE photographer_id = $1
ORDER BY sort_order ASC, id ASC;

-- name: GetServicePhotographerID :one
SELECT photographer_id FROM services WHERE id = $1;

-- name: GetBookableServicePhotographerID :one
SELECT photographer_id FROM services WHERE id = $1 AND is_active = true;

-- name: InsertService :one
INSERT INTO services (name, price, description, duration, photographer_id, is_active, sort_order, price_unit, photo_count)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: UpdateService :execrows
UPDATE services
SET name = $3, price = $4, description = $5, duration = $6,
    is_active = COALESCE($7, is_active),
    sort_order = COALESCE($8, sort_order),
    price_unit = COALESCE($9, price_unit),
    photo_count = COALESCE($10, photo_count)
WHERE id = $1 AND photographer_id = $2;

-- name: DeleteService :execrows
DELETE FROM services WHERE id = $1 AND photographer_id = $2;

-- name: GetServicePriceSummaries :many
-- 最低价按计价单位分组聚合：整套价与按张价的量纲互不可比，¥10/张 不能进「¥399 起」。
SELECT photographer_id,
       MIN(price) FILTER (WHERE price > 0 AND price_unit = 'fixed')     AS min_price,
       MIN(price) FILTER (WHERE price > 0 AND price_unit = 'per_photo') AS min_price_per_photo,
       COALESCE(bool_or(price = 0), false)     AS has_free,
       COALESCE(bool_or(price IS NULL), false) AS has_negotiable,
       count(*)                                AS active_count
FROM services
WHERE is_active AND photographer_id = ANY($1::bigint[])
GROUP BY photographer_id;
