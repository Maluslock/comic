DELETE FROM notifications WHERE user_id IS NULL;
ALTER TABLE notifications ALTER COLUMN user_id SET NOT NULL;
