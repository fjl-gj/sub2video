ALTER TABLE generation_tasks
  ADD COLUMN IF NOT EXISTS source_meta JSONB NOT NULL DEFAULT '{}'::jsonb;

INSERT INTO schema_migrations (version) VALUES ('008_task_source_meta')
ON CONFLICT (version) DO NOTHING;
