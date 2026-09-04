-- Fixed client creation modes, model policies, hidden supplier offerings, and quote snapshots.
CREATE TABLE IF NOT EXISTS creation_modes (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(64) NOT NULL UNIQUE,
  code            VARCHAR(64) NOT NULL UNIQUE,
  name            VARCHAR(128) NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  status          VARCHAR(20) NOT NULL DEFAULT 'published',
  sort_order      INT NOT NULL DEFAULT 0,
  input_contract  JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mode_model_policies (
  id                 BIGSERIAL PRIMARY KEY,
  mode_id            BIGINT NOT NULL REFERENCES creation_modes(id),
  model_id           BIGINT NOT NULL REFERENCES models(id),
  enabled            BOOLEAN NOT NULL DEFAULT TRUE,
  recommendation_rank INT NOT NULL DEFAULT 100,
  display_badge      VARCHAR(64) NOT NULL DEFAULT '',
  default_duration   INT NOT NULL DEFAULT 5,
  default_resolution VARCHAR(16) NOT NULL DEFAULT '768P',
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (mode_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_mode_model_policies_lookup
  ON mode_model_policies (mode_id, enabled, recommendation_rank);

CREATE TABLE IF NOT EXISTS supplier_offerings (
  id                       BIGSERIAL PRIMARY KEY,
  public_id                VARCHAR(64) NOT NULL UNIQUE,
  supplier_id              BIGINT NOT NULL REFERENCES suppliers(id),
  model_id                 BIGINT NOT NULL REFERENCES models(id),
  mode_id                  BIGINT NOT NULL REFERENCES creation_modes(id),
  name                     VARCHAR(128) NOT NULL DEFAULT '',
  upstream_workflow_id     VARCHAR(128) NOT NULL,
  adapter_code             VARCHAR(64) NOT NULL DEFAULT 'autodl_h3_v1',
  enabled                  BOOLEAN NOT NULL DEFAULT TRUE,
  health_status            VARCHAR(20) NOT NULL DEFAULT 'unknown',
  priority                 INT NOT NULL DEFAULT 100,
  min_duration_sec         INT NOT NULL DEFAULT 1,
  max_duration_sec         INT NOT NULL DEFAULT 15,
  resolutions              JSONB NOT NULL DEFAULT '[]'::jsonb,
  aspect_ratios            JSONB NOT NULL DEFAULT '[]'::jsonb,
  min_images               INT NOT NULL DEFAULT 0,
  max_images               INT NOT NULL DEFAULT 0,
  supports_generated_audio BOOLEAN NOT NULL DEFAULT FALSE,
  parameter_mapping        JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (supplier_id, model_id, mode_id, upstream_workflow_id)
);

CREATE INDEX IF NOT EXISTS idx_supplier_offerings_match
  ON supplier_offerings (mode_id, model_id, enabled, priority);

CREATE TABLE IF NOT EXISTS supplier_cost_rules (
  id                 BIGSERIAL PRIMARY KEY,
  public_id          VARCHAR(64) NOT NULL UNIQUE,
  offering_id        BIGINT NOT NULL REFERENCES supplier_offerings(id),
  resolution         VARCHAR(16) NOT NULL,
  generate_audio     BOOLEAN NOT NULL DEFAULT FALSE,
  price_per_sec_peak NUMERIC(12,4) NOT NULL,
  price_per_sec_offpeak NUMERIC(12,4) NOT NULL,
  currency           VARCHAR(8) NOT NULL DEFAULT 'CNY',
  version            VARCHAR(32) NOT NULL DEFAULT 'v1',
  status             VARCHAR(20) NOT NULL DEFAULT 'active',
  effective_from     TIMESTAMPTZ NULL,
  effective_to       TIMESTAMPTZ NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (offering_id, resolution, generate_audio)
);

CREATE INDEX IF NOT EXISTS idx_supplier_cost_rules_match
  ON supplier_cost_rules (offering_id, resolution, generate_audio, status);

ALTER TABLE supplier_cost_rules
  ADD COLUMN IF NOT EXISTS version VARCHAR(32) NOT NULL DEFAULT 'v1';

CREATE TABLE IF NOT EXISTS retail_price_rules (
  id                 BIGSERIAL PRIMARY KEY,
  public_id          VARCHAR(64) NOT NULL UNIQUE,
  mode_id            BIGINT NOT NULL REFERENCES creation_modes(id),
  model_id           BIGINT NOT NULL REFERENCES models(id),
  resolution         VARCHAR(16) NOT NULL,
  generate_audio     BOOLEAN NOT NULL DEFAULT FALSE,
  name               VARCHAR(128) NOT NULL DEFAULT '',
  price_per_sec      NUMERIC(12,4) NOT NULL,
  currency           VARCHAR(8) NOT NULL DEFAULT 'CNY',
  version            VARCHAR(32) NOT NULL DEFAULT 'v1',
  status             VARCHAR(20) NOT NULL DEFAULT 'active',
  effective_from     TIMESTAMPTZ NULL,
  effective_to       TIMESTAMPTZ NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (mode_id, model_id, resolution, generate_audio, version)
);

CREATE INDEX IF NOT EXISTS idx_retail_price_rules_match
  ON retail_price_rules (mode_id, model_id, resolution, generate_audio, status);

CREATE TABLE IF NOT EXISTS task_quotes (
  id                BIGSERIAL PRIMARY KEY,
  public_id         VARCHAR(64) NOT NULL UNIQUE,
  user_id           BIGINT NOT NULL REFERENCES users(id),
  mode_id           BIGINT NOT NULL REFERENCES creation_modes(id),
  model_id          BIGINT NOT NULL REFERENCES models(id),
  offering_id       BIGINT NOT NULL REFERENCES supplier_offerings(id),
  retail_rule_id    BIGINT NOT NULL REFERENCES retail_price_rules(id),
  cost_rule_id      BIGINT NOT NULL REFERENCES supplier_cost_rules(id),
  request_snapshot  JSONB NOT NULL DEFAULT '{}'::jsonb,
  retail_snapshot   JSONB NOT NULL DEFAULT '{}'::jsonb,
  cost_snapshot     JSONB NOT NULL DEFAULT '{}'::jsonb,
  offering_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  expires_at        TIMESTAMPTZ NOT NULL,
  consumed_at       TIMESTAMPTZ NULL,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_quotes_consume
  ON task_quotes (public_id, user_id, expires_at, consumed_at);

ALTER TABLE generation_tasks
  ADD COLUMN IF NOT EXISTS mode_public_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS model_public_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS generate_audio BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS quote_public_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS retail_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS cost_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS offering_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS input_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

INSERT INTO schema_migrations (version) VALUES ('009_creation_catalog')
ON CONFLICT (version) DO NOTHING;
