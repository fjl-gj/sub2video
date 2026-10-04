#!/bin/bash
set -euo pipefail

archive=${SUB2VIDEO_FRONTEND_ARCHIVE:-/tmp/sub2video-frontend-dist.tgz}
root=${SUB2VIDEO_FRONTEND_DIR:-/opt/sub2video/frontend}
expected_sha=${1:-${SUB2VIDEO_FRONTEND_EXPECTED_SHA256:-}}
target="$root/dist"

if [[ ! "$expected_sha" =~ ^[[:xdigit:]]{64}$ ]]; then
  printf 'Usage: %s <archive-sha256>\n' "$0" >&2
  exit 2
fi
test -s "$archive"
test -d "$root"
test -d "$target"
expected_sha=${expected_sha,,}
if [ "$(sha256sum "$archive" | cut -d ' ' -f 1)" != "$expected_sha" ]; then
  printf 'Frontend archive SHA-256 mismatch; published files unchanged.\n' >&2
  exit 1
fi

exec 9>"$root/.deploy.lock"
flock -n 9 || { printf 'Another frontend release is in progress.\n' >&2; exit 1; }
release_id="$(date +%Y%m%d%H%M%S)-$$"
staging=$(mktemp -d "$root/.dist-staging.XXXXXX")
backup="$root/.dist-pre-${release_id}"
failed="$root/.dist-failed-${release_id}"
probe="$root/.dist-probe-${release_id}"

cleanup() {
  rm -f -- "$probe"
  if [ -d "$staging" ]; then
    printf 'Unpublished staging retained: %s\n' "$staging" >&2
  fi
}
trap cleanup EXIT

tar -xzf "$archive" -C "$staging"
test -s "$staging/index.html"
index_sha=$(sha256sum "$staging/index.html" | cut -d ' ' -f 1)

# Open SPAs may request lazy chunks from either side of a release or rollback.
# Preserve content-addressed assets without replacing the candidate's files.
if [ -d "$target/assets" ]; then
  mkdir -p "$staging/assets"
  cp -an "$target/assets/." "$staging/assets/"
fi
nginx -t

verify_index() {
  local wanted_sha=$1
  curl -fsS --connect-timeout 2 --max-time 10 http://127.0.0.1:12000/ -o "$probe" &&
    test "$(sha256sum "$probe" | cut -d ' ' -f 1)" = "$wanted_sha"
}

rollback() {
  local previous_sha source_assets retained_candidate
  test -s "$backup/index.html" || { printf 'Frontend backup is missing; manual recovery required.\n' >&2; return 1; }
  previous_sha=$(sha256sum "$backup/index.html" | cut -d ' ' -f 1)
  source_assets="$target/assets"
  retained_candidate="$failed"
  if [ ! -d "$target" ]; then
    source_assets="$staging/assets"
    retained_candidate="$staging"
  fi
  if [ -d "$source_assets" ]; then
    mkdir -p "$backup/assets" || return 1
    cp -an "$source_assets/." "$backup/assets/" || return 1
  fi
  if [ -d "$target" ]; then mv -- "$target" "$failed" || return 1; fi
  mv -- "$backup" "$target" || return 1
  if ! nginx -t || ! systemctl reload nginx || ! verify_index "$previous_sha"; then
    printf 'Previous frontend files restored, but serving them could not be verified.\n' >&2
    return 1
  fi
  printf 'Previous frontend restored, retaining new hashed assets. Candidate copy: %s\n' "$retained_candidate" >&2
}

printf 'Frontend backup: %s\nArchive retained: %s\n' "$backup" "$archive"
mv -- "$target" "$backup"
if ! mv -- "$staging" "$target" || ! nginx -t || ! systemctl reload nginx || ! verify_index "$index_sha"; then
  if ! rollback; then
    printf 'Frontend rollback failed; inspect %s and %s before further action.\n' "$target" "$backup" >&2
  fi
  exit 1
fi

printf 'Frontend activated; index SHA-256: %s\n' "$index_sha"
printf 'Entry document verified; browser and business acceptance remain required.\n'
