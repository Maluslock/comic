-- Re-add the bookings.service_id foreign key dropped in the up migration.
-- Guarded so it is re-runnable.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'bookings_service_id_fkey'
  ) THEN
    ALTER TABLE bookings
      ADD CONSTRAINT bookings_service_id_fkey
      FOREIGN KEY (service_id) REFERENCES services(id);
  END IF;
END
$$;
