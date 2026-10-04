#!/bin/bash
set -euo pipefail

# The release runbook must verify migrations and bootstrap behavior before use.
# This script never migrates a database or changes the service environment.
# Set SUB2VIDEO_ROLLBACK_COMPATIBLE=true only after reviewing the old binary's
# schema compatibility and confirming that its startup cannot replay seeds.
candidate=${SUB2VIDEO_API_CANDIDATE:-/tmp/sub2video-server}
backend_dir=${SUB2VIDEO_BACKEND_DIR:-/opt/sub2video/backend}
target="$backend_dir/sub2video-server"
expected_sha=${1:-${SUB2VIDEO_API_EXPECTED_SHA256:-}}
rollback_compatible=${SUB2VIDEO_ROLLBACK_COMPATIBLE:-false}

if [[ ! "$expected_sha" =~ ^[[:xdigit:]]{64}$ ]]; then
  printf 'Usage: %s <candidate-sha256>\n' "$0" >&2
  exit 2
fi
case "$rollback_compatible" in true|false) ;; *) printf 'Invalid rollback compatibility flag.\n' >&2; exit 2 ;; esac
test -d "$backend_dir"
test -s "$candidate"
test -s "$target"
expected_sha=${expected_sha,,}
actual_sha=$(sha256sum "$candidate" | cut -d ' ' -f 1)
if [ "$actual_sha" != "$expected_sha" ]; then
  printf 'Candidate SHA-256 mismatch; service unchanged.\n' >&2
  exit 1
fi

exec 9>"$backend_dir/.deploy.lock"
flock -n 9 || { printf 'Another API release is in progress.\n' >&2; exit 1; }
release_id="$(date +%Y%m%d%H%M%S)-$$"
backup="${target}.pre-${release_id}"
staging="${target}.candidate-${release_id}"
cleanup() { rm -f -- "$staging"; }
trap cleanup EXIT

# Both copies are complete before the atomic same-directory replacement.
install -m 755 "$target" "$backup"
install -m 755 "$candidate" "$staging"
test "$(sha256sum "$staging" | cut -d ' ' -f 1)" = "$expected_sha"
printf 'API backup: %s\nCandidate retained: %s\n' "$backup" "$candidate"

wait_for_health() {
  local attempt
  for attempt in {1..15}; do
    if systemctl is-active --quiet sub2video-backend &&
       curl -fsS --connect-timeout 2 --max-time 3 http://127.0.0.1:12080/health >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

recover_failure() {
  printf 'API activation failed. The database has not been rolled back.\n' >&2
  if ! systemctl stop sub2video-backend; then
    printf 'Could not stop the failed service; manual recovery required. Backup: %s\n' "$backup" >&2
    return 1
  fi
  if [ "$rollback_compatible" != true ]; then
    printf 'Service stopped; automatic rollback was not authorized as schema/bootstrap compatible. Backup: %s\n' "$backup" >&2
    return 1
  fi
  if ! install -m 755 "$backup" "$staging" || ! mv -f -- "$staging" "$target" ||
     ! systemctl start sub2video-backend || ! wait_for_health; then
    if ! systemctl stop sub2video-backend; then
      printf 'Could not stop the unverified rollback service.\n' >&2
    fi
    printf 'API rollback could not be verified; manual recovery is required. Backup: %s\n' "$backup" >&2
    return 1
  fi
  printf 'Previous API restored and responding. Business acceptance remains required.\n' >&2
}

if ! mv -f -- "$staging" "$target"; then
  printf 'API file switch failed; previous service was not restarted. Backup: %s\n' "$backup" >&2
  exit 1
fi
if ! systemctl restart sub2video-backend || ! wait_for_health; then
  recover_failure || true
  exit 1
fi

printf 'API activated; SHA-256: %s\n' "$expected_sha"
printf 'Only process liveness was verified; run the release business checks before accepting this release.\n'
