-- The direct-routing contract prices audio separately. Existing UAT data had
-- costs and executions for multi-image audio, but no matching platform retail
-- rules, causing estimate and submission to fail before routing.
INSERT INTO retail_price_rules (public_id, mode_id, model_id, resolution, generate_audio, name, price_per_sec, currency, version, status)
SELECT value.public_id, mode.id, NULL, value.resolution, TRUE, value.name, value.price_per_sec, 'CNY', 'v1', 'active'
FROM (
  VALUES
    ('ret_mode_multi_480_audio', '480P', '平台统一售价 / 480P / 自动音频', 0.02::numeric),
    ('ret_mode_multi_768_audio', '768P', '平台统一售价 / 768P / 自动音频', 0.03::numeric),
    ('ret_mode_multi_1080_audio', '1080P', '平台统一售价 / 1080P / 自动音频', 0.09::numeric)
) AS value(public_id, resolution, name, price_per_sec)
JOIN creation_modes mode ON mode.code='multi_image_to_video'
WHERE NOT EXISTS (
  SELECT 1 FROM retail_price_rules existing
  WHERE existing.mode_id=mode.id
    AND existing.model_id IS NULL
    AND existing.resolution=value.resolution
    AND existing.generate_audio=TRUE
    AND existing.version='v1'
);

INSERT INTO schema_migrations (version) VALUES ('017_seed_multi_image_audio_retail_prices')
ON CONFLICT (version) DO NOTHING;
