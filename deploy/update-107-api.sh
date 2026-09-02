#!/bin/bash
set -euo pipefail
systemctl stop sub2video-backend || true
install -m 755 /tmp/sub2video-server /opt/sub2video/backend/sub2video-server
systemctl start sub2video-backend
sleep 2
systemctl is-active sub2video-backend
curl -s http://127.0.0.1:12080/health
echo
RESP=$(curl -s -X POST http://127.0.0.1:12080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"demo@sub2video.local","password":"Demo1234!"}')
TOKEN=$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["token"]["accessToken"])' "$RESP")
curl -s -X POST http://127.0.0.1:12080/api/v1/client/tasks/quote -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"workflowId":"wf-h3-text","durationSeconds":15,"resolution":"768P"}'
echo
curl -s -X POST http://127.0.0.1:12080/api/v1/client/tasks -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"workflowId":"wf-h3-text","title":"API验证任务","prompt":"雨夜霓虹","durationSeconds":15,"resolution":"768P","priceSnapshot":{"ruleId":"pr-1","version":"v1.2","pricePerSecond":0.5,"estimatedCost":7.5}}'
echo
