-- carriers / models / suppliers / routes + task provider fields
CREATE TABLE IF NOT EXISTS carriers (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  code            VARCHAR(64) NOT NULL UNIQUE,
  name            VARCHAR(128) NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order      INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS models (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  carrier_id      BIGINT NOT NULL REFERENCES carriers(id),
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  client_workflow_id VARCHAR(64) NOT NULL DEFAULT '',
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order      INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (carrier_id, code)
);

CREATE INDEX IF NOT EXISTS idx_models_carrier ON models (carrier_id, enabled);

CREATE TABLE IF NOT EXISTS suppliers (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  code            VARCHAR(64) NOT NULL UNIQUE,
  name            VARCHAR(128) NOT NULL,
  base_url        TEXT NOT NULL DEFAULT 'https://autodl.art',
  auth_header     VARCHAR(64) NOT NULL DEFAULT 'Authorization',
  token_encrypted TEXT NOT NULL DEFAULT '',
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  status          VARCHAR(32) NOT NULL DEFAULT 'unknown',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- supplier workflow bindings (routes)
CREATE TABLE IF NOT EXISTS supplier_routes (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  supplier_id     BIGINT NOT NULL REFERENCES suppliers(id),
  model_id        BIGINT NOT NULL REFERENCES models(id),
  workflow_id     VARCHAR(128) NOT NULL,
  name            VARCHAR(128) NOT NULL DEFAULT '',
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  -- cost per second: peak (08-24) / offpeak (00-08)
  cost_per_sec_peak    NUMERIC(12,4) NOT NULL DEFAULT 0.03,
  cost_per_sec_offpeak NUMERIC(12,4) NOT NULL DEFAULT 0.02,
  -- conditions JSON: {"max_duration":15,"resolutions":["480P","768P"],"require_source":["text"],"aspect_ratios":["9:16","16:9","1:1"]}
  conditions      JSONB NOT NULL DEFAULT '{}'::jsonb,
  priority        INT NOT NULL DEFAULT 100,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routes_model_enabled ON supplier_routes (model_id, enabled);
CREATE INDEX IF NOT EXISTS idx_routes_supplier ON supplier_routes (supplier_id);

ALTER TABLE generation_tasks
  ADD COLUMN IF NOT EXISTS carrier_code VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS model_code VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS supplier_code VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS supplier_route_id VARCHAR(32) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS upstream_workflow_id VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS upstream_task_id VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS upstream_status VARCHAR(32) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS result_url TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS result_oss_url TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS upstream_duration_sec INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS route_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS fail_reason TEXT NOT NULL DEFAULT '';

INSERT INTO schema_migrations (version) VALUES ('003_supply_chain')
ON CONFLICT (version) DO NOTHING;
