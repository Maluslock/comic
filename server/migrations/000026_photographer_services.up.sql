ALTER TABLE services ADD COLUMN IF NOT EXISTS photographer_id BIGINT;
ALTER TABLE services ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE services ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE services ALTER COLUMN price DROP NOT NULL;
CREATE INDEX IF NOT EXISTS idx_services_photographer ON services(photographer_id);

INSERT INTO services (name, price, description, duration, photographer_id, is_active, sort_order)
SELECT s.name, s.price, s.description, s.duration, p.id, true, s.id
FROM services s
CROSS JOIN photographers p
WHERE s.photographer_id IS NULL
  AND NOT EXISTS (SELECT 1 FROM services x WHERE x.photographer_id = p.id);
