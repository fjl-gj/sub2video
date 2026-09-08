-- Execution readiness, supplier health accounting, and administrator audit trail.
ALTER TABLE supplier_offerings
  ADD COLUMN IF NOT EXISTS failure_count INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_failure_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMPTZ NULL;

ALTER TABLE generation_tasks
  ADD COLUMN IF NOT EXISTS execution_mode VARCHAR(32) NOT NULL DEFAULT 'provider';

-- Historical tasks may keep the development simulator path. New quote-backed
-- tasks are always provider executions and are never advanced by the simulator.
UPDATE generation_tasks
SET execution_mode = 'simulated_legacy'
WHERE quote_public_id = '' AND execution_mode = 'provider';

CREATE TABLE IF NOT EXISTS admin_audit_logs (
  id            BIGSERIAL PRIMARY KEY,
  public_id     VARCHAR(64) NOT NULL UNIQUE,
  actor_user_id BIGINT NOT NULL REFERENCES users(id),
  action        VARCHAR(96) NOT NULL,
  target_type   VARCHAR(64) NOT NULL,
  target_id     VARCHAR(128) NOT NULL,
  details       JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_recent
  ON admin_audit_logs (created_at DESC, id DESC);

INSERT INTO schema_migrations (version) VALUES ('010_execution_audit')
ON CONFLICT (version) DO NOTHING;
