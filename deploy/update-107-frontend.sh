#!/bin/bash
set -euo pipefail

archive=/tmp/sub2video-frontend-dist.tgz
root=/opt/sub2video/frontend
target="$root/dist"
staging="$root/.dist-staging-$$"
backup="$root/.dist-pre-$(date +%Y%m%d%H%M%S)"

test -s "$archive"
test -d "$root"
test -d "$target"

cleanup() {
  rm -rf "$staging"
}
trap cleanup EXIT

mkdir -p "$staging"
tar -xzf "$archive" -C "$staging"
test -s "$staging/index.html"

rollback() {
  if [ -d "$backup" ]; then
    rm -rf "$target"
    mv "$backup" "$target"
    nginx -t && systemctl reload nginx || true
  fi
}

mv "$target" "$backup"
if ! mv "$staging" "$target" || ! nginx -t || ! systemctl reload nginx || ! curl -fsS http://127.0.0.1:12000/ >/dev/null; then
  rollback
  exit 1
fi

rm -f "$archive"
trap - EXIT
