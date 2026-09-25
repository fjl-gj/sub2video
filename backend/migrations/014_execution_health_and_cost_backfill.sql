ALTER TABLE supplier_executions
  ADD COLUMN IF NOT EXISTS failure_count INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_failure_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMPTZ NULL;

UPDATE supplier_cost_rules cost
SET execution_id = execution.id
FROM supplier_executions execution
WHERE cost.execution_id IS NULL AND execution.legacy_offering_id = cost.offering_id;

UPDATE supplier_executions execution
SET resolutions=offering.resolutions,
    aspect_ratios=offering.aspect_ratios,
    min_duration_sec=offering.min_duration_sec,
    max_duration_sec=offering.max_duration_sec,
    min_images=offering.min_images,
    max_images=offering.max_images,
    supports_generated_audio=offering.supports_generated_audio,
    updated_at=NOW()
FROM supplier_offerings offering
WHERE execution.legacy_offering_id=offering.id;

INSERT INTO schema_migrations (version) VALUES ('014_execution_health_and_cost_backfill')
ON CONFLICT (version) DO NOTHING;
