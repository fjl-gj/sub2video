-- AutoDL workflow 8 only accepts image, resolution and optional reference
-- audio fields. Its routing capability remains audio-enabled, but generic
-- prompt/duration fields must not be passed to that workflow.
UPDATE supplier_offerings
SET parameter_mapping='{"omit":["prompt","duration"],"audioDurationField":"audio_duration"}'::jsonb,
    updated_at=NOW()
WHERE workflow_code='minimax_h3_image_audio_to_video';

UPDATE supplier_executions
SET parameter_mapping='{"omit":["prompt","duration"],"audioDurationField":"audio_duration"}'::jsonb,
    updated_at=NOW()
WHERE workflow_code='minimax_h3_image_audio_to_video';

INSERT INTO schema_migrations (version) VALUES ('018_autodl_reference_audio_mapping')
ON CONFLICT (version) DO NOTHING;
