-- Supplier procurement costs belong to an executable API/workflow
-- configuration. Legacy offering_id remains a nullable compatibility pointer.
ALTER TABLE supplier_cost_rules
  ALTER COLUMN offering_id DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_supplier_cost_execution_match
  ON supplier_cost_rules (execution_id, resolution, generate_audio)
  WHERE execution_id IS NOT NULL;

INSERT INTO schema_migrations (version) VALUES ('020_execution_cost_rules')
ON CONFLICT (version) DO NOTHING;
