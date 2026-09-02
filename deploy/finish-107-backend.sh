#!/bin/bash
set -euo pipefail

docker exec sub2video-postgres psql -U sub2video -d sub2video -c \
  "UPDATE workspaces SET name = '李维工作室' WHERE user_id = (SELECT id FROM users WHERE email = 'demo@sub2video.local');"

docker exec sub2video-postgres psql -U sub2video -d sub2video -c \
  "UPDATE wallets SET balance = 256 WHERE user_id = (SELECT id FROM users WHERE email = 'demo@sub2video.local');"

ln -sf /etc/nginx/sites-available/sub2video-uat.conf /etc/nginx/conf.d/sub2video-uat.conf
nginx -t
systemctl reload nginx
systemctl daemon-reload
systemctl restart sub2video-backend
sleep 2
systemctl is-active sub2video-backend
echo "--- health ---"
curl -s http://127.0.0.1:12080/health
echo
echo "--- login ---"
curl -s -X POST http://127.0.0.1:12080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@sub2video.local","password":"Demo1234!"}'
echo
echo "--- nginx health ---"
curl -s http://127.0.0.1:12000/health
echo
