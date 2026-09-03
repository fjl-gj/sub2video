-- COS + commerce settings defaults
INSERT INTO settings (key, value) VALUES
  ('cos_secret_id', ''),
  ('cos_secret_key', ''),
  ('cos_bucket', ''),
  ('cos_region', ''),
  ('cos_use_intranet', 'false')
ON CONFLICT (key) DO NOTHING;

INSERT INTO schema_migrations (version) VALUES ('006_cos_settings')
ON CONFLICT (version) DO NOTHING;
