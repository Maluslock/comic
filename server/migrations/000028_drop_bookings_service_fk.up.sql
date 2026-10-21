-- Drop the bookings.service_id foreign key so a photographer can delete a package
-- that has booking history. Order history must survive package deletion (spec §6.1/§9):
-- bookings.service_name (migration 000027) already snapshots the display name for
-- historical orders, and the app enforces that a booked package belongs to the
-- photographer. service_id is therefore kept as a soft reference; joined reads
-- COALESCE the (now possibly missing) services.name to ''.
-- Idempotent: safe to replay.
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_service_id_fkey;
