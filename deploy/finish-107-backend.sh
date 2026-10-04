#!/bin/bash
set -euo pipefail

# Kept as a refusal entry point for old operator commands and bookmarks.
# The historical script changed account/wallet data and exposed login responses.
printf '%s\n' \
  'This deployment script is retired and performs no operations.' \
  'Use deploy/update-107-api.sh and deploy/update-107-frontend.sh only after candidate verification, database backup and migration review.' >&2
exit 2
