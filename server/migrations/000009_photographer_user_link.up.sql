-- Link photographers to users so chat sessions have a real peer identity.
-- Each photographer becomes a user account (role context demo: see AGENTS.md).
ALTER TABLE photographers ADD COLUMN IF NOT EXISTS user_id BIGINT;
CREATE INDEX IF NOT EXISTS idx_photographers_user ON photographers(user_id);
