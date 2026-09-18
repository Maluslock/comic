-- name: CreateBooking :one
INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price, remarks)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, photographer_id, coser_id, service_id, date, time, status, total_price, remarks, created_at, updated_at;

-- name: GetBookingsByUser :many
SELECT id, photographer_id, coser_id, service_id, date, time, status, total_price, remarks, created_at, updated_at
FROM bookings
WHERE coser_id = $1
ORDER BY created_at DESC;
