-- 000041 down：回滚套餐按张计价
-- 依赖逆序：先删 CHECK 约束，再删列。

ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_photo_count_check;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_price_unit_check;
ALTER TABLE services DROP CONSTRAINT IF EXISTS services_per_photo_price_check;
ALTER TABLE services DROP CONSTRAINT IF EXISTS services_photo_count_check;
ALTER TABLE services DROP CONSTRAINT IF EXISTS services_price_unit_check;

ALTER TABLE bookings DROP COLUMN IF EXISTS price_unit;
ALTER TABLE bookings DROP COLUMN IF EXISTS photo_count;

ALTER TABLE services DROP COLUMN IF EXISTS photo_count;
ALTER TABLE services DROP COLUMN IF EXISTS price_unit;
