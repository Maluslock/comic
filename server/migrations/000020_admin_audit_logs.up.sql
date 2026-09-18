CREATE TABLE IF NOT EXISTS admin_audit_logs (
  id         BIGSERIAL PRIMARY KEY,
  admin_id   BIGINT NOT NULL,
  method     VARCHAR(10) NOT NULL,
  path       TEXT NOT NULL,
  status     INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_created ON admin_audit_logs (created_at DESC);
