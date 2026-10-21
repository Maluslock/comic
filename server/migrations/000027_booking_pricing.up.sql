ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price_mode VARCHAR(10) NOT NULL DEFAULT 'fixed';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS quote_price INTEGER;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price_status VARCHAR(20) NOT NULL DEFAULT 'agreed';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS service_name VARCHAR(100);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS service_duration INTEGER;

UPDATE bookings b
SET service_name = COALESCE(b.service_name, s.name),
    service_duration = COALESCE(b.service_duration, s.duration)
FROM services s
WHERE s.id = b.service_id AND (b.service_name IS NULL OR b.service_duration IS NULL);
