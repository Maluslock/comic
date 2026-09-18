DELETE FROM services WHERE photographer_id IS NOT NULL;
UPDATE services SET price = COALESCE(price, 0) WHERE price IS NULL;
DROP INDEX IF EXISTS idx_services_photographer;
ALTER TABLE services DROP COLUMN IF EXISTS sort_order;
ALTER TABLE services DROP COLUMN IF EXISTS is_active;
ALTER TABLE services DROP COLUMN IF EXISTS photographer_id;
ALTER TABLE services ALTER COLUMN price SET NOT NULL;
