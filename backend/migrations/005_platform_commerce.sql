-- Platform workflows + sell-side pricing (cost stays on supplier_routes)
CREATE TABLE IF NOT EXISTS platform_workflows (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(64) NOT NULL UNIQUE,
  code            VARCHAR(64) NOT NULL UNIQUE,
  name            VARCHAR(128) NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  model_id        BIGINT NULL REFERENCES models(id),
  status          VARCHAR(20) NOT NULL DEFAULT 'draft', -- published | draft | disabled
  badge           VARCHAR(64) NOT NULL DEFAULT '',
  sort_order      INT NOT NULL DEFAULT 0,
  params          JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_workflows_status ON platform_workflows (status, sort_order);

CREATE TABLE IF NOT EXISTS pricing_rules (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(64) NOT NULL UNIQUE,
  workflow_id     BIGINT NOT NULL REFERENCES platform_workflows(id),
  name            VARCHAR(128) NOT NULL,
  resolution      VARCHAR(16) NOT NULL,
  price_per_sec   NUMERIC(12,4) NOT NULL,
  -- display hint only; real procurement cost is on supplier_routes
  cost_hint_per_sec NUMERIC(12,4) NOT NULL DEFAULT 0,
  version         VARCHAR(32) NOT NULL DEFAULT 'v1',
  status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active | draft
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workflow_id, resolution, version)
);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_wf ON pricing_rules (workflow_id, status);

ALTER TABLE generation_tasks
  ALTER COLUMN public_id TYPE VARCHAR(64),
  ADD COLUMN IF NOT EXISTS workflow_public_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS price_rule_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS price_version VARCHAR(32) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS price_per_sec NUMERIC(12,4) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS cost_per_sec NUMERIC(12,4) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS price_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS ledger_public_id VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE wallet_ledger
  ALTER COLUMN public_id TYPE VARCHAR(64);

INSERT INTO schema_migrations (version) VALUES ('005_platform_commerce')
ON CONFLICT (version) DO NOTHING;
