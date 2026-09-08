-- Retire the pre-catalog contract without deleting historic route or task data.
-- A model with client_workflow_id was a client workflow masquerading as a
-- model. New supply uses creation_modes, mode_model_policies, and
-- supplier_offerings instead.
UPDATE supplier_routes r
SET enabled = FALSE, updated_at = NOW()
FROM models m
WHERE r.model_id = m.id
  AND COALESCE(m.client_workflow_id, '') <> ''
  AND r.enabled;

UPDATE models
SET enabled = FALSE, updated_at = NOW()
WHERE COALESCE(client_workflow_id, '') <> ''
  AND enabled;

UPDATE carriers c
SET enabled = FALSE, updated_at = NOW()
WHERE c.enabled
  AND NOT EXISTS (
    SELECT 1
    FROM models m
    WHERE m.carrier_id = c.id
      AND COALESCE(m.client_workflow_id, '') = ''
  );

INSERT INTO schema_migrations (version) VALUES ('011_retire_legacy_workflow_contract')
ON CONFLICT (version) DO NOTHING;
