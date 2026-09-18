CREATE TABLE IF NOT EXISTS photographer_cert_applications (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  photographer_id BIGINT NOT NULL REFERENCES photographers(id),
  evidence_images TEXT[] NOT NULL,
  evidence_desc TEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  review_reason TEXT,
  admin_id BIGINT REFERENCES admins(id),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cert_app_user ON photographer_cert_applications(user_id);
CREATE INDEX IF NOT EXISTS idx_cert_app_status ON photographer_cert_applications(status, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cert_app_photographer_active ON photographer_cert_applications(photographer_id) WHERE status <> 'rejected';
