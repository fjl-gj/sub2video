#!/bin/bash
set -euo pipefail

# File-operation and failure-path tests only. No real service/network commands.
repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
test_root=${1:-"$repo_root/.tmp_release_20260929_01/deploy"}
mkdir -p "$test_root"
test_root=$(cd -- "$test_root" && pwd)
case "$test_root" in "$repo_root"/.tmp_release_*/deploy) ;; *) printf 'Test root must be a release temporary deploy directory inside this repository.\n' >&2; exit 2 ;; esac
run_dir=$(mktemp -d "$test_root/mock-run.XXXXXX")
mock_bin="$run_dir/bin"
mkdir -p "$mock_bin"
export SUB2VIDEO_DEPLOY_TEST_REAL_MV
SUB2VIDEO_DEPLOY_TEST_REAL_MV=$(command -v mv)
for command_name in systemctl nginx curl flock sleep mv; do
  install -m 755 "$repo_root/deploy/tests/mock-command.sh" "$mock_bin/$command_name"
done
export PATH="$mock_bin:$PATH"

passed=0
case_dir=
begin_case() {
  case_dir="$run_dir/$1"
  mkdir -p "$case_dir"
  export SUB2VIDEO_DEPLOY_TEST_CASE="$case_dir"
  export SUB2VIDEO_DEPLOY_TEST_FAILURE=none
  export SUB2VIDEO_ROLLBACK_COMPATIBLE=false
}
pass() { passed=$((passed + 1)); printf 'PASS %s\n' "${case_dir##*/}"; }
expect_status() {
  local expected=$1 status=0
  shift
  "$@" > "$case_dir/output.log" 2>&1 || status=$?
  if [ "$status" -ne "$expected" ]; then
    printf 'FAIL %s: expected status %s, got %s; see %s\n' "${case_dir##*/}" "$expected" "$status" "$case_dir/output.log" >&2
    exit 1
  fi
}
api_fixture() {
  export SUB2VIDEO_BACKEND_DIR="$case_dir/backend"
  export SUB2VIDEO_API_CANDIDATE="$case_dir/candidate"
  mkdir -p "$SUB2VIDEO_BACKEND_DIR"
  printf 'old-api\n' > "$SUB2VIDEO_BACKEND_DIR/sub2video-server"
  printf 'new-api\n' > "$SUB2VIDEO_API_CANDIDATE"
  api_sha=$(sha256sum "$SUB2VIDEO_API_CANDIDATE" | cut -d ' ' -f 1)
}
frontend_fixture() {
  export SUB2VIDEO_FRONTEND_DIR="$case_dir/frontend"
  export SUB2VIDEO_FRONTEND_ARCHIVE="$case_dir/dist.tgz"
  mkdir -p "$SUB2VIDEO_FRONTEND_DIR/dist/assets" "$case_dir/new/assets"
  printf 'old-index\n' > "$SUB2VIDEO_FRONTEND_DIR/dist/index.html"
  printf 'old-chunk\n' > "$SUB2VIDEO_FRONTEND_DIR/dist/assets/old-HASH.js"
  printf 'new-index\n' > "$case_dir/new/index.html"
  printf 'new-chunk\n' > "$case_dir/new/assets/new-HASH.js"
  tar -czf "$SUB2VIDEO_FRONTEND_ARCHIVE" -C "$case_dir/new" .
  frontend_sha=$(sha256sum "$SUB2VIDEO_FRONTEND_ARCHIVE" | cut -d ' ' -f 1)
}

begin_case api-success
api_fixture
expect_status 0 bash "$repo_root/deploy/update-107-api.sh" "$api_sha"
grep -q new-api "$SUB2VIDEO_BACKEND_DIR/sub2video-server"
grep -q old-api "$SUB2VIDEO_BACKEND_DIR"/sub2video-server.pre-*
test -s "$SUB2VIDEO_API_CANDIDATE"
grep -q 'Only process liveness' "$case_dir/output.log"
pass

begin_case api-sha-rejected
api_fixture
expect_status 1 bash "$repo_root/deploy/update-107-api.sh" "$(printf '%064d' 0)"
grep -q old-api "$SUB2VIDEO_BACKEND_DIR/sub2video-server"
test ! -e "$case_dir/actions.log"
pass

