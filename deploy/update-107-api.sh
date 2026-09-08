#!/bin/bash
set -euo pipefail

candidate=/tmp/sub2video-server
target=/opt/sub2video/backend/sub2video-server
backup="${target}.pre-$(date +%Y%m%d%H%M%S)"

test -s "$candidate"
test -s "$target"
install -m 755 "$target" "$backup"

rollback() {
  systemctl stop sub2video-backend || true
  install -m 755 "$backup" "$target"
  systemctl start sub2video-backend || true
}

wait_for_health() {
  local attempt
  for attempt in {1..15}; do
    if curl -fs http://127.0.0.1:12080/health >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

if ! install -m 755 "$candidate" "$target" || ! systemctl restart sub2video-backend || ! wait_for_health; then
  rollback
  exit 1
fi

rm -f "$candidate"
systemctl is-active --quiet sub2video-backend
