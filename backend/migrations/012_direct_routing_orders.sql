-- Direct routing and submission-time billing. Historical quote/route data stays intact.
ALTER TABLE supplier_offerings
  ADD COLUMN IF NOT EXISTS execution_type VARCHAR(16) NOT NULL DEFAULT 'workflow',
  ADD COLUMN IF NOT EXISTS api_model_code VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS api_version VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS workflow_code VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS quota_status VARCHAR(20) NOT NULL DEFAULT 'unknown',
  ADD COLUMN IF NOT EXISTS result_mapping JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE supplier_offerings
SET workflow_code = upstream_workflow_id
WHERE execution_type = 'workflow' AND workflow_code = '' AND upstream_workflow_id <> '';

CREATE TABLE IF NOT EXISTS supplier_model_bindings (
  id          BIGSERIAL PRIMARY KEY,
  public_id   VARCHAR(64) NOT NULL UNIQUE,
  supplier_id BIGINT NOT NULL REFERENCES suppliers(id),
  model_id    BIGINT NOT NULL REFERENCES models(id),
  enabled     BOOLEAN NOT NULL DEFAULT TRUE,
  status      VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (supplier_id, model_id)
);

CREATE TABLE IF NOT EXISTS supplier_executions (
  id                       BIGSERIAL PRIMARY KEY,
  public_id                VARCHAR(64) NOT NULL UNIQUE,
  binding_id               BIGINT NOT NULL REFERENCES supplier_model_bindings(id),
  mode_id                  BIGINT NOT NULL REFERENCES creation_modes(id),
  execution_type           VARCHAR(16) NOT NULL CHECK (execution_type IN ('api', 'workflow')),
  api_model_code           VARCHAR(128) NOT NULL DEFAULT '',
  api_version              VARCHAR(64) NOT NULL DEFAULT '',
  workflow_code            VARCHAR(128) NOT NULL DEFAULT '',
  adapter_code             VARCHAR(64) NOT NULL DEFAULT '',
  enabled                  BOOLEAN NOT NULL DEFAULT TRUE,
  health_status            VARCHAR(20) NOT NULL DEFAULT 'unknown',
  quota_status             VARCHAR(20) NOT NULL DEFAULT 'unknown',
  priority                 INT NOT NULL DEFAULT 100,
  min_duration_sec         INT NOT NULL DEFAULT 1,
  max_duration_sec         INT NOT NULL DEFAULT 60,
  resolutions              JSONB NOT NULL DEFAULT '[]'::jsonb,
  aspect_ratios            JSONB NOT NULL DEFAULT '[]'::jsonb,
  min_images               INT NOT NULL DEFAULT 0,
  max_images               INT NOT NULL DEFAULT 0,
  supports_generated_audio BOOLEAN NOT NULL DEFAULT FALSE,
  parameter_mapping       JSONB NOT NULL DEFAULT '{}'::jsonb,
  result_mapping          JSONB NOT NULL DEFAULT '{}'::jsonb,
  legacy_offering_id      BIGINT NULL UNIQUE REFERENCES supplier_offerings(id),
  created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK ((execution_type = 'api' AND api_model_code <> '' AND workflow_code = '') OR
         (execution_type = 'workflow' AND workflow_code <> ''))
);

INSERT INTO supplier_model_bindings (public_id, supplier_id, model_id)
SELECT 'bind_' || md5(supplier.id::text || ':' || model.id::text), supplier.id, model.id
FROM supplier_offerings offering
JOIN suppliers supplier ON supplier.id=offering.supplier_id
JOIN models model ON model.id=offering.model_id
ON CONFLICT (supplier_id, model_id) DO NOTHING;

INSERT INTO supplier_executions (
  public_id, binding_id, mode_id, execution_type, api_model_code, api_version,
  workflow_code, adapter_code, enabled, health_status, quota_status, priority,
  min_duration_sec, max_duration_sec, resolutions, aspect_ratios, min_images,
  max_images, supports_generated_audio, result_mapping, legacy_offering_id
)
SELECT 'exec_' || offering.public_id, binding.id, offering.mode_id, offering.execution_type,
  offering.api_model_code, offering.api_version, offering.workflow_code, offering.adapter_code,
  offering.enabled, offering.health_status, offering.quota_status, offering.priority,
  offering.min_duration_sec, offering.max_duration_sec, offering.resolutions, offering.aspect_ratios,
  offering.min_images, offering.max_images, offering.supports_generated_audio, offering.result_mapping,
  offering.id
FROM supplier_offerings offering
JOIN supplier_model_bindings binding ON binding.supplier_id=offering.supplier_id AND binding.model_id=offering.model_id
WHERE offering.execution_type IN ('api', 'workflow')
ON CONFLICT (legacy_offering_id) DO NOTHING;

ALTER TABLE supplier_cost_rules
  ADD COLUMN IF NOT EXISTS execution_id BIGINT NULL REFERENCES supplier_executions(id);

UPDATE supplier_cost_rules cost
SET execution_id = execution.id
FROM supplier_executions execution
WHERE cost.execution_id IS NULL AND execution.legacy_offering_id = cost.offering_id;

ALTER TABLE retail_price_rules
  ALTER COLUMN model_id DROP NOT NULL;

-- Seed model-independent retail rows from the existing model-specific rules.
INSERT INTO retail_price_rules (public_id, mode_id, model_id, resolution, generate_audio, name, price_per_sec, currency, version, status)
SELECT 'ret_mode_' || md5(mode_id::text || ':' || resolution || ':' || generate_audio::text || ':' || version),
       mode_id, NULL, resolution, generate_audio, MIN(name), MIN(price_per_sec), MIN(currency), version, MIN(status)
FROM retail_price_rules
GROUP BY mode_id, resolution, generate_audio, version
ON CONFLICT DO NOTHING;

CREATE UNIQUE INDEX IF NOT EXISTS ux_retail_mode_resolution_audio_version
  ON retail_price_rules (mode_id, resolution, generate_audio, version)
  WHERE model_id IS NULL;

CREATE TABLE IF NOT EXISTS consumption_orders (
  id                    BIGSERIAL PRIMARY KEY,
  public_id             VARCHAR(64) NOT NULL UNIQUE,
  user_id               BIGINT NOT NULL REFERENCES users(id),
  task_id               BIGINT NULL REFERENCES generation_tasks(id),
  client_request_id     VARCHAR(128) NOT NULL,
  amount                NUMERIC(12,2) NOT NULL,
  currency              VARCHAR(8) NOT NULL DEFAULT 'CNY',
  status                VARCHAR(20) NOT NULL DEFAULT 'paid',
  retail_snapshot       JSONB NOT NULL DEFAULT '{}'::jsonb,
  cost_snapshot         JSONB NOT NULL DEFAULT '{}'::jsonb,
  execution_snapshot    JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  paid_at               TIMESTAMPTZ NULL,
  refunded_at           TIMESTAMPTZ NULL,
  UNIQUE (user_id, client_request_id)
);

CREATE INDEX IF NOT EXISTS idx_consumption_orders_user_time
  ON consumption_orders (user_id, created_at DESC);

ALTER TABLE generation_tasks
  ADD COLUMN IF NOT EXISTS client_request_id VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS consumption_order_public_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS execution_type VARCHAR(16) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS execution_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

INSERT INTO schema_migrations (version) VALUES ('012_direct_routing_orders')
ON CONFLICT (version) DO NOTHING;
