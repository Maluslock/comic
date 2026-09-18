CREATE TABLE IF NOT EXISTS admins (
  id            BIGSERIAL PRIMARY KEY,
  username      VARCHAR(64) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role          VARCHAR(20) NOT NULL DEFAULT 'admin',
  token         VARCHAR(255),
  token_expires_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- 种子管理员 admin / admin123（bcrypt cost 10）
INSERT INTO admins (username, password_hash) VALUES ('admin', '$2a$10$aVPo9Dq40lS35p3J8u/rd.cb4Uw5FPpgR7OHevi0SXaSkuoYK/cuC') ON CONFLICT (username) DO NOTHING;
