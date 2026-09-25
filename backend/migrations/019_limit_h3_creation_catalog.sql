-- Initial creator acceptance is intentionally restricted to four verified H3
-- workflow codes. Purge all other supply records. Generation tasks, orders and
-- wallet ledgers retain immutable JSON snapshots, while obsolete task quotes
-- are deleted first to satisfy their supply/pricing foreign keys.

DELETE FROM task_quotes AS q
USING supplier_offerings AS offering, supplier_cost_rules AS cost,
      retail_price_rules AS retail, creation_modes AS mode
WHERE q.offering_id=offering.id
  AND q.cost_rule_id=cost.id
  AND q.retail_rule_id=retail.id
  AND retail.mode_id=mode.id
  AND (
    NOT (
      offering.execution_type='workflow'
      AND offering.workflow_code IN (
        'minimax_h3_zm_u24',
        'minimax_h3_zm_u08',
        'minimax_h3_lightx2v',
        'minimax_h3_lightx2v_no_pic'
      )
    )
    OR cost.execution_id IN (
      SELECT execution.id
      FROM supplier_executions execution
      WHERE NOT (
        execution.execution_type='workflow'
        AND execution.workflow_code IN (
          'minimax_h3_zm_u24',
          'minimax_h3_zm_u08',
          'minimax_h3_lightx2v',
          'minimax_h3_lightx2v_no_pic'
        )
      )
    )
    OR cost.resolution NOT IN ('480P','768P')
    OR (
      cost.generate_audio
      AND offering.workflow_code NOT IN ('minimax_h3_zm_u24','minimax_h3_zm_u08')
    )
    OR retail.model_id IS NOT NULL
    OR NOT (
      (mode.code IN ('text_to_video','first_last_frame_to_video')
        AND retail.resolution IN ('480P','768P') AND retail.generate_audio=FALSE)
      OR
      (mode.code='multi_image_to_video' AND retail.resolution IN ('480P','768P'))
    )
  );

DELETE FROM supplier_cost_rules AS cost
WHERE cost.execution_id IN (
  SELECT execution.id
  FROM supplier_executions execution
  WHERE NOT (
    execution.execution_type='workflow'
    AND execution.workflow_code IN (
      'minimax_h3_zm_u24',
      'minimax_h3_zm_u08',
      'minimax_h3_lightx2v',
      'minimax_h3_lightx2v_no_pic'
    )
  )
);

DELETE FROM supplier_cost_rules AS cost
USING supplier_offerings AS offering
WHERE offering.id=cost.offering_id
  AND (
    NOT (
      offering.execution_type='workflow'
      AND offering.workflow_code IN (
        'minimax_h3_zm_u24',
        'minimax_h3_zm_u08',
        'minimax_h3_lightx2v',
        'minimax_h3_lightx2v_no_pic'
      )
    )
    OR cost.resolution NOT IN ('480P','768P')
    OR (
      cost.generate_audio
      AND offering.workflow_code NOT IN ('minimax_h3_zm_u24','minimax_h3_zm_u08')
    )
  );

DELETE FROM supplier_executions AS execution
WHERE NOT (
  execution.execution_type='workflow'
  AND execution.workflow_code IN (
    'minimax_h3_zm_u24',
    'minimax_h3_zm_u08',
    'minimax_h3_lightx2v',
    'minimax_h3_lightx2v_no_pic'
  )
);

DELETE FROM supplier_offerings AS offering
WHERE NOT (
  offering.execution_type='workflow'
  AND offering.workflow_code IN (
    'minimax_h3_zm_u24',
    'minimax_h3_zm_u08',
    'minimax_h3_lightx2v',
    'minimax_h3_lightx2v_no_pic'
  )
);

DELETE FROM supplier_routes AS route_row
WHERE route_row.workflow_id LIKE 'minimax_h3%'
  AND route_row.workflow_id NOT IN (
    'minimax_h3_zm_u24',
    'minimax_h3_zm_u08',
    'minimax_h3_lightx2v',
    'minimax_h3_lightx2v_no_pic'
  );

-- The four retained workflows all accept 480P/768P. The two ZM workflows
-- retain their configured audio capability; no 1080P execution is advertised.
UPDATE supplier_offerings
SET execution_type='workflow',
    api_model_code='',
    api_version='',
    workflow_code=upstream_workflow_id,
    resolutions='["480P","768P"]'::jsonb,
    aspect_ratios='["16:9","9:16","1:1"]'::jsonb,
    min_duration_sec=1,
    max_duration_sec=CASE WHEN workflow_code='minimax_h3_lightx2v' THEN 10 ELSE 15 END,
    min_images=CASE WHEN workflow_code IN ('minimax_h3_zm_u24','minimax_h3_zm_u08') THEN 1 ELSE 0 END,
    max_images=CASE WHEN workflow_code IN ('minimax_h3_zm_u24','minimax_h3_zm_u08') THEN 9 ELSE 0 END,
    supports_generated_audio=(workflow_code IN ('minimax_h3_zm_u24','minimax_h3_zm_u08')),
    updated_at=NOW()
WHERE workflow_code IN (
  'minimax_h3_zm_u24',
  'minimax_h3_zm_u08',
  'minimax_h3_lightx2v',
  'minimax_h3_lightx2v_no_pic'
);

UPDATE supplier_executions execution
SET execution_type=offering.execution_type,
    api_model_code=offering.api_model_code,
    api_version=offering.api_version,
    workflow_code=offering.workflow_code,
    resolutions=offering.resolutions,
    aspect_ratios=offering.aspect_ratios,
    min_duration_sec=offering.min_duration_sec,
    max_duration_sec=offering.max_duration_sec,
    min_images=offering.min_images,
    max_images=offering.max_images,
    supports_generated_audio=offering.supports_generated_audio,
    updated_at=NOW()
FROM supplier_offerings offering
WHERE execution.legacy_offering_id=offering.id;

UPDATE supplier_cost_rules cost
SET price_per_sec_peak=CASE cost.resolution WHEN '480P' THEN 0.02 WHEN '768P' THEN 0.03 ELSE cost.price_per_sec_peak END,
    price_per_sec_offpeak=CASE cost.resolution WHEN '480P' THEN 0.01 WHEN '768P' THEN 0.02 ELSE cost.price_per_sec_offpeak END,
    status='active',
    updated_at=NOW()
FROM supplier_offerings offering
WHERE offering.id=cost.offering_id
  AND offering.workflow_code IN (
    'minimax_h3_zm_u24',
    'minimax_h3_zm_u08',
    'minimax_h3_lightx2v',
    'minimax_h3_lightx2v_no_pic'
  )
  AND cost.resolution IN ('480P','768P')
  AND (NOT cost.generate_audio OR offering.supports_generated_audio);

DELETE FROM retail_price_rules AS retail
USING creation_modes AS mode
WHERE retail.mode_id=mode.id
  AND retail.model_id IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM task_quotes AS q WHERE q.retail_rule_id=retail.id
  )
  AND NOT (
      (mode.code IN ('text_to_video','first_last_frame_to_video')
        AND retail.resolution IN ('480P','768P') AND retail.generate_audio=FALSE)
      OR
      (mode.code='multi_image_to_video'
        AND retail.resolution IN ('480P','768P'))
  );

DELETE FROM retail_price_rules AS retail
WHERE retail.model_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM task_quotes AS q WHERE q.retail_rule_id=retail.id
  );

INSERT INTO schema_migrations (version) VALUES ('019_limit_h3_creation_catalog')
ON CONFLICT (version) DO NOTHING;
