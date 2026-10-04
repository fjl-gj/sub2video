#!/bin/bash
set -euo pipefail

# Invoked only from update-107.test.sh through copies named for each command.
test -n "${SUB2VIDEO_DEPLOY_TEST_CASE:-}"
test -d "$SUB2VIDEO_DEPLOY_TEST_CASE"
command_name=${0##*/}
printf '%s %s\n' "$command_name" "$*" >> "$SUB2VIDEO_DEPLOY_TEST_CASE/actions.log"
failure=${SUB2VIDEO_DEPLOY_TEST_FAILURE:-none}

case "$command_name" in
  flock)
    [ "$failure" != locked ]
    ;;
  sleep|nginx)
    exit 0
    ;;
  systemctl)
    if [ "$1" = restart ] && [[ "$failure" = api_restart || "$failure" = api_rollback ]]; then exit 1; fi
    if [ "$1" = start ] && [ "$failure" = api_rollback ]; then exit 1; fi
    if [ "$1" = reload ] && [ "$failure" = frontend_rollback ]; then exit 1; fi
    exit 0
    ;;
  curl)
    output=/dev/stdout
    url=
    while [ "$#" -gt 0 ]; do
      case "$1" in
        -o|--output) output=$2; shift 2 ;;
        --connect-timeout|--max-time) shift 2 ;;
        http://*) url=$1; shift ;;
        *) shift ;;
      esac
    done
    case "$url" in
      http://127.0.0.1:12080/health)
        if [ "$failure" = api_health ] && grep -q new-api "$SUB2VIDEO_BACKEND_DIR/sub2video-server"; then exit 1; fi
        printf '{"status":"ok"}\n' > "$output"
        ;;
      http://127.0.0.1:12000/)
        if [ "$failure" = frontend_probe ] && grep -q new-index "$SUB2VIDEO_FRONTEND_DIR/dist/index.html"; then
          printf 'stale-proxy-response\n' > "$output"
        else
          cp "$SUB2VIDEO_FRONTEND_DIR/dist/index.html" "$output"
        fi
        ;;
      *) printf 'Unexpected network request rejected by mock: %s\n' "$url" >&2; exit 99 ;;
    esac
    ;;
  mv)
    if [ "$failure" = frontend_move ] && [[ "$*" = *".dist-staging."* ]]; then exit 1; fi
    exec "$SUB2VIDEO_DEPLOY_TEST_REAL_MV" "$@"
    ;;
  *) printf 'Unknown mocked command: %s\n' "$command_name" >&2; exit 99 ;;
esac
