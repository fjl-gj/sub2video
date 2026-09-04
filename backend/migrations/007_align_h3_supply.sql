-- Align supplier routes with platform workflow codes (MINMAX / H3)
ALTER TABLE supplier_routes
  ADD COLUMN IF NOT EXISTS client_workflow_id VARCHAR(64) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_routes_client_workflow
  ON supplier_routes (client_workflow_id, enabled);

INSERT INTO schema_migrations (version) VALUES ('007_align_h3_supply')
ON CONFLICT (version) DO NOTHING;
