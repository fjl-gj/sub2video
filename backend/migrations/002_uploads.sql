-- uploads + task source refs
CREATE TABLE IF NOT EXISTS uploads (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  user_id         BIGINT NOT NULL REFERENCES users(id),
  file_name       VARCHAR(256) NOT NULL,
  mime_type       VARCHAR(128) NOT NULL,
  size_bytes      BIGINT NOT NULL DEFAULT 0,
  storage_path    TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uploads_user ON uploads (user_id, created_at DESC);

ALTER TABLE generation_tasks
  ADD COLUMN IF NOT EXISTS source_type VARCHAR(16) NOT NULL DEFAULT 'text',
  ADD COLUMN IF NOT EXISTS source_ref VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS prompt TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS aspect_ratio VARCHAR(8) NOT NULL DEFAULT '16:9';

INSERT INTO schema_migrations (version) VALUES ('002_uploads')
ON CONFLICT (version) DO NOTHING;
