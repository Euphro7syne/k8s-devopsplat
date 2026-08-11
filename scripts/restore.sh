#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${DB_PATH:-data/ops-platform.db}"

if [[ -z "${BACKUP_FILE:-}" ]]; then
  echo "BACKUP_FILE is required" >&2
  exit 1
fi

mkdir -p "$(dirname "${DB_PATH}")"
sqlite3 "${DB_PATH}" ".restore '${BACKUP_FILE}'"
