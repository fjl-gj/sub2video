-- Provider execution leases, retry scheduling, deadlines, and durable events.
ALTER TABLE generation_tasks
  ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS claim_expires_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS claim_token VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS submit_started BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS submit_started_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS submit_uncertain BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS execution_deadline TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS poll_attempts INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS next_poll_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS provider_request_id VARCHAR(256) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_generation_tasks_provider_due
  ON generation_tasks (status, next_poll_at, created_at)
  WHERE execution_mode = 'provider';

CREATE TABLE IF NOT EXISTS provider_task_events (
  id                  BIGSERIAL PRIMARY KEY,
  task_id             BIGINT NOT NULL REFERENCES generation_tasks(id) ON DELETE CASCADE,
  event_type          VARCHAR(32) NOT NULL,
  execution_type      VARCHAR(16) NOT NULL DEFAULT '',
  provider_request_id VARCHAR(256) NOT NULL DEFAULT '',
  upstream_status     VARCHAR(64) NOT NULL DEFAULT '',
  detail              JSONB NOT NULL DEFAULT '{}'::jsonb,
  elapsed_ms          BIGINT NOT NULL DEFAULT 0,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_task_events_task_time
  ON provider_task_events (task_id, created_at DESC, id DESC);

INSERT INTO schema_migrations (version) VALUES ('021_provider_execution_reliability')
ON CONFLICT (version) DO NOTHING;
