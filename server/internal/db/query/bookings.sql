-- sqlc 源文件。仓库里实际使用的是 internal/repository/bookings.sql.go（手工维护，
-- 因为构建环境无法联网跑 sqlc）。两边列清单必须逐字一致：bookings 的 SELECT 列表是
-- 27 列（含 photo_count、price_unit），少一列就会让 scanBooking 的目标数与返回值错位。

-- name: CreateBooking :one
INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price, remarks, price_mode, price_status, service_name, service_duration, photo_count, price_unit)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, photographer_id, coser_id, service_id, date, time, status, sub_status, total_price, remarks, created_at, updated_at, price_mode, quote_price, price_status, service_name, service_duration, checkin_code, checkin_at, delivered_at, accept_deadline, delivery_evidence, cancel_policy, cancelled_by, noshow_party, photo_count, price_unit;

-- name: GetBookingsByUser :many
SELECT id, photographer_id, coser_id, service_id, date, time, status, sub_status, total_price, remarks, created_at, updated_at, price_mode, quote_price, price_status, service_name, service_duration, checkin_code, checkin_at, delivered_at, accept_deadline, delivery_evidence, cancel_policy, cancelled_by, noshow_party, photo_count, price_unit
FROM bookings
WHERE coser_id = $1
ORDER BY created_at DESC;

-- name: GetBookingByID :one
SELECT id, photographer_id, coser_id, service_id, date, time, status, sub_status, total_price, remarks, created_at, updated_at, price_mode, quote_price, price_status, service_name, service_duration, checkin_code, checkin_at, delivered_at, accept_deadline, delivery_evidence, cancel_policy, cancelled_by, noshow_party, photo_count, price_unit
FROM bookings
WHERE id = $1;

-- name: UpdateBookingStatus :one
UPDATE bookings
SET status = $2, updated_at = NOW()
WHERE id = $1 AND status = $3
RETURNING id, photographer_id, coser_id, service_id, date, time, status, sub_status, total_price, remarks, created_at, updated_at, price_mode, quote_price, price_status, service_name, service_duration, checkin_code, checkin_at, delivered_at, accept_deadline, delivery_evidence, cancel_policy, cancelled_by, noshow_party, photo_count, price_unit;

-- name: GetBookingsByUserWithDetails :many
SELECT b.id, b.photographer_id, b.coser_id, b.service_id, b.date, b.time, b.status, b.sub_status,
       b.total_price, b.remarks, b.created_at, b.updated_at, b.price_mode, b.quote_price,
       b.price_status, b.service_name, b.service_duration, b.checkin_code, b.checkin_at,
       b.delivered_at, b.accept_deadline, b.delivery_evidence, b.cancel_policy,
       b.cancelled_by, b.noshow_party, b.photo_count, b.price_unit,
       p.name AS photographer_name, COALESCE(p.avatar, '') AS photographer_avatar, p.user_id AS photographer_user_id,
       COALESCE(s.name, '') AS service_name_join, COALESCE(s.price, 0) AS service_price
FROM bookings b
LEFT JOIN photographers p ON p.id = b.photographer_id
LEFT JOIN services s ON s.id = b.service_id
WHERE b.coser_id = $1
ORDER BY b.created_at DESC;

-- name: GetBookingsByPhotographer :many
SELECT b.id, b.photographer_id, b.coser_id, b.service_id, b.date, b.time, b.status, b.sub_status,
       b.total_price, b.remarks, b.created_at, b.updated_at, b.price_mode, b.quote_price,
       b.price_status, b.service_name, b.service_duration, b.checkin_code, b.checkin_at,
       b.delivered_at, b.accept_deadline, b.delivery_evidence, b.cancel_policy,
       b.cancelled_by, b.noshow_party, b.photo_count, b.price_unit,
       COALESCE(u.name, '') AS coser_name, COALESCE(u.avatar, '') AS coser_avatar,
       CASE WHEN b.status IN ('confirmed', 'completed') THEN COALESCE(u.phone, '') ELSE '' END AS coser_phone
FROM bookings b
LEFT JOIN users u ON u.id = b.coser_id
WHERE b.photographer_id = $1
ORDER BY b.created_at DESC;
