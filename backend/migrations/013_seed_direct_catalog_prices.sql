-- Normalize seeded direct-routing capabilities and model-independent retail prices.
UPDATE supplier_offerings
SET resolutions = CASE
  WHEN public_id IN ('off_h3_first_last_b99', 'off_h3_text_b99') THEN '["480P","768P"]'::jsonb
  ELSE resolutions
END,
    max_duration_sec = CASE
      WHEN public_id = 'off_h3_first_last_lightx' THEN 10
      WHEN public_id IN ('off_h3_multi_v2','off_h3_multi_lightx') THEN 10
      WHEN public_id IN ('off_h3_multi_zm_u24','off_h3_multi_zm_u08','off_h3_multi_v2_15s','off_h3_multi_lightx_15s','off_h3_multi_audio') THEN 15
      ELSE max_duration_sec
    END,
    workflow_code = COALESCE(NULLIF(workflow_code,''), upstream_workflow_id)
WHERE public_id LIKE 'off_h3_%';

INSERT INTO supplier_cost_rules (public_id, offering_id, execution_id, resolution, generate_audio, price_per_sec_peak, price_per_sec_offpeak, version, status)
SELECT 'cost_direct_' || offering.public_id || '_' || resolution || '_' || audio::text,
       offering.id, execution.id, resolution, audio, CASE WHEN resolution='1080P' THEN 0.05 ELSE 0.02 END,
       CASE WHEN resolution='480P' THEN 0.01 ELSE 0.02 END, 'v1', 'active'
FROM supplier_offerings offering
JOIN supplier_executions execution ON execution.legacy_offering_id=offering.id
CROSS JOIN LATERAL jsonb_array_elements_text(offering.resolutions) AS r(resolution)
CROSS JOIN (VALUES (FALSE), (TRUE)) AS a(audio)
WHERE offering.public_id LIKE 'off_h3_%'
  AND (audio = FALSE OR offering.supports_generated_audio)
ON CONFLICT (offering_id, resolution, generate_audio) DO NOTHING;

UPDATE retail_price_rules
SET price_per_sec = CASE resolution WHEN '480P' THEN 0.02 WHEN '768P' THEN 0.03 WHEN '1080P' THEN 0.09 ELSE price_per_sec END,
    name = '平台统一售价 / ' || resolution
WHERE model_id IS NULL;

INSERT INTO schema_migrations (version) VALUES ('013_seed_direct_catalog_prices')
ON CONFLICT (version) DO NOTHING;
