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

INSERT INTO schema_migrations (version) VALUES ('015_sync_execution_capabilities')
ON CONFLICT (version) DO NOTHING;
