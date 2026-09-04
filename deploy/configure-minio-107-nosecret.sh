#!/bin/bash
set -euo pipefail
# Apply non-secret MinIO settings. Secret Key must be set via Admin UI.
docker exec -i sub2video-postgres psql -U sub2video -d sub2video <<'SQL'
INSERT INTO settings (key, value) VALUES
  ('storage_provider', 'minio'),
  ('minio_endpoint', '192.168.0.103:9000'),
  ('minio_access_key', 'api_user'),
  ('minio_bucket', 'sub2video'),
  ('minio_use_ssl', 'false'),
  ('minio_public_base', 'http://192.168.0.103:9000/sub2video')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();
SELECT key, value FROM settings
WHERE key IN ('storage_provider','minio_endpoint','minio_access_key','minio_bucket','minio_use_ssl','minio_public_base')
ORDER BY 1;
SQL
systemctl restart sub2video-backend
sleep 2
systemctl is-active sub2video-backend
journalctl -u sub2video-backend -n 8 --no-pager
