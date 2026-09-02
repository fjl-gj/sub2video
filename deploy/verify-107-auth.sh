#!/bin/bash
set -euo pipefail
RESP=$(curl -s -X POST http://127.0.0.1:12000/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@sub2video.local","password":"Demo1234!"}')
TOKEN=$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["token"]["accessToken"])' "$RESP")
echo "login ok"
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:12000/api/v1/client/overview
echo
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:12000/api/v1/auth/me
echo