begin_case api-lock-rejected
api_fixture
export SUB2VIDEO_DEPLOY_TEST_FAILURE=locked
expect_status 1 bash "$repo_root/deploy/update-107-api.sh" "$api_sha"
grep -q old-api "$SUB2VIDEO_BACKEND_DIR/sub2video-server"
! grep -q systemctl "$case_dir/actions.log"
pass

begin_case api-health-rollback
api_fixture
export SUB2VIDEO_DEPLOY_TEST_FAILURE=api_health SUB2VIDEO_ROLLBACK_COMPATIBLE=true
expect_status 1 bash "$repo_root/deploy/update-107-api.sh" "$api_sha"
grep -q old-api "$SUB2VIDEO_BACKEND_DIR/sub2video-server"
grep -q 'Previous API restored' "$case_dir/output.log"
test -s "$SUB2VIDEO_API_CANDIDATE"
pass

begin_case api-unsafe-rollback-stops
api_fixture
export SUB2VIDEO_DEPLOY_TEST_FAILURE=api_restart
expect_status 1 bash "$repo_root/deploy/update-107-api.sh" "$api_sha"
grep -q new-api "$SUB2VIDEO_BACKEND_DIR/sub2video-server"
grep -q 'systemctl stop ' "$case_dir/actions.log"
! grep -q 'systemctl start ' "$case_dir/actions.log"
grep -q 'Service stopped; automatic rollback' "$case_dir/output.log"
pass

begin_case api-rollback-failure-visible
api_fixture
export SUB2VIDEO_DEPLOY_TEST_FAILURE=api_rollback SUB2VIDEO_ROLLBACK_COMPATIBLE=true
expect_status 1 bash "$repo_root/deploy/update-107-api.sh" "$api_sha"
grep -q old-api "$SUB2VIDEO_BACKEND_DIR/sub2video-server"
grep -q 'rollback could not be verified' "$case_dir/output.log"
pass

begin_case frontend-success
frontend_fixture
expect_status 0 bash "$repo_root/deploy/update-107-frontend.sh" "$frontend_sha"
grep -q new-index "$SUB2VIDEO_FRONTEND_DIR/dist/index.html"
test -s "$SUB2VIDEO_FRONTEND_DIR/dist/assets/old-HASH.js"
test -s "$SUB2VIDEO_FRONTEND_DIR/dist/assets/new-HASH.js"
test -s "$SUB2VIDEO_FRONTEND_ARCHIVE"
pass

begin_case frontend-sha-rejected
frontend_fixture
expect_status 1 bash "$repo_root/deploy/update-107-frontend.sh" "$(printf '%064d' 0)"
grep -q old-index "$SUB2VIDEO_FRONTEND_DIR/dist/index.html"
test ! -e "$case_dir/actions.log"
pass

begin_case frontend-stale-response-rollback
frontend_fixture
export SUB2VIDEO_DEPLOY_TEST_FAILURE=frontend_probe
expect_status 1 bash "$repo_root/deploy/update-107-frontend.sh" "$frontend_sha"
grep -q old-index "$SUB2VIDEO_FRONTEND_DIR/dist/index.html"
test -s "$SUB2VIDEO_FRONTEND_DIR/dist/assets/old-HASH.js"
test -s "$SUB2VIDEO_FRONTEND_DIR/dist/assets/new-HASH.js"
grep -q 'Previous frontend restored' "$case_dir/output.log"
pass

begin_case frontend-switch-failure-rollback
frontend_fixture
export SUB2VIDEO_DEPLOY_TEST_FAILURE=frontend_move
expect_status 1 bash "$repo_root/deploy/update-107-frontend.sh" "$frontend_sha"
grep -q old-index "$SUB2VIDEO_FRONTEND_DIR/dist/index.html"
test -s "$SUB2VIDEO_FRONTEND_DIR/dist/assets/new-HASH.js"
grep -q 'Previous frontend restored' "$case_dir/output.log"
pass

begin_case frontend-rollback-failure-visible
frontend_fixture
export SUB2VIDEO_DEPLOY_TEST_FAILURE=frontend_rollback
expect_status 1 bash "$repo_root/deploy/update-107-frontend.sh" "$frontend_sha"
grep -q old-index "$SUB2VIDEO_FRONTEND_DIR/dist/index.html"
test -s "$SUB2VIDEO_FRONTEND_DIR/dist/assets/new-HASH.js"
grep -q 'Frontend rollback failed' "$case_dir/output.log"
pass

printf '%s deploy script tests passed. Evidence: %s\n' "$passed" "$run_dir"
