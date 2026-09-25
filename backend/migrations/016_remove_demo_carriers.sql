-- Remove the two historical demo carrier records from UAT and prevent them from
-- reappearing through the runtime catalog seed. Only catalog records with no
-- execution history are removed, so existing production routes/tasks remain intact.
DELETE FROM models
WHERE carrier_id IN (
  SELECT id FROM carriers
  WHERE public_id IN ('car_volcengine', 'car_3140e6d17852')
)
AND NOT EXISTS (
  SELECT 1 FROM supplier_offerings offering WHERE offering.model_id = models.id
)
AND NOT EXISTS (
  SELECT 1 FROM supplier_model_bindings binding WHERE binding.model_id = models.id
)
AND NOT EXISTS (
  SELECT 1 FROM mode_model_policies policy WHERE policy.model_id = models.id
)
AND NOT EXISTS (
  SELECT 1 FROM generation_tasks task WHERE task.model_public_id = models.public_id
);

DELETE FROM carriers
WHERE public_id IN ('car_volcengine', 'car_3140e6d17852')
AND NOT EXISTS (SELECT 1 FROM models model WHERE model.carrier_id = carriers.id)
AND NOT EXISTS (
  SELECT 1 FROM generation_tasks task
  WHERE task.carrier_code = carriers.code
);

INSERT INTO schema_migrations (version) VALUES ('016_remove_demo_carriers')
ON CONFLICT (version) DO NOTHING;
