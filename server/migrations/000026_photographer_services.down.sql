DELETE FROM services WHERE photographer_id IS NOT NULL;
DROP INDEX IF EXISTS idx_services_photographer;
ALTER TABLE services DROP COLUMN IF EXISTS sort_order;
ALTER TABLE services DROP COLUMN IF EXISTS is_active;
ALTER TABLE services DROP COLUMN IF EXISTS photographer_id;
