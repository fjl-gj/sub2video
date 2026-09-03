-- platform settings (SMTP / auth switches) + email_verified helpers
CREATE TABLE IF NOT EXISTS settings (
  key         VARCHAR(128) PRIMARY KEY,
  value       TEXT NOT NULL DEFAULT '',
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO settings (key, value) VALUES
  ('smtp_host', ''),
  ('smtp_port', '587'),
  ('smtp_username', ''),
  ('smtp_password', ''),
  ('smtp_from', ''),
  ('smtp_from_name', 'Sub2Video'),
  ('smtp_use_tls', 'true'),
  ('email_verify_enabled', 'false'),
  ('password_reset_enabled', 'false'),
  ('registration_enabled', 'true'),
  ('frontend_url', 'http://192.168.0.107:12000'),
  ('input_retention_days', '7'),
  ('result_retention_days', '30'),
  ('notify_task', 'true'),
  ('audit_enabled', 'true')
ON CONFLICT (key) DO NOTHING;

INSERT INTO schema_migrations (version) VALUES ('004_settings')
ON CONFLICT (version) DO NOTHING;
